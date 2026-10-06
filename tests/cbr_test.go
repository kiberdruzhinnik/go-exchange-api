package tests

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/kiberdruzhinnik/go-exchange-api/api"
	customerrors "github.com/kiberdruzhinnik/go-exchange-api/errors"
)

func TestCbrGetTickerParsesCurrencyHistory(t *testing.T) {
	useTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host != "www.cbr.ru" || !strings.Contains(req.URL.Path, "/scripts/XML_dynamic.asp") {
			t.Errorf("unexpected CBR request: %s", req.URL)
		}
		if got := req.URL.Query().Get("VAL_NM_RQ"); got != "R01235" {
			t.Errorf("VAL_NM_RQ = %q, want %q", got, "R01235")
		}
		body := `<ValCurs Date="06.02.2024" name="Foreign Currency Market"><Record Date="05.02.2024"><Nominal>1</Nominal><Value>91,50</Value><VunitRate>91,5000</VunitRate></Record></ValCurs>`
		return response(req, http.StatusOK, body), nil
	}))

	cbr := api.NewCbrAPI()
	history, err := cbr.GetTicker("usd")
	if err != nil {
		t.Fatalf("GetTicker() error = %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("GetTicker() returned %d entries, want 1", len(history))
	}
	wantDate := time.Date(2024, time.February, 5, 0, 0, 0, 0, time.UTC)
	if !history[0].Date.Equal(wantDate) || history[0].Close != 91.5 || history[0].Facevalue != 1 {
		t.Errorf("currency history entry = %+v, unexpected values", history[0])
	}
}

func TestCbrGetTickerRejectsUnknownCurrency(t *testing.T) {
	useTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Errorf("unexpected upstream request for unsupported currency: %s", req.URL)
		return response(req, http.StatusInternalServerError, ""), nil
	}))

	cbr := api.NewCbrAPI()
	if _, err := cbr.GetTicker("gbp"); err != customerrors.ErrorNotFound {
		t.Fatalf("GetTicker() error = %v, want %v", err, customerrors.ErrorNotFound)
	}
}

func TestCbrGetTickerRejectsDisallowedURL(t *testing.T) {
	useTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Errorf("unexpected request to disallowed URL: %s", req.URL)
		return response(req, http.StatusOK, ""), nil
	}))
	cbr := api.NewCbrAPI()
	cbr.BaseURL = "https://example.com"
	if _, err := cbr.GetTicker("usd"); err != customerrors.ErrorNotAllowed {
		t.Fatalf("GetTicker() error = %v, want %v", err, customerrors.ErrorNotAllowed)
	}
}
