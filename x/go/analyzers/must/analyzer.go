// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package must

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"go/types"

	"github.com/synnaxlabs/x/set"
	"golang.org/x/tools/go/analysis"
)

const testutilImport = `"github.com/synnaxlabs/x/testutil"`

var Analyzer = &analysis.Analyzer{
	Name: "must",
	Doc: `detects errors that a test captures in a variable before it asserts on them.

This analyzer finds statements where an error is assigned from a function call and then
checked with Expect(err). These can assert on the call itself:

  result, err := f()                    →  result := MustSucceed(f())
  Expect(err).ToNot(HaveOccurred())

  err := f()                            →  Expect(f()).To(Succeed())
  Expect(err).To(Succeed())

  _, err := f()                         →  Expect(f()).Error().To(MatchError(e))
  Expect(err).To(MatchError(e))

Error() also asserts that every other result is zero. The Expect(err) check does not
need to be immediately adjacent to the assignment; the analyzer walks backwards to find
the binding assignment as long as no intervening statement references err.`,
	Run: run,
}

// asyncGomegaFuncs are Gomega functions whose callbacks should NOT be rewritten to use
// MustSucceed, because these functions retry on failure and MustSucceed would panic
// instead of allowing the retry.
var asyncGomegaFuncs = set.New("Eventually", "Consistently")

func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		info := fileImportInfo(file)
		var needsImport bool
		skipFuncLits := collectAsyncFuncLits(file)
		ast.Inspect(file, func(n ast.Node) bool {
			block, ok := n.(*ast.BlockStmt)
			if !ok {
				return true
			}
			if skipFuncLits.Contains(block) {
				return false
			}
			if analyzeBlock(pass, block.List) {
				needsImport = true
			}
			return true
		})
		if needsImport && !info.hasTestutil && info.insertPos.IsValid() {
			pass.Report(analysis.Diagnostic{
				Pos:     info.insertPos,
				End:     info.insertPos,
				Message: "missing testutil import for MustSucceed",
				SuggestedFixes: []analysis.SuggestedFix{
					{
						Message: "Add testutil import",
						TextEdits: []analysis.TextEdit{
							{
								Pos:     info.insertPos,
								End:     info.insertPos,
								NewText: []byte(info.insertText),
							},
						},
					},
				},
			})
		}
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
			// Only treat as "has testutil" if it's a dot import, since
			// the fixes emit unqualified MustSucceed calls.
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

// collectAsyncFuncLits walks the file and returns the set of BlockStmts belonging to
// FuncLits that are arguments to Eventually/Consistently. These blocks should be
// skipped because those Gomega helpers retry on failure, and MustSucceed would panic
// instead of allowing retries.
func collectAsyncFuncLits(file *ast.File) set.Set[*ast.BlockStmt] {
	skip := make(set.Set[*ast.BlockStmt])
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		name := ""
		switch fn := call.Fun.(type) {
		case *ast.Ident:
			name = fn.Name
		case *ast.SelectorExpr:
			name = fn.Sel.Name
		}
		if !asyncGomegaFuncs.Contains(name) {
			return true
		}
		for _, arg := range call.Args {
			if funcLit, ok := arg.(*ast.FuncLit); ok {
				skip.Add(funcLit.Body)
			}
		}
		return true
	})
	return skip
}

// analyzeBlock returns true if any MustSucceed replacement was emitted.
func analyzeBlock(pass *analysis.Pass, stmts []ast.Stmt) bool {
	var found bool
	for i := 1; i < len(stmts); i++ {
		expectStmt, ok := stmts[i].(*ast.ExprStmt)
		if !ok {
			continue
		}
		if errName, ok := matchExpectErrSucceeds(expectStmt.X); ok {
			assignStmt, assignIdx, rhsCall := findBinding(stmts, i, errName)
			if assignStmt != nil && reportDiagnostic(
				pass,
				assignStmt,
				expectStmt,
				rhsCall,
				errName,
				assignIdx == i-1,
			) {
				found = true
			}
			continue
		}
		m, ok := matchExpectErrMatches(expectStmt.X)
		if !ok {
			continue
		}
		assignStmt, assignIdx, rhsCall := findBinding(stmts, i, m.errName)
		if assignStmt == nil ||
			!bindsOnlyError(pass, assignStmt, m.errName) ||
			usesIdentIn(stmts[i+1:], m.errName) {
			continue
		}
		reportInlineDiagnostic(
			pass,
			assignStmt,
			expectStmt,
			rhsCall,
			m,
			assignIdx == i-1,
		)
	}
	return found
}

// usesIdent reports whether any identifier with the given name appears anywhere inside
// the given AST node. Used to check whether intervening statements between an
// assignment and an Expect(err) call still reference err.
//
// The match is purely by identifier name, not by lexical scope or types.Object — an
// unrelated identifier that happens to share name (a struct field accessed as obj.err,
// a label, a parameter in a nested function literal) will also stop the backward walk.
// This is a deliberate, conservative trade-off: a stricter scope-aware check would
// require running the analyzer with type information, and the cost of a false-negative
// here is only that a valid MustSucceed rewrite is skipped — never that an incorrect
// rewrite is emitted.
func usesIdent(node ast.Node, name string) bool {
	var found bool
	ast.Inspect(node, func(n ast.Node) bool {
		if found {
			return false
		}
		if ident, ok := n.(*ast.Ident); ok && ident.Name == name {
			found = true
			return false
		}
		return true
	})
	return found
}

// usesIdentIn reports whether any of the given statements references name.
func usesIdentIn(stmts []ast.Stmt, name string) bool {
	for _, s := range stmts {
		if usesIdent(s, name) {
			return true
		}
	}
	return false
}

// findBinding walks backwards from stmts[i] to the assignment that binds name from a
// single call. It stops at the first statement that references name, since name must
// stay bound for that statement.
func findBinding(
	stmts []ast.Stmt,
	i int,
	name string,
) (*ast.AssignStmt, int, *ast.CallExpr) {
	for j := i - 1; j >= 0; j-- {
		if a, ok := stmts[j].(*ast.AssignStmt); ok && assignsToName(a, name) {
			if len(a.Rhs) != 1 {
				return nil, -1, nil
			}
			call, ok := a.Rhs[0].(*ast.CallExpr)
			if !ok {
				return nil, -1, nil
			}
			return a, j, call
		}
		if usesIdent(stmts[j], name) {
			return nil, -1, nil
		}
	}
	return nil, -1, nil
}

// matchExpectErrSucceeds checks if an expression matches:
//
//	Expect(<name>).ToNot(HaveOccurred())
//	Expect(<name>).To(Not(HaveOccurred()))
//	Expect(<name>).To(Succeed())
//	Expect(<name>).Should(Succeed())
//
// Returns the name used in Expect() and true if matched.
func matchExpectErrSucceeds(expr ast.Expr) (string, bool) {
	outerCall, ok := expr.(*ast.CallExpr)
	if !ok {
		return "", false
	}
	sel, ok := outerCall.Fun.(*ast.SelectorExpr)
	if !ok || len(outerCall.Args) != 1 {
		return "", false
	}
	arg := outerCall.Args[0]
	switch sel.Sel.Name {
	case "ToNot", "NotTo":
		if !isCallTo(arg, "HaveOccurred", 0) {
			return "", false
		}
	case "To", "Should":
		if isCallTo(arg, "Succeed", 0) {
			break
		}
		notCall, ok := arg.(*ast.CallExpr)
		if !ok || sel.Sel.Name != "To" {
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
	return expectedIdent(sel.X)
}

// errorMatchers are the matchers that fail on a nil error, so asserting them through
// Expect(f()).Error() keeps the meaning of the original assertion.
var errorMatchers = set.New(
	"MatchError",
	"SatisfyAll",
	"SatisfyAny",
	"And",
	"Or",
	"Equal",
)

// errMatch is an Expect(<errName>).<verb>(<matcher>) assertion on a captured error.
type errMatch struct {
	errName string
	verb    string
	matcher *ast.CallExpr
}

// matchExpectErrMatches checks if an expression matches Expect(<name>).To(<m>) or
// Expect(<name>).Should(<m>), where <m> is one of errorMatchers.
func matchExpectErrMatches(expr ast.Expr) (errMatch, bool) {
	outerCall, ok := expr.(*ast.CallExpr)
	if !ok || len(outerCall.Args) != 1 {
		return errMatch{}, false
	}
	sel, ok := outerCall.Fun.(*ast.SelectorExpr)
	if !ok || (sel.Sel.Name != "To" && sel.Sel.Name != "Should") {
		return errMatch{}, false
	}
	matcher, ok := outerCall.Args[0].(*ast.CallExpr)
	if !ok || !isErrorMatcher(matcher) {
		return errMatch{}, false
	}
	expectCall, ok := sel.X.(*ast.CallExpr)
	if !ok || !isNamedFunc(expectCall, "Expect") || len(expectCall.Args) != 1 {
		return errMatch{}, false
	}
	ident, ok := expectCall.Args[0].(*ast.Ident)
	if !ok {
		return errMatch{}, false
	}
	return errMatch{errName: ident.Name, verb: sel.Sel.Name, matcher: matcher}, true
}

func isErrorMatcher(call *ast.CallExpr) bool {
	for name := range errorMatchers {
		if !isNamedFunc(call, name) {
			continue
		}
		if name == "Equal" {
			return len(call.Args) == 1 && !isNil(call.Args[0])
		}
		return len(call.Args) > 0
	}
	return false
}

func isNil(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "nil"
}

var errorType = types.Universe.Lookup("error").Type().Underlying().(*types.Interface)

// bindsOnlyError reports whether assign binds errName, of an error type, as its last
// value and discards every other value.
func bindsOnlyError(pass *analysis.Pass, assign *ast.AssignStmt, errName string) bool {
	if pass.TypesInfo == nil {
		return false
	}
	last, ok := assign.Lhs[len(assign.Lhs)-1].(*ast.Ident)
	if !ok || last.Name != errName {
		return false
	}
	t := pass.TypesInfo.TypeOf(last)
	if t == nil || !types.Implements(t, errorType) {
		return false
	}
	for _, lhs := range assign.Lhs[:len(assign.Lhs)-1] {
		if !isBlank(lhs) {
			return false
		}
	}
	return true
}

// reportInlineDiagnostic reports an Expect(err) assertion on a captured error that
// can assert on the call itself. A call with more than one result asserts through
// Error(), which also checks that every other result is zero.
func reportInlineDiagnostic(
	pass *analysis.Pass,
	assign *ast.AssignStmt,
	expectStmt *ast.ExprStmt,
	rhsCall *ast.CallExpr,
	m errMatch,
	adjacent bool,
) {
	callStr := nodeString(pass.Fset, rhsCall)
	errorCall := ""
	if len(assign.Lhs) > 1 {
		errorCall = ".Error()"
	}
	fixText := fmt.Sprintf(
		"Expect(%s)%s.%s(%s)",
		callStr,
		errorCall,
		m.verb,
		nodeString(pass.Fset, m.matcher),
	)
	pass.Report(analysis.Diagnostic{
		Pos: assign.Pos(),
		End: expectStmt.End(),
		Message: fmt.Sprintf(
			"assert the error inline with Expect(%s)%s.%s(...)",
			callStr,
			errorCall,
			m.verb,
		),
		SuggestedFixes: []analysis.SuggestedFix{
			{
				Message: "Assert the error inline",
				TextEdits: replacementEdits(
					pass.Fset,
					assign,
					expectStmt,
					fixText,
					adjacent,
				),
			},
		},
	})
}

// expectedIdent returns the name of the identifier that an Expect or ExpectWithOffset
// call asserts on.
func expectedIdent(expr ast.Expr) (string, bool) {
	expectCall, ok := expr.(*ast.CallExpr)
	if !ok {
		return "", false
	}
	if !isNamedFunc(expectCall, "Expect") &&
		!isNamedFunc(expectCall, "ExpectWithOffset") {
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

func isBlank(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "_"
}

func assignsToName(assign *ast.AssignStmt, name string) bool {
	for _, lhs := range assign.Lhs {
		if ident, ok := lhs.(*ast.Ident); ok && ident.Name == name {
			return true
		}
	}
	return false
}

// reportDiagnostic reports a single code replacement diagnostic. The adjacent flag
// indicates whether the assignment and Expect statements are directly adjacent in the
// source — when they are, a single text edit can rewrite both at once; otherwise the
// fix needs two edits (rewrite the assignment, delete the Expect line). Returns true if
// the replacement uses MustSucceed/MustSucceed2 (meaning the testutil import is
// needed).
func reportDiagnostic(
	pass *analysis.Pass,
	assign *ast.AssignStmt,
	expectStmt *ast.ExprStmt,
	rhsCall *ast.CallExpr,
	errName string,
	adjacent bool,
) bool {
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
			errName,
			callStr,
		)
		fixText = fmt.Sprintf("Expect(%s).To(Succeed())", callStr)
	} else if numLHS == 2 && errIdx == 1 {
		needsImport = true
		if isBlank(assign.Lhs[0]) {
			msg = fmt.Sprintf(
				"Expect(%s).ToNot(HaveOccurred()) can be replaced with MustSucceed",
				errName,
			)
			fixText = fmt.Sprintf("MustSucceed(%s)", callStr)
		} else {
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
		}
	} else if numLHS == 3 && errIdx == 2 {
		needsImport = true
		if isBlank(assign.Lhs[0]) && isBlank(assign.Lhs[1]) {
			msg = fmt.Sprintf(
				"Expect(%s).ToNot(HaveOccurred()) can be replaced with MustSucceed2",
				errName,
			)
			fixText = fmt.Sprintf("MustSucceed2(%s)", callStr)
		} else {
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
		}
	} else {
		return false
	}

	pass.Report(analysis.Diagnostic{
		Pos:     assign.Pos(),
		End:     expectStmt.End(),
		Message: msg,
		SuggestedFixes: []analysis.SuggestedFix{
			{
				Message: "Replace with MustSucceed",
				TextEdits: replacementEdits(
					pass.Fset,
					assign,
					expectStmt,
					fixText,
					adjacent,
				),
			},
		},
	})
	return needsImport
}

// replacementEdits replaces an assignment and its Expect statement with fixText. When
// the two are adjacent, one edit rewrites both. Otherwise, the assignment becomes
// fixText and the Expect statement's lines are deleted.
func replacementEdits(
	fset *token.FileSet,
	assign *ast.AssignStmt,
	expectStmt *ast.ExprStmt,
	fixText string,
	adjacent bool,
) []analysis.TextEdit {
	if adjacent {
		return []analysis.TextEdit{{
			Pos:     assign.Pos(),
			End:     expectStmt.End(),
			NewText: []byte(fixText),
		}}
	}
	deleteStart, deleteEnd := lineRange(fset, expectStmt)
	return []analysis.TextEdit{
		{Pos: assign.Pos(), End: assign.End(), NewText: []byte(fixText)},
		{Pos: deleteStart, End: deleteEnd},
	}
}

// lineRange returns the start of the first line that stmt occupies and the start of the
// line after the last line stmt occupies — i.e. the half-open range that covers every
// physical line of the statement, including leading whitespace and the trailing
// newline. Multi-line statements (e.g. an Expect chain split across lines) are covered
// in full so that deleting the range does not leave dangling tokens behind.
func lineRange(fset *token.FileSet, stmt ast.Stmt) (token.Pos, token.Pos) {
	f := fset.File(stmt.Pos())
	startLine := fset.Position(stmt.Pos()).Line
	endLine := fset.Position(stmt.End()).Line
	start := f.LineStart(startLine)
	var end token.Pos
	if endLine < f.LineCount() {
		end = f.LineStart(endLine + 1)
	} else {
		end = token.Pos(f.Base() + f.Size())
	}
	return start, end
}

func nodeString(fset *token.FileSet, node ast.Node) string {
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, node); err != nil {
		return ""
	}
	return buf.String()
}
