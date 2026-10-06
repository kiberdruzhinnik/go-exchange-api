package tests

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/kiberdruzhinnik/go-exchange-api/api"
	"github.com/kiberdruzhinnik/go-exchange-api/utils"
)

func TestMoexGetTickerParsesHistoryAndCurrentQuote(t *testing.T) {
	useTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case strings.HasSuffix(req.URL.Path, "/iss/securities/sber.json"):
			return response(req, http.StatusOK, `{"boards":{"columns":["boardid","market","engine","is_primary"],"data":[["TQBR","shares","stock",1]]}}`), nil
		case strings.Contains(req.URL.Path, "/iss/history/engines/stock/markets/shares/boards/TQBR/securities/sber.json"):
			return response(req, http.StatusOK, `{"history":{"columns":["TRADEDATE","CLOSE","HIGH","LOW","VOLUME","FACEVALUE"],"data":[["2024-01-02",100.5,102,99,1000,1]]}}`), nil
		case strings.Contains(req.URL.Path, "/iss/engines/stock/markets/shares/securities/sber.json"):
			return response(req, http.StatusOK, `{"marketdata":{"columns":["BOARDID","LAST","HIGH","LOW","VOLTODAY"],"data":[["TQBR",101,103,100,1500]]}}`), nil
		default:
			t.Errorf("unexpected MOEX request: %s", req.URL)
			return response(req, http.StatusNotFound, `{}`), nil
		}
	}))

	moex := api.NewMoexAPI(utils.RedisClient{})
	history, err := moex.GetTicker("sber")
	if err != nil {
		t.Fatalf("GetTicker() error = %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("GetTicker() returned %d entries, want 2", len(history))
	}

	wantHistoryDate := time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC)
	if !history[0].Date.Equal(wantHistoryDate) || history[0].Close != 100.5 || history[0].High != 102 || history[0].Low != 99 || history[0].Volume != 1000 || history[0].Facevalue != 1 {
		t.Errorf("historical entry = %+v, unexpected values", history[0])
	}
	if history[1].Close != 101 || history[1].High != 103 || history[1].Low != 100 || history[1].Volume != 1500 || history[1].Facevalue != 1 {
		t.Errorf("current quote = %+v, unexpected values", history[1])
	}
	if history[1].Date.IsZero() {
		t.Error("current quote date is zero")
	}
}
