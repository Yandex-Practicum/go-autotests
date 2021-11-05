package main

import (
	"flag"
)

var (
	flagGophermartBinaryPath  string
	flagGophermartHost        string
	flagGophermartPort        string
	flagGophermartDatabaseURI string

	flagAccrualBinaryPath  string
	flagAccrualHost        string
	flagAccrualPort        string
	flagAccrualDatabaseURI string
)

func init() {
	flag.StringVar(&flagGophermartBinaryPath, "gophermart-binary-path", "", "path to gophermart HTTP server binary")
	flag.StringVar(&flagGophermartHost, "gophermart-host", "", "host to run gophermart HTTP server on")
	flag.StringVar(&flagGophermartPort, "gophermart-port", "", "port to run gophermart HTTP server on")
	flag.StringVar(&flagGophermartDatabaseURI, "gophermart-database-uri", "", "connection string to gophermart database")

	flag.StringVar(&flagAccrualBinaryPath, "accrual-binary-path", "", "path to accrual HTTP server binary")
	flag.StringVar(&flagAccrualHost, "accrual-host", "", "host to run accrual HTTP server on")
	flag.StringVar(&flagAccrualPort, "accrual-port", "", "port to run accrual HTTP server on")
	flag.StringVar(&flagAccrualDatabaseURI, "accrual-database-uri", "", "connection string to accrual database")
}
