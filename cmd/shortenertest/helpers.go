package main

import (
	"crypto/rand"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math/big"
	mathrand "math/rand"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/ast/astutil"
)

// generateTestURL returns valid random URL
func generateTestURL(t *testing.T) string {
	t.Helper()

	// generate PROTO
	proto := "http://"
	if mathrand.Float32() < 0.5 {
		proto = "https://"
	}

	// generate DOMAIN
	var letters = "0123456789abcdefghijklmnopqrstuvwxyz"

	minLen, maxLen := 5, 15
	domainLen := mathrand.Intn(maxLen-minLen) + minLen

	lettersLen := big.NewInt(int64(len(letters)))

	ret := make([]byte, domainLen)
	for i := 0; i < domainLen; i++ {
		num, err := rand.Int(rand.Reader, lettersLen)
		require.NoError(t, err)
		ret[i] = letters[num.Int64()]
	}
	domain := string(ret)

	// generate ZONE
	var zones = []string{".com", ".ru", ".net", ".biz", ".yandex"}
	zone := zones[mathrand.Intn(len(zones))]

	return proto + domain + zone
}

// importsKnownPackage checks if given file imports
func importsKnownPackage(t *testing.T, filepath string, knownPackages []string) (*ast.ImportSpec, error) {
	t.Helper()

	fset := token.NewFileSet()
	sf, err := parser.ParseFile(fset, filepath, nil, parser.ImportsOnly)
	if err != nil {
		return nil, fmt.Errorf("cannot parse file: %w", err)
	}

	importSpecs := astutil.Imports(fset, sf)
	for _, paragraph := range importSpecs {
		for _, importSpec := range paragraph {
			for _, knownImport := range knownPackages {
				if strings.Contains(importSpec.Path.Value, knownImport) {
					return importSpec, nil
				}
			}
		}
	}

	return nil, nil
}
