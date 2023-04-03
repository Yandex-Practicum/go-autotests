package main

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"net"
	"net/http"
	"net/http/httputil"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/tools/go/ast/astutil"

	"github.com/Yandex-Practicum/go-autotests/internal/random"
)

// generateTestURL возвращает валидный псевдослучайный URL
func generateTestURL(t *testing.T) string {
	t.Helper()
	return random.URL().String()
}

// usesKnownPackage проверяет, что хотя бы в одном файле, начиная с указанной директории rootdir,
// содержится хотя бы один пакет из списка knownPackages
func usesKnownPackage(t *testing.T, rootdir string, knownPackages []string) error {
	// запускаем рекурсивное прохождение по дереву директорий начиная с rootdir
	err := filepath.WalkDir(rootdir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// объект является директорией
		if d.IsDir() {
			// пропускаем служебные директории
			if d.Name() == "vendor" || d.Name() == ".git" {
				// возвращаем специальную ошибку, сигнализирующую, что необходимо пропустить
				// рекурсивное сканирование директории
				return filepath.SkipDir
			}
			// углубляемся в директорию
			return nil
		}

		// пропускаем файлы с тестами или без расширения .go
		if !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}

		// пытаемся получить import запись из файла с исходным кодом
		spec, err := importsKnownPackage(t, path, knownPackages)
		if err != nil {
			return fmt.Errorf("невозможно проинспектировать файл %s: %w", path, err)
		}
		// запись не пустая и импортирована явно
		if spec != nil && spec.Name.String() != "_" {
			// возвращаем специализированную ошибку, сообщающую о нахождении импорта
			return errUsageFound
		}

		// продолжаем сканирование файлов
		return nil
	})

	// рекурсия не вернула никакой ошибки = мы не нашли искомого импорта ни в одном файле
	if err == nil {
		return errUsageNotFound
	}
	// получена специализированная ошибка = мы нашли искомый импорт в файлах
	if errors.Is(err, errUsageFound) {
		return nil
	}
	// неизвестная ошибка - возвращаем ее вызывающей функции
	return err
}

// importsKnownPackage зовращает import запись первого найденного импорта из списка knownPackages в файле filepath
func importsKnownPackage(t *testing.T, filepath string, knownPackages []string) (*ast.ImportSpec, error) {
	t.Helper()

	// парсим файл с исходным кодом
	fset := token.NewFileSet()
	sf, err := parser.ParseFile(fset, filepath, nil, parser.ImportsOnly)
	if err != nil {
		return nil, fmt.Errorf("невозможно распарсить файл: %w", err)
	}

	// итерируемся по import записям файла
	importSpecs := astutil.Imports(fset, sf)
	// импорты могут быть объединены в группы внутри круглых скобок
	for _, paragraph := range importSpecs {
		for _, importSpec := range paragraph {
			for _, knownImport := range knownPackages {
				// проверяем совпадение с искомым импортом
				if strings.Contains(importSpec.Path.Value, knownImport) {
					return importSpec, nil
				}
			}
		}
	}

	return nil, nil
}

// dialContextFunc является сигнатурой функции, которую можно передать в (*http.Transport).DialContext
type dialContextFunc = func(ctx context.Context, network, addr string) (net.Conn, error)

// mockResolver возращает функцию dialContextFunc, которая перехватывает запрос на определение
// доменного имени и подменяет результат
func mockResolver(network, requestAddress, responseIP string) dialContextFunc {
	dialer := &net.Dialer{
		Timeout:   time.Second,
		KeepAlive: 30 * time.Second,
	}
	return func(ctx context.Context, net, addr string) (net.Conn, error) {
		if net == network && addr == requestAddress {
			addr = responseIP
		}
		return dialer.DialContext(ctx, net, addr)
	}
}

// dumpRequest - это httputil.DumpRequest, который возвращает только байты запроса
func dumpRequest(req *http.Request, body bool) (dump []byte) {
	if req != nil {
		dump, _ = httputil.DumpRequest(req, body)
	}
	return
}

// dumpResponse - это httputil.DumpResponse, который возвращает только байты ответа
func dumpResponse(resp *http.Response, body bool) (dump []byte) {
	if resp != nil {
		dump, _ = httputil.DumpResponse(resp, body)
	}
	return
}
