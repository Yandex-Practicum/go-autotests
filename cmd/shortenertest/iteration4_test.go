package main

import (
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"golang.org/x/tools/go/ast/astutil"
)

var (
	knownEncodingLibs = []string{
		"encoding/json",
		"github.com/mailru/easyjson",
		"github.com/pquerna/ffjson",
	}
)

// TestIteration4 checks that students code:
// - uses known JSON encoding library
// - exposes new API handler
func TestIteration4(t *testing.T) {
	iteration4_AstTest(t)
	Iteration4_HandlerTest(t)
}

func iteration4_AstTest(t *testing.T) {
	fset := token.NewFileSet()

	err := filepath.WalkDir(config.SourceRoot, func(path string, d fs.DirEntry, err error) error {
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

		sf, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return fmt.Errorf("cannot parse AST of file: %s: %w", path, err)
		}

		importSpecs := astutil.Imports(fset, sf)
		if importsKnownPackage(importSpecs, knownEncodingLibs) {
			return importFound
		}

		return nil
	})

	if errors.Is(err, importFound) {
		return
	}

	if err == nil {
		t.Error("No import of known HTTP framework has been found")
		return
	}

	t.Errorf("unexpected error: %s", err)
}

type shortenRequest struct {
	URL string `json:"url"`
}

type shortenResponse struct {
	Result string `json:"result"`
}

func Iteration4_HandlerTest(t *testing.T) {
	endpointURL := config.TargetAddress + "/api/shorten"
	targetURL := generateTestURL(t)

	// create HTTP client without redirects support
	errRedirectBlocked := errors.New("HTTP redirect blocked")
	httpc := resty.New().
		SetRedirectPolicy(resty.RedirectPolicyFunc(func(_ *http.Request, _ []*http.Request) error {
			return errRedirectBlocked
		}),
		)

	var result shortenResponse

	resp, err := httpc.R().
		SetHeader("Content-Type", "application/json").
		SetBody(&shortenRequest{
			URL: targetURL,
		}).
		SetResult(&result).
		Post(endpointURL)
	if !assert.NoError(t, err) {
		return
	}

	shortenURL := result.Result

	assert.Equal(t, http.StatusCreated, resp.StatusCode())
	assert.NoError(t, func() error {
		_, err := url.Parse(shortenURL)
		return err
	}())

	// expand URL
	resp, err = httpc.R().Get(shortenURL)
	if !errors.Is(err, errRedirectBlocked) && !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode())
	assert.Equal(t, targetURL, resp.Header().Get("Location"))
}