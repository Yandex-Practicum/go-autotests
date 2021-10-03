package random

import (
	"net"
)

// Port returns random port in given range
func Port(from, to int) int {
	if from <= 0 {
		from = 1024
	}
	if to <= 0 || to > 65535 {
		to = 65535
	}
	return rnd.Intn(to-from) + from
}

// UnusedPort returns random unused port
func UnusedPort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
