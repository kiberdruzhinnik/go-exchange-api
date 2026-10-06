package api

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kiberdruzhinnik/go-exchange-api/constants"
	custom_errors "github.com/kiberdruzhinnik/go-exchange-api/errors"
)

type SpbexAPI struct {
	BaseURL string
}

// spbexCandle is the response from the exchange's public chart data feed.
// Prices are pointers so that missing values cannot silently become zero quotes.
type spbexCandle struct {
	Time  int64    `json:"bar_unixtime"`
	Close *float64 `json:"close"`
	High  *float64 `json:"high"`
	Low   *float64 `json:"low"`
}

func NewSpbexAPI() SpbexAPI {
	return SpbexAPI{BaseURL: constants.SpbexBaseApiURL}
}

// GetTicker returns available daily history, including the latest session's
// candle. Its close is the latest price available from the public chart feed.
func (api *SpbexAPI) GetTicker(ticker string) (HistoryEntries, error) {
	return api.getHistory(ticker, 0, time.Now().Unix())
}

// GetCurrentPrice returns the latest available daily candle, preserving its
// actual date even when the instrument has not traded recently.
func (api *SpbexAPI) GetCurrentPrice(ticker string) (HistoryEntry, error) {
	history, err := api.GetTicker(ticker)
	if err != nil {
		return HistoryEntry{}, err
	}
	return history[len(history)-1], nil
}

func (api *SpbexAPI) getHistory(ticker string, from, to int64) (HistoryEntries, error) {
	ticker = strings.ToUpper(strings.TrimSpace(ticker))
	if ticker == "" {
		return nil, custom_errors.ErrorNotFound
	}
	query := url.Values{
		"symbol": {ticker}, "resolution": {"1440"},
		"from": {strconv.FormatInt(from, 10)}, "to": {strconv.FormatInt(to, 10)},
	}
	endpoint := strings.TrimRight(api.BaseURL, "/") + "/reader/marketdata/charts/chistory?" + query.Encode()
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("SPB Exchange history: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SPB Exchange history: HTTP %d", resp.StatusCode)
	}
	// Limit memory consumption if the upstream returns an unexpected response.
	const maxResponseSize = 16 << 20
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize+1))
	if err != nil {
		return nil, fmt.Errorf("SPB Exchange history: %w", err)
	}
	if len(body) > maxResponseSize {
		return nil, fmt.Errorf("SPB Exchange history response too large")
	}
	var candles []spbexCandle
	if err := json.Unmarshal(body, &candles); err != nil {
		return nil, fmt.Errorf("SPB Exchange history JSON: %w", err)
	}
	if len(candles) == 0 {
		return nil, custom_errors.ErrorNotFound
	}
	entries := make(HistoryEntries, 0, len(candles))
	seen := make(map[int64]bool, len(candles))
	for _, candle := range candles {
		if candle.Time <= 0 || !validSpbexPrice(candle.Close) || !validSpbexPrice(candle.High) || !validSpbexPrice(candle.Low) || *candle.Low > *candle.High || *candle.Close < *candle.Low || *candle.Close > *candle.High {
			return nil, fmt.Errorf("SPB Exchange history: invalid candle at %d", candle.Time)
		}
		if seen[candle.Time] {
			return nil, fmt.Errorf("SPB Exchange history: duplicate candle at %d", candle.Time)
		}
		seen[candle.Time] = true
		entries = append(entries, HistoryEntry{
			Date: time.Unix(candle.Time, 0).UTC(), Close: *candle.Close,
			High: *candle.High, Low: *candle.Low, Facevalue: 1,
		})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Date.Before(entries[j].Date) })
	return entries, nil
}

func validSpbexPrice(price *float64) bool {
	return price != nil && !math.IsNaN(*price) && !math.IsInf(*price, 0) && *price > 0
}
