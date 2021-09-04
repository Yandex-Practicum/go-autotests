package main

import (
	"flag"
)

var (
	flagTargetBinaryPath string
	flagTargetSourcePath string
)

func ParseFlags() {
	flag.StringVar(&flagTargetBinaryPath, "binary-path", "", "path to target HTTP server binary")
	flag.StringVar(&flagTargetSourcePath, "source-path", "", "path to target HTTP server source")
	flag.Parse()
}
