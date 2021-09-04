package fork

import (
	"time"
)

type ProcessOpt = func(p *BackgroundProcess)

// WithEnv adds env KEY=VALUE pairs to process
func WithEnv(env ...string) ProcessOpt {
	return func(p *BackgroundProcess) {
		p.cmd.Env = append(p.cmd.Env, env...)
	}
}

// WithArgs adds command line arguments to process
func WithArgs(args ...string) ProcessOpt {
	return func(p *BackgroundProcess) {
		p.cmd.Args = append(p.cmd.Args, args...)
	}
}

// WaitPortConnTimeout sets connection timeout while waiting for port
func WaitPortConnTimeout(d time.Duration) ProcessOpt {
	return func(p *BackgroundProcess) {
		p.waitPortConnTimeout = d
	}
}

// WaitPortInterval sets port check interval
func WaitPortInterval(d time.Duration) ProcessOpt {
	return func(p *BackgroundProcess) {
		p.waitPortInterval = d
	}
}
