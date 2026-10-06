package tests

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	exchangeapi "github.com/kiberdruzhinnik/go-exchange-api/api"
	"github.com/kiberdruzhinnik/go-exchange-api/constants"
	custom_errors "github.com/kiberdruzhinnik/go-exchange-api/errors"
)

func TestSpbexHistoryAndCurrentPrice(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/reader/marketdata/charts/chistory" || r.URL.Query().Get("symbol") != "SPBE&OTHER" || r.URL.Query().Get("resolution") != "1440" || r.URL.Query().Get("from") != "0" || r.URL.Query().Get("to") == "" {
			t.Errorf("unexpected request: %s", r.URL)
		}
		if got := r.UserAgent(); got != constants.BrowserUserAgent {
			t.Errorf("User-Agent = %q, want %q", got, constants.BrowserUserAgent)
		}
		w.Header().Set("Content-Type", "application/json")
		// Deliberately unsorted: the latest quote must be chosen by its timestamp.
		w.Write([]byte(`[{"bar_unixtime":1704240000,"close":101,"high":103,"low":100},{"bar_unixtime":1704153600,"close":100,"high":102,"low":99}]`))
	}))
	defer server.Close()
	api := exchangeapi.SpbexAPI{BaseURL: server.URL}
	history, err := api.GetTicker("spbe&other")
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 2 || history[0].Close != 100 || history[1].Close != 101 || history[1].High != 103 || history[1].Low != 100 || history[1].Volume != 0 || history[1].Facevalue != 1 || history[1].Date.Location() != time.UTC {
		t.Fatalf("unexpected history: %+v", history)
	}
	current, err := api.GetCurrentPrice("spbe&other")
	if err != nil || current != history[1] {
		t.Fatalf("current = %+v, err = %v", current, err)
	}
}

func TestSpbexRejectsBadResponses(t *testing.T) {
	cases := []struct {
		name, body string
		status     int
		notFound   bool
	}{
		{"unknown ticker", `[]`, 200, true},
		{"null response", `null`, 200, true},
		{"upstream error", `[]`, 503, false},
		{"API error", `{"status":400,"message":"incorrect data type"}`, 200, false},
		{"invalid JSON", `<html>error</html>`, 200, false},
		{"missing price", `[{"bar_unixtime":1704153600,"high":102,"low":99}]`, 200, false},
		{"missing date", `[{"close":100,"high":102,"low":99}]`, 200, false},
		{"invalid range", `[{"bar_unixtime":1704153600,"close":100,"high":99,"low":102}]`, 200, false},
		{"duplicate date", `[{"bar_unixtime":1704153600,"close":100,"high":102,"low":99},{"bar_unixtime":1704153600,"close":100,"high":102,"low":99}]`, 200, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(tc.status); w.Write([]byte(tc.body)) }))
			defer server.Close()
			api := exchangeapi.SpbexAPI{BaseURL: server.URL}
			_, err := api.GetTicker("SPBE")
			if err == nil {
				t.Fatal("expected error")
			}
			if errors.Is(err, custom_errors.ErrorNotFound) != tc.notFound {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
