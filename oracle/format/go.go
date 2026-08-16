// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package format

import (
	"bytes"
	"context"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
)

// Go is a Formatter that runs `gofmt -s` over its input via stdin/stdout,
// first collapsing a parenthesized single-import declaration to the unwrapped
// form gofmt itself never rewrites.
type Go struct{}

// NewGo returns a Go formatter.
func NewGo() *Go { return &Go{} }

// Format runs gofmt -s with content on stdin.
func (g *Go) Format(ctx context.Context, content []byte, _ string) ([]byte, error) {
	content = unwrapSingleImport(content)
	return stdinRun{Name: "gofmt", Args: []string{"-s"}, Stdin: content}.run(ctx)
}

// unwrapSingleImport rewrites `import ( x "path" )` to `import x "path"` when
// the declaration holds exactly one spec. Content that fails to parse is
// returned unchanged so gofmt reports the real error.
func unwrapSingleImport(content []byte) []byte {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", content, parser.ParseComments)
	if err != nil {
		return content
	}
	changed := false
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.IMPORT {
			continue
		}
		if gd.Lparen.IsValid() && len(gd.Specs) == 1 {
			gd.Lparen = token.NoPos
			gd.Rparen = token.NoPos
			changed = true
		}
	}
	if !changed {
		return content
	}
	var buf bytes.Buffer
	if err := (&printer.Config{
		Mode:     printer.UseSpaces | printer.TabIndent,
		Tabwidth: 8,
	}).Fprint(&buf, fset, f); err != nil {
		return content
	}
	return buf.Bytes()
}
