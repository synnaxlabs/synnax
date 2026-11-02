// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package errmatch

import (
	"go/ast"
	"go/types"

	"github.com/synnaxlabs/x/set"
	"golang.org/x/tools/go/analysis"
)

// Analyzer reports each Gomega assertion that checks only that an error occurred.
var Analyzer = &analysis.Analyzer{
	Name: "errmatch",
	Doc: `forbids error assertions that do not identify the error.

HaveOccurred(), Not(Succeed()), and Not(BeNil()) on an error pass when the code fails
for the wrong reason. Match the error instead: MatchError(sentinel),
MatchError(ContainSubstring("...")), or both through SatisfyAll(...).`,
	Run: run,
}

var (
	positiveAsserts = set.New("To", "Should")
	negativeAsserts = set.New("ToNot", "NotTo", "ShouldNot")
	asyncFuncs      = set.New(
		"Eventually",
		"Consistently",
		"EventuallyWithOffset",
		"ConsistentlyWithOffset",
	)
	errorType = types.Universe.Lookup("error").Type().Underlying().(*types.Interface)
)

// actualIndex maps each assertion function to the position of its actual value.
var actualIndex = map[string]int{
	"Expect":                 0,
	"Ω":                      0,
	"Eventually":             0,
	"Consistently":           0,
	"ExpectWithOffset":       1,
	"EventuallyWithOffset":   1,
	"ConsistentlyWithOffset": 1,
}

const message = "assertion only checks that an error occurred; " +
	"identify it with MatchError(...)"

func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			var negated bool
			switch {
			case positiveAsserts.Contains(sel.Sel.Name):
			case negativeAsserts.Contains(sel.Sel.Name):
				negated = true
			default:
				return true
			}
			matcher, negated := unwrapNot(call.Args[0], negated)
			if bare(pass, matcher, negated, sel.X) {
				pass.Report(analysis.Diagnostic{
					Pos:     call.Pos(),
					End:     call.End(),
					Message: message,
				})
			}
			return true
		})
	}
	return nil, nil
}

// unwrapNot strips Not(...) wrappers from a matcher, flipping negated once per wrapper.
func unwrapNot(matcher ast.Expr, negated bool) (*ast.CallExpr, bool) {
	for {
		call, ok := matcher.(*ast.CallExpr)
		if !ok {
			return nil, negated
		}
		if calleeName(call) != "Not" || len(call.Args) != 1 {
			return call, negated
		}
		matcher, negated = call.Args[0], !negated
	}
}

// bare reports whether matcher, applied with the given polarity to the assertion
// receiver, only checks that an error occurred.
func bare(
	pass *analysis.Pass,
	matcher *ast.CallExpr,
	negated bool,
	receiver ast.Expr,
) bool {
	if matcher == nil || len(matcher.Args) != 0 {
		return false
	}
	switch calleeName(matcher) {
	case "HaveOccurred":
		return !negated
	case "Succeed":
		return negated
	case "BeNil":
		return negated && assertsError(pass, receiver)
	}
	return false
}

// assertsError reports whether the assertion receiver holds a single error value:
// Expect(err), Expect(f()).Error(), or Eventually(f) where f returns only an error.
func assertsError(pass *analysis.Pass, receiver ast.Expr) bool {
	call, ok := receiver.(*ast.CallExpr)
	if !ok {
		return false
	}
	name := calleeName(call)
	if name == "Error" {
		return true
	}
	// Skip async options such as WithTimeout to reach Eventually or Consistently.
	for _, isAssert := actualIndex[name]; !isAssert; _, isAssert = actualIndex[name] {
		sel, isSel := call.Fun.(*ast.SelectorExpr)
		if !isSel {
			return false
		}
		if call, ok = sel.X.(*ast.CallExpr); !ok {
			return false
		}
		name = calleeName(call)
	}
	i, ok := actualIndex[name]
	if !ok || len(call.Args) <= i {
		return false
	}
	t := pass.TypesInfo.TypeOf(call.Args[i])
	if t == nil {
		return false
	}
	if sig, isFunc := t.Underlying().(*types.Signature); isFunc &&
		asyncFuncs.Contains(name) {
		if sig.Results().Len() != 1 {
			return false
		}
		t = sig.Results().At(0).Type()
	}
	if _, isTuple := t.(*types.Tuple); isTuple {
		return false
	}
	return types.Implements(t, errorType)
}

// calleeName returns the called function's identifier for both unqualified F(...) (the
// dot-imported case) and qualified pkg.F(...) calls. It returns "" for any other callee
// expression.
func calleeName(call *ast.CallExpr) string {
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		return fn.Name
	case *ast.SelectorExpr:
		return fn.Sel.Name
	}
	return ""
}
