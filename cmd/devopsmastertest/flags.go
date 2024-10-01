package main

import (
	"flag"
)

// Доступные для тест-сьютов флаги командной строки
var (
	flagTargetBinaryPath string // путь до бинарного файла проекта
)

func init() {
	flag.StringVar(&flagTargetBinaryPath, "binary-path", "", "path to target script binary")
}
