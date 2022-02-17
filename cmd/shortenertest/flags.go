package main

import (
	"flag"
)

// Доступные для тест-сьютов флаги командной строки
var (
	flagTargetBinaryPath  string // путь до бинарного файла проекта
	flagTargetSourcePath  string // путь до исходного кода проекта
	flagServerHost        string // адрес хоста на котором запущен проект
	flagServerPort        string // номер порта на котором запущен проект
	flagServerBaseURL     string // базовый URL проекта
	flagFileStoragePath   string // путь до файла с данными проекта
	flagDatabaseDSN       string // строка для подключения к базе данных
	flagBaseProfilePath   string
	flagResultProfilePath string
	flagPackageName       string
)

func init() {
	flag.StringVar(&flagTargetBinaryPath, "binary-path", "", "path to target HTTP server binary")
	flag.StringVar(&flagTargetSourcePath, "source-path", "", "path to target HTTP server source")
	flag.StringVar(&flagServerHost, "server-host", "", "host of target HTTP address")
	flag.StringVar(&flagServerPort, "server-port", "", "port of target HTTP address")
	flag.StringVar(&flagServerBaseURL, "server-base-url", "", "base URL of target HTTP address")
	flag.StringVar(&flagFileStoragePath, "file-storage-path", "", "path to persistent file storage")
	flag.StringVar(&flagDatabaseDSN, "database-dsn", "", "connection string to database")
	flag.StringVar(&flagBaseProfilePath, "base-profile-path", "", "path to base pprof profile")
	flag.StringVar(&flagResultProfilePath, "result-profile-path", "", "path to result pprof profile")
	flag.StringVar(&flagPackageName, "package-name", "", "name of package to be tested")
}
