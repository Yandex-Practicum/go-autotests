package main

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Yandex-Practicum/go-autotests/internal/random"
	"github.com/goccy/go-yaml"
	yamlast "github.com/goccy/go-yaml/ast"
	yamlparser "github.com/goccy/go-yaml/parser"
	"github.com/stretchr/testify/suite"
)

//go:embed kuber_golden.yaml
var goldenYAML []byte

// Lesson02Suite является сьютом с тестами урока
type Lesson02Suite struct {
	suite.Suite
}

func (suite *Lesson02Suite) TestValidateYAML() {
	// проверяем наличие необходимых флагов
	suite.Require().NotEmpty(flagTargetBinaryPath, "-binary-path non-empty flag required")

	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	// сгененрируем новое содержимое YAML файла
	suite.T().Log("creating test YAML file")
	fpath, modifications, err := newYAMLFile(rnd)
	suite.Require().NoError(err, "cannot generate new YAML file content")

	// не забудем удалить за собой временный файл
	defer os.Remove(fpath)

	// запускаем бинарник скрипта
	suite.T().Log("creating process")
	binctx, bincancel := context.WithTimeout(context.Background(), time.Minute)
	defer bincancel()

	var scriptOut bytes.Buffer
	cmd := exec.CommandContext(binctx, flagTargetBinaryPath, fpath)
	cmd.Stdout = &scriptOut

	// ждем завершения скрипта
	var exiterr *exec.ExitError
	if err := cmd.Run(); errors.As(err, &exiterr) {
		suite.Require().NotEqualf(-1, exiterr.ExitCode(), "скрипт завершился аварийно, вывод:\n\n%s", scriptOut.String())
	}

	// соберем и отфильтруем вывод скрипта
	linesOut := strings.Split(scriptOut.String(), "\n")
	linesOut = slices.DeleteFunc(linesOut, func(line string) bool {
		return strings.TrimSpace(line) == ""
	})

	// проверим вывод скрипта
	var expectedMessages []string
	for _, modification := range modifications {
		expectedMessages = append(expectedMessages, modification.message)
	}

	matches := suite.Assert().ElementsMatch(expectedMessages, linesOut, "вывод скрипта (List B) не совпадает с ожидаемым (List A)")
	if !matches {
		content, err := os.ReadFile(fpath)
		suite.Require().NoError(err, "невозможно прочитать содержимое YAML файла")
		suite.T().Logf("Содержимое тестового YAML файла:\n\n%s\n", content)
	}
}

func newYAMLFile(rnd *rand.Rand) (fpath string, modifications []yamlModification, err error) {
	// сгенерируем случайное имя файла и путь
	fname := random.ASCIIString(5, 10) + ".yaml"
	fpath = filepath.Join(os.TempDir(), fname)

	// декодируем файл в промежуточное представление
	ast, err := yamlparser.ParseBytes(goldenYAML, 0)
	if err != nil {
		return "", nil, fmt.Errorf("cannot build YAML AST: %w", err)
	}

	// модифицируем YAML дерево
	modifications, err = applyYAMLModifications(rnd, ast)
	if err != nil {
		return "", nil, fmt.Errorf("cannot perform YAML tree modifications: %w", err)
	}
	// обогощаем информацию о модификациях
	for i, m := range modifications {
		m.message = fmt.Sprintf("%s:%d %s", fname, m.lineno, m.message)
		modifications[i] = m
	}

	// запишем модифицированные данные в файл
	if err := os.WriteFile(fpath, []byte(ast.String()), 0444); err != nil {
		return "", nil, fmt.Errorf("cannot write modified YAML file: %w", err)
	}
	return fpath, modifications, nil
}

type yamlModification struct {
	lineno  int
	message string
}

func applyYAMLModifications(rnd *rand.Rand, root *yamlast.File) ([]yamlModification, error) {
	if root == nil {
		return nil, errors.New("root YAML node expected")
	}

	funcs := []yamlModifierFunc{
		modifyYAMLNop, // с определенной вероятностью файл не будет модифицирован вообще
		modifyYAMLSpecOS,
		modifyYAMLRemoveRequired,
		modifyYAMLPortOutOfRange,
		modifyYAMLInvalidType,
	}

	rnd.Shuffle(len(funcs), func(i, j int) {
		funcs[i], funcs[j] = funcs[j], funcs[i]
	})

	modificationsCount := intInRange(rnd, 1, len(funcs))
	var modifications []yamlModification
	for _, fn := range funcs[:modificationsCount] {
		mods, err := fn(rnd, root)
		if err != nil {
			return nil, fmt.Errorf("cannot apply modification: %w", err)
		}
		modifications = append(modifications, mods...)
	}

	return modifications, nil
}

// yamlModifierFunc функция, которая умеет модифицировать одну или более ноду YAML дерева
type yamlModifierFunc func(rnd *rand.Rand, root *yamlast.File) ([]yamlModification, error)

// modifyYAMLNop не делает с YAML деревом ничего
func modifyYAMLNop(_ *rand.Rand, root *yamlast.File) ([]yamlModification, error) {
	return nil, nil
}

// modifyYAMLSpecOS заменяет значение `spec.os` на не валидное
func modifyYAMLSpecOS(_ *rand.Rand, root *yamlast.File) ([]yamlModification, error) {
	badValue := random.ASCIIString(3, 10)

	path, err := yaml.PathString("$.spec.os")
	if err != nil {
		return nil, fmt.Errorf("bad field path given: %w", err)
	}

	node, err := path.FilterFile(root)
	if err != nil {
		return nil, fmt.Errorf("cannot filter 'spec.os' node: %w", err)
	}

	lineno := node.GetToken().Position.Line
	path.ReplaceWithReader(root, strings.NewReader(badValue))
	return []yamlModification{
		{
			lineno:  lineno,
			message: fmt.Sprintf("%s has unsupported value '%s'", basename(node.GetPath()), badValue),
		},
	}, nil
}

// modifyYAMLRemoveRequired удаляет случайную обязательную ноду
func modifyYAMLRemoveRequired(rnd *rand.Rand, root *yamlast.File) ([]yamlModification, error) {
	paths := []string{
		"$.spec.containers[0].name",
		"$.metadata.name",
	}

	path, err := yaml.PathString(paths[rnd.Intn(len(paths))])
	if err != nil {
		return nil, fmt.Errorf("bad field path given: %w", err)
	}

	node, err := path.FilterFile(root)
	if err != nil {
		return nil, fmt.Errorf("cannot filter node by path '%s': %w", path, err)
	}

	lineno := node.GetToken().Position.Line
	path.ReplaceWithReader(root, strings.NewReader(`""`))
	return []yamlModification{
		{
			lineno:  lineno,
			message: fmt.Sprintf("%s is required", basename(node.GetPath())),
		},
	}, nil
}

// modifyYAMLPortOutOfRange устанавливает значение порта за пределами границ
func modifyYAMLPortOutOfRange(rnd *rand.Rand, root *yamlast.File) ([]yamlModification, error) {
	paths := []string{
		"$.spec.containers[0].ports[0].containerPort",
		"$.spec.containers[0].readinessProbe.httpGet.port",
		"$.spec.containers[0].livenessProbe.httpGet.port",
	}

	port := rnd.Intn(100000)
	if port < 65536 {
		port *= -1
	}

	path, err := yaml.PathString(paths[rnd.Intn(len(paths))])
	if err != nil {
		return nil, fmt.Errorf("bad field path given: %w", err)
	}

	node, err := path.FilterFile(root)
	if err != nil {
		return nil, fmt.Errorf("cannot filter node by path '%s': %w", path, err)
	}

	lineno := node.GetToken().Position.Line
	path.ReplaceWithReader(root, strings.NewReader(fmt.Sprint(port)))
	return []yamlModification{
		{
			lineno:  lineno,
			message: fmt.Sprintf("%s value out of range", basename(node.GetPath())),
		},
	}, nil
}

// modifyYAMLInvalidType меняет тип на недопустимый
func modifyYAMLInvalidType(rnd *rand.Rand, root *yamlast.File) ([]yamlModification, error) {
	paths := []string{
		"$.spec.containers[0].resources.limits.cpu",
		"$.spec.containers[0].resources.requests.cpu",
	}

	path, err := yaml.PathString(paths[rnd.Intn(len(paths))])
	if err != nil {
		return nil, fmt.Errorf("bad field path given: %w", err)
	}

	node, err := path.FilterFile(root)
	if err != nil {
		return nil, fmt.Errorf("cannot filter node by path '%s': %w", path, err)
	}

	lineno := node.GetToken().Position.Line
	path.ReplaceWithReader(root, strings.NewReader(`"`+node.String()+`"`))
	return []yamlModification{
		{
			lineno:  lineno,
			message: fmt.Sprintf("%s must be int", basename(node.GetPath())),
		},
	}, nil
}

func basename(path string) string {
	idx := strings.LastIndex(path, ".")
	if idx == -1 {
		return path
	}
	return path[idx+1:]
}
