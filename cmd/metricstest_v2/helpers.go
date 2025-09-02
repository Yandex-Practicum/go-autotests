package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"net/http"
	"net/http/httputil"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/go/ast/astutil"
)

// PackageRules это набор правил для поиска используемых пакетов
type PackageRules []PackageRule

// PackageList возвращает строковый список пакетов из правил
func (p PackageRules) PackageList() string {
	var b strings.Builder
	for i, r := range p {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(r.Name)
	}
	return b.String()
}

// PackageRule это правило для поиска используемого пакета
type PackageRule struct {
	// имя пакета для поиска
	Name string
	// разрешать ли импортирование через "_"
	AllowBlank bool
}

// usesKnownPackage проверяет, что хотя бы в одном файле, начиная с указанной директории rootdir,
// содержится хотя бы один пакет из списка knownPackages
func usesKnownPackage(t *testing.T, rootdir string, rules ...PackageRule) error {
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

		// проверяем файл на наличие искомых пакетов
		return importsKnownPackage(t, path, rules...)
	})

	// рекурсия не вернула никакой ошибки = мы не нашли искомого импорта ни в одном файле
	if err == nil {
		// возвращаем специализированную ошибку
		return errUsageNotFound
	}
	// здесь мы возращаем либо специализированную ошибку errUsageFound, либо любую другую неизвестную ошибку
	return err
}

// importsKnownPackage возвращает import запись первого найденного импорта из списка knownPackages в файле filepath
func importsKnownPackage(t *testing.T, filepath string, rules ...PackageRule) error {
	t.Helper()

	// парсим файл с исходным кодом
	fset := token.NewFileSet()
	sf, err := parser.ParseFile(fset, filepath, nil, parser.ImportsOnly)
	if err != nil {
		return fmt.Errorf("невозможно распарсить файл: %w", err)
	}

	// итерируемся по import записям файла
	importSpecs := astutil.Imports(fset, sf)
	// импорты могут быть объединены в группы внутри круглых скобок
	for _, paragraph := range importSpecs {
		for _, importSpec := range paragraph {
			for _, rule := range rules {
				// пропускаем не подходящий импорт
				if !strings.Contains(importSpec.Path.Value, rule.Name) {
					continue
				}
				// найден "пустой" импорт, хотя это запрещено правилом
				if importSpec.Name != nil && importSpec.Name.String() == "_" && !rule.AllowBlank {
					return nil
				}
				// возвращаем специализированную ошибку, сообщающую о нахождении импорта
				return errUsageFound
			}
		}
	}

	return nil
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
