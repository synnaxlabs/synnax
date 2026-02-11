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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/errors"
)

var _ = Describe("Diagnostics", func() {
	Describe("Severity", func() {
		It("Should have correct string representation for Error", func() {
			Expect(diagnostics.SeverityError.String()).To(Equal("error"))
		})

		It("Should have correct string representation for Warning", func() {
			Expect(diagnostics.SeverityWarning.String()).To(Equal("warning"))
		})

		It("Should have correct string representation for Info", func() {
			Expect(diagnostics.SeverityInfo.String()).To(Equal("info"))
		})

		It("Should have correct string representation for Hint", func() {
			Expect(diagnostics.SeverityHint.String()).To(Equal("hint"))
		})

		It("Should handle unknown severity", func() {
			unknown := diagnostics.Severity(99)
			Expect(unknown.String()).To(Equal("Severity(99)"))
		})
	})

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
			d.Add(diagnostics.Diagnostic{Start: diagnostics.Position{Line: 1, Col: 0}, Message: "error one", Severity: diagnostics.SeverityError})
			d.Add(diagnostics.Diagnostic{Start: diagnostics.Position{Line: 1, Col: 0}, Message: "error two", Severity: diagnostics.SeverityError})
			Expect(d).To(HaveLen(2))
		})

		It("Should allow errors with same message at different locations", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{Start: diagnostics.Position{Line: 1, Col: 0}, Message: "same error", Severity: diagnostics.SeverityError})
			d.Add(diagnostics.Diagnostic{Start: diagnostics.Position{Line: 2, Col: 0}, Message: "same error", Severity: diagnostics.SeverityError})
			d.Add(diagnostics.Diagnostic{Start: diagnostics.Position{Line: 1, Col: 5}, Message: "same error", Severity: diagnostics.SeverityError})
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
			d.Add(diagnostics.Diagnostic{Start: diagnostics.Position{Line: 1, Col: 0}, Message: "same message", Severity: diagnostics.SeverityError})
			d.Add(diagnostics.Diagnostic{Start: diagnostics.Position{Line: 1, Col: 0}, Message: "same message", Severity: diagnostics.SeverityWarning})
			Expect(d).To(HaveLen(1))
			Expect(d[0].Severity).To(Equal(diagnostics.SeverityError))
		})

		It("Should replace warning with error when error comes second", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{Start: diagnostics.Position{Line: 1, Col: 0}, Message: "same message", Severity: diagnostics.SeverityWarning})
			d.Add(diagnostics.Diagnostic{Start: diagnostics.Position{Line: 1, Col: 0}, Message: "same message", Severity: diagnostics.SeverityError})
			Expect(d).To(HaveLen(1))
			Expect(d[0].Severity).To(Equal(diagnostics.SeverityError))
		})

		It("Should keep error when hint comes second", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{Start: diagnostics.Position{Line: 1, Col: 0}, Message: "issue", Severity: diagnostics.SeverityError})
			d.Add(diagnostics.Diagnostic{Start: diagnostics.Position{Line: 1, Col: 0}, Message: "issue", Severity: diagnostics.SeverityHint})
			Expect(d).To(HaveLen(1))
			Expect(d[0].Severity).To(Equal(diagnostics.SeverityError))
		})

		It("Should replace info with warning", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{Start: diagnostics.Position{Line: 1, Col: 0}, Message: "x", Severity: diagnostics.SeverityInfo})
			d.Add(diagnostics.Diagnostic{Start: diagnostics.Position{Line: 1, Col: 0}, Message: "x", Severity: diagnostics.SeverityWarning})
			Expect(d).To(HaveLen(1))
			Expect(d[0].Severity).To(Equal(diagnostics.SeverityWarning))
		})

		It("Should replace hint with info", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{Start: diagnostics.Position{Line: 1, Col: 0}, Message: "x", Severity: diagnostics.SeverityHint})
			d.Add(diagnostics.Diagnostic{Start: diagnostics.Position{Line: 1, Col: 0}, Message: "x", Severity: diagnostics.SeverityInfo})
			Expect(d).To(HaveLen(1))
			Expect(d[0].Severity).To(Equal(diagnostics.SeverityInfo))
		})

		It("Should keep warning over hint", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{Start: diagnostics.Position{Line: 1, Col: 0}, Message: "x", Severity: diagnostics.SeverityWarning})
			d.Add(diagnostics.Diagnostic{Start: diagnostics.Position{Line: 1, Col: 0}, Message: "x", Severity: diagnostics.SeverityHint})
			Expect(d).To(HaveLen(1))
			Expect(d[0].Severity).To(Equal(diagnostics.SeverityWarning))
		})

		It("Should converge to highest severity across multiple adds", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{Start: diagnostics.Position{Line: 1, Col: 0}, Message: "x", Severity: diagnostics.SeverityHint})
			d.Add(diagnostics.Diagnostic{Start: diagnostics.Position{Line: 1, Col: 0}, Message: "x", Severity: diagnostics.SeverityInfo})
			d.Add(diagnostics.Diagnostic{Start: diagnostics.Position{Line: 1, Col: 0}, Message: "x", Severity: diagnostics.SeverityWarning})
			d.Add(diagnostics.Diagnostic{Start: diagnostics.Position{Line: 1, Col: 0}, Message: "x", Severity: diagnostics.SeverityError})
			Expect(d).To(HaveLen(1))
			Expect(d[0].Severity).To(Equal(diagnostics.SeverityError))
		})

		It("Should track multiple different messages at same location", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{Start: diagnostics.Position{Line: 5, Col: 10}, Message: "msg1", Severity: diagnostics.SeverityError})
			d.Add(diagnostics.Diagnostic{Start: diagnostics.Position{Line: 5, Col: 10}, Message: "msg2", Severity: diagnostics.SeverityWarning})
			d.Add(diagnostics.Diagnostic{Start: diagnostics.Position{Line: 5, Col: 10}, Message: "msg3", Severity: diagnostics.SeverityHint})
			Expect(d).To(HaveLen(3))
		})
	})

	Describe("AtLocation", func() {
		It("Should return empty slice when no diagnostics at location", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{Start: diagnostics.Position{Line: 1, Col: 0}, Message: "x", Severity: diagnostics.SeverityError})
			indices := d.AtLocation(diagnostics.Position{Line: 2, Col: 0})
			Expect(indices).To(BeEmpty())
		})

		It("Should return single index when one diagnostic at location", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{Start: diagnostics.Position{Line: 1, Col: 0}, Message: "x", Severity: diagnostics.SeverityError})
			indices := d.AtLocation(diagnostics.Position{Line: 1, Col: 0})
			Expect(indices).To(HaveLen(1))
			Expect(indices[0]).To(Equal(0))
		})

		It("Should return multiple indices when multiple diagnostics at location", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{Start: diagnostics.Position{Line: 1, Col: 0}, Message: "msg1", Severity: diagnostics.SeverityError})
			d.Add(diagnostics.Diagnostic{Start: diagnostics.Position{Line: 2, Col: 5}, Message: "other", Severity: diagnostics.SeverityWarning})
			d.Add(diagnostics.Diagnostic{Start: diagnostics.Position{Line: 1, Col: 0}, Message: "msg2", Severity: diagnostics.SeverityInfo})
			d.Add(diagnostics.Diagnostic{Start: diagnostics.Position{Line: 1, Col: 0}, Message: "msg3", Severity: diagnostics.SeverityHint})
			indices := d.AtLocation(diagnostics.Position{Line: 1, Col: 0})
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
				Start:    diagnostics.Position{Line: 10, Col: 5},
				Severity: diagnostics.SeverityError,
				Message:  "undefined symbol",
			})
			Expect(d.String()).To(Equal("10:5 error: undefined symbol"))
		})

		It("Should format multiple diagnostics with newlines", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{
				Start:    diagnostics.Position{Line: 1, Col: 0},
				Severity: diagnostics.SeverityError,
				Message:  "first error",
			})
			d.Add(diagnostics.Diagnostic{
				Start:    diagnostics.Position{Line: 2, Col: 10},
				Severity: diagnostics.SeverityWarning,
				Message:  "a warning",
			})
			expected := "1:0 error: first error\n2:10 warning: a warning"
			Expect(d.String()).To(Equal(expected))
		})
	})

	Describe("Add methods with nil context", func() {
		It("Should handle nil context for Error", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Error(errors.New("error"), nil))
			Expect(d).To(HaveLen(1))
			Expect(d[0].Start.Line).To(Equal(0))
			Expect(d[0].Start.Col).To(Equal(0))
		})

		It("Should handle nil context for Warningf", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Warningf(nil, "warning"))
			Expect(d).To(HaveLen(1))
			Expect(d[0].Severity).To(Equal(diagnostics.SeverityWarning))
		})

		It("Should handle nil context for Infof", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Infof(nil, "info"))
			Expect(d).To(HaveLen(1))
			Expect(d[0].Severity).To(Equal(diagnostics.SeverityInfo))
		})

		It("Should handle nil context for Hintf", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Hintf(nil, "hint"))
			Expect(d).To(HaveLen(1))
			Expect(d[0].Severity).To(Equal(diagnostics.SeverityHint))
		})
	})

	Describe("Error Codes", func() {
		It("Should add error with code", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Errorf(nil, "type error").WithCode(diagnostics.ErrorCodeTypeMismatch))
			Expect(d).To(HaveLen(1))
			Expect(d[0].Code).To(Equal(diagnostics.ErrorCodeTypeMismatch))
			Expect(d[0].Message).To(Equal("type error"))
		})

		It("Should format error code in string output", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{
				Start:    diagnostics.Position{Line: 1, Col: 5},
				Severity: diagnostics.SeverityError,
				Code:     diagnostics.ErrorCodeFuncArgCount,
				Message:  "wrong arg count",
			})
			Expect(d.String()).To(Equal("1:5 error [ARC3001]: wrong arg count"))
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
			d.Add(diagnostics.Errorf(nil, "wrong type").WithCode(diagnostics.ErrorCodeFuncArgType).WithNote("signature: add(x i64, y i64) i64"))
			Expect(d).To(HaveLen(1))
			Expect(d[0].Code).To(Equal(diagnostics.ErrorCodeFuncArgType))
			Expect(d[0].Notes).To(HaveLen(1))
			Expect(d[0].Notes[0].Message).To(Equal("signature: add(x i64, y i64) i64"))
		})

		It("Should format notes in string output", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{
				Start:    diagnostics.Position{Line: 1, Col: 0},
				Severity: diagnostics.SeverityError,
				Message:  "error msg",
				Notes:    []diagnostics.Note{{Message: "additional context"}},
			})
			str := d.String()
			Expect(str).To(ContainSubstring("error msg"))
			Expect(str).To(ContainSubstring("note: additional context"))
		})

		It("Should format note with position", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{
				Start:    diagnostics.Position{Line: 5, Col: 2},
				Severity: diagnostics.SeverityError,
				Message:  "error here",
				Notes: []diagnostics.Note{{
					Message: "related to this",
					Start:   diagnostics.Position{Line: 3, Col: 10},
				}},
			})
			str := d.String()
			Expect(str).To(ContainSubstring("3:10 note: related to this"))
		})

		It("Should skip empty note", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Error(errors.New("error"), nil).WithNote(""))
			Expect(d).To(HaveLen(1))
			Expect(d[0].Notes).To(BeEmpty())
		})
	})
})
