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

type mockResolver struct {
	vals map[string]float64
}

func (m *mockResolver) Resolve(s string) (float64, error) {
	return m.vals[s], nil
}

func (m *mockResolver) Set(s string, v float64) error {
	m.vals[s] = v
	return nil
}

var _ = Describe("Calc", func() {
	Describe("Build", func() {
		Describe("Build with math expressions", func() {
			It("Should build 1+1", func() {
				e := calc.Expression{}
				err := e.Build("1 + 1")
				Expect(err).ToNot(HaveOccurred())
				expectedTree := &ast.BinaryExpr{X: &ast.BasicLit{Kind: token.FLOAT, Value: "1"}, Op: token.ADD, Y: &ast.BasicLit{Kind: token.FLOAT, Value: "1"}}
				actualTree := e.Tree().(*ast.BinaryExpr)
				Expect(treesEqual(expectedTree, actualTree)).To(BeTrue())
			})
			It("Should build 1+2*4", func() {
				e := calc.Expression{}
				err := e.Build("1 + 2 * 4")
				Expect(err).ToNot(HaveOccurred())
				expectedTree := &ast.BinaryExpr{X: &ast.BasicLit{Kind: token.FLOAT, Value: "1"}, Op: token.ADD, Y: &ast.BinaryExpr{X: &ast.BasicLit{Kind: token.FLOAT, Value: "2"}, Op: token.MUL, Y: &ast.BasicLit{Kind: token.FLOAT, Value: "4"}}}
				actualTree := e.Tree().(*ast.BinaryExpr)
				Expect(treesEqual(expectedTree, actualTree)).To(BeTrue())
			})
			It("Should build 2*4+1", func() {
				e := calc.Expression{}
				err := e.Build("2 * 4 + 1")
				Expect(err).ToNot(HaveOccurred())
				expectedTree := &ast.BinaryExpr{X: &ast.BinaryExpr{X: &ast.BasicLit{Kind: token.FLOAT, Value: "2"}, Op: token.MUL, Y: &ast.BasicLit{Kind: token.FLOAT, Value: "4"}}, Op: token.ADD, Y: &ast.BasicLit{Kind: token.FLOAT, Value: "1"}}
				actualTree := e.Tree().(*ast.BinaryExpr)
				Expect(treesEqual(expectedTree, actualTree)).To(BeTrue())
			})
			It("Should build 2*(4+1)", func() {
				e := calc.Expression{}
				err := e.Build("2 * (4 + 1)")
				Expect(err).ToNot(HaveOccurred())
				expectedTree := &ast.BinaryExpr{X: &ast.BasicLit{Kind: token.FLOAT, Value: "2"}, Op: token.MUL, Y: &ast.BinaryExpr{X: &ast.BasicLit{Kind: token.FLOAT, Value: "4"}, Op: token.ADD, Y: &ast.BasicLit{Kind: token.FLOAT, Value: "1"}}}
				actualTree := e.Tree().(*ast.BinaryExpr)
				Expect(treesEqual(expectedTree, actualTree)).To(BeTrue())
			})
			It("Should handle nested parentheses", func() {
				e := calc.Expression{}
				err := e.Build("2 * (4 + (1 / 2))")
				Expect(err).ToNot(HaveOccurred())
				expectedTree := &ast.BinaryExpr{X: &ast.BasicLit{Kind: token.FLOAT, Value: "2"}, Op: token.MUL, Y: &ast.BinaryExpr{X: &ast.BasicLit{Kind: token.FLOAT, Value: "4"}, Op: token.ADD, Y: &ast.BinaryExpr{X: &ast.BasicLit{Kind: token.FLOAT, Value: "1"}, Op: token.QUO, Y: &ast.BasicLit{Kind: token.FLOAT, Value: "2"}}}}
				actualTree := e.Tree().(*ast.BinaryExpr)
				Expect(treesEqual(expectedTree, actualTree)).To(BeTrue())
			})
			It("Should handle nested parentheses", func() {
				e := calc.Expression{}
				err := e.Build("2 * ((4 + 1) / 2)")
				Expect(err).ToNot(HaveOccurred())
				expectedTree := &ast.BinaryExpr{X: &ast.BasicLit{Kind: token.FLOAT, Value: "2"}, Op: token.MUL, Y: &ast.BinaryExpr{X: &ast.BinaryExpr{X: &ast.BasicLit{Kind: token.FLOAT, Value: "4"}, Op: token.ADD, Y: &ast.BasicLit{Kind: token.FLOAT, Value: "1"}}, Op: token.QUO, Y: &ast.BasicLit{Kind: token.FLOAT, Value: "2"}}}
				actualTree := e.Tree().(*ast.BinaryExpr)
				Expect(treesEqual(expectedTree, actualTree)).To(BeTrue())
			})
			It("Should handle exponents", func() {
				e := calc.Expression{}
				err := e.Build("2 * (4 + 1) ^ 2")
				Expect(err).ToNot(HaveOccurred())
				expectedTree := &ast.BinaryExpr{X: &ast.BasicLit{Kind: token.FLOAT, Value: "2"}, Op: token.MUL, Y: &ast.BinaryExpr{X: &ast.BinaryExpr{X: &ast.BasicLit{Kind: token.FLOAT, Value: "4"}, Op: token.ADD, Y: &ast.BasicLit{Kind: token.FLOAT, Value: "1"}}, Op: token.XOR, Y: &ast.BasicLit{Kind: token.FLOAT, Value: "2"}}}
				actualTree := e.Tree().(*ast.BinaryExpr)
				Expect(treesEqual(expectedTree, actualTree)).To(BeTrue())
			})
			It("Should handle negative numbers", func() {
				e := calc.Expression{}
				err := e.Build("-1 * 2")
				Expect(err).ToNot(HaveOccurred())
				expectedTree := &ast.BinaryExpr{X: &ast.BasicLit{Kind: token.FLOAT, Value: "-1"}, Op: token.MUL, Y: &ast.BasicLit{Kind: token.FLOAT, Value: "2"}}
				actualTree := e.Tree().(*ast.BinaryExpr)
				Expect(treesEqual(expectedTree, actualTree)).To(BeTrue())
			})
			It("Should handle negative numbers", func() {
				e := calc.Expression{}
				err := e.Build("2 * -1")
				Expect(err).ToNot(HaveOccurred())
				expectedTree := &ast.BinaryExpr{X: &ast.BasicLit{Kind: token.FLOAT, Value: "2"}, Op: token.MUL, Y: &ast.BasicLit{Kind: token.FLOAT, Value: "-1"}}
				actualTree := e.Tree().(*ast.BinaryExpr)
				Expect(treesEqual(expectedTree, actualTree)).To(BeTrue())
			})
			It("Should build 3+4*2/(1-5)^2^3", func() {
				e := calc.Expression{}
				err := e.Build("3+4*2/(1-5)^2^3")
				Expect(err).ToNot(HaveOccurred())
				expectedTree := &ast.BinaryExpr{X: &ast.BasicLit{Kind: token.FLOAT, Value: "3"}, Op: token.ADD, Y: &ast.BinaryExpr{X: &ast.BinaryExpr{X: &ast.BasicLit{Kind: token.FLOAT, Value: "4"}, Op: token.MUL, Y: &ast.BasicLit{Kind: token.FLOAT, Value: "2"}}, Op: token.QUO, Y: &ast.BinaryExpr{X: &ast.BinaryExpr{X: &ast.BasicLit{Kind: token.FLOAT, Value: "1"}, Op: token.SUB, Y: &ast.BasicLit{Kind: token.FLOAT, Value: "5"}}, Op: token.XOR, Y: &ast.BinaryExpr{X: &ast.BasicLit{Kind: token.FLOAT, Value: "2"}, Op: token.XOR, Y: &ast.BasicLit{Kind: token.FLOAT, Value: "3"}}}}}
				actualTree := e.Tree().(*ast.BinaryExpr)
				Expect(treesEqual(expectedTree, actualTree)).To(BeTrue())
			})
		})
		Describe("Build with expressions that have variables", func() {
			It("Should build 1+x", func() {
				e := calc.Expression{}
				err := e.Build("1+x")
				Expect(err).ToNot(HaveOccurred())
				expectedTree := &ast.BinaryExpr{X: &ast.BasicLit{Kind: token.FLOAT, Value: "1"}, Op: token.ADD, Y: &ast.Ident{Name: "x"}}
				actualTree := e.Tree().(*ast.BinaryExpr)
				Expect(treesEqual(expectedTree, actualTree)).To(BeTrue())
			})
			It("Should build token1 * 5", func() {
				e := calc.Expression{}
				err := e.Build("token1 * 5")
				Expect(err).ToNot(HaveOccurred())
				expectedTree := &ast.BinaryExpr{X: &ast.Ident{Name: "token1"}, Op: token.MUL, Y: &ast.BasicLit{Kind: token.FLOAT, Value: "5"}}
				actualTree := e.Tree().(*ast.BinaryExpr)
				Expect(treesEqual(expectedTree, actualTree)).To(BeTrue())
			})
			It("Should build work with no spaces", func() {
				e := calc.Expression{}
				err := e.Build("token1*5")
				Expect(err).ToNot(HaveOccurred())
				expectedTree := &ast.BinaryExpr{X: &ast.Ident{Name: "token1"}, Op: token.MUL, Y: &ast.BasicLit{Kind: token.FLOAT, Value: "5"}}
				actualTree := e.Tree().(*ast.BinaryExpr)
				Expect(treesEqual(expectedTree, actualTree)).To(BeTrue())
			})
		})
		Describe("Build with invalid expressions", func() {
			It("Should return an error for 1+", func() {
				e := calc.Expression{}
				err := e.Build("1+")
				Expect(err).To(HaveOccurred())
			})
			It("Should return an error for 1+(2*4", func() {
				e := calc.Expression{}
				err := e.Build("1+(2*4")
				Expect(err).To(HaveOccurred())
			})
			It("Should return an error for 1+2*4)", func() {
				e := calc.Expression{}
				err := e.Build("1+2*4)")
				Expect(err).To(HaveOccurred())
			})
		})
	})
	Describe("Evaluate", func() {
		It("Should evaluate 1+1", func() {
			e := calc.Expression{}
			err := e.Build("1 + 1")
			Expect(err).ToNot(HaveOccurred())
			Expect(e.Evaluate(nil)).To(Equal(float64(2)))
		})
		It("Should evaluate 3+4*2/(1-5)^2^3", func() {
			e := calc.Expression{}
			err := e.Build("3+4*2/(1-5)^2^3")
			Expect(err).ToNot(HaveOccurred())
			Expect(e.Evaluate(nil)).To(Equal(float64(3.0001220703125)))
		})
		It("Should evaluate pi*r^2", func() {
			e := calc.Expression{}
			err := e.Build("pi*r^2")
			Expect(err).ToNot(HaveOccurred())
			r := &mockResolver{vals: map[string]float64{"pi": 3.14159, "r": 2}}
			Expect(e.Evaluate(r)).To(Equal(float64(12.56636)))
		})
		It("Should evaluate the pythagorean theorem", func() {
			e := calc.Expression{}
			err := e.Build("(a^2 + b^2)^(1/2)")
			Expect(err).ToNot(HaveOccurred())
			r := &mockResolver{vals: map[string]float64{"a": 3, "b": 4}}
			Expect(e.Evaluate(r)).To(Equal(float64(5)))
		})
	})
})
