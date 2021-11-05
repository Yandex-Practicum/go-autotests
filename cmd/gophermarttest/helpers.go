package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"strconv"
	"testing"

	"github.com/Yandex-Practicum/go-autotests/internal/random"
)

// dumpRequest is a shorthand to httputil.DumpRequest
func dumpRequest(t *testing.T, req *http.Request, body io.Reader) []byte {
	t.Helper()
	if req == nil {
		return nil
	}

	dump, _ := httputil.DumpRequest(req, false)
	if body != nil {
		b, err := io.ReadAll(body)
		if err == nil {
			dump = append(dump, '\n')
			dump = append(dump, b...)
			dump = append(dump, '\n')
			dump = append(dump, '\n')
		}
	}

	return dump
}

func generateOrderNumber(t *testing.T) (string, error) {
	t.Helper()
	ds := random.DigitString(5, 15)
	cd, err := luhnCheckDigit(ds)
	if err != nil {
		return "", fmt.Errorf("cannot calculate check digit: %s", err)
	}
	return ds + strconv.FormatUint(uint64(cd), 10), nil
}

func luhnCheckDigit(s string) (uint8, error) {
	if s == "" {
		return 0, errors.New("empty string given")
	}

	var sum uint
	var alter bool
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] < '0' || s[i] > '9' {
			return 0, errors.New("non-numeric character found")
		}

		d := uint(s[i] - '0')
		if alter {
			d *= 2
		}
		sum += d / 10
		sum += d % 10
		alter = !alter
	}

	return uint8(sum % 10), nil
}
