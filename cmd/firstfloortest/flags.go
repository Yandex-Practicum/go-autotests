package main

import "flag"

var (
	flagServerBinaryPath string // путь до бинарного файла сервера
)

func init() {
	flag.StringVar(&flagServerBinaryPath, "server-binary-path", "", "path to server binary")
}
