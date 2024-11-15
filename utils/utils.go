package utils

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"unicode"

	"github.com/kiberdruzhinnik/go-exchange-api/constants"
	"github.com/kiberdruzhinnik/go-exchange-api/errors"
)

var URLS_ALLOW_LIST []string = []string{
	constants.MoexBaseApiURL,
	constants.SpbexBaseApiURL,
}

func CheckSafeURL(url string) bool {
	for _, u := range URLS_ALLOW_LIST {
		if strings.HasPrefix(url, u) {
			return true
		}
	}
	return false
}

func HttpGet(url string) ([]byte, error) {

	if !CheckSafeURL(url) {
		return nil, errors.ErrorNotAllowed
	}

	resp, err := http.Get(url)

	if err != nil {
		return []byte{}, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return []byte{}, err
	}

	return body, nil
}

func StringAllowlist(s string) string {
	valid := []*unicode.RangeTable{
		unicode.Letter,
		unicode.Digit,
		{R16: []unicode.Range16{{'_', '_', 1}}},
	}
	return strings.Map(
		func(r rune) rune {
			if unicode.IsOneOf(valid, r) {
				return r
			}
			return -1
		},
		s,
	)
}

func GetFloat64(v interface{}) float64 {
	if v == nil {
		return 0
	}

	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case string:
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return 0
}
