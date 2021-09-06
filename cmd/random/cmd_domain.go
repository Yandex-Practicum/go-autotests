package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/Yandex-Practicum/go-autotests/internal/random"
)

var domainFlags = flag.NewFlagSet("domain", flag.ExitOnError)

var (
	flagDomainMinLen = domainFlags.Int("min-len", 5, "minimum length of domain")
	flagDomainMaxLen = domainFlags.Int("max-len", 15, "maximum length of domain")
	flagDomainZones  = domainFlags.String("zones", "", "comma separated list of desired zones to choose from")
)

var domainCmd = cmd{
	name:      "domain",
	shortHelp: "generates random domain name",
	do:        generateDomain,
	flags:     domainFlags,
}

func generateDomain() {
	var zones []string
	if *flagDomainZones != "" {
		zones = strings.Split(*flagDomainZones, ",")
	}

	domain := random.Domain(*flagDomainMinLen, *flagDomainMaxLen, zones...)
	fmt.Print(domain)
}
