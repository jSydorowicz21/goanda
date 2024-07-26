package goanda

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"time"
)

const (
	apiUserAgent = "v20-golang/0.0.1"
	httpTimeout  = time.Second * 5
)

// ConnectionConfig is used to configure new connections
// Defaults;
//
//	UserAgent	= v20-golang/0.0.1
//	Timeout		= 5 seconds
//	Live		= False
type ConnectionConfig struct {
	UserAgent string
	Timeout   time.Duration
	Live      bool
}

// Connection describes a connection to the Oanda v20 API
// It is thread safe
type Connection struct {
	hostname   string
	accountID  string
	authHeader string
	userAgent  string
	client     *http.Client
}

// NewConnection creates a new connection
// This function calls Connection.CheckConnection(), returning any errors
// Supplying a config is optional, with sane defaults (paper trading) being used otherwise.
func NewConnection(accountID string, token string, config *ConnectionConfig) (*Connection, error) {
	// Make new connection with defaults
	nc := &Connection{
		hostname:   "https://api-fxpractice.oanda.com/v3",
		accountID:  accountID,
		authHeader: "Bearer " + token,
		userAgent:  apiUserAgent,
		client: &http.Client{
			Timeout: httpTimeout,
			Transport: &loggingTransport{
				transport: http.DefaultTransport,
			},
		},
	}

	// Overwrite things if we've been given configuration for them
	if config != nil {
		if config.Live {
			nc.hostname = "https://api-fxtrade.oanda.com/v3"
		}

		if config.Timeout != 0 {
			nc.client.Timeout = config.Timeout
		}

		if config.UserAgent != "" {
			nc.userAgent = config.UserAgent
		}
	}

	return nc, nc.CheckConnection()
}

// CheckConnection performs a request, returning any errors encountered
func (c *Connection) CheckConnection() error {
	_, err := c.Get("/accounts/" + c.accountID)
	return err
}

// Get performs a generic http get on the api
func (c *Connection) Get(endpoint string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, c.hostname+endpoint, nil)
	if err != nil {
		return nil, err
	}

	return c.makeRequest(endpoint, req)
}

// Post performs a generic http post on the api
func (c *Connection) Post(endpoint string, data []byte) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, c.hostname+endpoint, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	return c.makeRequest(endpoint, req)
}

// Put performs a generic http put on the api
func (c *Connection) Put(endpoint string, data []byte) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPut, c.hostname+endpoint, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	return c.makeRequest(endpoint, req)
}

func (c *Connection) getAndUnmarshal(endpoint string, receive interface{}) error {
	response, err := c.Get(endpoint)
	if err != nil {
		return err
	}

	return json.Unmarshal(response, receive)
}

func (c *Connection) postAndUnmarshal(endpoint string, send interface{}, receive interface{}) error {
	data, err := json.Marshal(send)
	if err != nil {
		return err
	}

	response, err := c.Post(endpoint, data)
	if err != nil {
		return err
	}

	return json.Unmarshal(response, receive)
}

func (c *Connection) putAndUnmarshal(endpoint string, send interface{}, receive interface{}) error {
	data, err := json.Marshal(send)
	if err != nil {
		return err
	}

	response, err := c.Put(endpoint, data)
	if err != nil {
		return err
	}

	return json.Unmarshal(response, receive)
}

func (c *Connection) makeRequest(endpoint string, req *http.Request) ([]byte, error) {
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Authorization", c.authHeader)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode >= 400 {
		return nil, newAPIError(req, res)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// loggingTransport for logging requests and responses
type loggingTransport struct {
	transport http.RoundTripper
}

func (t *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Log request
	reqDump, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		log.Println("Request dump error:", err)
		return nil, err
	}
	log.Printf("Request:\n%s\n", reqDump)

	// Perform request
	resp, err := t.transport.RoundTrip(req)
	if err != nil {
		log.Println("RoundTrip error:", err)
		return nil, err
	}

	// Log response
	respDump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		log.Println("Response dump error:", err)
		return nil, err
	}
	log.Printf("Response:\n%s\n", respDump)

	return resp, nil
}
