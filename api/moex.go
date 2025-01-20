package api

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/kiberdruzhinnik/go-exchange-api/constants"
	custom_errors "github.com/kiberdruzhinnik/go-exchange-api/errors"
	"github.com/kiberdruzhinnik/go-exchange-api/utils"
)

const PAGE_SIZE = 100

type MoexAPI struct {
	BaseURL string
	Redis   utils.RedisClient
}

type MoexSecurityParameters struct {
	Board  string `json:"board"`
	Market string `json:"market"`
	Engine string `json:"engine"`
}

func (params MoexSecurityParameters) MarshalBinary() ([]byte, error) {
	return json.Marshal(params)
}

type MoexSecurityParametersJSON struct {
	Boards struct {
		Columns []string `json:"columns"`
		Data    [][]any  `json:"data"`
	} `json:"boards"`
}

type MoexHistoryJSON struct {
	History struct {
		Columns []string        `json:"columns"`
		Data    [][]interface{} `json:"data"`
	} `json:"history"`
}

type MoexPriceJSON struct {
	Marketdata struct {
		Columns []string `json:"columns"`
		Data    [][]any  `json:"data"`
	} `json:"marketdata"`
}

type MoexCbrfPriceJSON struct {
	Cbrf struct {
		Columns []string `json:"columns"`
		Data    [][]any  `json:"data"`
	} `json:"cbrf"`
	WapRates struct {
		Columns []string `json:"columns"`
		Data    [][]any  `json:"data"`
	} `json:"wap_rates"`
}

func NewMoexAPI(redis utils.RedisClient) MoexAPI {
	return MoexAPI{
		BaseURL: constants.MoexBaseApiURL,
		Redis:   redis,
	}
}

func (api *MoexAPI) getRegularTicker(ticker string) (HistoryEntries, error) {
	security, err := api.getSecurityParameters(ticker)
	if err != nil {
		log.Println(err)
		return HistoryEntries{}, err
	}

	var history HistoryEntries
	offset := uint(0)
	for {
		entryHistory, err := api.getSecurityHistoryOffset(ticker, security, offset)
		if err != nil {
			log.Println(err)
			return HistoryEntries{}, err
		}
		// last entry
		if len(entryHistory) == 0 {
			break
		}
		// last entry but contains data
		if len(entryHistory) != PAGE_SIZE {
			history = append(history, entryHistory...)
			break
		}
		offset += PAGE_SIZE
		history = append(history, entryHistory...)
	}

	currentPrice, err := api.getSecurityCurrentPrice(ticker, security)
	if err == nil {
		history = append(history, currentPrice)
		if len(history) > 1 {
			history[len(history)-1].Facevalue = history[len(history)-2].Facevalue
		}
	}
	if err == custom_errors.ErrorNoData {
		log.Println("No current price data. Returning only history data.")
		err = nil
	}

	return history, err
}

func (api *MoexAPI) getCbrfTicker(ticker string) (HistoryEntries, error) {

	if ticker != "cbrf_usd" && ticker != "cbrf_eur" {
		return HistoryEntries{}, custom_errors.ErrorNotFound
	}

	url := fmt.Sprintf("%s/iss/statistics/engines/currency/markets/selt/rates.json?iss.meta=off&"+
		"cbrf.columns=CBRF_USD_LAST,CBRF_USD_TRADEDATE,CBRF_EUR_LAST,CBRF_EUR_TRADEDATE",
		api.BaseURL)

	log.Printf("Fetching price data from url %s for %s\n", url, ticker)
	data, err := utils.HttpGet(url)
	if err != nil {
		return HistoryEntries{}, err
	}

	var moexCbrfJSON MoexCbrfPriceJSON
	err = json.Unmarshal(data, &moexCbrfJSON)
	if err != nil {
		return HistoryEntries{}, err
	}

	var moexHistoryEntry HistoryEntry
	if ticker == "cbrf_usd" {
		moexHistoryEntry.Close = utils.GetFloat64(moexCbrfJSON.Cbrf.Data[0][0])
		time, err := time.Parse("2006-01-02", moexCbrfJSON.Cbrf.Data[0][1].(string))
		if err != nil {
			return HistoryEntries{}, err
		}
		moexHistoryEntry.Date = time
	} else if ticker == "cbrf_eur" {
		moexHistoryEntry.Close = utils.GetFloat64(moexCbrfJSON.Cbrf.Data[0][2])
		time, err := time.Parse("2006-01-02", moexCbrfJSON.Cbrf.Data[0][3].(string))
		if err != nil {
			return HistoryEntries{}, err
		}
		moexHistoryEntry.Date = time
	}

	return HistoryEntries{moexHistoryEntry}, nil

}

func (api *MoexAPI) GetTicker(ticker string) (HistoryEntries, error) {
	if strings.HasPrefix(ticker, "cbrf_") {
		return api.getCbrfTicker(ticker)
	}
	return api.getRegularTicker(ticker)
}

func (api *MoexAPI) getSecurityParametersFromCache(ticker string) (MoexSecurityParameters, error) {
	log.Printf("Getting security parameters data from cache for %s\n", ticker)
	data, err := api.Redis.Client.Get(api.Redis.Context, ticker).Bytes()
	if err != nil {
		log.Printf("No security parameters data from cache for %s\n", ticker)
		return MoexSecurityParameters{}, err
	}

	log.Printf("Got security parameters from cache for %s\n", ticker)
	var params MoexSecurityParameters
	err = json.Unmarshal(data, &params)
	if err != nil {
		return MoexSecurityParameters{}, err
	}

	return params, nil
}

func (api *MoexAPI) setSecurityParametersToCache(ticker string, params MoexSecurityParameters) error {
	log.Printf("Saving security parameters data to cache for %s\n", ticker)
	return api.Redis.Client.Set(api.Redis.Context, ticker, params, 0).Err()
}

func (api *MoexAPI) getSecurityParameters(ticker string) (MoexSecurityParameters, error) {
	var moexJson MoexSecurityParametersJSON

	url := fmt.Sprintf("%s/iss/securities/%s.json?"+
		"iss.only=boards&iss.meta=off&"+
		"boards.columns=boardid,market,engine,is_primary",
		api.BaseURL, ticker)

	var output MoexSecurityParameters

	if api.Redis.Client != nil {
		output, err := api.getSecurityParametersFromCache(ticker)
		if err == nil {
			return output, nil
		}
	}

	log.Printf("Getting security parameters data from url %s for %s\n", url, ticker)
	data, err := utils.HttpGet(url)
	if err != nil {
		return MoexSecurityParameters{}, custom_errors.ErrorCouldNotFetchData
	}

	err = json.Unmarshal(data, &moexJson)
	if err != nil {
		return MoexSecurityParameters{}, custom_errors.ErrorCouldNotParseJSON
	}

	for _, entry := range moexJson.Boards.Data {
		isPrimary := int(entry[3].(float64))
		if isPrimary == 1 {
			output.Board = entry[0].(string)
			output.Market = entry[1].(string)
			output.Engine = entry[2].(string)
		}
	}

	if output.Board == "" || output.Market == "" || output.Engine == "" {
		return MoexSecurityParameters{}, custom_errors.ErrorNotFound
	}

	if api.Redis.Client != nil {
		err := api.setSecurityParametersToCache(ticker, output)
		if err != nil {
			return MoexSecurityParameters{}, err
		}
	}

	return output, err
}

func (api *MoexAPI) getSecurityHistoryOffsetFromCache(key string) (HistoryEntries, error) {
	log.Printf("Getting history data from cache for %s\n", key)
	data, err := api.Redis.Client.Get(api.Redis.Context, key).Bytes()
	if err != nil {
		log.Printf("Got no history data from cache for %s\n", key)
		return HistoryEntries{}, err
	}

	log.Printf("Got history data from cache for %s\n", key)
	var moexHistory HistoryEntries
	err = json.Unmarshal(data, &moexHistory)
	if err != nil {
		return HistoryEntries{}, err
	}

	return moexHistory, nil
}

func (api *MoexAPI) setSecurityHistoryOffsetToCache(key string,
	value HistoryEntries, duration time.Duration) error {
	log.Printf("Saving history data to cache for %s for %d seconds\n", key, uint64(duration.Seconds()))
	return api.Redis.Client.Set(api.Redis.Context, key, value, duration).Err()
}

func (api *MoexAPI) getSecurityHistoryOffset(ticker string,
	params MoexSecurityParameters,
	offset uint) (HistoryEntries, error) {
	url := fmt.Sprintf("%s/iss/history/engines/%s/markets/%s/boards/%s/"+
		"securities/%s.json?iss.meta=off&start=%d&history.columns=TRADEDATE"+
		",CLOSE,HIGH,LOW,VOLUME,FACEVALUE",
		api.BaseURL, params.Engine, params.Market, params.Board, ticker, offset)

	var moexHistory HistoryEntries
	cacheKey := fmt.Sprintf("%s-%s-%s-%s-%d", params.Board, params.Market, params.Engine, ticker, offset)

	if api.Redis.Client != nil {
		moexHistory, err := api.getSecurityHistoryOffsetFromCache(cacheKey)
		if err == nil {
			return moexHistory, nil
		}
	}

	log.Printf("Fetching history data from url %s for %s\n", url, ticker)
	data, err := utils.HttpGet(url)
	if err != nil {
		return HistoryEntries{}, err
	}

	var moexHistoryJSON MoexHistoryJSON
	err = json.Unmarshal(data, &moexHistoryJSON)
	if err != nil {
		return HistoryEntries{}, err
	}

	if len(moexHistoryJSON.History.Data) == 0 {
		// end of data
		return HistoryEntries{}, nil
	}

	moexHistory = make(HistoryEntries, len(moexHistoryJSON.History.Data))

	for i, entry := range moexHistoryJSON.History.Data {
		time, err := time.Parse("2006-01-02", entry[0].(string))
		if err != nil {
			return HistoryEntries{}, err
		}
		moexHistory[i].Date = time

		if entry[1] == nil || entry[2] == nil || entry[3] == nil {
			continue
		}

		moexHistory[i].Close = utils.GetFloat64(entry[1])
		moexHistory[i].High = utils.GetFloat64(entry[2])
		moexHistory[i].Low = utils.GetFloat64(entry[3].(float64))

		if len(entry) > 4 {
			moexHistory[i].Volume = uint64(utils.GetFloat64(entry[4]))
		} else {
			moexHistory[i].Volume = 0
		}

		if len(entry) > 5 {
			moexHistory[i].Facevalue = entry[5].(float64)
		} else {
			moexHistory[i].Facevalue = 1.0
		}
	}

	if api.Redis.Client != nil {
		var duration time.Duration
		if len(moexHistory)%PAGE_SIZE == 0 {
			// forever
			duration = time.Duration(0)

		} else {
			// cache until tomorrow
			now := time.Now().UTC()
			tomorrow := time.Date(
				now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC,
			).AddDate(0, 0, 1)
			duration = tomorrow.Sub(now)
		}
		err = api.setSecurityHistoryOffsetToCache(cacheKey, moexHistory, duration)
		if err != nil {
			return HistoryEntries{}, err
		}
	}

	return moexHistory, nil
}

func (api *MoexAPI) getSecurityCurrentPrice(ticker string, params MoexSecurityParameters) (HistoryEntry, error) {
	url := fmt.Sprintf(
		"%s/iss/engines/%s/markets/%s/securities/%s.json?iss.meta=off&iss.only=marketdata&marketdata.columns=BOARDID,LAST,HIGH,LOW,VOLTODAY",
		api.BaseURL, params.Engine, params.Market, ticker,
	)
	log.Printf("Fetching price data from url %s for %s\n", url, ticker)
	data, err := utils.HttpGet(url)
	if err != nil {
		return HistoryEntry{}, err
	}

	var moexPriceJSON MoexPriceJSON
	err = json.Unmarshal(data, &moexPriceJSON)
	if err != nil {
		return HistoryEntry{}, err
	}

	for _, entry := range moexPriceJSON.Marketdata.Data {
		if entry[0] != params.Board {
			continue
		}

		var moexHistory HistoryEntry

		if entry[1] == nil {
			return HistoryEntry{}, custom_errors.ErrorNoData
		} else {
			moexHistory.Close = utils.GetFloat64(entry[1].(float64))
		}

		if entry[2] != nil {
			moexHistory.High = utils.GetFloat64(entry[2].(float64))
		}

		if entry[3] != nil {
			moexHistory.Low = utils.GetFloat64(entry[3])
		}

		if len(entry) > 4 && entry[4] != nil {
			moexHistory.Volume = uint64(utils.GetFloat64(entry[4]))
		} else {
			moexHistory.Volume = 0
		}

		now := time.Now().UTC()
		year, month, day := now.Date()

		moexHistory.Date = time.Date(year, month, day, 0, 0, 0, 0, time.UTC)

		return moexHistory, nil
	}

	return HistoryEntry{}, custom_errors.ErrorNotFound

}
