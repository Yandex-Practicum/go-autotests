package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/go/ast/astutil"
)

var (
	importFound = errors.New("known import found")

	knownHTTPFrameworks = []string{
		"aahframework.org",
		"confetti-framework.com",
		"github.com/abahmed/gearbox",
		"github.com/aerogo/aero",
		"github.com/aisk/vox",
		"github.com/ant0ine/go-json-rest",
		"github.com/aofei/air",
		"github.com/appist/appy",
		"github.com/astaxie/beego",
		"github.com/beatlabs/patron",
		"github.com/bnkamalesh/webgo",
		"github.com/buaazp/fasthttprouter",
		"github.com/claygod/Bxog",
		"github.com/claygod/microservice",
		"github.com/dimfeld/httptreemux",
		"github.com/dinever/golf",
		"github.com/fulldump/golax",
		"github.com/gernest/alien",
		"github.com/gernest/utron",
		"github.com/gin-gonic/gin",
		"github.com/go-chi/chi",
		"github.com/go-goyave/goyave",
		"github.com/go-macaron/macaron",
		"github.com/go-ozzo/ozzo-routing",
		"github.com/go-playground/lars",
		"github.com/go-playground/pure",
		"github.com/go-zoo/bone",
		"github.com/goa-go/goa",
		"github.com/goadesign/goa",
		"github.com/goanywhere/rex",
		"github.com/gocraft/web",
		"github.com/gofiber/fiber",
		"github.com/goji/goji",
		"github.com/gookit/rux",
		"github.com/gorilla/mux",
		"github.com/goroute/route",
		"github.com/gotuna/gotuna",
		"github.com/gowww/router",
		"github.com/GuilhermeCaruso/bellt",
		"github.com/hidevopsio/hiboot",
		"github.com/husobee/vestigo",
		"github.com/i-love-flamingo/flamingo",
		"github.com/i-love-flamingo/flamingo-commerce",
		"github.com/ivpusic/neo",
		"github.com/julienschmidt/httprouter",
		"github.com/labstack/echo",
		"github.com/lunny/tango",
		"github.com/mustafaakin/gongular",
		"github.com/nbari/violetear",
		"github.com/nsheremet/banjo",
		"github.com/NYTimes/gizmo",
		"github.com/paulbellamy/mango",
		"github.com/rainycape/gondola",
		"github.com/razonyang/fastrouter",
		"github.com/rcrowley/go-tigertonic",
		"github.com/resoursea/api",
		"github.com/revel/revel",
		"github.com/rs/xmux",
		"github.com/twharmon/goweb",
		"github.com/uadmin/uadmin",
		"github.com/ungerik/go-rest",
		"github.com/vardius/gorouter",
		"github.com/VividCortex/siesta",
		"github.com/xujiajun/gorouter",
		"github.com/xxjwxc/ginrpc",
		"github.com/yarf-framework/yarf",
		"github.com/zpatrick/fireball",
		"gobuffalo.io",
		"rest-layer.io",
	}
)

// TestIteration3 checks that students code uses known 3rd party HTTP framework
func TestIteration3(t *testing.T) {
	fset := token.NewFileSet()

	err := filepath.WalkDir(config.SourceRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			// skip vendor directory
			if d.Name() == "vendor" || d.Name() == ".git" {
				return filepath.SkipDir
			}
			// dive into regular directory
			return nil
		}

		// skip test files or non-Go files
		if !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}

		sf, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return fmt.Errorf("cannot parse AST of file: %s: %w", path, err)
		}

		importSpecs := astutil.Imports(fset, sf)
		if importsKnownHTTPFrameworks(importSpecs) {
			return importFound
		}

		return nil
	})

	if errors.Is(err, importFound) {
		return
	}

	if err == nil {
		t.Error("No import of known HTTP framework has been found")
		return
	}

	t.Errorf("unexpected error: %s", err)
}

func importsKnownHTTPFrameworks(imports [][]*ast.ImportSpec) bool {
	for _, paragraph := range imports {
		for _, importSpec := range paragraph {
			for _, knownImport := range knownHTTPFrameworks {
				if strings.Contains(importSpec.Path.Value, knownImport) {
					return true
				}
			}
		}
	}
	return false
}