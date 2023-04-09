package main

import (
	"flag"
	"strconv"
	"time"
)

const (
	serverDefaultHost          = "localhost"
	serverDefaultPort          = 8080
	agentDefaultReportInterval = 10 * time.Second
	agentDefaultPollInterval   = 2 * time.Second
)

var (
	flagAgentBinaryPath  string
	flagServerBinaryPath string
	flagTargetSourcePath string
	flagServerHost       string
	flagServerPort       string
	flagServerBaseURL    string
	flagFileStoragePath  string
	flagDatabaseDSN      string
	flagSHA256Key        string
)

func init() {
	flag.StringVar(&flagAgentBinaryPath, "agent-binary-path", "", "path to target agent binary")
	flag.StringVar(&flagServerBinaryPath, "binary-path", "", "path to target server binary")
	flag.StringVar(&flagTargetSourcePath, "source-path", "", "path to target server source")
	flag.StringVar(&flagServerHost, "server-host", serverDefaultHost, "host of target address")
	flag.StringVar(&flagServerPort, "server-port", strconv.Itoa(serverDefaultPort), "port of target address")
	flag.StringVar(&flagServerBaseURL, "server-base-url", "", "base URL of target address")
	flag.StringVar(&flagFileStoragePath, "file-storage-path", "", "path to persistent file storage")
	flag.StringVar(&flagDatabaseDSN, "database-dsn", "", "connection string to database")
	flag.StringVar(&flagSHA256Key, "key", "", "sha256 key for hashing")
}
