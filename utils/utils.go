package utils

import (
	"io"
	"net/http"
	"strings"
	"unicode"
)

func HttpGet(url string) ([]byte, error) {
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

func RemoveNonAlnum(s string) string {
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
