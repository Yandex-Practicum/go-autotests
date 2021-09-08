package main

import (
	"fmt"

	"github.com/Yandex-Practicum/go-autotests/internal/random"
)

var unusedPortCmd = cmd{
	name:      "unused-port",
	shortHelp: "finds and returns random unused port number",
	do:        generateUnusedPort,
}

func generateUnusedPort() {
	port, err := random.UnusedPort()
	if err != nil {
		fatalf("cannot find unused port: %s", err)
	}
	fmt.Print(port)
}
