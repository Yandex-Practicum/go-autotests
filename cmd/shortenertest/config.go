package main

import (
	"os"
)

var config = struct {
	TargetAddress string
}{
	TargetAddress: func() string {
		if val := os.Getenv("TARGET_ADDRESS"); val != "" {
			return val
		}
		return "http://localhost:8080/"
	}(),
}
