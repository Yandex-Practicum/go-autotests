package main

import (
	"io"
	"net/http"
	"net/http/httputil"
	"testing"
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
