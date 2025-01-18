package api

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kiberdruzhinnik/go-exchange-api/constants"
	custom_errors "github.com/kiberdruzhinnik/go-exchange-api/errors"
	"github.com/kiberdruzhinnik/go-exchange-api/utils"
	"golang.org/x/text/encoding/charmap"
)

// https://www.cbr.ru/scripts/XML_val.asp?d=0
var CBR_CURRENCIES = map[string]string{
	"usd": "R01235",
	"cny": "R01375",
	"eur": "R01239",
}

type CbrAPI struct {
	BaseURL string
}

func NewCbrAPI() CbrAPI {
	return CbrAPI{
		BaseURL: constants.CbrBaseApiURL,
	}
}

type ValCurs struct {
	XMLName xml.Name `xml:"ValCurs"`
	Records []Record `xml:"Record"`
}

type Record struct {
	Date      string `xml:"Date,attr"`
	Nominal   string `xml:"Nominal"`
	Value     string `xml:"Value"`
	VunitRate string `xml:"VunitRate"`
}

func (api *CbrAPI) GetTicker(ticker string) (HistoryEntries, error) {

	if !utils.Contains(CBR_CURRENCIES, ticker) {
		return HistoryEntries{}, custom_errors.ErrorNotFound
	}

	endDate := time.Now()
	startDate := time.Date(2014, 01, 01, 01, 01, 01, 01, time.UTC)
	dateFormat := "02/01/2006"

	url := fmt.Sprintf(
		"https://www.cbr.ru/scripts/XML_dynamic.asp?date_req1=%s&date_req2=%s&VAL_NM_RQ=%s",
		startDate.Format(dateFormat),
		endDate.Format(dateFormat),
		CBR_CURRENCIES[ticker],
	)
	log.Printf("Getting data from %s\n", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Error creating request: %v\n", err)
		return HistoryEntries{}, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error making request: %v\n", err)
		return HistoryEntries{}, err
	}
	defer resp.Body.Close()

	d := xml.NewDecoder(resp.Body)
	d.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		switch charset {
		case "windows-1251":
			return charmap.Windows1251.NewDecoder().Reader(input), nil
		default:
			return nil, fmt.Errorf("unknown charset: %s", charset)
		}
	}

	var valCurs ValCurs
	err = d.Decode(&valCurs)
	if err != nil {
		log.Printf("Error parsing XML: %v\n", err)
		return HistoryEntries{}, err
	}

	historyEntries := make(HistoryEntries, len(valCurs.Records))

	for i := range valCurs.Records {
		time, err := time.Parse("02.01.2006", valCurs.Records[i].Date)
		if err != nil {
			return HistoryEntries{}, err
		}
		historyEntries[i].Date = time

		valueStr := strings.Replace(valCurs.Records[i].VunitRate, ",", ".", -1)
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			log.Printf("Error parsing value: %v\n", err)
			continue
		}
		historyEntries[i].Close = value
		historyEntries[i].Facevalue = 1
	}

	return historyEntries, nil
}
