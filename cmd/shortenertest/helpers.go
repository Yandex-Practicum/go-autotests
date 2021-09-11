package main

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"testing"
	"time"

	"golang.org/x/tools/go/ast/astutil"

	"github.com/Yandex-Practicum/go-autotests/internal/random"
)

// generateTestURL returns valid random URL
func generateTestURL(t *testing.T) string {
	t.Helper()
	return random.URL().String()
}

// importsKnownPackage checks if given file imports
func importsKnownPackage(t *testing.T, filepath string, knownPackages []string) (*ast.ImportSpec, error) {
	t.Helper()

	fset := token.NewFileSet()
	sf, err := parser.ParseFile(fset, filepath, nil, parser.ImportsOnly)
	if err != nil {
		return nil, fmt.Errorf("cannot parse file: %w", err)
	}

	importSpecs := astutil.Imports(fset, sf)
	for _, paragraph := range importSpecs {
		for _, importSpec := range paragraph {
			for _, knownImport := range knownPackages {
				if strings.Contains(importSpec.Path.Value, knownImport) {
					return importSpec, nil
				}
			}
		}
	}

	return nil, nil
}

// dialContextFunc is a function that is suitable to be setted as an (*http.Transport).DialContext
type dialContextFunc = func(ctx context.Context, network, addr string) (net.Conn, error)

// mockResolver returns dialContextFunc that intercepts network requests
// and resolves given address as custom IP address
func mockResolver(network, requestAddress, responseIP string) dialContextFunc {
	dialer := &net.Dialer{
		Timeout:   time.Second,
		KeepAlive: 30 * time.Second,
	}
	return func(ctx context.Context, net, addr string) (net.Conn, error) {
		if net == network && addr == requestAddress {
			addr = responseIP
		}
		return dialer.DialContext(ctx, net, addr)
	}
}

// dumpRequest is a shorthand to httputil.DumpRequest
func dumpRequest(req *http.Request, body bool) (dump []byte) {
	if req != nil {
		dump, _ = httputil.DumpRequest(req, body)
	}
	return
}
