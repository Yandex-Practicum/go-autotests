package fork

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
)

// BackgroundProcess является удобной оберткой над exec.Cmd
// для работы с запущенными процессами
type BackgroundProcess struct {
	cmd    *exec.Cmd
	stdout *buffer
	stderr *buffer

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

	p.stdout = new(buffer)
	p.cmd.Stdout = p.stdout
	p.stderr = new(buffer)
	p.cmd.Stderr = p.stdout

	return p
}

// Start является аналогом (*exec.Cmd).Start с поддержкой контекста
func (p *BackgroundProcess) Start(ctx context.Context) error {
	startChan := make(chan error, 1)
	go func() {
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

// WaitPort позволяет дождаться занятия порта процессом
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

// ListenPort позволяет проверить наличие свободного порта
func (p *BackgroundProcess) ListenPort(ctx context.Context, network, port string) error {
	ticker := time.NewTicker(p.waitPortInterval)
	defer ticker.Stop()

	port = strings.TrimLeft(port, ":")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			lc := &net.ListenConfig{}
			ln, _ := lc.Listen(ctx, network, ":"+port)
			if ln != nil {
				defer ln.Close()
				done := make(chan struct{})
				go func() {
					conn, _ := ln.Accept()
					if conn != nil {
						_ = conn.Close()
					}
					close(done)
				}()
				select {
				case <-done:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
	}
}

// Stdout вычитывает и возвращает новый блок данных из stdout
func (p *BackgroundProcess) Stdout(ctx context.Context) []byte {
	return p.stdout.Bytes()
}

// Stderr вычитывает и возвращает новый блок данных из stderr
func (p *BackgroundProcess) Stderr(ctx context.Context) []byte {
	return p.stderr.Bytes()
}

// Stop пытается остановить процесс последовательной передачей процессу данных сигналов
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

// String возвращает человекочитаемую команду, которая породила процесс
func (p *BackgroundProcess) String() string {
	return p.cmd.String()
}
