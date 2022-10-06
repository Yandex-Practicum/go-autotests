package main

// Basic imports
import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/tools/go/ast/inspector"
)

// Iteration17Suite является сьютом с тестами и состоянием для инкремента
type Iteration17Suite struct {
	suite.Suite
}

// SetupSuite подготавливает необходимые зависимости
func (suite *Iteration17Suite) SetupSuite() {
	// check required flags
	suite.Require().NotEmpty(flagTargetSourcePath, "-source-path non-empty flag required")
}

// TestDocsComments пробует проверить налиция документационных комментариев в коде
func (suite *Iteration17Suite) TestDocsComments() {
	var undocumentedFiles []string
	err := filepath.WalkDir(flagTargetSourcePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			// пропускаем служебные директории
			if d.Name() == "vendor" || d.Name() == ".git" {
				return filepath.SkipDir
			}
			// проваливаемся в директорию
			return nil
		}

		// проверяем только Go файлы, но не тесты
		if strings.HasSuffix(d.Name(), ".go") &&
			!strings.HasSuffix(d.Name(), "_test.go") &&
			undocumentedFile(suite.T(), path) {
			// сохраняем плохой файл в слайс
			undocumentedFiles = append(undocumentedFiles, path)
		}

		return nil
	})

	suite.NoError(err, "Неожиданная ошибка")
	suite.Emptyf(undocumentedFiles,
		"Найдены файлы с недокументированной сущностями:\n\n%s",
		strings.Join(undocumentedFiles, "\n"),
	)
}

// TestExamplePresence пробует рекурсивно найти хотя бы один файл example_test.go в директории с исходным кодом проекта
func (suite *Iteration17Suite) TestExamplePresence() {
	err := filepath.WalkDir(flagTargetSourcePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			// пропускаем служебные директории
			if d.Name() == "vendor" || d.Name() == ".git" {
				return filepath.SkipDir
			}
			// проваливаемся в директорию
			return nil
		}

		// проверяем имя файла
		if strings.HasSuffix(d.Name(), "example_test.go") {
			// возвращаем сигнальную ошибку
			return errUsageFound
		}

		return nil
	})

	// проверяем сигнальную ошибку
	if errors.Is(err, errUsageFound) {
		// найден хотя бы один файл
		return
	}

	if err == nil {
		suite.T().Error("Не найден ни один файл example_test.go")
		return
	}
	suite.T().Errorf("Неожиданная ошибка при поиске файла example_test.go: %s", err)
}

func undocumentedFile(t *testing.T, filepath string) bool {
	t.Helper()

	fset := token.NewFileSet()
	sf, err := parser.ParseFile(fset, filepath, nil, parser.ParseComments)
	require.NoError(t, err)

	ins := inspector.New([]*ast.File{sf})
	nodeFilter := []ast.Node{
		(*ast.GenDecl)(nil),
		(*ast.FuncDecl)(nil),
	}

	var undocumentedFound bool
	ins.Nodes(nodeFilter, func(node ast.Node, push bool) (proceed bool) {
		switch nt := node.(type) {
		case *ast.GenDecl:
			if undocumentedGenDecl(nt) {
				undocumentedFound = true
			}
		case *ast.FuncDecl:
			if nt.Name.IsExported() && nt.Doc == nil {
				undocumentedFound = true
			}
		}

		return !undocumentedFound
	})

	return undocumentedFound
}

// undocumentedGenDecl проверяет, что экспортированная декларация является недокументированной
func undocumentedGenDecl(decl *ast.GenDecl) bool {
	for _, spec := range decl.Specs {
		switch st := spec.(type) {
		case *ast.TypeSpec:
			if st.Name.IsExported() && decl.Doc == nil {
				return true
			}
		case *ast.ValueSpec:
			for _, name := range st.Names {
				if name.IsExported() && decl.Doc == nil {
					return true
				}
			}
		}
	}
	return false
}
