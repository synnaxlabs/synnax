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
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/diagnostics"
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
			d.AddError(err, nil)
			d.AddError(err, nil)
			d.AddError(err, nil)
			Expect(d).To(HaveLen(1))
		})

		It("Should allow errors with different messages at same location", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{Line: 1, Column: 0, Message: "error one", Severity: diagnostics.SeverityError})
			d.Add(diagnostics.Diagnostic{Line: 1, Column: 0, Message: "error two", Severity: diagnostics.SeverityError})
			Expect(d).To(HaveLen(2))
		})

		It("Should allow errors with same message at different locations", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{Line: 1, Column: 0, Message: "same error", Severity: diagnostics.SeverityError})
			d.Add(diagnostics.Diagnostic{Line: 2, Column: 0, Message: "same error", Severity: diagnostics.SeverityError})
			d.Add(diagnostics.Diagnostic{Line: 1, Column: 5, Message: "same error", Severity: diagnostics.SeverityError})
			Expect(d).To(HaveLen(3))
		})

		It("Should deduplicate warnings", func() {
			var d diagnostics.Diagnostics
			err := errors.New("unused variable")
			d.AddWarning(err, nil)
			d.AddWarning(err, nil)
			Expect(d).To(HaveLen(1))
		})

		It("Should deduplicate info messages", func() {
			var d diagnostics.Diagnostics
			err := errors.New("info message")
			d.AddInfo(err, nil)
			d.AddInfo(err, nil)
			Expect(d).To(HaveLen(1))
		})

		It("Should deduplicate hints", func() {
			var d diagnostics.Diagnostics
			err := errors.New("hint message")
			d.AddHint(err, nil)
			d.AddHint(err, nil)
			Expect(d).To(HaveLen(1))
		})

		It("Should keep higher severity when error comes first", func() {
			// Same location + message is considered duplicate even with different severity.
			// This prevents confusing output where the same issue is reported as both error and warning.
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{Line: 1, Column: 0, Message: "same message", Severity: diagnostics.SeverityError})
			d.Add(diagnostics.Diagnostic{Line: 1, Column: 0, Message: "same message", Severity: diagnostics.SeverityWarning})
			Expect(d).To(HaveLen(1))
			Expect(d[0].Severity).To(Equal(diagnostics.SeverityError)) // Error has higher severity
		})

		It("Should replace warning with error when error comes second", func() {
			// When a warning is added first, a later error at the same location should replace it.
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{Line: 1, Column: 0, Message: "same message", Severity: diagnostics.SeverityWarning})
			d.Add(diagnostics.Diagnostic{Line: 1, Column: 0, Message: "same message", Severity: diagnostics.SeverityError})
			Expect(d).To(HaveLen(1))
			Expect(d[0].Severity).To(Equal(diagnostics.SeverityError)) // Error replaces warning
		})

		It("Should keep error when hint comes second", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{Line: 1, Column: 0, Message: "issue", Severity: diagnostics.SeverityError})
			d.Add(diagnostics.Diagnostic{Line: 1, Column: 0, Message: "issue", Severity: diagnostics.SeverityHint})
			Expect(d).To(HaveLen(1))
			Expect(d[0].Severity).To(Equal(diagnostics.SeverityError))
		})

		It("Should replace info with warning", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{Line: 1, Column: 0, Message: "x", Severity: diagnostics.SeverityInfo})
			d.Add(diagnostics.Diagnostic{Line: 1, Column: 0, Message: "x", Severity: diagnostics.SeverityWarning})
			Expect(d).To(HaveLen(1))
			Expect(d[0].Severity).To(Equal(diagnostics.SeverityWarning))
		})

		It("Should replace hint with info", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{Line: 1, Column: 0, Message: "x", Severity: diagnostics.SeverityHint})
			d.Add(diagnostics.Diagnostic{Line: 1, Column: 0, Message: "x", Severity: diagnostics.SeverityInfo})
			Expect(d).To(HaveLen(1))
			Expect(d[0].Severity).To(Equal(diagnostics.SeverityInfo))
		})

		It("Should keep warning over hint", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{Line: 1, Column: 0, Message: "x", Severity: diagnostics.SeverityWarning})
			d.Add(diagnostics.Diagnostic{Line: 1, Column: 0, Message: "x", Severity: diagnostics.SeverityHint})
			Expect(d).To(HaveLen(1))
			Expect(d[0].Severity).To(Equal(diagnostics.SeverityWarning))
		})

		It("Should converge to highest severity across multiple adds", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{Line: 1, Column: 0, Message: "x", Severity: diagnostics.SeverityHint})
			d.Add(diagnostics.Diagnostic{Line: 1, Column: 0, Message: "x", Severity: diagnostics.SeverityInfo})
			d.Add(diagnostics.Diagnostic{Line: 1, Column: 0, Message: "x", Severity: diagnostics.SeverityWarning})
			d.Add(diagnostics.Diagnostic{Line: 1, Column: 0, Message: "x", Severity: diagnostics.SeverityError})
			Expect(d).To(HaveLen(1))
			Expect(d[0].Severity).To(Equal(diagnostics.SeverityError))
		})

		It("Should track multiple different messages at same location", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{Line: 5, Column: 10, Message: "msg1", Severity: diagnostics.SeverityError})
			d.Add(diagnostics.Diagnostic{Line: 5, Column: 10, Message: "msg2", Severity: diagnostics.SeverityWarning})
			d.Add(diagnostics.Diagnostic{Line: 5, Column: 10, Message: "msg3", Severity: diagnostics.SeverityHint})
			Expect(d).To(HaveLen(3))
		})
	})

	Describe("AtLocation", func() {
		It("Should return empty slice when no diagnostics at location", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{Line: 1, Column: 0, Message: "x", Severity: diagnostics.SeverityError})
			indices := d.AtLocation(2, 0)
			Expect(indices).To(BeEmpty())
		})

		It("Should return single index when one diagnostic at location", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{Line: 1, Column: 0, Message: "x", Severity: diagnostics.SeverityError})
			indices := d.AtLocation(1, 0)
			Expect(indices).To(HaveLen(1))
			Expect(indices[0]).To(Equal(0))
		})

		It("Should return multiple indices when multiple diagnostics at location", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{Line: 1, Column: 0, Message: "msg1", Severity: diagnostics.SeverityError})
			d.Add(diagnostics.Diagnostic{Line: 2, Column: 5, Message: "other", Severity: diagnostics.SeverityWarning})
			d.Add(diagnostics.Diagnostic{Line: 1, Column: 0, Message: "msg2", Severity: diagnostics.SeverityInfo})
			d.Add(diagnostics.Diagnostic{Line: 1, Column: 0, Message: "msg3", Severity: diagnostics.SeverityHint})
			indices := d.AtLocation(1, 0)
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
			d.AddWarning(errors.New("warning"), nil)
			Expect(d.Ok()).To(BeTrue())
		})

		It("Should return true when only info exists", func() {
			var d diagnostics.Diagnostics
			d.AddInfo(errors.New("info"), nil)
			Expect(d.Ok()).To(BeTrue())
		})

		It("Should return true when only hints exist", func() {
			var d diagnostics.Diagnostics
			d.AddHint(errors.New("hint"), nil)
			Expect(d.Ok()).To(BeTrue())
		})

		It("Should return false when errors exist", func() {
			var d diagnostics.Diagnostics
			d.AddError(errors.New("error"), nil)
			Expect(d.Ok()).To(BeFalse())
		})

		It("Should return false when errors exist alongside warnings", func() {
			var d diagnostics.Diagnostics
			d.AddWarning(errors.New("warning"), nil)
			d.AddError(errors.New("error"), nil)
			d.AddHint(errors.New("hint"), nil)
			Expect(d.Ok()).To(BeFalse())
		})
	})

	Describe("Errors", func() {
		It("Should return empty slice when no errors", func() {
			var d diagnostics.Diagnostics
			d.AddWarning(errors.New("warning"), nil)
			d.AddHint(errors.New("hint"), nil)
			Expect(d.Errors()).To(BeEmpty())
		})

		It("Should return only error-level diagnostics", func() {
			var d diagnostics.Diagnostics
			d.AddError(errors.New("error1"), nil)
			d.AddWarning(errors.New("warning"), nil)
			d.AddError(errors.New("error2"), nil)
			d.AddHint(errors.New("hint"), nil)
			errs := d.Errors()
			Expect(errs).To(HaveLen(2))
			Expect(errs[0].Message).To(Equal("error1"))
			Expect(errs[1].Message).To(Equal("error2"))
		})
	})

	Describe("Warnings", func() {
		It("Should return empty slice when no warnings", func() {
			var d diagnostics.Diagnostics
			d.AddError(errors.New("error"), nil)
			d.AddHint(errors.New("hint"), nil)
			Expect(d.Warnings()).To(BeEmpty())
		})

		It("Should return only warning-level diagnostics", func() {
			var d diagnostics.Diagnostics
			d.AddWarning(errors.New("warning1"), nil)
			d.AddError(errors.New("error"), nil)
			d.AddWarning(errors.New("warning2"), nil)
			d.AddHint(errors.New("hint"), nil)
			warnings := d.Warnings()
			Expect(warnings).To(HaveLen(2))
			Expect(warnings[0].Message).To(Equal("warning1"))
			Expect(warnings[1].Message).To(Equal("warning2"))
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
				Line:     10,
				Column:   5,
				Severity: diagnostics.SeverityError,
				Message:  "undefined symbol",
			})
			Expect(d.String()).To(Equal("10:5 error: undefined symbol"))
		})

		It("Should format multiple diagnostics with newlines", func() {
			var d diagnostics.Diagnostics
			d.Add(diagnostics.Diagnostic{
				Line:     1,
				Column:   0,
				Severity: diagnostics.SeverityError,
				Message:  "first error",
			})
			d.Add(diagnostics.Diagnostic{
				Line:     2,
				Column:   10,
				Severity: diagnostics.SeverityWarning,
				Message:  "a warning",
			})
			expected := "1:0 error: first error\n2:10 warning: a warning"
			Expect(d.String()).To(Equal(expected))
		})
	})

	Describe("Error interface", func() {
		It("Should implement error interface", func() {
			var d diagnostics.Diagnostics
			d.AddError(errors.New("test error"), nil)
			var err error = d
			Expect(err.Error()).To(ContainSubstring("test error"))
		})
	})

	Describe("Add methods with nil context", func() {
		It("Should handle nil context for AddError", func() {
			var d diagnostics.Diagnostics
			d.AddError(errors.New("error"), nil)
			Expect(d).To(HaveLen(1))
			Expect(d[0].Line).To(Equal(0))
			Expect(d[0].Column).To(Equal(0))
		})

		It("Should handle nil context for AddWarning", func() {
			var d diagnostics.Diagnostics
			d.AddWarning(errors.New("warning"), nil)
			Expect(d).To(HaveLen(1))
			Expect(d[0].Severity).To(Equal(diagnostics.SeverityWarning))
		})

		It("Should handle nil context for AddInfo", func() {
			var d diagnostics.Diagnostics
			d.AddInfo(errors.New("info"), nil)
			Expect(d).To(HaveLen(1))
			Expect(d[0].Severity).To(Equal(diagnostics.SeverityInfo))
		})

		It("Should handle nil context for AddHint", func() {
			var d diagnostics.Diagnostics
			d.AddHint(errors.New("hint"), nil)
			Expect(d).To(HaveLen(1))
			Expect(d[0].Severity).To(Equal(diagnostics.SeverityHint))
		})
	})
})
