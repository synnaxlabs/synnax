// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mustsucceedlint

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"

	"golang.org/x/tools/go/analysis"
)

const testutilImport = `"github.com/synnaxlabs/x/testutil"`

var Analyzer = &analysis.Analyzer{
	Name: "mustsucceedlint",
	Doc: `detects Expect(err).ToNot(HaveOccurred()) patterns that can be replaced with MustSucceed.

This analyzer finds consecutive statements where an error is assigned from a function
call and then checked with Expect(err).ToNot(HaveOccurred()). These can be simplified:

  result, err := f()                    →  result := MustSucceed(f())
  Expect(err).ToNot(HaveOccurred())

  err := f()                            →  Expect(f()).To(Succeed())
  Expect(err).ToNot(HaveOccurred())`,
	Run: run,
}

func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		info := fileImportInfo(file)
		ast.Inspect(file, func(n ast.Node) bool {
			block, ok := n.(*ast.BlockStmt)
			if !ok {
				return true
			}
			analyzeBlock(pass, block.List, info)
			return true
		})
	}
	return nil, nil
}

// importInfo holds pre-computed import information for a file.
type importInfo struct {
	hasTestutil bool
	// insertPos is the position where a new import line should be inserted. This is
	// right before the closing paren of the import block, or at the end of a single
	// import declaration.
	insertPos token.Pos
	// insertText is the text to insert (includes newline and tab).
	insertText string
}

func fileImportInfo(file *ast.File) importInfo {
	info := importInfo{}
	for _, imp := range file.Imports {
		if imp.Path.Value == testutilImport {
			info.hasTestutil = true
			// Check if it's a dot import
			if imp.Name != nil && imp.Name.Name == "." {
				info.hasTestutil = true
			}
			return info
		}
	}
	// Find the import block to determine insertion point.
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.IMPORT {
			continue
		}
		if genDecl.Lparen.IsValid() {
			// Grouped import: insert before closing paren.
			info.insertPos = genDecl.Rparen
			info.insertText = "\t. " + testutilImport + "\n"
		} else if len(genDecl.Specs) > 0 {
			// Single import: insert after the existing import line.
			info.insertPos = genDecl.End()
			info.insertText = "\nimport . " + testutilImport
		}
		break
	}
	if !info.insertPos.IsValid() {
		// No import block at all: insert after the package clause.
		info.insertPos = file.Name.End()
		info.insertText = "\n\nimport . " + testutilImport
	}
	return info
}

func analyzeBlock(pass *analysis.Pass, stmts []ast.Stmt, info importInfo) {
	for i := 1; i < len(stmts); i++ {
		expectStmt, ok := stmts[i].(*ast.ExprStmt)
		if !ok {
			continue
		}
		errName, isExpectErr := matchExpectErrNotHaveOccurred(expectStmt.X)
		if !isExpectErr {
			continue
		}
		assignStmt, ok := stmts[i-1].(*ast.AssignStmt)
		if !ok {
			continue
		}
		if !assignsToName(assignStmt, errName) {
			continue
		}
		if len(assignStmt.Rhs) != 1 {
			continue
		}
		rhsCall, ok := assignStmt.Rhs[0].(*ast.CallExpr)
		if !ok {
			continue
		}
		reportDiagnostic(pass, assignStmt, expectStmt, rhsCall, errName, info)
	}
}

// matchExpectErrNotHaveOccurred checks if an expression matches:
//
//	Expect(<name>).ToNot(HaveOccurred())
//	Expect(<name>).To(Not(HaveOccurred()))
//
// Returns the name used in Expect() and true if matched.
func matchExpectErrNotHaveOccurred(expr ast.Expr) (string, bool) {
	outerCall, ok := expr.(*ast.CallExpr)
	if !ok {
		return "", false
	}
	sel, ok := outerCall.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	switch sel.Sel.Name {
	case "ToNot", "NotTo":
		if len(outerCall.Args) != 1 {
			return "", false
		}
		if !isCallTo(outerCall.Args[0], "HaveOccurred", 0) {
			return "", false
		}
	case "To":
		if len(outerCall.Args) != 1 {
			return "", false
		}
		notCall, ok := outerCall.Args[0].(*ast.CallExpr)
		if !ok {
			return "", false
		}
		if !isNamedFunc(notCall, "Not") || len(notCall.Args) != 1 {
			return "", false
		}
		if !isCallTo(notCall.Args[0], "HaveOccurred", 0) {
			return "", false
		}
	default:
		return "", false
	}
	expectCall, ok := sel.X.(*ast.CallExpr)
	if !ok {
		return "", false
	}
	if !isNamedFunc(expectCall, "Expect") && !isNamedFunc(expectCall, "ExpectWithOffset") {
		return "", false
	}
	var errArg ast.Expr
	if isNamedFunc(expectCall, "Expect") {
		if len(expectCall.Args) != 1 {
			return "", false
		}
		errArg = expectCall.Args[0]
	} else {
		if len(expectCall.Args) != 2 {
			return "", false
		}
		errArg = expectCall.Args[1]
	}
	ident, ok := errArg.(*ast.Ident)
	if !ok {
		return "", false
	}
	return ident.Name, true
}

func isCallTo(expr ast.Expr, name string, nArgs int) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	return isNamedFunc(call, name) && len(call.Args) == nArgs
}

func isNamedFunc(call *ast.CallExpr, name string) bool {
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		return fn.Name == name
	case *ast.SelectorExpr:
		return fn.Sel.Name == name
	}
	return false
}

func assignsToName(assign *ast.AssignStmt, name string) bool {
	for _, lhs := range assign.Lhs {
		if ident, ok := lhs.(*ast.Ident); ok && ident.Name == name {
			return true
		}
	}
	return false
}

func reportDiagnostic(
	pass *analysis.Pass,
	assign *ast.AssignStmt,
	expectStmt *ast.ExprStmt,
	rhsCall *ast.CallExpr,
	errName string,
	info importInfo,
) {
	callStr := nodeString(pass.Fset, rhsCall)
	numLHS := len(assign.Lhs)
	errIdx := -1
	for i, lhs := range assign.Lhs {
		if ident, ok := lhs.(*ast.Ident); ok && ident.Name == errName {
			errIdx = i
			break
		}
	}
	var (
		msg         string
		fixText     string
		needsImport bool
	)
	if numLHS == 1 && errIdx == 0 {
		msg = fmt.Sprintf(
			"Expect(%s).ToNot(HaveOccurred()) can be replaced with Expect(%s).To(Succeed())",
			errName, callStr,
		)
		fixText = fmt.Sprintf("Expect(%s).To(Succeed())", callStr)
	} else if numLHS == 2 && errIdx == 1 {
		resultName := nodeString(pass.Fset, assign.Lhs[0])
		tok := ":="
		if assign.Tok == token.ASSIGN {
			tok = "="
		}
		msg = fmt.Sprintf(
			"Expect(%s).ToNot(HaveOccurred()) can be replaced with MustSucceed",
			errName,
		)
		fixText = fmt.Sprintf("%s %s MustSucceed(%s)", resultName, tok, callStr)
		needsImport = true
	} else if numLHS == 3 && errIdx == 2 {
		r1 := nodeString(pass.Fset, assign.Lhs[0])
		r2 := nodeString(pass.Fset, assign.Lhs[1])
		tok := ":="
		if assign.Tok == token.ASSIGN {
			tok = "="
		}
		msg = fmt.Sprintf(
			"Expect(%s).ToNot(HaveOccurred()) can be replaced with MustSucceed2",
			errName,
		)
		fixText = fmt.Sprintf("%s, %s %s MustSucceed2(%s)", r1, r2, tok, callStr)
		needsImport = true
	} else {
		return
	}

	edits := []analysis.TextEdit{
		{Pos: assign.Pos(), End: expectStmt.End(), NewText: []byte(fixText)},
	}
	if needsImport && !info.hasTestutil && info.insertPos.IsValid() {
		edits = append(edits, analysis.TextEdit{
			Pos:     info.insertPos,
			End:     info.insertPos,
			NewText: []byte(info.insertText),
		})
	}

	pass.Report(analysis.Diagnostic{
		Pos:     assign.Pos(),
		End:     expectStmt.End(),
		Message: msg,
		SuggestedFixes: []analysis.SuggestedFix{
			{
				Message:   "Replace with MustSucceed",
				TextEdits: edits,
			},
		},
	})
}

func nodeString(fset *token.FileSet, node ast.Node) string {
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, node); err != nil {
		return ""
	}
	return buf.String()
}
