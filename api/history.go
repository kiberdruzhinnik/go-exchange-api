package api

import (
	"encoding/json"
	"time"
)

type HistoryEntry struct {
	Date      time.Time `json:"date"`
	Close     float64   `json:"close"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Volume    uint64    `json:"volume"`
	Facevalue float64   `json:"facevalue"`
}

type HistoryEntries []HistoryEntry

func (entries HistoryEntries) MarshalBinary() ([]byte, error) {
	return json.Marshal(entries)
}
