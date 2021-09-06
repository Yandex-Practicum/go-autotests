package main

// Basic imports
import (
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

// Iteration5Suite is a suite of autotests
type Iteration5Suite struct {
	suite.Suite

	serverAddress string
	serverBaseURL string
	serverProcess *fork.BackgroundProcess
}

// SetupSuite bootstraps suite dependencies
func (suite *Iteration5Suite) SetupSuite() {
	// check required flags
	suite.Require().NotEmpty(flagTargetBinaryPath, "-binary-path non-empty flag required")
	suite.Require().NotEmpty(flagServerPort, "-server-port non-empty flag required")

	// start server
	{
		suite.serverAddress = "localhost:" + flagServerPort
		suite.serverBaseURL = "http://" + suite.serverAddress

		p := fork.NewBackgroundProcess(context.Background(), flagTargetBinaryPath,
			fork.WithEnv(
				"SERVER_ADDRESS="+suite.serverAddress,
				"BASE_URL="+suite.serverBaseURL,
			),
		)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := p.Start(ctx)
		if err != nil {
			suite.T().Errorf("cannot start process with command %s: %s", p, err)
			return
		}

		err = p.WaitPort(ctx, "tcp", flagServerPort)
		if err != nil {
			suite.T().Errorf("unable to wait for port %s to become available: %s", flagServerPort, err)
			return
		}

		suite.serverProcess = p
	}
}

// TearDownSuite teardowns suite dependencies
func (suite *Iteration5Suite) TearDownSuite() {
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

// TestHandlers attempts to:
// - generate and send random URL to shorten handler
// - generate and send random URL to shorten API handler
// - fetch original URLs by sending shorten URLs to expand handler one by one
func (suite *Iteration5Suite) TestHandlers() {
	originalURL := generateTestURL(suite.T())
	var shortenURLs []string

	// create HTTP client without redirects support and custom resolver
	errRedirectBlocked := errors.New("HTTP redirect blocked")

	restyClient := resty.New()
	transport := restyClient.GetClient().Transport.(*http.Transport)

	// mock all network requests to be resolved at localhost
	resolveIP := "127.0.0.1:" + flagServerPort
	transport.DialContext = mockResolver("tcp", suite.serverAddress, resolveIP)

	httpc := restyClient.
		SetTransport(transport).
		SetHostURL(suite.serverBaseURL).
		SetRedirectPolicy(
			resty.RedirectPolicyFunc(func(_ *http.Request, _ []*http.Request) error {
				return errRedirectBlocked
			}),
		)

	suite.Run("shorten", func() {
		resp, err := httpc.R().
			SetBody(originalURL).
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
