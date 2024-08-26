package goanda

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewStreamingConnection(t *testing.T) {
	defer logTestResult(t, "TestNewStreamingConnection")
	conn := &Connection{
		hostname: "https://api-fxpractice.oanda.com/v3",
	}
	sc := NewStreamingConnection(conn)

	if sc.streamURL != "https://stream-fxpractice.oanda.com/v3" {
		t.Errorf("Expected streamURL to be https://stream-fxpractice.oanda.com/v3, got %s", sc.streamURL)
	}

	conn.hostname = "https://api-fxtrade.oanda.com/v3"
	sc = NewStreamingConnection(conn)

	if sc.streamURL != "https://stream-fxtrade.oanda.com/v3" {
		t.Errorf("Expected streamURL to be https://stream-fxtrade.oanda.com/v3, got %s", sc.streamURL)
	}
}

func TestStreamPrices(t *testing.T) {
	defer logTestResult(t, "TestStreamPrices")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the request is properly formed
		if r.URL.Path != "/accounts/test-account/pricing/stream" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}

		instruments := r.URL.Query().Get("instruments")
		if instruments != "EUR_USD" {
			t.Errorf("Unexpected instruments: %s", instruments)
			http.Error(w, "Invalid instruments", http.StatusBadRequest)
			return
		}

		response := PricingStreamResponse{
			Type:       "PRICE",
			Time:       time.Now().Format(time.RFC3339),
			Instrument: "EUR_USD",
			Bids: []struct {
				Price     string `json:"price"`
				Liquidity int    `json:"liquidity"`
			}{{Price: "1.1000", Liquidity: 1000000}},
			Asks: []struct {
				Price     string `json:"price"`
				Liquidity int    `json:"liquidity"`
			}{{Price: "1.1001", Liquidity: 1000000}},
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	conn := &Connection{
		hostname:   server.URL,
		accountID:  "test-account",
		authHeader: "Bearer test-token",
		client:     *server.Client(),
	}
	sc := NewStreamingConnection(conn)

	// Override the streamURL to use the test server
	sc.streamURL = server.URL

	instruments := []string{"EUR_USD"}
	err := sc.StreamPrices(instruments, func(response PricingStreamResponse) {
		if response.Type != "PRICE" {
			t.Errorf("Expected response type to be PRICE, got %s", response.Type)
		}
		if response.Instrument != "EUR_USD" {
			t.Errorf("Expected instrument to be EUR_USD, got %s", response.Instrument)
		}
		if len(response.Bids) == 0 {
			t.Errorf("Expected non-empty Bids slice")
		} else if response.Bids[0].Price != "1.1000" {
			t.Errorf("Expected bid price to be 1.1000, got %s", response.Bids[0].Price)
		}
		if len(response.Asks) == 0 {
			t.Errorf("Expected non-empty Asks slice")
		} else if response.Asks[0].Price != "1.1001" {
			t.Errorf("Expected ask price to be 1.1001, got %s", response.Asks[0].Price)
		}
	})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestStreamTransactions(t *testing.T) {
	defer logTestResult(t, "TestStreamTransactions")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/accounts/test-account/transactions/stream" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}

		response := TransactionStreamResponse{
			Type:          "TRANSACTION",
			Time:          time.Now().Format(time.RFC3339),
			TransactionID: "1234",
			AccountID:     "test-account",
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	conn := &Connection{
		hostname:   server.URL,
		accountID:  "test-account",
		authHeader: "Bearer test-token",
		client:     *server.Client(),
	}
	sc := NewStreamingConnection(conn)

	// Override the streamURL to use the test server
	sc.streamURL = server.URL

	err := sc.StreamTransactions(func(response TransactionStreamResponse) {
		if response.Type != "TRANSACTION" {
			t.Errorf("Expected response type to be TRANSACTION, got %s", response.Type)
		}
		if response.AccountID != "test-account" {
			t.Errorf("Expected account ID to be test-account, got %s", response.AccountID)
		}
		if response.TransactionID != "1234" {
			t.Errorf("Expected transaction ID to be 1234, got %s", response.TransactionID)
		}
	})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestStreamAccountChanges(t *testing.T) {
	defer logTestResult(t, "TestStreamAccountChanges")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/accounts/test-account/changes/stream" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}

		response := AccountChangesStreamResponse{
			Type:              "ACCOUNT_CHANGES",
			Time:              time.Now().Format(time.RFC3339),
			LastTransactionID: "5678",
			Changes:           json.RawMessage(`{"orders":[],"trades":[]}`),
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	conn := &Connection{
		hostname:   server.URL,
		accountID:  "test-account",
		authHeader: "Bearer test-token",
		client:     *server.Client(),
	}
	sc := NewStreamingConnection(conn)

	// Override the streamURL to use the test server
	sc.streamURL = server.URL

	err := sc.StreamAccountChanges(func(response AccountChangesStreamResponse) {
		if response.Type != "ACCOUNT_CHANGES" {
			t.Errorf("Expected response type to be ACCOUNT_CHANGES, got %s", response.Type)
		}
		if response.LastTransactionID != "5678" {
			t.Errorf("Expected last transaction ID to be 5678, got %s", response.LastTransactionID)
		}
	})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestStreamCandles(t *testing.T) {
	defer logTestResult(t, "TestStreamCandles")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/accounts/test-account/instruments/EUR_USD/candles/stream" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}

		if r.URL.Query().Get("granularity") != "M1" {
			t.Errorf("Unexpected granularity: %s", r.URL.Query().Get("granularity"))
			http.Error(w, "Invalid granularity", http.StatusBadRequest)
			return
		}

		response := CandlestickStreamResponse{
			Type:        "CANDLESTICK",
			Time:        time.Now().Format(time.RFC3339),
			Instrument:  "EUR_USD",
			Granularity: "M1",
			Candles: []struct {
				Time     string `json:"time"`
				Bid      Candle `json:"bid,omitempty"`
				Ask      Candle `json:"ask,omitempty"`
				Mid      Candle `json:"mid,omitempty"`
				Volume   int    `json:"volume"`
				Complete bool   `json:"complete"`
			}{
				{
					Time: time.Now().Format(time.RFC3339),
					Mid: Candle{
						Open: 1.1000,
						High: 1.1001,
						Low:  1.0999,
						Close: 1.1000,
					},
					Volume:   100,
					Complete: true,
				},
			},
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	conn := &Connection{
		hostname:   server.URL,
		accountID:  "test-account",
		authHeader: "Bearer test-token",
		client:     *server.Client(),
	}
	sc := NewStreamingConnection(conn)

	// Override the streamURL to use the test server
	sc.streamURL = server.URL

	err := sc.StreamCandles("EUR_USD", "M1", func(response CandlestickStreamResponse) {
		if response.Type != "CANDLESTICK" {
			t.Errorf("Expected response type to be CANDLESTICK, got %s", response.Type)
		}
		if response.Instrument != "EUR_USD" {
			t.Errorf("Expected instrument to be EUR_USD, got %s", response.Instrument)
		}
		if response.Granularity != "M1" {
			t.Errorf("Expected granularity to be M1, got %s", response.Granularity)
		}
		if len(response.Candles) == 0 {
			t.Errorf("Expected at least one candle, got none")
		} else {
			candle := response.Candles[0]
			if candle.Mid.Open != 1.1000 || candle.Mid.High != 1.1001 || candle.Mid.Low != 1.0999 || candle.Mid.Close != 1.1000 {
				t.Errorf("Unexpected candle data: %+v", candle.Mid)
			}
		}
	})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestStreamHeartbeat(t *testing.T) {
	defer logTestResult(t, "TestStreamHeartbeat")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := HeartbeatResponse{
			Type: "HEARTBEAT",
			Time: time.Now().Format(time.RFC3339),
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	conn := &Connection{
		hostname:   server.URL,
		accountID:  "test-account",
		authHeader: "Bearer test-token",
		client:     *server.Client(),
	}
	sc := NewStreamingConnection(conn)

	// Override the streamURL to use the test server
	sc.streamURL = server.URL

	// This test is a bit tricky because heartbeats are handled internally.
	// We'll use the StreamPrices function, but send a heartbeat instead.
	err := sc.StreamPrices([]string{"EUR_USD"}, func(response PricingStreamResponse) {
		t.Errorf("Unexpected pricing response: %+v", response)
	})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	// If we reach this point without errors, it means the heartbeat was properly handled
}
