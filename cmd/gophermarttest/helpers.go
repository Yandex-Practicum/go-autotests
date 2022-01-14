package main

import (
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
	return ds + strconv.Itoa(cd), nil
}

func luhnCheckDigit(s string) (int, error) {
	number, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}

	checkNumber := luhnChecksum(number)

	if checkNumber == 0 {
		return 0, nil
	}
	return 10 - checkNumber, nil
}

func luhnChecksum(number int) int {
	var luhn int

	for i := 0; number > 0; i++ {
		cur := number % 10

		if i%2 == 0 { // even
			cur = cur * 2
			if cur > 9 {
				cur = cur%10 + cur/10
			}
		}

		luhn += cur
		number = number / 10
	}
	return luhn % 10
}
