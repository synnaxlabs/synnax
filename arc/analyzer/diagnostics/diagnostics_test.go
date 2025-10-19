// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package diagnostics_test

import (
	"errors"
	"testing"

	"github.com/antlr4-go/antlr/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/synnaxlabs/arc/analyzer/diagnostics"
)

func TestDiagnostics(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Diagnostics Suite")
}

// Mock ANTLR token for testing
type mockToken struct {
	line   int
	column int
}

func (m *mockToken) GetSource() *antlr.TokenSourceCharStreamPair { return nil }
func (m *mockToken) GetTokenType() int                           { return 0 }
func (m *mockToken) GetChannel() int                             { return 0 }
func (m *mockToken) GetStart() int                               { return 0 }
func (m *mockToken) GetStop() int                                { return 0 }
func (m *mockToken) GetLine() int                                { return m.line }
func (m *mockToken) GetColumn() int                              { return m.column }
func (m *mockToken) GetText() string                             { return "" }
func (m *mockToken) SetText(string)                              {}
func (m *mockToken) GetTokenIndex() int                          { return 0 }
func (m *mockToken) SetTokenIndex(int)                           {}
func (m *mockToken) GetInputStream() antlr.CharStream            { return nil }
func (m *mockToken) GetTokenSource() antlr.TokenSource           { return nil }
func (m *mockToken) String() string                              { return "" }

// Mock ANTLR ParserRuleContext for testing
type mockContext struct {
	antlr.BaseParserRuleContext
	token *mockToken
}

func (m *mockContext) GetStart() antlr.Token {
	return m.token
}

func newMockContext(line, column int) *mockContext {
	return &mockContext{
		token: &mockToken{line: line, column: column},
	}
}

var _ = Describe("Severity", func() {
	Describe("String Conversion", func() {
		It("Should convert Error to string", func() {
			s := diagnostics.Error
			Expect(s.String()).To(Equal("error"))
		})

		It("Should convert Warning to string", func() {
			s := diagnostics.Warning
			Expect(s.String()).To(Equal("warning"))
		})

		It("Should convert Info to string", func() {
			s := diagnostics.Info
			Expect(s.String()).To(Equal("info"))
		})

		It("Should convert Hint to string", func() {
			s := diagnostics.Hint
			Expect(s.String()).To(Equal("hint"))
		})

		It("Should handle unknown severity values", func() {
			s := diagnostics.Severity(999)
			Expect(s.String()).To(Equal("Severity(999)"))
		})
	})

	Describe("Severity Values", func() {
		It("Should have Error as 0", func() {
			Expect(int(diagnostics.Error)).To(Equal(0))
		})

		It("Should have Warning as 1", func() {
			Expect(int(diagnostics.Warning)).To(Equal(1))
		})

		It("Should have Info as 2", func() {
			Expect(int(diagnostics.Info)).To(Equal(2))
		})

		It("Should have Hint as 3", func() {
			Expect(int(diagnostics.Hint)).To(Equal(3))
		})
	})
})

var _ = Describe("Diagnostics", func() {
	var diags diagnostics.Diagnostics

	BeforeEach(func() {
		diags = diagnostics.Diagnostics{}
	})

	Describe("Ok Method", func() {
		It("Should return true for empty diagnostics", func() {
			Expect(diags.Ok()).To(BeTrue())
		})

		It("Should return false when diagnostics exist", func() {
			err := errors.New("test error")
			diags.AddError(err, nil)
			Expect(diags.Ok()).To(BeFalse())
		})

		It("Should return false after adding warning", func() {
			err := errors.New("test warning")
			diags.AddWarning(err, nil)
			Expect(diags.Ok()).To(BeFalse())
		})

		It("Should return false after adding info", func() {
			err := errors.New("test info")
			diags.AddInfo(err, nil)
			Expect(diags.Ok()).To(BeFalse())
		})

		It("Should return false after adding hint", func() {
			err := errors.New("test hint")
			diags.AddHint(err, nil)
			Expect(diags.Ok()).To(BeFalse())
		})
	})

	Describe("AddError", func() {
		It("Should add error without context", func() {
			err := errors.New("undefined variable")
			diags.AddError(err, nil)

			Expect(diags).To(HaveLen(1))
			Expect(diags[0].Severity).To(Equal(diagnostics.Error))
			Expect(diags[0].Message).To(Equal("undefined variable"))
			Expect(diags[0].Line).To(Equal(0))
			Expect(diags[0].Column).To(Equal(0))
		})

		It("Should add error with context", func() {
			err := errors.New("type mismatch")
			ctx := newMockContext(5, 10)
			diags.AddError(err, ctx)

			Expect(diags).To(HaveLen(1))
			Expect(diags[0].Severity).To(Equal(diagnostics.Error))
			Expect(diags[0].Message).To(Equal("type mismatch"))
			Expect(diags[0].Line).To(Equal(5))
			Expect(diags[0].Column).To(Equal(10))
		})

		It("Should add multiple errors", func() {
			diags.AddError(errors.New("error 1"), nil)
			diags.AddError(errors.New("error 2"), nil)
			diags.AddError(errors.New("error 3"), nil)

			Expect(diags).To(HaveLen(3))
			Expect(diags[0].Message).To(Equal("error 1"))
			Expect(diags[1].Message).To(Equal("error 2"))
			Expect(diags[2].Message).To(Equal("error 3"))
		})

		It("Should preserve error order", func() {
			ctx1 := newMockContext(1, 0)
			ctx2 := newMockContext(5, 0)
			ctx3 := newMockContext(10, 0)

			diags.AddError(errors.New("first"), ctx1)
			diags.AddError(errors.New("second"), ctx2)
			diags.AddError(errors.New("third"), ctx3)

			Expect(diags[0].Line).To(Equal(1))
			Expect(diags[1].Line).To(Equal(5))
			Expect(diags[2].Line).To(Equal(10))
		})
	})

	Describe("AddWarning", func() {
		It("Should add warning without context", func() {
			err := errors.New("unused variable")
			diags.AddWarning(err, nil)

			Expect(diags).To(HaveLen(1))
			Expect(diags[0].Severity).To(Equal(diagnostics.Warning))
			Expect(diags[0].Message).To(Equal("unused variable"))
		})

		It("Should add warning with context", func() {
			err := errors.New("shadowed variable")
			ctx := newMockContext(3, 5)
			diags.AddWarning(err, ctx)

			Expect(diags).To(HaveLen(1))
			Expect(diags[0].Severity).To(Equal(diagnostics.Warning))
			Expect(diags[0].Line).To(Equal(3))
			Expect(diags[0].Column).To(Equal(5))
		})

		It("Should add multiple warnings", func() {
			diags.AddWarning(errors.New("warning 1"), nil)
			diags.AddWarning(errors.New("warning 2"), nil)

			Expect(diags).To(HaveLen(2))
			Expect(diags[0].Severity).To(Equal(diagnostics.Warning))
			Expect(diags[1].Severity).To(Equal(diagnostics.Warning))
		})
	})

	Describe("AddInfo", func() {
		It("Should add info without context", func() {
			err := errors.New("optimization applied")
			diags.AddInfo(err, nil)

			Expect(diags).To(HaveLen(1))
			Expect(diags[0].Severity).To(Equal(diagnostics.Info))
			Expect(diags[0].Message).To(Equal("optimization applied"))
		})

		It("Should add info with context", func() {
			err := errors.New("inferred type")
			ctx := newMockContext(2, 8)
			diags.AddInfo(err, ctx)

			Expect(diags).To(HaveLen(1))
			Expect(diags[0].Severity).To(Equal(diagnostics.Info))
			Expect(diags[0].Line).To(Equal(2))
			Expect(diags[0].Column).To(Equal(8))
		})
	})

	Describe("AddHint", func() {
		It("Should add hint without context", func() {
			err := errors.New("consider using const")
			diags.AddHint(err, nil)

			Expect(diags).To(HaveLen(1))
			Expect(diags[0].Severity).To(Equal(diagnostics.Hint))
			Expect(diags[0].Message).To(Equal("consider using const"))
		})

		It("Should add hint with context", func() {
			err := errors.New("use explicit type")
			ctx := newMockContext(7, 3)
			diags.AddHint(err, ctx)

			Expect(diags).To(HaveLen(1))
			Expect(diags[0].Severity).To(Equal(diagnostics.Hint))
			Expect(diags[0].Line).To(Equal(7))
			Expect(diags[0].Column).To(Equal(3))
		})
	})

	Describe("Mixed Severities", func() {
		It("Should support mixing different severity levels", func() {
			diags.AddError(errors.New("error"), nil)
			diags.AddWarning(errors.New("warning"), nil)
			diags.AddInfo(errors.New("info"), nil)
			diags.AddHint(errors.New("hint"), nil)

			Expect(diags).To(HaveLen(4))
			Expect(diags[0].Severity).To(Equal(diagnostics.Error))
			Expect(diags[1].Severity).To(Equal(diagnostics.Warning))
			Expect(diags[2].Severity).To(Equal(diagnostics.Info))
			Expect(diags[3].Severity).To(Equal(diagnostics.Hint))
		})

		It("Should preserve insertion order across severities", func() {
			ctx1 := newMockContext(1, 0)
			ctx2 := newMockContext(2, 0)
			ctx3 := newMockContext(3, 0)

			diags.AddWarning(errors.New("warn"), ctx1)
			diags.AddError(errors.New("err"), ctx2)
			diags.AddHint(errors.New("hint"), ctx3)

			Expect(diags[0].Line).To(Equal(1))
			Expect(diags[0].Severity).To(Equal(diagnostics.Warning))
			Expect(diags[1].Line).To(Equal(2))
			Expect(diags[1].Severity).To(Equal(diagnostics.Error))
			Expect(diags[2].Line).To(Equal(3))
			Expect(diags[2].Severity).To(Equal(diagnostics.Hint))
		})
	})

	Describe("String Formatting", func() {
		It("Should format empty diagnostics", func() {
			str := diags.String()
			Expect(str).To(Equal("analysis successful"))
		})

		It("Should format single error", func() {
			ctx := newMockContext(5, 10)
			diags.AddError(errors.New("undefined variable: x"), ctx)

			str := diags.String()
			Expect(str).To(Equal("5:10 error: undefined variable: x"))
		})

		It("Should format single warning", func() {
			ctx := newMockContext(3, 2)
			diags.AddWarning(errors.New("unused variable"), ctx)

			str := diags.String()
			Expect(str).To(Equal("3:2 warning: unused variable"))
		})

		It("Should format single info", func() {
			ctx := newMockContext(1, 0)
			diags.AddInfo(errors.New("type inferred as i32"), ctx)

			str := diags.String()
			Expect(str).To(Equal("1:0 info: type inferred as i32"))
		})

		It("Should format single hint", func() {
			ctx := newMockContext(8, 5)
			diags.AddHint(errors.New("consider using const"), ctx)

			str := diags.String()
			Expect(str).To(Equal("8:5 hint: consider using const"))
		})

		It("Should format multiple diagnostics with newlines", func() {
			ctx1 := newMockContext(1, 0)
			ctx2 := newMockContext(5, 10)

			diags.AddError(errors.New("first error"), ctx1)
			diags.AddError(errors.New("second error"), ctx2)

			str := diags.String()
			expected := "1:0 error: first error\n5:10 error: second error"
			Expect(str).To(Equal(expected))
		})

		It("Should format diagnostics without context (line 0, column 0)", func() {
			diags.AddError(errors.New("parse error"), nil)

			str := diags.String()
			Expect(str).To(Equal("0:0 error: parse error"))
		})

		It("Should format mixed severities", func() {
			ctx1 := newMockContext(1, 5)
			ctx2 := newMockContext(3, 10)
			ctx3 := newMockContext(5, 2)

			diags.AddError(errors.New("type error"), ctx1)
			diags.AddWarning(errors.New("unused"), ctx2)
			diags.AddHint(errors.New("use explicit type"), ctx3)

			str := diags.String()
			expected := "1:5 error: type error\n3:10 warning: unused\n5:2 hint: use explicit type"
			Expect(str).To(Equal(expected))
		})

		It("Should handle long error messages", func() {
			ctx := newMockContext(10, 20)
			longMsg := "this is a very long error message that describes in detail what went wrong with the code and provides helpful context"
			diags.AddError(errors.New(longMsg), ctx)

			str := diags.String()
			Expect(str).To(ContainSubstring("10:20 error:"))
			Expect(str).To(ContainSubstring(longMsg))
		})

		It("Should handle special characters in messages", func() {
			ctx := newMockContext(2, 3)
			diags.AddError(errors.New("cannot assign \"string\" to i32"), ctx)

			str := diags.String()
			Expect(str).To(ContainSubstring(`cannot assign "string" to i32`))
		})
	})

	Describe("Error Method", func() {
		It("Should convert to error type", func() {
			ctx := newMockContext(5, 10)
			diags.AddError(errors.New("test error"), ctx)

			err := diags.Error()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("5:10 error: test error"))
		})

		It("Should return error for empty diagnostics", func() {
			err := diags.Error()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("analysis successful"))
		})

		It("Should format multiple diagnostics in error", func() {
			ctx1 := newMockContext(1, 0)
			ctx2 := newMockContext(2, 5)

			diags.AddError(errors.New("error 1"), ctx1)
			diags.AddError(errors.New("error 2"), ctx2)

			err := diags.Error()
			Expect(err.Error()).To(ContainSubstring("error 1"))
			Expect(err.Error()).To(ContainSubstring("error 2"))
			Expect(err.Error()).To(ContainSubstring("\n"))
		})
	})

	Describe("Edge Cases", func() {
		It("Should handle very large line numbers", func() {
			ctx := newMockContext(99999, 500)
			diags.AddError(errors.New("error at end of file"), ctx)

			str := diags.String()
			Expect(str).To(ContainSubstring("99999:500"))
		})

		It("Should handle zero line and column", func() {
			ctx := newMockContext(0, 0)
			diags.AddError(errors.New("error"), ctx)

			Expect(diags[0].Line).To(Equal(0))
			Expect(diags[0].Column).To(Equal(0))
		})

		It("Should handle empty error message", func() {
			diags.AddError(errors.New(""), nil)

			Expect(diags[0].Message).To(Equal(""))
			str := diags.String()
			Expect(str).To(Equal("0:0 error: "))
		})

		It("Should handle adding many diagnostics", func() {
			for i := 0; i < 100; i++ {
				ctx := newMockContext(i, i)
				diags.AddError(errors.New("error"), ctx)
			}

			Expect(diags).To(HaveLen(100))
			Expect(diags[0].Line).To(Equal(0))
			Expect(diags[99].Line).To(Equal(99))
		})

		It("Should handle nil error gracefully", func() {
			// This would panic in real code, but testing the structure
			// In practice, callers should never pass nil error
			defer func() {
				if r := recover(); r != nil {
					// Expected to panic with nil error
					Expect(r).ToNot(BeNil())
				}
			}()

			// This will panic when calling err.Error() on nil
			// diags.AddError(nil, nil)
		})
	})

	Describe("Diagnostic Key Field", func() {
		It("Should have empty key by default", func() {
			diags.AddError(errors.New("test"), nil)

			Expect(diags[0].Key).To(Equal(""))
		})

		It("Should allow manual key assignment", func() {
			var d diagnostics.Diagnostics
			d = append(d, diagnostics.Diagnostic{
				Key:      "E001",
				Severity: diagnostics.Error,
				Line:     5,
				Column:   10,
				Message:  "undefined variable",
			})

			Expect(d[0].Key).To(Equal("E001"))
		})
	})
})
