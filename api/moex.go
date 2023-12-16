package api

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	custom_errors "github.com/kiberdruzhinnik/go-exchange-api/errors"
	"github.com/kiberdruzhinnik/go-exchange-api/utils"
	"github.com/shopspring/decimal"
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

type MoexHistoryEntry struct {
	Date      time.Time       `json:"date"`
	Close     decimal.Decimal `json:"close"`
	High      decimal.Decimal `json:"high"`
	Low       decimal.Decimal `json:"low"`
	Volume    uint64          `json:"volume"`
	Facevalue decimal.Decimal `json:"facevalue"`
}

type MoexHistoryEntries []MoexHistoryEntry

func (entries MoexHistoryEntries) MarshalBinary() ([]byte, error) {
	return json.Marshal(entries)
}

type MoexHistoryJSON struct {
	History struct {
		Columns []string        `json:"columns"`
		Data    [][]interface{} `json:"data"`
	} `json:"history"`
}

func NewMoexAPI(redis utils.RedisClient) MoexAPI {
	return MoexAPI{
		BaseURL: "https://iss.moex.com",
		Redis:   redis,
	}
}

func (api *MoexAPI) GetTicker(ticker string) (MoexHistoryEntries, error) {
	security, err := api.getSecurityParameters(ticker)
	if err != nil {
		log.Println(err)
		return MoexHistoryEntries{}, err
	}

	var history MoexHistoryEntries
	offset := uint(0)
	for {
		entryHistory, err := api.getSecurityHistoryOffset(ticker, security, offset)
		if err != nil {
			log.Println(err)
			return MoexHistoryEntries{}, err
		}
		if len(entryHistory) == 0 {
			break
		}
		offset += PAGE_SIZE
		history = append(history, entryHistory...)
	}

	return history, err
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

	// var output MoexSecurityParameters
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

func (api *MoexAPI) getSecurityHistoryOffsetFromCache(key string) (MoexHistoryEntries, error) {
	log.Printf("Getting history data from cache for %s\n", key)
	data, err := api.Redis.Client.Get(api.Redis.Context, key).Bytes()
	if err != nil {
		log.Printf("Got no history data from cache for %s\n", key)
		return MoexHistoryEntries{}, err
	}

	log.Printf("Got history data from cache for %s\n", key)
	var moexHistory MoexHistoryEntries
	err = json.Unmarshal(data, &moexHistory)
	if err != nil {
		return MoexHistoryEntries{}, err
	}

	return moexHistory, nil
}

func (api *MoexAPI) setSecurityHistoryOffsetToCache(key string, value MoexHistoryEntries) error {
	log.Printf("Saving history data to cache for %s\n", key)
	return api.Redis.Client.Set(api.Redis.Context, key, value, 0).Err()
}

func (api *MoexAPI) getSecurityHistoryOffset(ticker string,
	params MoexSecurityParameters,
	offset uint) (MoexHistoryEntries, error) {
	url := fmt.Sprintf("%s/iss/history/engines/%s/markets/%s/boards/%s/"+
		"securities/%s.json?iss.meta=off&start=%d&history.columns=TRADEDATE"+
		",CLOSE,HIGH,LOW,VOLUME,FACEVALUE",
		api.BaseURL, params.Engine, params.Market, params.Board, ticker, offset)

	var moexHistory MoexHistoryEntries
	var cacheKey string

	if api.Redis.Client != nil {
		cacheKey := fmt.Sprintf("%s-%s-%s-%s-%d", params.Board, params.Market, params.Engine, ticker, offset)
		moexHistory, err := api.getSecurityHistoryOffsetFromCache(cacheKey)
		if err == nil {
			return moexHistory, nil
		}
	}

	log.Printf("Fetching history data from url %s for %s\n", url, ticker)
	data, err := utils.HttpGet(url)
	if err != nil {
		return MoexHistoryEntries{}, err
	}

	var moexHistoryJSON MoexHistoryJSON
	err = json.Unmarshal(data, &moexHistoryJSON)
	if err != nil {
		return MoexHistoryEntries{}, err
	}

	if len(moexHistoryJSON.History.Data) == 0 {
		// end of data
		return MoexHistoryEntries{}, nil
	}

	moexHistory = make(MoexHistoryEntries, len(moexHistoryJSON.History.Data))

	for i, entry := range moexHistoryJSON.History.Data {
		time, err := time.Parse("2006-01-02", entry[0].(string))
		if err != nil {
			return MoexHistoryEntries{}, err
		}
		moexHistory[i].Date = time

		if entry[1] == nil || entry[2] == nil || entry[3] == nil {
			continue
		}
		moexHistory[i].Close = decimal.NewFromFloat(entry[1].(float64))
		moexHistory[i].High = decimal.NewFromFloat(entry[2].(float64))
		moexHistory[i].Low = decimal.NewFromFloat(entry[3].(float64))

		if len(entry) > 4 {
			moexHistory[i].Volume = uint64(entry[4].(float64))
		} else {
			moexHistory[i].Volume = 0
		}

		if len(entry) > 5 {
			moexHistory[i].Facevalue = decimal.NewFromFloat(entry[5].(float64))
		} else {
			moexHistory[i].Facevalue = decimal.Zero
		}
	}

	if api.Redis.Client != nil {
		if len(moexHistory)%PAGE_SIZE == 0 {
			err = api.setSecurityHistoryOffsetToCache(cacheKey, moexHistory)
			if err != nil {
				return MoexHistoryEntries{}, err
			}
		}
	}

	return moexHistory, nil
}
