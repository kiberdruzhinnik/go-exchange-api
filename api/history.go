package api

import (
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
)

type HistoryEntry struct {
	Date      time.Time       `json:"date"`
	Close     decimal.Decimal `json:"close"`
	High      decimal.Decimal `json:"high"`
	Low       decimal.Decimal `json:"low"`
	Volume    uint64          `json:"volume"`
	Facevalue decimal.Decimal `json:"facevalue"`
}

type HistoryEntries []HistoryEntry

func (entries HistoryEntries) MarshalBinary() ([]byte, error) {
	return json.Marshal(entries)
}
