// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package blank

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
	"strings"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "blank",
	Doc: `detects blank identifiers that can be removed from function declarations.

This analyzer finds method receivers named _ and parameter lists where every parameter
is named _. Both carry no information: Go allows an anonymous receiver (func (*T) M())
and an unnamed parameter list (func f(int, string)). Each diagnostic includes a
suggested fix that removes the blank names.`,
	Run: run,
}

func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			checkReceiver(pass, fd)
			checkParams(pass, fd)
		}
	}
	return nil, nil
}

func checkReceiver(pass *analysis.Pass, fd *ast.FuncDecl) {
	if fd.Recv == nil || len(fd.Recv.List) != 1 {
		return
	}
	field := fd.Recv.List[0]
	if len(field.Names) != 1 || field.Names[0].Name != "_" {
		return
	}
	name := field.Names[0]
	pass.Report(analysis.Diagnostic{
		Pos:     name.Pos(),
		End:     name.End(),
		Message: "blank receiver can be removed",
		SuggestedFixes: []analysis.SuggestedFix{{
			Message: "Remove blank receiver",
			TextEdits: []analysis.TextEdit{{
				Pos: name.Pos(),
				End: field.Type.Pos(),
			}},
		}},
	})
}

func checkParams(pass *analysis.Pass, fd *ast.FuncDecl) {
	params := fd.Type.Params
	if params == nil || len(params.List) == 0 {
		return
	}
	for _, field := range params.List {
		if len(field.Names) == 0 {
			return
		}
		for _, name := range field.Names {
			if name.Name != "_" {
				return
			}
		}
	}
	var edits []analysis.TextEdit
	for _, field := range params.List {
		if len(field.Names) == 1 {
			edits = append(edits, analysis.TextEdit{
				Pos: field.Names[0].Pos(),
				End: field.Type.Pos(),
			})
			continue
		}
		// A multi-name field like "_, _ int" must expand to one type per name.
		typeStr := nodeString(pass.Fset, field.Type)
		expanded := make([]string, len(field.Names))
		for i := range expanded {
			expanded[i] = typeStr
		}
		edits = append(edits, analysis.TextEdit{
			Pos:     field.Pos(),
			End:     field.End(),
			NewText: []byte(strings.Join(expanded, ", ")),
		})
	}
	pass.Report(analysis.Diagnostic{
		Pos:     params.Pos(),
		End:     params.End(),
		Message: "all parameters are blank; parameter names can be removed",
		SuggestedFixes: []analysis.SuggestedFix{{
			Message:   "Remove blank parameter names",
			TextEdits: edits,
		}},
	})
}

func nodeString(fset *token.FileSet, node ast.Node) string {
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, node); err != nil {
		return ""
	}
	return buf.String()
}
