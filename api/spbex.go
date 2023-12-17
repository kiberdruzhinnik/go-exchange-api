package api

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	custom_errors "github.com/kiberdruzhinnik/go-exchange-api/errors"
	"github.com/kiberdruzhinnik/go-exchange-api/utils"
)

type SpbexAPI struct {
	BaseURL string
}

type TimeRange struct {
	Start uint64
	End   uint64
}

type SpbexSecurityJSON struct {
	Time   []int     `json:"t"`
	Open   []float64 `json:"o"`
	High   []float64 `json:"h"`
	Low    []float64 `json:"l"`
	Close  []float64 `json:"c"`
	Status string    `json:"s"`
}

func NewSpbexAPI() SpbexAPI {
	return SpbexAPI{
		BaseURL: "https://investcab.ru/api",
	}
}

func (api *SpbexAPI) GetTicker(ticker string) (HistoryEntries, error) {

	jsonHistory, err := api.getHistory(ticker)
	if err != nil {
		return nil, err
	}

	historyEntries := make(HistoryEntries, len(jsonHistory.Time))
	for i := 0; i < len(jsonHistory.Time); i++ {

		entry := HistoryEntry{
			Date:      api.parseTime(int64(jsonHistory.Time[i])),
			Close:     jsonHistory.Close[i],
			High:      jsonHistory.High[i],
			Low:       jsonHistory.Low[i],
			Volume:    0,
			Facevalue: 1.0,
		}
		historyEntries[i] = entry
	}

	return historyEntries, nil
}

func (api *SpbexAPI) parseTime(timestamp int64) time.Time {
	return time.Unix(timestamp, 0)
}

func (api *SpbexAPI) getHistory(ticker string) (SpbexSecurityJSON, error) {

	timeRange := api.getTimeRange()
	url := api.getUrl(ticker, "D", timeRange)

	log.Printf("Fetching history data from url %s for %s\n", url, ticker)
	data, err := utils.HttpGet(url)
	if err != nil {
		return SpbexSecurityJSON{}, err
	}

	var rawJson string
	err = json.Unmarshal(data, &rawJson)
	if err != nil {
		return SpbexSecurityJSON{}, err
	}

	var spbexSecurityJson SpbexSecurityJSON
	err = json.Unmarshal([]byte(rawJson), &spbexSecurityJson)
	if err != nil {
		return SpbexSecurityJSON{}, err
	}

	if len(spbexSecurityJson.Time) == 0 {
		return SpbexSecurityJSON{}, custom_errors.ErrorNotFound
	}

	return spbexSecurityJson, nil
}

func (api *SpbexAPI) getTimeRange() TimeRange {
	return TimeRange{
		Start: 0,
		End:   uint64(time.Now().Unix()),
	}
}

func (api *SpbexAPI) getUrl(ticker string, resolution string, timeRange TimeRange) string {
	return fmt.Sprintf(
		"%s/chistory?symbol=%s&resolution=%s&from=%d&to=%d",
		api.BaseURL, ticker, resolution, timeRange.Start, timeRange.End,
	)
}
