package main

import (
	"flag"
)

func init() {

}

var (
	flagTargetBinaryPath string

	config struct {
		TargetBinary string
	}
)

func ParseConfig() {
	flag.StringVar(&flagTargetBinaryPath, "binary-path", "", "path to target HTTP server binary")
	flag.Parse()

	config.TargetBinary = flagTargetBinaryPath
}
