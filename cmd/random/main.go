package main

//go:generate go build -o=../../bin/random

import (
	"flag"
	"fmt"
	"os"
)

var cmds = []cmd{
	unusedPortCmd,
	domainCmd,
	tempfileCmd,
}

type cmd struct {
	name      string
	shortHelp string
	do        func()
	flags     *flag.FlagSet
}

const Usage = `random is a tool to generate various random values.

Usage: random <command>

The commands are:
	help	show this help message
`

func help() {
	fmt.Print(Usage)

	for _, cmd := range cmds {
		fmt.Printf("\t%s\t%s\n", cmd.name, cmd.shortHelp)
	}

	os.Exit(2)
}

func fatalf(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, "yo: "+format+"\n", args...)
	os.Exit(1)
}

func main() {
	if len(os.Args) < 2 || os.Args[1] == "help" {
		help()
	}

	for _, cmd := range cmds {
		if os.Args[1] == cmd.name {
			if cmd.flags != nil {
				if err := cmd.flags.Parse(os.Args[2:]); err != nil {
					fatalf("cannot parse argument", err)
					help()
					return
				}
			}

			cmd.do()
			return
		}
	}

	help()
}
