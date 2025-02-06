// Copyright 2025 Synnax Labs, Inc.
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
	case *ast.UnaryExpr:
		t2, ok := t2.(*ast.UnaryExpr)
		if !ok {
			return false
		}
		return t1.Op == t2.Op && treesEqual(t1.X, t2.X)
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
				expectedTree := &ast.BinaryExpr{X: &ast.BasicLit{Kind: token.FLOAT, Value: "1"}, Op: token.ADD,
					Y: &ast.BasicLit{Kind: token.FLOAT, Value: "1"}}
				actualTree := e.Tree().(*ast.BinaryExpr)
				Expect(treesEqual(expectedTree, actualTree)).To(BeTrue())
			})
			It("Should build 1+2*4", func() {
				e := calc.Expression{}
				err := e.Build("1 + 2 * 4")
				Expect(err).ToNot(HaveOccurred())
				expectedTree := &ast.BinaryExpr{
					X:  &ast.BasicLit{Kind: token.FLOAT, Value: "1"},
					Op: token.ADD,
					Y: &ast.BinaryExpr{
						X:  &ast.BasicLit{Kind: token.FLOAT, Value: "2"},
						Op: token.MUL,
						Y:  &ast.BasicLit{Kind: token.FLOAT, Value: "4"},
					},
				}
				actualTree := e.Tree().(*ast.BinaryExpr)
				Expect(treesEqual(expectedTree, actualTree)).To(BeTrue())
			})
			It("Should build 2*4+1", func() {
				e := calc.Expression{}
				err := e.Build("2 * 4 + 1")
				Expect(err).ToNot(HaveOccurred())
				expectedTree := &ast.BinaryExpr{
					X: &ast.BinaryExpr{
						X:  &ast.BasicLit{Kind: token.FLOAT, Value: "2"},
						Op: token.MUL, Y: &ast.BasicLit{Kind: token.FLOAT, Value: "4"},
					},
					Op: token.ADD,
					Y:  &ast.BasicLit{Kind: token.FLOAT, Value: "1"},
				}
				actualTree := e.Tree().(*ast.BinaryExpr)
				Expect(treesEqual(expectedTree, actualTree)).To(BeTrue())
			})
			It("Should build 2*(4+1)", func() {
				e := calc.Expression{}
				err := e.Build("2 * (4 + 1)")
				Expect(err).ToNot(HaveOccurred())
				expectedTree := &ast.BinaryExpr{
					X:  &ast.BasicLit{Kind: token.FLOAT, Value: "2"},
					Op: token.MUL,
					Y: &ast.BinaryExpr{
						X:  &ast.BasicLit{Kind: token.FLOAT, Value: "4"},
						Op: token.ADD,
						Y:  &ast.BasicLit{Kind: token.FLOAT, Value: "1"},
					},
				}
				actualTree := e.Tree().(*ast.BinaryExpr)
				Expect(treesEqual(expectedTree, actualTree)).To(BeTrue())
			})
			It("Should handle nested parentheses", func() {
				e := calc.Expression{}
				err := e.Build("2 * (4 + (1 / 2))")
				Expect(err).ToNot(HaveOccurred())
				expectedTree := &ast.BinaryExpr{
					X:  &ast.BasicLit{Kind: token.FLOAT, Value: "2"},
					Op: token.MUL,
					Y: &ast.BinaryExpr{
						X:  &ast.BasicLit{Kind: token.FLOAT, Value: "4"},
						Op: token.ADD,
						Y: &ast.BinaryExpr{
							X:  &ast.BasicLit{Kind: token.FLOAT, Value: "1"},
							Op: token.QUO,
							Y:  &ast.BasicLit{Kind: token.FLOAT, Value: "2"},
						},
					},
				}
				actualTree := e.Tree().(*ast.BinaryExpr)
				Expect(treesEqual(expectedTree, actualTree)).To(BeTrue())
			})
			It("Should handle nested parentheses", func() {
				e := calc.Expression{}
				err := e.Build("2 * ((4 + 1) / 2)")
				Expect(err).ToNot(HaveOccurred())
				expectedTree := &ast.BinaryExpr{
					X:  &ast.BasicLit{Kind: token.FLOAT, Value: "2"},
					Op: token.MUL,
					Y: &ast.BinaryExpr{
						X: &ast.BinaryExpr{
							X:  &ast.BasicLit{Kind: token.FLOAT, Value: "4"},
							Op: token.ADD,
							Y:  &ast.BasicLit{Kind: token.FLOAT, Value: "1"},
						},
						Op: token.QUO,
						Y:  &ast.BasicLit{Kind: token.FLOAT, Value: "2"},
					},
				}
				actualTree := e.Tree().(*ast.BinaryExpr)
				Expect(treesEqual(expectedTree, actualTree)).To(BeTrue())
			})
			It("Should handle exponents", func() {
				e := calc.Expression{}
				err := e.Build("2 * (4 + 1) ^ 2")
				Expect(err).ToNot(HaveOccurred())
				expectedTree := &ast.BinaryExpr{
					X:  &ast.BasicLit{Kind: token.FLOAT, Value: "2"},
					Op: token.MUL,
					Y: &ast.BinaryExpr{
						X: &ast.BinaryExpr{
							X:  &ast.BasicLit{Kind: token.FLOAT, Value: "4"},
							Op: token.ADD,
							Y:  &ast.BasicLit{Kind: token.FLOAT, Value: "1"},
						},
						Op: token.XOR,
						Y:  &ast.BasicLit{Kind: token.FLOAT, Value: "2"},
					},
				}
				actualTree := e.Tree().(*ast.BinaryExpr)
				Expect(treesEqual(expectedTree, actualTree)).To(BeTrue())
			})
			It("Should handle negative numbers", func() {
				e := calc.Expression{}
				err := e.Build("-1 * 2")
				Expect(err).ToNot(HaveOccurred())
				expectedTree := &ast.BinaryExpr{
					X: &ast.BinaryExpr{
						X:  &ast.BasicLit{Kind: token.FLOAT, Value: "-1"},
						Op: token.MUL,
						Y:  &ast.BasicLit{Kind: token.FLOAT, Value: "1"},
					},
					Op: token.MUL,
					Y:  &ast.BasicLit{Kind: token.FLOAT, Value: "2"},
				}
				actualTree := e.Tree().(*ast.BinaryExpr)
				Expect(treesEqual(expectedTree, actualTree)).To(BeTrue())
			})
			It("Should handle negative numbers", func() {
				e := calc.Expression{}
				err := e.Build("2 * -1")
				Expect(err).ToNot(HaveOccurred())
				expectedTree := &ast.BinaryExpr{
					X:  &ast.BasicLit{Kind: token.FLOAT, Value: "2"},
					Op: token.MUL,
					Y: &ast.BinaryExpr{
						X:  &ast.BasicLit{Kind: token.FLOAT, Value: "-1"},
						Op: token.MUL,
						Y:  &ast.BasicLit{Kind: token.FLOAT, Value: "1"},
					},
				}
				actualTree := e.Tree().(*ast.BinaryExpr)
				Expect(treesEqual(expectedTree, actualTree)).To(BeTrue())
			})
			It("Should build 3+4*2/(1-5)^2^3", func() {
				e := calc.Expression{}
				err := e.Build("3+4*2/(1-5)^2^3")
				Expect(err).ToNot(HaveOccurred())
				expectedTree := &ast.BinaryExpr{
					X:  &ast.BasicLit{Kind: token.FLOAT, Value: "3"},
					Op: token.ADD,
					Y: &ast.BinaryExpr{
						X: &ast.BinaryExpr{
							X:  &ast.BasicLit{Kind: token.FLOAT, Value: "4"},
							Op: token.MUL,
							Y:  &ast.BasicLit{Kind: token.FLOAT, Value: "2"},
						},
						Op: token.QUO,
						Y: &ast.BinaryExpr{
							X: &ast.BinaryExpr{
								X:  &ast.BasicLit{Kind: token.FLOAT, Value: "1"},
								Op: token.SUB,
								Y:  &ast.BasicLit{Kind: token.FLOAT, Value: "5"},
							},
							Op: token.XOR,
							Y: &ast.BinaryExpr{
								X:  &ast.BasicLit{Kind: token.FLOAT, Value: "2"},
								Op: token.XOR,
								Y:  &ast.BasicLit{Kind: token.FLOAT, Value: "3"},
							},
						},
					},
				}
				actualTree := e.Tree().(*ast.BinaryExpr)
				Expect(treesEqual(expectedTree, actualTree)).To(BeTrue())
			})
		})
		Describe("Build with expressions that have variables", func() {
			It("Should build 1+x", func() {
				e := calc.Expression{}
				err := e.Build("1+x")
				Expect(err).ToNot(HaveOccurred())
				expectedTree := &ast.BinaryExpr{
					X:  &ast.BasicLit{Kind: token.FLOAT, Value: "1"},
					Op: token.ADD,
					Y:  &ast.Ident{Name: "x"},
				}
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
			It("Should build with boolean expressions", func() {
				e := calc.Expression{}
				err := e.Build("token1*5 > 10")
				Expect(err).ToNot(HaveOccurred())
				expectedTree := &ast.BinaryExpr{
					X: &ast.BinaryExpr{
						X:  &ast.Ident{Name: "token1"},
						Op: token.MUL,
						Y:  &ast.BasicLit{Kind: token.FLOAT, Value: "5"},
					},
					Op: token.GTR,
					Y:  &ast.BasicLit{Kind: token.FLOAT, Value: "10"},
				}
				actualTree := e.Tree().(*ast.BinaryExpr)
				Expect(treesEqual(expectedTree, actualTree)).To(BeTrue())
			})
			It("Should build with multiple boolean expressions", func() {
				e := calc.Expression{}
				err := e.Build("token1*5 > 10 && token2*5 < 20")
				Expect(err).ToNot(HaveOccurred())
				expectedTree := &ast.BinaryExpr{
					X: &ast.BinaryExpr{
						X: &ast.BinaryExpr{
							X:  &ast.Ident{Name: "token1"},
							Op: token.MUL,
							Y:  &ast.BasicLit{Kind: token.FLOAT, Value: "5"},
						},
						Op: token.GTR,
						Y:  &ast.BasicLit{Kind: token.FLOAT, Value: "10"},
					},
					Op: token.LAND,
					Y: &ast.BinaryExpr{
						X: &ast.BinaryExpr{
							X:  &ast.Ident{Name: "token2"},
							Op: token.MUL,
							Y:  &ast.BasicLit{Kind: token.FLOAT, Value: "5"},
						},
						Op: token.LSS,
						Y:  &ast.BasicLit{Kind: token.FLOAT, Value: "20"},
					},
				}
				actualTree := e.Tree().(*ast.BinaryExpr)
				Expect(treesEqual(expectedTree, actualTree)).To(BeTrue())
			})
			It("Should build boolean expressions with parenthesis", func() {
				e := calc.Expression{}
				err := e.Build("(23 >= 0 && 3 < 0) || 1 == 1")
				Expect(err).ToNot(HaveOccurred())
				expectedTree := &ast.BinaryExpr{
					X: &ast.BinaryExpr{
						X: &ast.BinaryExpr{
							X:  &ast.BasicLit{Kind: token.FLOAT, Value: "23"},
							Op: token.GEQ,
							Y:  &ast.BasicLit{Kind: token.FLOAT, Value: "0"},
						},
						Op: token.LAND,
						Y: &ast.BinaryExpr{
							X:  &ast.BasicLit{Kind: token.FLOAT, Value: "3"},
							Op: token.LSS,
							Y:  &ast.BasicLit{Kind: token.FLOAT, Value: "0"},
						},
					},
					Op: token.LOR,
					Y: &ast.BinaryExpr{
						X:  &ast.BasicLit{Kind: token.FLOAT, Value: "1"},
						Op: token.EQL,
						Y:  &ast.BasicLit{Kind: token.FLOAT, Value: "1"},
					},
				}
				actualTree := e.Tree().(*ast.BinaryExpr)
				Expect(treesEqual(expectedTree, actualTree)).To(BeTrue())
			})
			It("Should build with negate operator", func() {
				e := calc.Expression{}
				err := e.Build("p || !q")
				Expect(err).ToNot(HaveOccurred())
				expectedTree := &ast.BinaryExpr{
					X:  &ast.Ident{Name: "p"},
					Op: token.LOR,
					Y: &ast.UnaryExpr{
						Op: token.NOT,
						X:  &ast.Ident{Name: "q"},
					},
				}
				actualTree := e.Tree().(*ast.BinaryExpr)
				Expect(treesEqual(expectedTree, actualTree)).To(BeTrue())
			})
			It("Should build complex unary expressions", func() {
				e := calc.Expression{}
				err := e.Build("!(p || !q)")
				Expect(err).ToNot(HaveOccurred())
				expectedTree := &ast.UnaryExpr{
					Op: token.NOT,
					X: &ast.BinaryExpr{
						X:  &ast.Ident{Name: "p"},
						Op: token.LOR,
						Y: &ast.UnaryExpr{
							Op: token.NOT,
							X:  &ast.Ident{Name: "q"},
						},
					},
				}
				actualTree := e.Tree().(*ast.UnaryExpr)
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
			It("Should return an error for an unbalanced boolean expression", func() {
				e := calc.Expression{}
				err := e.Build("5 ==")
				Expect(err).To(HaveOccurred())
			})
			It("Should return an error for an unbalanced unary expression", func() {
				e := calc.Expression{}
				err := e.Build("!")
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
		It("Should evaluate 3+4*2/(-5)^2^3", func() {
			e := calc.Expression{}
			err := e.Build("3+4*2/(-5)^2^3")
			Expect(err).ToNot(HaveOccurred())
			Expect(e.Evaluate(nil)).To(Equal(float64(3.00002048)))
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
		It("Should evaluate boolean expressions", func() {
			e := calc.Expression{}
			err := e.Build("a > b")
			Expect(err).ToNot(HaveOccurred())
			r := &mockResolver{vals: map[string]float64{"a": 3, "b": 4}}
			Expect(e.Evaluate(r)).To(Equal(float64(0)))
		})
		It("Should evaluate boolean expressions", func() {
			e := calc.Expression{}
			err := e.Build("45 + 23 > 0 && 12 - 3 < 10")
			Expect(err).ToNot(HaveOccurred())
			Expect(e.Evaluate(nil)).To(Equal(float64(1)))
		})
		It("Should evaluate boolean expressions with parenthesis", func() {
			e := calc.Expression{}
			err := e.Build("(45 + 23 > 0 && 12 - 3 < 0) || 1 == 1")
			Expect(err).ToNot(HaveOccurred())
			Expect(e.Evaluate(nil)).To(Equal(float64(1)))
		})
	})
	Describe("Boolean Logic Tests", func() {
		Describe("&& Tests", func() {
			It("1 && 1 == 1", func() {
				e := calc.Expression{}
				err := e.Build("1 && 1")
				Expect(err).ToNot(HaveOccurred())
				Expect(e.Evaluate(nil)).To(Equal(float64(1)))
			})
			It("1 && 0 == 0", func() {
				e := calc.Expression{}
				err := e.Build("1 && 0")
				Expect(err).ToNot(HaveOccurred())
				Expect(e.Evaluate(nil)).To(Equal(float64(0)))
			})
			It("0 && 1 == 0", func() {
				e := calc.Expression{}
				err := e.Build("0 && 1")
				Expect(err).ToNot(HaveOccurred())
				Expect(e.Evaluate(nil)).To(Equal(float64(0)))
			})
			It("0 && 0 == 0", func() {
				e := calc.Expression{}
				err := e.Build("0 && 0")
				Expect(err).ToNot(HaveOccurred())
				Expect(e.Evaluate(nil)).To(Equal(float64(0)))
			})
		})
		Describe("|| Tests", func() {
			It("1 || 1 == 1", func() {
				e := calc.Expression{}
				err := e.Build("1 || 1")
				Expect(err).ToNot(HaveOccurred())
				Expect(e.Evaluate(nil)).To(Equal(float64(1)))
			})
			It("1 || 0 == 1", func() {
				e := calc.Expression{}
				err := e.Build("1 || 0")
				Expect(err).ToNot(HaveOccurred())
				Expect(e.Evaluate(nil)).To(Equal(float64(1)))
			})
			It("0 || 1 == 1", func() {
				e := calc.Expression{}
				err := e.Build("0 || 1")
				Expect(err).ToNot(HaveOccurred())
				Expect(e.Evaluate(nil)).To(Equal(float64(1)))
			})
			It("0 || 0 == 0", func() {
				e := calc.Expression{}
				err := e.Build("0 || 0")
				Expect(err).ToNot(HaveOccurred())
				Expect(e.Evaluate(nil)).To(Equal(float64(0)))
			})
		})
		Describe("! Tests", func() {
			It("!1 == 0", func() {
				e := calc.Expression{}
				err := e.Build("!1")
				Expect(err).ToNot(HaveOccurred())
				Expect(e.Evaluate(nil)).To(Equal(float64(0)))
			})
			It("!0 == 1", func() {
				e := calc.Expression{}
				err := e.Build("!0")
				Expect(err).ToNot(HaveOccurred())
				Expect(e.Evaluate(nil)).To(Equal(float64(1)))
			})
		})
		Describe("DeMorgan's Law", func() {
			It("!(a && b) == (!a || !b)", func() {
				e := calc.Expression{}
				err := e.Build("!(a && b) == (!a || !b)")
				Expect(err).ToNot(HaveOccurred())
				r1 := &mockResolver{vals: map[string]float64{"a": 1, "b": 1}}
				r2 := &mockResolver{vals: map[string]float64{"a": 1, "b": 0}}
				r3 := &mockResolver{vals: map[string]float64{"a": 0, "b": 1}}
				r4 := &mockResolver{vals: map[string]float64{"a": 0, "b": 0}}
				Expect(e.Evaluate(r1)).To(Equal(float64(1)))
				Expect(e.Evaluate(r2)).To(Equal(float64(1)))
				Expect(e.Evaluate(r3)).To(Equal(float64(1)))
				Expect(e.Evaluate(r4)).To(Equal(float64(1)))
			})
			It("!(a || b) == (!a && !b)", func() {
				e := calc.Expression{}
				err := e.Build("!(a || b) == (!a && !b)")
				Expect(err).ToNot(HaveOccurred())
				r1 := &mockResolver{vals: map[string]float64{"a": 1, "b": 1}}
				r2 := &mockResolver{vals: map[string]float64{"a": 1, "b": 0}}
				r3 := &mockResolver{vals: map[string]float64{"a": 0, "b": 1}}
				r4 := &mockResolver{vals: map[string]float64{"a": 0, "b": 0}}
				Expect(e.Evaluate(r1)).To(Equal(float64(1)))
				Expect(e.Evaluate(r2)).To(Equal(float64(1)))
				Expect(e.Evaluate(r3)).To(Equal(float64(1)))
				Expect(e.Evaluate(r4)).To(Equal(float64(1)))
			})
			It("!(a -> b) == (a && !b)", func() {
				e := calc.Expression{}
				err := e.Build("!(!a || b) == (a && !b)")
				Expect(err).ToNot(HaveOccurred())
				r1 := &mockResolver{vals: map[string]float64{"a": 1, "b": 1}}
				r2 := &mockResolver{vals: map[string]float64{"a": 1, "b": 0}}
				r3 := &mockResolver{vals: map[string]float64{"a": 0, "b": 1}}
				r4 := &mockResolver{vals: map[string]float64{"a": 0, "b": 0}}
				Expect(e.Evaluate(r1)).To(Equal(float64(1)))
				Expect(e.Evaluate(r2)).To(Equal(float64(1)))
				Expect(e.Evaluate(r3)).To(Equal(float64(1)))
				Expect(e.Evaluate(r4)).To(Equal(float64(1)))
			})
		})
		Describe("Comparator Tests", func() {
			It("==", func() {
				e := calc.Expression{}
				err := e.Build("p == 1")
				Expect(err).ToNot(HaveOccurred())
				r1 := &mockResolver{vals: map[string]float64{"p": 1}}
				r2 := &mockResolver{vals: map[string]float64{"p": 0}}
				Expect(e.Evaluate(r1)).To(Equal(float64(1)))
				Expect(e.Evaluate(r2)).To(Equal(float64(0)))
			})
			It("!=", func() {
				e := calc.Expression{}
				err := e.Build("p != 1")
				Expect(err).ToNot(HaveOccurred())
				r1 := &mockResolver{vals: map[string]float64{"p": 1}}
				r2 := &mockResolver{vals: map[string]float64{"p": 0}}
				Expect(e.Evaluate(r1)).To(Equal(float64(0)))
				Expect(e.Evaluate(r2)).To(Equal(float64(1)))
			})
			It(">", func() {
				e := calc.Expression{}
				err := e.Build("p > 1")
				Expect(err).ToNot(HaveOccurred())
				r1 := &mockResolver{vals: map[string]float64{"p": 1}}
				r2 := &mockResolver{vals: map[string]float64{"p": 0}}
				Expect(e.Evaluate(r1)).To(Equal(float64(0)))
				Expect(e.Evaluate(r2)).To(Equal(float64(0)))
			})
			It(">=", func() {
				e := calc.Expression{}
				err := e.Build("p >= 1")
				Expect(err).ToNot(HaveOccurred())
				r1 := &mockResolver{vals: map[string]float64{"p": 1}}
				r2 := &mockResolver{vals: map[string]float64{"p": 0}}
				Expect(e.Evaluate(r1)).To(Equal(float64(1)))
				Expect(e.Evaluate(r2)).To(Equal(float64(0)))
			})
			It("<", func() {
				e := calc.Expression{}
				err := e.Build("p < 1")
				Expect(err).ToNot(HaveOccurred())
				r1 := &mockResolver{vals: map[string]float64{"p": 1}}
				r2 := &mockResolver{vals: map[string]float64{"p": 0}}
				Expect(e.Evaluate(r1)).To(Equal(float64(0)))
				Expect(e.Evaluate(r2)).To(Equal(float64(1)))
			})
			It("<=", func() {
				e := calc.Expression{}
				err := e.Build("p <= 1")
				Expect(err).ToNot(HaveOccurred())
				r1 := &mockResolver{vals: map[string]float64{"p": 1}}
				r2 := &mockResolver{vals: map[string]float64{"p": 0}}
				Expect(e.Evaluate(r1)).To(Equal(float64(1)))
				Expect(e.Evaluate(r2)).To(Equal(float64(1)))
			})
		})
	})
})
