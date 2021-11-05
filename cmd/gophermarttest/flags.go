package main

import (
	"flag"
)

var (
	flagTargetBinaryPath   string
	flagServerHost         string
	flagServerPort         string
	flagDatabaseURI        string
	flagAccrualServiceAddr string
)

func init() {
	flag.StringVar(&flagTargetBinaryPath, "binary-path", "", "path to target HTTP server binary")
	flag.StringVar(&flagServerHost, "server-host", "", "host to run HTTP server on")
	flag.StringVar(&flagServerPort, "server-port", "", "port to run HTTP server on")
	flag.StringVar(&flagDatabaseURI, "database-uri", "", "connection string to database")
	flag.StringVar(&flagAccrualServiceAddr, "accrual-addr", "", "accrual service address")
}
