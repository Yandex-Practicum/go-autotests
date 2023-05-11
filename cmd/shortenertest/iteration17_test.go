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
	var reports []string
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

		// проускаем не Go файлы и Go тесты
		if !strings.HasSuffix(d.Name(), ".go") ||
			strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}

		reported := undocumentedNodes(suite.T(), path)
		if len(reported) > 0 {
			reports = append(reports, reported...)
		}

		return nil
	})

	suite.NoError(err, "Неожиданная ошибка")
	if len(reports) > 0 {
		suite.Failf("Найдены файлы с недокументированной сущностями",
			strings.Join(reports, "\n"),
		)
	}
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

func undocumentedNodes(t *testing.T, filepath string) []string {
	t.Helper()

	fset := token.NewFileSet()
	sf, err := parser.ParseFile(fset, filepath, nil, parser.ParseComments)
	require.NoError(t, err)

	// пропускаем автоматически сгенерированные файлы
	if isGenerated(sf) {
		return nil
	}

	var reports []string

	for _, decl := range sf.Decls {
		switch node := decl.(type) {
		case *ast.GenDecl:
			if undocumentedGenDecl(node) {
				reports = append(reports, fset.Position(node.Pos()).String())
			}
		case *ast.FuncDecl:
			if node.Name.IsExported() && node.Doc == nil {
				reports = append(reports, fset.Position(node.Pos()).String())
			}
		}
	}

	return reports
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

// isGenerated проверяет сгенерирован ли файл автоматически
// на основании правил, описанных в https://golang.org/s/generatedcode.
func isGenerated(file *ast.File) bool {
	const (
		genCommentPrefix = "// Code generated "
		genCommentSuffix = " DO NOT EDIT."
	)

	for _, group := range file.Comments {
		for _, comment := range group.List {
			if strings.HasPrefix(comment.Text, genCommentPrefix) &&
				strings.HasSuffix(comment.Text, genCommentSuffix) &&
				len(comment.Text) > len(genCommentPrefix)+len(genCommentSuffix) {
				return true
			}
		}
	}

	return false
}
