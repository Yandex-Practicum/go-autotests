package main

// Basic imports
import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"net/http"
	"net/url"
	"syscall"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/suite"

	"github.com/Yandex-Practicum/go-autotests/internal/fork"
)

// Iteration8Suite is a suite of autotests
type Iteration8Suite struct {
	suite.Suite

	serverAddress string
	serverProcess *fork.BackgroundProcess
}

// SetupSuite bootstraps suite dependencies
func (suite *Iteration8Suite) SetupSuite() {
	// check required flags
	suite.Require().NotEmpty(flagTargetBinaryPath, "-binary-path non-empty flag required")

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

		err = p.WaitPort(ctx, "tcp", "8080")
		if err != nil {
			suite.T().Errorf("unable to wait for port %s to become available: %s", "8080", err)
			return
		}

		suite.serverProcess = p
	}
}

// TearDownSuite teardowns suite dependencies
func (suite *Iteration8Suite) TearDownSuite() {
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

// TestGzipCompress attempts to:
// - generate and send random URL to shorten handler (with gzip)
// - generate and send random URL to shorten API handler (without gzip)
// - fetch original URLs by sending shorten URLs to expand handler one by one
func (suite *Iteration8Suite) TestGzipCompress() {
	originalURL := generateTestURL(suite.T())
	var shortenURLs []string

	// create HTTP client without redirects support and custom resolver
	errRedirectBlocked := errors.New("HTTP redirect blocked")

	httpc := resty.New().
		SetHostURL(suite.serverAddress).
		SetRedirectPolicy(
			resty.RedirectPolicyFunc(func(_ *http.Request, _ []*http.Request) error {
				return errRedirectBlocked
			}),
		)

	suite.Run("shorten", func() {
		// gzip request body for base shorten handler
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		_, _ = zw.Write([]byte(originalURL))
		_ = zw.Close()

		resp, err := httpc.R().
			SetBody(buf.Bytes()).
			SetHeader("Accept-Encoding", "gzip").
			SetHeader("Content-Encoding", "gzip").
			Post("/")
		suite.Require().NoError(err)

		shortenURL := string(resp.Body())

		suite.Assert().Equal(http.StatusCreated, resp.StatusCode())
		suite.Assert().NoError(func() error {
			_, err := url.Parse(shortenURL)
			return err
		}())

		shortenURLs = append(shortenURLs, shortenURL)
	})

	suite.Run("shorten_api", func() {
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

		shortenURL := result.Result

		suite.Assert().Equal(http.StatusCreated, resp.StatusCode())
		suite.Assert().NoError(func() error {
			_, err := url.Parse(shortenURL)
			return err
		}())

		shortenURLs = append(shortenURLs, shortenURL)
	})

	suite.Run("expand", func() {
		for _, shortenURL := range shortenURLs {
			resp, err := httpc.R().Get(shortenURL)
			if !errors.Is(err, errRedirectBlocked) {
				suite.Assert().NoErrorf(err, "URL to expand: %s", shortenURL)
			}

			suite.Assert().Equalf(http.StatusTemporaryRedirect, resp.StatusCode(), "URL to expand: %s", shortenURL)
			suite.Assert().Equalf(originalURL, resp.Header().Get("Location"), "URL to expand: %s", shortenURL)
		}
	})
}
