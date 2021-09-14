package main

import (
	"flag"
)

var (
	flagTargetBinaryPath string
	flagTargetSourcePath string
	flagServerHost       string
	flagServerPort       string
	flagServerBaseURL    string
	flagFileStoragePath  string
	flagDatabaseDSN      string
)

func init() {
	flag.StringVar(&flagTargetBinaryPath, "binary-path", "", "path to target HTTP server binary")
	flag.StringVar(&flagTargetSourcePath, "source-path", "", "path to target HTTP server source")
	flag.StringVar(&flagServerHost, "server-host", "", "host of target HTTP address")
	flag.StringVar(&flagServerPort, "server-port", "", "port of target HTTP address")
	flag.StringVar(&flagServerBaseURL, "server-base-url", "", "base URL of target HTTP address")
	flag.StringVar(&flagFileStoragePath, "file-storage-path", "", "path to persistent file storage")
	flag.StringVar(&flagDatabaseDSN, "database-dsn", "", "connection string to database")
}
