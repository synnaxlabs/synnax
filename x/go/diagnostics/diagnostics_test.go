// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package diagnostics_test

import (
	"github.com/antlr4-go/antlr/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/errors"
	"go.lsp.dev/protocol"
)

// fakeToken is a minimal antlr.Token with a fixed line, column, and text.
type fakeToken struct {
	line, column int
	text         string
}

var _ antlr.Token = (*fakeToken)(nil)

func (*fakeToken) GetSource() *antlr.TokenSourceCharStreamPair { return nil }
func (*fakeToken) GetTokenType() int                           { return 0 }
func (*fakeToken) GetChannel() int                             { return 0 }
func (*fakeToken) GetStart() int                               { return 0 }
func (*fakeToken) GetStop() int                                { return 0 }
func (t *fakeToken) GetLine() int                              { return t.line }
func (t *fakeToken) GetColumn() int                            { return t.column }
func (t *fakeToken) GetText() string                           { return t.text }
func (t *fakeToken) SetText(text string)                       { t.text = text }
func (*fakeToken) GetTokenIndex() int                          { return 0 }
func (*fakeToken) SetTokenIndex(int)                           {}
func (*fakeToken) GetTokenSource() antlr.TokenSource           { return nil }
func (*fakeToken) GetInputStream() antlr.CharStream            { return nil }
func (t *fakeToken) String() string                            { return t.text }

// ruleCtxAt builds a parser rule context spanning the given tokens; stop may be nil.
func ruleCtxAt(start, stop antlr.Token) antlr.ParserRuleContext {
	ctx := antlr.NewBaseParserRuleContext(nil, -1)
	ctx.SetStart(start)
	ctx.SetStop(stop)
	return ctx
}

// hintedError is an error carrying a fix hint via GetHint.
type hintedError struct{ msg, hint string }

func (e hintedError) Error() string   { return e.msg }
func (e hintedError) GetHint() string { return e.hint }

var _ = Describe("Diagnostics", func() {
	Describe("Deduplication", func() {
		It("Should not add duplicate errors with same location and message", func() {
			var d diagnostics.Diagnostics
			err := errors.New("undefined symbol: x")
			d.Add(diagnostics.Error(err, nil))
			d.Add(diagnostics.Error(err, nil))
			d.Add(diagnostics.Error(err, nil))
			Expect(d).To(HaveLen(1))
		})

		It("Should allow errors with different messages at same location", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{Range: protocol.Range{Start: protocol.Position{Line: 1, Character: 0}}, Message: "error one", Severity: protocol.DiagnosticSeverityError})
			d.Add(diagnostics.Diagnostic{Range: protocol.Range{Start: protocol.Position{Line: 1, Character: 0}}, Message: "error two", Severity: protocol.DiagnosticSeverityError})
			Expect(d).To(HaveLen(2))
		})

		It("Should allow errors with same message at different locations", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{Range: protocol.Range{Start: protocol.Position{Line: 1, Character: 0}}, Message: "same error", Severity: protocol.DiagnosticSeverityError})
			d.Add(diagnostics.Diagnostic{Range: protocol.Range{Start: protocol.Position{Line: 2, Character: 0}}, Message: "same error", Severity: protocol.DiagnosticSeverityError})
			d.Add(diagnostics.Diagnostic{Range: protocol.Range{Start: protocol.Position{Line: 1, Character: 5}}, Message: "same error", Severity: protocol.DiagnosticSeverityError})
			Expect(d).To(HaveLen(3))
		})

		It("Should deduplicate warnings", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Warningf(nil, "unused variable"))
			d.Add(diagnostics.Warningf(nil, "unused variable"))
			Expect(d).To(HaveLen(1))
		})

		It("Should deduplicate info messages", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Infof(nil, "info message"))
			d.Add(diagnostics.Infof(nil, "info message"))
			Expect(d).To(HaveLen(1))
		})

		It("Should deduplicate hints", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Hintf(nil, "hint message"))
			d.Add(diagnostics.Hintf(nil, "hint message"))
			Expect(d).To(HaveLen(1))
		})

		It("Should keep higher severity when error comes first", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{Range: protocol.Range{Start: protocol.Position{Line: 1, Character: 0}}, Message: "same message", Severity: protocol.DiagnosticSeverityError})
			d.Add(diagnostics.Diagnostic{Range: protocol.Range{Start: protocol.Position{Line: 1, Character: 0}}, Message: "same message", Severity: protocol.DiagnosticSeverityWarning})
			Expect(d).To(HaveLen(1))
			Expect(d[0].Severity).To(Equal(protocol.DiagnosticSeverityError))
		})

		It("Should replace warning with error when error comes second", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{Range: protocol.Range{Start: protocol.Position{Line: 1, Character: 0}}, Message: "same message", Severity: protocol.DiagnosticSeverityWarning})
			d.Add(diagnostics.Diagnostic{Range: protocol.Range{Start: protocol.Position{Line: 1, Character: 0}}, Message: "same message", Severity: protocol.DiagnosticSeverityError})
			Expect(d).To(HaveLen(1))
			Expect(d[0].Severity).To(Equal(protocol.DiagnosticSeverityError))
		})

		It("Should keep error when hint comes second", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{Range: protocol.Range{Start: protocol.Position{Line: 1, Character: 0}}, Message: "issue", Severity: protocol.DiagnosticSeverityError})
			d.Add(diagnostics.Diagnostic{Range: protocol.Range{Start: protocol.Position{Line: 1, Character: 0}}, Message: "issue", Severity: protocol.DiagnosticSeverityHint})
			Expect(d).To(HaveLen(1))
			Expect(d[0].Severity).To(Equal(protocol.DiagnosticSeverityError))
		})

		It("Should replace info with warning", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{Range: protocol.Range{Start: protocol.Position{Line: 1, Character: 0}}, Message: "x", Severity: protocol.DiagnosticSeverityInformation})
			d.Add(diagnostics.Diagnostic{Range: protocol.Range{Start: protocol.Position{Line: 1, Character: 0}}, Message: "x", Severity: protocol.DiagnosticSeverityWarning})
			Expect(d).To(HaveLen(1))
			Expect(d[0].Severity).To(Equal(protocol.DiagnosticSeverityWarning))
		})

		It("Should replace hint with info", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{Range: protocol.Range{Start: protocol.Position{Line: 1, Character: 0}}, Message: "x", Severity: protocol.DiagnosticSeverityHint})
			d.Add(diagnostics.Diagnostic{Range: protocol.Range{Start: protocol.Position{Line: 1, Character: 0}}, Message: "x", Severity: protocol.DiagnosticSeverityInformation})
			Expect(d).To(HaveLen(1))
			Expect(d[0].Severity).To(Equal(protocol.DiagnosticSeverityInformation))
		})

		It("Should keep warning over hint", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{Range: protocol.Range{Start: protocol.Position{Line: 1, Character: 0}}, Message: "x", Severity: protocol.DiagnosticSeverityWarning})
			d.Add(diagnostics.Diagnostic{Range: protocol.Range{Start: protocol.Position{Line: 1, Character: 0}}, Message: "x", Severity: protocol.DiagnosticSeverityHint})
			Expect(d).To(HaveLen(1))
			Expect(d[0].Severity).To(Equal(protocol.DiagnosticSeverityWarning))
		})

		It("Should converge to highest severity across multiple adds", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{Range: protocol.Range{Start: protocol.Position{Line: 1, Character: 0}}, Message: "x", Severity: protocol.DiagnosticSeverityHint})
			d.Add(diagnostics.Diagnostic{Range: protocol.Range{Start: protocol.Position{Line: 1, Character: 0}}, Message: "x", Severity: protocol.DiagnosticSeverityInformation})
			d.Add(diagnostics.Diagnostic{Range: protocol.Range{Start: protocol.Position{Line: 1, Character: 0}}, Message: "x", Severity: protocol.DiagnosticSeverityWarning})
			d.Add(diagnostics.Diagnostic{Range: protocol.Range{Start: protocol.Position{Line: 1, Character: 0}}, Message: "x", Severity: protocol.DiagnosticSeverityError})
			Expect(d).To(HaveLen(1))
			Expect(d[0].Severity).To(Equal(protocol.DiagnosticSeverityError))
		})

		It("Should track multiple different messages at same location", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{Range: protocol.Range{Start: protocol.Position{Line: 5, Character: 10}}, Message: "msg1", Severity: protocol.DiagnosticSeverityError})
			d.Add(diagnostics.Diagnostic{Range: protocol.Range{Start: protocol.Position{Line: 5, Character: 10}}, Message: "msg2", Severity: protocol.DiagnosticSeverityWarning})
			d.Add(diagnostics.Diagnostic{Range: protocol.Range{Start: protocol.Position{Line: 5, Character: 10}}, Message: "msg3", Severity: protocol.DiagnosticSeverityHint})
			Expect(d).To(HaveLen(3))
		})
	})

	Describe("AtLocation", func() {
		It("Should return empty slice when no diagnostics at location", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{Range: protocol.Range{Start: protocol.Position{Line: 1, Character: 0}}, Message: "x", Severity: protocol.DiagnosticSeverityError})
			indices := d.AtLocation(protocol.Position{Line: 2, Character: 0})
			Expect(indices).To(BeEmpty())
		})

		It("Should return single index when one diagnostic at location", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{Range: protocol.Range{Start: protocol.Position{Line: 1, Character: 0}}, Message: "x", Severity: protocol.DiagnosticSeverityError})
			indices := d.AtLocation(protocol.Position{Line: 1, Character: 0})
			Expect(indices).To(HaveLen(1))
			Expect(indices[0]).To(Equal(0))
		})

		It("Should return multiple indices when multiple diagnostics at location", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{Range: protocol.Range{Start: protocol.Position{Line: 1, Character: 0}}, Message: "msg1", Severity: protocol.DiagnosticSeverityError})
			d.Add(diagnostics.Diagnostic{Range: protocol.Range{Start: protocol.Position{Line: 2, Character: 5}}, Message: "other", Severity: protocol.DiagnosticSeverityWarning})
			d.Add(diagnostics.Diagnostic{Range: protocol.Range{Start: protocol.Position{Line: 1, Character: 0}}, Message: "msg2", Severity: protocol.DiagnosticSeverityInformation})
			d.Add(diagnostics.Diagnostic{Range: protocol.Range{Start: protocol.Position{Line: 1, Character: 0}}, Message: "msg3", Severity: protocol.DiagnosticSeverityHint})
			indices := d.AtLocation(protocol.Position{Line: 1, Character: 0})
			Expect(indices).To(HaveLen(3))
			Expect(indices).To(ContainElements(0, 2, 3))
		})
	})

	Describe("Ok", func() {
		It("Should return true when empty", func() {
			var d diagnostics.Diagnostics
			Expect(d.Ok()).To(BeTrue())
		})

		It("Should return true when only warnings exist", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Warningf(nil, "warning"))
			Expect(d.Ok()).To(BeTrue())
		})

		It("Should return true when only info exists", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Infof(nil, "info"))
			Expect(d.Ok()).To(BeTrue())
		})

		It("Should return true when only hints exist", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Hintf(nil, "hint"))
			Expect(d.Ok()).To(BeTrue())
		})

		It("Should return false when errors exist", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Errorf(nil, "error"))
			Expect(d.Ok()).To(BeFalse())
		})

		It("Should return false when errors exist alongside warnings", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Warningf(nil, "warning"))
			d.Add(diagnostics.Errorf(nil, "error"))
			d.Add(diagnostics.Hintf(nil, "hint"))
			Expect(d.Ok()).To(BeFalse())
		})
	})

	Describe("Errors", func() {
		It("Should return empty slice when no errors", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Warningf(nil, "warning"))
			d.Add(diagnostics.Hintf(nil, "hint"))
			Expect(d.Errors()).To(BeEmpty())
		})

		It("Should return only error-level diagnostics", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Errorf(nil, "error1"))
			d.Add(diagnostics.Warningf(nil, "warning"))
			d.Add(diagnostics.Errorf(nil, "error2"))
			d.Add(diagnostics.Hintf(nil, "hint"))
			errs := d.Errors()
			Expect(errs).To(HaveLen(2))
			Expect(errs[0].Message).To(Equal("error1"))
			Expect(errs[1].Message).To(Equal("error2"))
		})
	})

	Describe("Warnings", func() {
		It("Should return empty slice when no warnings", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Errorf(nil, "error"))
			d.Add(diagnostics.Hintf(nil, "hint"))
			Expect(d.Warnings()).To(BeEmpty())
		})

		It("Should return only warning-level diagnostics", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Warningf(nil, "warning1"))
			d.Add(diagnostics.Errorf(nil, "error"))
			d.Add(diagnostics.Warningf(nil, "warning2"))
			d.Add(diagnostics.Hintf(nil, "hint"))
			warnings := d.Warnings()
			Expect(warnings).To(HaveLen(2))
			Expect(warnings[0].Message).To(Equal("warning1"))
			Expect(warnings[1].Message).To(Equal("warning2"))
		})
	})

	Describe("Empty", func() {
		It("Should return true when no diagnostics", func() {
			var d diagnostics.Diagnostics
			Expect(d.Empty()).To(BeTrue())
		})

		It("Should return false when diagnostics exist", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Warningf(nil, "warning"))
			Expect(d.Empty()).To(BeFalse())
		})
	})

	Describe("Merge", func() {
		It("Should merge diagnostics from another collection", func() {
			var d1, d2 diagnostics.Diagnostics
			d1.Add(diagnostics.Errorf(nil, "error1"))
			d2.Add(diagnostics.Errorf(nil, "error2"))
			d1.Merge(d2)
			Expect(d1).To(HaveLen(2))
		})

		It("Should deduplicate when merging", func() {
			var d1, d2 diagnostics.Diagnostics
			d1.Add(diagnostics.Errorf(nil, "same error"))
			d2.Add(diagnostics.Errorf(nil, "same error"))
			d1.Merge(d2)
			Expect(d1).To(HaveLen(1))
		})
	})

	Describe("String", func() {
		It("Should return success message when empty", func() {
			var d diagnostics.Diagnostics
			Expect(d.String()).To(Equal("analysis successful"))
		})

		It("Should format single diagnostic", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{
				Range:    protocol.Range{Start: protocol.Position{Line: 10, Character: 5}},
				Severity: protocol.DiagnosticSeverityError,
				Message:  "undefined symbol",
			})
			Expect(d.String()).To(Equal("11:5 error: undefined symbol"))
		})

		It("Should format multiple diagnostics with newlines", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{
				Range:    protocol.Range{Start: protocol.Position{Line: 1, Character: 0}},
				Severity: protocol.DiagnosticSeverityError,
				Message:  "first error",
			})
			d.Add(diagnostics.Diagnostic{
				Range:    protocol.Range{Start: protocol.Position{Line: 2, Character: 10}},
				Severity: protocol.DiagnosticSeverityWarning,
				Message:  "a warning",
			})
			expected := "2:0 error: first error\n3:10 warning: a warning"
			Expect(d.String()).To(Equal(expected))
		})

		DescribeTable("Should render each severity label",
			func(sev protocol.DiagnosticSeverity, label string) {
				var d diagnostics.Diagnostics
				d.Add(diagnostics.Diagnostic{
					Range:    protocol.Range{Start: protocol.Position{Line: 1, Character: 0}},
					Severity: sev,
					Message:  "m",
				})
				Expect(d.String()).To(Equal("2:0 " + label + ": m"))
			},
			Entry("info", protocol.DiagnosticSeverityInformation, "info"),
			Entry("hint", protocol.DiagnosticSeverityHint, "hint"),
			Entry("unknown", protocol.DiagnosticSeverity(99), "severity(99)"),
		)
	})

	Describe("Add methods with nil context", func() {
		It("Should handle nil context for Error", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Error(errors.New("error"), nil))
			Expect(d).To(HaveLen(1))
			Expect(d[0].Range.Start.Line).To(BeZero())
			Expect(d[0].Range.Start.Character).To(BeZero())
		})

		It("Should handle nil context for Warningf", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Warningf(nil, "warning"))
			Expect(d).To(HaveLen(1))
			Expect(d[0].Severity).To(Equal(protocol.DiagnosticSeverityWarning))
		})

		It("Should handle nil context for Infof", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Infof(nil, "info"))
			Expect(d).To(HaveLen(1))
			Expect(d[0].Severity).To(Equal(protocol.DiagnosticSeverityInformation))
		})

		It("Should handle nil context for Hintf", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Hintf(nil, "hint"))
			Expect(d).To(HaveLen(1))
			Expect(d[0].Severity).To(Equal(protocol.DiagnosticSeverityHint))
		})
	})

	Describe("Error Codes", func() {
		It("Should add error with code", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Errorf(nil, "type error").WithCode("TEST001"))
			Expect(d).To(HaveLen(1))
			Expect(d[0].Code).To(Equal(diagnostics.ErrorCode("TEST001")))
			Expect(d[0].Message).To(Equal("type error"))
		})

		It("Should format error code in string output", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{
				Range:    protocol.Range{Start: protocol.Position{Line: 1, Character: 5}},
				Severity: protocol.DiagnosticSeverityError,
				Code:     "TEST002",
				Message:  "wrong arg count",
			})
			Expect(d.String()).To(Equal("2:5 error [TEST002]: wrong arg count"))
		})
	})

	Describe("Notes", func() {
		It("Should add error with note", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Error(errors.New("type mismatch"), nil).WithNote("expected i64"))
			Expect(d).To(HaveLen(1))
			Expect(d[0].Notes).To(HaveLen(1))
			Expect(d[0].Notes[0].Message).To(Equal("expected i64"))
		})

		It("Should add error with code and note", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Errorf(nil, "wrong type").WithCode("TEST003").WithNote("signature: add(x i64, y i64) i64"))
			Expect(d).To(HaveLen(1))
			Expect(d[0].Code).To(Equal(diagnostics.ErrorCode("TEST003")))
			Expect(d[0].Notes).To(HaveLen(1))
			Expect(d[0].Notes[0].Message).To(Equal("signature: add(x i64, y i64) i64"))
		})

		It("Should format notes in string output", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{
				Range:    protocol.Range{Start: protocol.Position{Line: 1, Character: 0}},
				Severity: protocol.DiagnosticSeverityError,
				Message:  "error msg",
				Notes:    []protocol.DiagnosticRelatedInformation{{Message: "additional context"}},
			})
			str := d.String()
			Expect(str).To(ContainSubstring("error msg"))
			Expect(str).To(ContainSubstring("note: additional context"))
		})

		It("Should format note with position", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{
				Range:    protocol.Range{Start: protocol.Position{Line: 5, Character: 2}},
				Severity: protocol.DiagnosticSeverityError,
				Message:  "error here",
				Notes: []protocol.DiagnosticRelatedInformation{{
					Message: "related to this",
					Location: protocol.Location{Range: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 10},
					}},
				}},
			})
			str := d.String()
			Expect(str).To(ContainSubstring("4:10 note: related to this"))
		})

		It("Should skip empty note", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Error(errors.New("error"), nil).WithNote(""))
			Expect(d).To(HaveLen(1))
			Expect(d[0].Notes).To(BeEmpty())
		})
	})

	Describe("Position.Advance", func() {
		DescribeTable("Should walk body bytes and reset column on newlines",
			func(body string, start protocol.Position, off int, expected protocol.Position) {
				Expect(diagnostics.Advance(start, body, off)).To(Equal(expected))
			},
			Entry("zero offset returns start unchanged",
				"abc", protocol.Position{Line: 1, Character: 1}, 0,
				protocol.Position{Line: 1, Character: 1}),
			Entry("walks N non-newline bytes incrementing Col",
				"abc", protocol.Position{Line: 1, Character: 1}, 2,
				protocol.Position{Line: 1, Character: 3}),
			Entry("newline bumps Line and resets Col to 0",
				"a\nb", protocol.Position{Line: 1, Character: 1}, 2,
				protocol.Position{Line: 2, Character: 0}),
			Entry("consecutive newlines each reset Col",
				"\n\n", protocol.Position{Line: 1, Character: 1}, 2,
				protocol.Position{Line: 3, Character: 0}),
			Entry("mixed text and newline",
				"abc\ndef", protocol.Position{Line: 1, Character: 1}, 6,
				protocol.Position{Line: 2, Character: 2}),
			Entry("offset past len(body) clamps at end of body",
				"abc", protocol.Position{Line: 1, Character: 1}, 100,
				protocol.Position{Line: 1, Character: 4}),
			Entry("empty body returns start unchanged",
				"", protocol.Position{Line: 5, Character: 7}, 0,
				protocol.Position{Line: 5, Character: 7}),
			Entry("empty body with non-zero offset still returns start",
				"", protocol.Position{Line: 5, Character: 7}, 10,
				protocol.Position{Line: 5, Character: 7}),
			Entry("trailing newline lands on next line at col 0",
				"abc\n", protocol.Position{Line: 1, Character: 1}, 4,
				protocol.Position{Line: 2, Character: 0}),
			Entry("starts on a non-zero line and column",
				"xy", protocol.Position{Line: 7, Character: 3}, 2,
				protocol.Position{Line: 7, Character: 5}),
		)

		It("Should not mutate the receiver", func() {
			start := protocol.Position{Line: 1, Character: 1}
			_ = diagnostics.Advance(start, "a\nb", 3)
			Expect(start).To(Equal(protocol.Position{Line: 1, Character: 1}))
		})
	})

	Describe("Diagnostic.WithRange", func() {
		It("Should override Start and End regardless of prior values", func() {
			d := diagnostics.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: 1, Character: 0},
					End:   protocol.Position{Line: 1, Character: 5},
				},
			}
			out := d.WithRange(
				protocol.Position{Line: 3, Character: 2},
				protocol.Position{Line: 3, Character: 8},
			)
			Expect(out.Range.Start).To(Equal(protocol.Position{Line: 3, Character: 2}))
			Expect(out.Range.End).To(Equal(protocol.Position{Line: 3, Character: 8}))
		})

		It("Should return a copy and leave the original unchanged", func() {
			d := diagnostics.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: 1, Character: 0},
					End:   protocol.Position{Line: 1, Character: 5},
				},
			}
			_ = d.WithRange(
				protocol.Position{Line: 9, Character: 9},
				protocol.Position{Line: 9, Character: 9},
			)
			Expect(d.Range.Start).To(Equal(protocol.Position{Line: 1, Character: 0}))
			Expect(d.Range.End).To(Equal(protocol.Position{Line: 1, Character: 5}))
		})

		It("Should preserve unrelated fields", func() {
			d := diagnostics.Diagnostic{
				Severity: protocol.DiagnosticSeverityWarning,
				Message:  "msg",
				Code:     "C001",
				Notes:    []protocol.DiagnosticRelatedInformation{{Message: "n"}},
			}
			out := d.WithRange(
				protocol.Position{Line: 2, Character: 0},
				protocol.Position{Line: 2, Character: 4},
			)
			Expect(out.Severity).To(Equal(protocol.DiagnosticSeverityWarning))
			Expect(out.Message).To(Equal("msg"))
			Expect(out.Code).To(Equal(diagnostics.ErrorCode("C001")))
			Expect(out.Notes).To(HaveLen(1))
		})
	})
})

var _ = Describe("Diagnostic.SetRange", func() {
	It("Should convert 1-indexed ANTLR lines to a 0-indexed range", func() {
		var d diagnostics.Diagnostic
		d.SetRange(ruleCtxAt(
			&fakeToken{line: 3, column: 4, text: "foo"},
			&fakeToken{line: 5, column: 2, text: "bar"},
		))
		Expect(d.Range.Start).To(Equal(protocol.Position{Line: 2, Character: 4}))
		Expect(d.Range.End).To(Equal(protocol.Position{Line: 4, Character: 5}))
	})

	It("Should derive the end from the start token when stop is missing", func() {
		var d diagnostics.Diagnostic
		d.SetRange(ruleCtxAt(&fakeToken{line: 2, column: 1, text: "name"}, nil))
		Expect(d.Range.Start).To(Equal(protocol.Position{Line: 1, Character: 1}))
		Expect(d.Range.End).To(Equal(protocol.Position{Line: 1, Character: 5}))
	})

	It("Should leave the range zero for a nil context", func() {
		var d diagnostics.Diagnostic
		d.SetRange(nil)
		Expect(d.Range).To(Equal(protocol.Range{}))
	})
})

var _ = Describe("Error with HintProvider", func() {
	It("Should extract a hint into a note", func() {
		d := diagnostics.Error(hintedError{msg: "bad type", hint: "use i64"}, nil)
		Expect(d.Message).To(Equal("bad type"))
		Expect(d.Notes).To(HaveLen(1))
		Expect(d.Notes[0].Message).To(Equal("use i64"))
	})

	It("Should skip an empty hint", func() {
		d := diagnostics.Error(hintedError{msg: "bad type"}, nil)
		Expect(d.Notes).To(BeEmpty())
	})
})

var _ = Describe("Diagnostic.WithNoteAt", func() {
	It("Should attach a note pointing at the given position", func() {
		d := diagnostics.Errorf(nil, "boom").
			WithNoteAt("declared here", protocol.Position{Line: 4, Character: 7})
		Expect(d.Notes).To(HaveLen(1))
		Expect(d.Notes[0].Message).To(Equal("declared here"))
		Expect(d.Notes[0].Location.Range).To(Equal(protocol.Range{
			Start: protocol.Position{Line: 4, Character: 7},
			End:   protocol.Position{Line: 4, Character: 8},
		}))
	})

	It("Should skip an empty note", func() {
		d := diagnostics.Errorf(nil, "boom").
			WithNoteAt("", protocol.Position{Line: 4, Character: 7})
		Expect(d.Notes).To(BeEmpty())
	})
})

var _ = Describe("Diagnostics.Error", func() {
	It("Should report the formatted diagnostics as the error message", func() {
		d := diagnostics.Diagnostics{{
			Severity: protocol.DiagnosticSeverityError,
			Message:  "boom",
		}}
		var err error = &d
		Expect(err.Error()).To(Equal(d.String()))
	})
})
