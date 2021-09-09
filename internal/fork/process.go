package fork

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
)

type BackgroundProcess struct {
	cmd    *exec.Cmd
	stdout *bufio.Scanner
	stderr *bufio.Scanner

	waitPortInterval    time.Duration
	waitPortConnTimeout time.Duration
}

// NewBackgroundProcess returns new unstarted background process instance.
func NewBackgroundProcess(ctx context.Context, command string, opts ...ProcessOpt) *BackgroundProcess {
	p := &BackgroundProcess{
		cmd:                 exec.CommandContext(ctx, command),
		waitPortInterval:    100 * time.Millisecond,
		waitPortConnTimeout: 50 * time.Millisecond,
	}

	for _, opt := range opts {
		opt(p)
	}

	p.cmd.Stdout = io.Discard
	rStdout, wStdout, err := os.Pipe()
	if err == nil {
		p.cmd.Stdout = wStdout
		p.stdout = bufio.NewScanner(rStdout)
	}

	p.cmd.Stderr = io.Discard
	rStderr, wStderr, err := os.Pipe()
	if err == nil {
		p.cmd.Stderr = wStderr
		p.stderr = bufio.NewScanner(rStderr)
	}

	return p
}

// Start attempts to create OS process and start command execution.
func (p *BackgroundProcess) Start(ctx context.Context) error {
	startChan := make(chan error, 1)
	go func() {
		log.Printf("process envs: %+v", p.cmd.Env)
		startChan <- p.cmd.Start()
	}()

	for {
		select {
		case err := <-startChan:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// WaitPort tries to perform network connection to given port.
func (p *BackgroundProcess) WaitPort(ctx context.Context, network, port string) error {
	ticker := time.NewTicker(p.waitPortInterval)
	defer ticker.Stop()

	port = strings.TrimLeft(port, ":")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			conn, _ := net.DialTimeout(network, ":"+port, p.waitPortConnTimeout)
			if conn != nil {
				_ = conn.Close()
				return nil
			}
		}
	}
}

// Stdout reads and returns next portion of bytes from stdout.
// This function may block until next newline is present in output
func (p *BackgroundProcess) Stdout(ctx context.Context) []byte {
	return readOutput(ctx, p.stdout)
}

// Stderr reads and returns next portion of bytes from stderr.
// This function may block until next newline is present in output
func (p *BackgroundProcess) Stderr(ctx context.Context) []byte {
	return readOutput(ctx, p.stderr)
}

// Stop attempts to send given signals to process one by one.
// After first successful signal attempt exit code of process will be returned
func (p *BackgroundProcess) Stop(signals ...os.Signal) (exitCode int, err error) {
	for _, sig := range signals {
		err = p.cmd.Process.Signal(sig)
		if err == nil {
			break
		}
	}

	if err != nil {
		return -1, fmt.Errorf("error sending signal to process: %w", err)
	}

	state, err := p.cmd.Process.Wait()
	if state == nil {
		return -1, err
	}
	return state.ExitCode(), err
}

// String returns a human-readable representation of process command.
func (p *BackgroundProcess) String() string {
	return p.cmd.String()
}

// readOutput reads process output in a non-blocking way
func readOutput(ctx context.Context, output *bufio.Scanner) []byte {
	outChan := make(chan []byte, 1)
	go func() {
		output.Scan()
		outChan <- output.Bytes()
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case out := <-outChan:
			return out
		}
	}
}
