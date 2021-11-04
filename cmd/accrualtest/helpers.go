package main

import (
	"net/http"
	"net/http/httputil"
)

// dumpRequest is a shorthand to httputil.DumpRequest
func dumpRequest(req *http.Request, body bool) (dump []byte) {
	if req != nil {
		dump, _ = httputil.DumpRequest(req, body)
	}
	return
}
