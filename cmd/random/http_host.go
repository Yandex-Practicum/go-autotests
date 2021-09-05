package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/Yandex-Practicum/go-autotests/internal/random"
)

var httpHostFlags = flag.NewFlagSet("http-host", flag.ExitOnError)

var (
	flagHTTPHostMinLen = httpHostFlags.Int("min-len", 5, "minimum length of domain")
	flagHTTPHostMaxLen = httpHostFlags.Int("max-len", 15, "maximum length of domain")
	flagHTTPHostZones  = httpHostFlags.String("zones", "", "comma separated list of desired zones")
)

var httpHostCmd = cmd{
	name:      "http-host",
	shortHelp: "generates random http host",
	do:        generateHTTPHost,
	flags:     httpHostFlags,
}

func generateHTTPHost() {
	var zones []string
	if *flagHTTPHostZones != "" {
		zones = strings.Split(*flagHTTPHostZones, ",")
	}

	domain := random.Domain(*flagHTTPHostMinLen, *flagHTTPHostMaxLen, zones...)
	fmt.Print("http://" + domain)
}
