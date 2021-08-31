package main

import (
	"os"
)

var config = struct {
	TargetAddress string
	SourceRoot         string
	PersistentFilePath string
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

	PersistentFilePath: os.Getenv("PERSISTENT_FILE_PATH"),
}
