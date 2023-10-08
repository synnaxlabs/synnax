// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calc_test

import (
	"go/ast"
	"go/token"
	"math"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/calc"
)

func treesEqual(t1, t2 ast.Expr) bool {
	switch t1 := t1.(type) {
	case *ast.BinaryExpr:
		t2, ok := t2.(*ast.BinaryExpr)
		if !ok {
			return false
		}
		return t1.Op == t2.Op && treesEqual(t1.X, t2.X) && treesEqual(t1.Y, t2.Y)
	case *ast.BasicLit:
		t2, ok := t2.(*ast.BasicLit)
		if !ok {
			return false
		}
		if t1.Kind != t2.Kind {
			return false
		}
		switch t1.Kind {
		case token.FLOAT:
			f1, err := strconv.ParseFloat(t1.Value, 64)
			if err != nil {
				return false
			}
			f2, err := strconv.ParseFloat(t2.Value, 64)
			if err != nil {
				return false
			}
			return math.Abs(f1-f2) < 0.000000001
		}
	case *ast.Ident:
		t2, ok := t2.(*ast.Ident)
		if !ok {
			return false
		}
		return t1.Name == t2.Name
	}
	return false
}

var _ = Describe("Calc", func() {
	Describe("Build with math expressions", func() {
		It("Should build 1+1", func() {
			e := calc.Expression{}
			err := e.Build("1 + 1")
			Expect(err).ToNot(HaveOccurred())
			expectedTree := &ast.BinaryExpr{X: &ast.BasicLit{Kind: token.FLOAT, Value: "1"}, Op: token.ADD, Y: &ast.BasicLit{Kind: token.FLOAT, Value: "1"}}
			actualTree := e.GetTree().(*ast.BinaryExpr)
			Expect(treesEqual(expectedTree, actualTree)).To(BeTrue())
		})
	})
})
