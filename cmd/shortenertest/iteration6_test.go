package main

// Basic imports
import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"syscall"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/suite"

	"github.com/Yandex-Practicum/go-autotests/internal/fork"
)

// Iteration6Suite is a suite of autotests
type Iteration6Suite struct {
	suite.Suite

	serverAddress string
	serverProcess *fork.BackgroundProcess
}

// SetupSuite bootstraps suite dependencies
func (suite *Iteration6Suite) SetupSuite() {
	// check required flags
	suite.Require().NotEmpty(flagTargetBinaryPath, "-binary-path non-empty flag required")
	suite.Require().NotEmpty(flagFileStoragePath, "-file-storage-path non-empty flag required")

	suite.serverAddress = "http://localhost:8080"

	// start server
	{
		p := fork.NewBackgroundProcess(context.Background(), flagTargetBinaryPath,
			fork.WithEnv("FILE_STORAGE_PATH="+flagFileStoragePath),
		)

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
func (suite *Iteration6Suite) TearDownSuite() {
	suite.stopServer()
}

// TestPersistentFile attempts to:
// - call handlers as Iteration1Suite.TestHandlers does
// - check if file at flagFileStoragePath is not empty
func (suite *Iteration6Suite) TestPersistentFile() {
	originalURL := generateTestURL(suite.T())
	var shortenURL string

	// create HTTP client without redirects support
	errRedirectBlocked := errors.New("HTTP redirect blocked")
	httpc := resty.New().
		SetHostURL(suite.serverAddress).
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

		shortenURL = string(resp.Body())

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

	suite.Run("check_file", func() {
		// stop server in case of file flushed on exit
		suite.stopServer()

		suite.Assert().FileExists(flagFileStoragePath)
		b, err := os.ReadFile(flagFileStoragePath)
		suite.Require().NoError(err)
		suite.Assert().NotEmpty(b)
	})
}

// TearDownSuite teardowns suite dependencies
func (suite *Iteration6Suite) stopServer() {
	if suite.serverProcess == nil {
		return
	}

	exitCode, err := suite.serverProcess.Stop(syscall.SIGINT, syscall.SIGKILL)
	if err != nil && !errors.Is(err, os.ErrProcessDone) {
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
