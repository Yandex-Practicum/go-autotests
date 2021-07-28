package main

import (
	"os"
)

var config = struct {
	TargetAddress string
	SourceRoot    string
	GobFilePath   string
}{
	TargetAddress: func() string {
		if val := os.Getenv("TARGET_HTTP_ADDRESS"); val != "" {
			return val
		}
		return "http://localhost:8080"
	}(),

	SourceRoot: func() string {
		if val := os.Getenv("SOURCE_ROOT"); val != "" {
			return val
		}
		return "."
	}(),

	GobFilePath: func() string {
		if val := os.Getenv("GOB_FILE_PATH"); val != "" {
			return val
		}
		return "urls.gob"
	}(),
}
