package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kiberdruzhinnik/go-exchange-api/constants"
	"github.com/kiberdruzhinnik/go-exchange-api/utils"
	"golang.org/x/text/encoding/charmap"
)

// ValCurs represents the root XML element from CBR API
type ValCurs struct {
	XMLName xml.Name `xml:"ValCurs"`
	Records []Record `xml:"Record"`
}

// Record represents a single exchange rate record
type Record struct {
	Date    string `xml:"Date,attr"`
	Nominal string `xml:"Nominal"`
	Value   string `xml:"Value"`
}

func main() {
	// Set the date range for the last 30 days
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -30)

	// Format the dates as required by CBR API (dd/mm/yyyy)
	dateFormat := "02/01/2006"
	url := fmt.Sprintf(
		"%s/scripts/XML_dynamic.asp?date_req1=%s&date_req2=%s&VAL_NM_RQ=R01235",
		constants.CbrBaseApiURL,
		startDate.Format(dateFormat),
		endDate.Format(dateFormat),
	)

	// Make the HTTP request
	client, err := utils.NewHTTPClient()
	if err != nil {
		fmt.Printf("Error creating HTTP client: %v\n", err)
		return
	}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Printf("Error making request: %v\n", err)
		return
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error fetching CBR data: HTTP %d\n", resp.StatusCode)
		return
	}

	d := xml.NewDecoder(resp.Body)
	d.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		switch charset {
		case "windows-1251":
			return charmap.Windows1251.NewDecoder().Reader(input), nil
		default:
			return nil, fmt.Errorf("unknown charset: %s", charset)
		}
	}

	// Parse XML response
	var valCurs ValCurs
	err = d.Decode(&valCurs)
	if err != nil {
		fmt.Printf("Error parsing XML: %v\n", err)
		return
	}

	// Print the results
	fmt.Println("USD/RUB Exchange Rates:")
	fmt.Println("Date\t\tRate (RUB)")
	fmt.Println("------------------------")
	for _, record := range valCurs.Records {
		fmt.Printf("%s\t%s\n", record.Date, record.Value)
	}
}
