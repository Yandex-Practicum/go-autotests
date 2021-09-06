package main

// Basic imports
import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/suite"

	"github.com/Yandex-Practicum/go-autotests/internal/fork"
)

// Iteration4Suite is a suite of autotests
type Iteration4Suite struct {
	suite.Suite

	serverAddress string
	serverProcess *fork.BackgroundProcess

	knownEncodingLibs []string
}

// SetupSuite bootstraps suite dependencies
func (suite *Iteration4Suite) SetupSuite() {
	// check required flags
	suite.Require().NotEmpty(flagTargetSourcePath, "-source-path non-empty flag required")
	suite.Require().NotEmpty(flagTargetBinaryPath, "-binary-path non-empty flag required")

	suite.knownEncodingLibs = []string{
		"encoding/json",
		"github.com/mailru/easyjson",
		"github.com/pquerna/ffjson",
	}

	suite.serverAddress = "http://localhost:8080"

	// start server
	{
		p := fork.NewBackgroundProcess(context.Background(), flagTargetBinaryPath)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := p.Start(ctx)
		if err != nil {
			suite.T().Errorf("cannot start process with command %s: %s", p, err)
			return
		}

		port := "8080"
		err = p.WaitPort(ctx, "tcp", port)
		if err != nil {
			suite.T().Errorf("unable to wait for port %s to become available: %s", port, err)
			return
		}

		suite.serverProcess = p
	}
}

// TearDownSuite teardowns suite dependencies
func (suite *Iteration4Suite) TearDownSuite() {
	if suite.serverProcess == nil {
		return
	}

	exitCode, err := suite.serverProcess.Stop(syscall.SIGINT, syscall.SIGKILL)
	if err != nil {
		suite.T().Logf("unable to stop server via OS signals: %s", err)
		return
	}
	if exitCode > 0 {
		suite.T().Logf("server has exited with non-zero exit code: %s", err)

		// try to read stderr
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		out := suite.serverProcess.Stderr(ctx)
		if len(out) > 0 {
			suite.T().Logf("server process stderr log obtained:\n\n%s", string(out))
		}

		return
	}
}

// TestEncoderUsage attempts to recursively find usage of known HTTP frameworks in given sources
func (suite *Iteration4Suite) TestEncoderUsage() {
	err := filepath.WalkDir(flagTargetSourcePath, func(path string, d fs.DirEntry, err error) error {
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

		spec, err := importsKnownPackage(suite.T(), path, suite.knownEncodingLibs)
		if err != nil {
			// log error and continue traversing
			suite.T().Logf("error inspecting file %s: %s", path, err)
			return nil
		}
		if spec != nil && spec.Name.String() != "_" {
			return errUsageFound
		}

		return nil
	})

	if errors.Is(err, errUsageFound) {
		return
	}

	if err == nil {
		suite.T().Errorf("No usage of known encoding libraries has been found in %s", flagTargetSourcePath)
		return
	}
	suite.T().Errorf("unexpected error: %s", err)
}

// TestJSONHandler attempts to:
// - generate and send random URL to JSON API handler
// - fetch original URL by sending shorten URL to expand handler
func (suite *Iteration4Suite) TestJSONHandler() {
	// create HTTP client without redirects support
	errRedirectBlocked := errors.New("HTTP redirect blocked")
	httpc := resty.New().
		SetHostURL(suite.serverAddress).
		SetRedirectPolicy(
			resty.RedirectPolicyFunc(func(_ *http.Request, _ []*http.Request) error {
				return errRedirectBlocked
			}),
		)

	// declare and generate URLs
	originalURL := generateTestURL(suite.T())
	var shortenURL string

	suite.Run("shorten", func() {
		type shortenRequest struct {
			URL string `json:"url"`
		}

		type shortenResponse struct {
			Result string `json:"result"`
		}

		var result shortenResponse

		resp, err := httpc.R().
			SetHeader("Content-Type", "application/json").
			SetBody(&shortenRequest{
				URL: originalURL,
			}).
			SetResult(&result).
			Post("/api/shorten")
		suite.Require().NoError(err)

		shortenURL = result.Result

		suite.Assert().Equal(http.StatusCreated, resp.StatusCode())
		suite.Assert().NoError(func() error {
			_, err := url.Parse(shortenURL)
			return err
		}())
	})

	suite.Run("expand", func() {
		resp, err := httpc.R().Get(shortenURL)
		if !errors.Is(err, errRedirectBlocked) {
			suite.Require().NoError(err)
		}

		suite.Assert().Equal(http.StatusTemporaryRedirect, resp.StatusCode())
		suite.Assert().Equal(originalURL, resp.Header().Get("Location"))
	})
}
