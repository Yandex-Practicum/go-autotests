package main

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"net"
	"net/http"
	"net/http/httputil"
	"path/filepath"
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

// usesKnownPackage checks if any file in given rootdir uses at least one of given knownPackages
func usesKnownPackage(t *testing.T, rootdir string, knownPackages []string) error {
	err := filepath.WalkDir(rootdir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			// skip vendor directory
			if d.Name() == "vendor" || d.Name() == ".git" {
				return filepath.SkipDir
			}
			// dive into regular directory
			return nil
		}

		// skip test files or non-Go files
		if !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}

		spec, err := importsKnownPackage(t, path, knownPackages)
		if err != nil {
			return fmt.Errorf("невозможно проинспектировать файл %s: %w", path, err)
		}
		if spec != nil && spec.Name.String() != "_" {
			return fmt.Errorf("%s: %w", spec.Path.Value, errUsageFound)
		}

		return nil
	})

	return err
}

// importsKnownPackage checks if given file imports
func importsKnownPackage(t *testing.T, filepath string, knownPackages []string) (*ast.ImportSpec, error) {
	t.Helper()

	fset := token.NewFileSet()
	sf, err := parser.ParseFile(fset, filepath, nil, parser.ImportsOnly)
	if err != nil {
		return nil, fmt.Errorf("невозможно распарсить файл: %w", err)
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

// dumpResponse is a shorthand to httputil.DumpResponse
func dumpResponse(resp *http.Response, body bool) (dump []byte) {
	if resp != nil {
		dump, _ = httputil.DumpResponse(resp, body)
	}
	return
}
