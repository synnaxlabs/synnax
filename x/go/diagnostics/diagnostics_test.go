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
	"github.com/synnaxlabs/x/diagnostics"
)

var _ = Describe("Diagnostics", func() {
	Describe("Severity", func() {
		DescribeTable("String",
			func(s diagnostics.Severity, expected string) {
				Expect(s.String()).To(Equal(expected))
			},
			Entry("Error", diagnostics.SeverityError, "error"),
			Entry("SeverityWarning", diagnostics.SeverityWarning, "warning"),
			Entry("SeverityInfo", diagnostics.SeverityInfo, "info"),
			Entry("SeverityHint", diagnostics.SeverityHint, "hint"),
			Entry("Unknown", diagnostics.Severity(99), "Severity(99)"),
		)
	})

	Describe("Diagnostics Collection", func() {
		var d diagnostics.Diagnostics

		BeforeEach(func() {
			d = diagnostics.Diagnostics{}
		})

		Describe("Ok", func() {
			It("Should return true for empty collection", func() {
				Expect(d.Ok()).To(BeTrue())
			})
			It("Should return false when diagnostics exist", func() {
				d.Add(diagnostics.Diagnostic{Severity: diagnostics.SeverityWarning, Message: "warn"})
				Expect(d.Ok()).To(BeFalse())
			})
		})

		Describe("HasErrors", func() {
			It("Should return false for empty collection", func() {
				Expect(d.HasErrors()).To(BeFalse())
			})
			It("Should return false when only warnings exist", func() {
				d.Add(diagnostics.Diagnostic{Severity: diagnostics.SeverityWarning, Message: "warn"})
				Expect(d.HasErrors()).To(BeFalse())
			})
			It("Should return true when errors exist", func() {
				d.Add(diagnostics.Diagnostic{Severity: diagnostics.SeverityError, Message: "err"})
				Expect(d.HasErrors()).To(BeTrue())
			})
		})

		Describe("Empty", func() {
			It("Should return true for empty collection", func() {
				Expect(d.Empty()).To(BeTrue())
			})
			It("Should return false when diagnostics exist", func() {
				d.Add(diagnostics.Diagnostic{Severity: diagnostics.SeverityInfo, Message: "info"})
				Expect(d.Empty()).To(BeFalse())
			})
		})

		Describe("Add", func() {
			It("Should add a diagnostic to the collection", func() {
				d.Add(diagnostics.Diagnostic{
					Key:      "test-key",
					Severity: diagnostics.SeverityError,
					Line:     10,
					Column:   5,
					Message:  "test message",
					File:     "test.go",
				})
				Expect(d).To(HaveLen(1))
				Expect(d[0].Key).To(Equal("test-key"))
				Expect(d[0].Severity).To(Equal(diagnostics.SeverityError))
				Expect(d[0].Line).To(Equal(10))
				Expect(d[0].Column).To(Equal(5))
				Expect(d[0].Message).To(Equal("test message"))
				Expect(d[0].File).To(Equal("test.go"))
			})
		})

		Describe("AddError", func() {
			It("Should add an error-level diagnostic", func() {
				d.AddError(errors.New("something failed"), nil)
				Expect(d).To(HaveLen(1))
				Expect(d[0].Severity).To(Equal(diagnostics.SeverityError))
				Expect(d[0].Message).To(Equal("something failed"))
			})
			It("Should include file when provided", func() {
				d.AddError(errors.New("failed"), nil, "main.go")
				Expect(d[0].File).To(Equal("main.go"))
			})
		})

		Describe("AddErrorf", func() {
			It("Should format the error message", func() {
				d.AddErrorf(nil, "test.go", "expected %d but got %d", 1, 2)
				Expect(d).To(HaveLen(1))
				Expect(d[0].Severity).To(Equal(diagnostics.SeverityError))
				Expect(d[0].Message).To(Equal("expected 1 but got 2"))
				Expect(d[0].File).To(Equal("test.go"))
			})
		})

		Describe("AddWarning", func() {
			It("Should add a warning-level diagnostic", func() {
				d.AddWarning(errors.New("might fail"), nil)
				Expect(d).To(HaveLen(1))
				Expect(d[0].Severity).To(Equal(diagnostics.SeverityWarning))
				Expect(d[0].Message).To(Equal("might fail"))
			})
		})

		Describe("AddWarningf", func() {
			It("Should format the warning message", func() {
				d.AddWarningf(nil, "", "unused variable %s", "x")
				Expect(d[0].Severity).To(Equal(diagnostics.SeverityWarning))
				Expect(d[0].Message).To(Equal("unused variable x"))
			})
		})

		Describe("AddInfo", func() {
			It("Should add an info-level diagnostic", func() {
				d.AddInfo(errors.New("processing started"), nil)
				Expect(d[0].Severity).To(Equal(diagnostics.SeverityInfo))
			})
		})

		Describe("AddHint", func() {
			It("Should add a hint-level diagnostic", func() {
				d.AddHint(errors.New("consider using const"), nil)
				Expect(d[0].Severity).To(Equal(diagnostics.SeverityHint))
			})
		})

		Describe("Merge", func() {
			It("Should merge diagnostics from another collection", func() {
				d.Add(diagnostics.Diagnostic{Message: "first"})
				other := diagnostics.Diagnostics{
					{Message: "second"},
					{Message: "third"},
				}
				d.Merge(other)
				Expect(d).To(HaveLen(3))
				Expect(d[0].Message).To(Equal("first"))
				Expect(d[1].Message).To(Equal("second"))
				Expect(d[2].Message).To(Equal("third"))
			})
		})

		Describe("Errors", func() {
			It("Should return only error-level diagnostics", func() {
				d.Add(diagnostics.Diagnostic{Severity: diagnostics.SeverityError, Message: "err1"})
				d.Add(diagnostics.Diagnostic{Severity: diagnostics.SeverityWarning, Message: "warn"})
				d.Add(diagnostics.Diagnostic{Severity: diagnostics.SeverityError, Message: "err2"})
				d.Add(diagnostics.Diagnostic{Severity: diagnostics.SeverityInfo, Message: "info"})

				errs := d.Errors()
				Expect(errs).To(HaveLen(2))
				Expect(errs[0].Message).To(Equal("err1"))
				Expect(errs[1].Message).To(Equal("err2"))
			})
			It("Should return empty collection when no errors", func() {
				d.Add(diagnostics.Diagnostic{Severity: diagnostics.SeverityWarning, Message: "warn"})
				Expect(d.Errors()).To(BeEmpty())
			})
		})

		Describe("Warnings", func() {
			It("Should return only warning-level diagnostics", func() {
				d.Add(diagnostics.Diagnostic{Severity: diagnostics.SeverityError, Message: "err"})
				d.Add(diagnostics.Diagnostic{Severity: diagnostics.SeverityWarning, Message: "warn1"})
				d.Add(diagnostics.Diagnostic{Severity: diagnostics.SeverityWarning, Message: "warn2"})
				d.Add(diagnostics.Diagnostic{Severity: diagnostics.SeverityInfo, Message: "info"})

				warns := d.Warnings()
				Expect(warns).To(HaveLen(2))
				Expect(warns[0].Message).To(Equal("warn1"))
				Expect(warns[1].Message).To(Equal("warn2"))
			})
			It("Should return empty collection when no warnings", func() {
				d.Add(diagnostics.Diagnostic{Severity: diagnostics.SeverityError, Message: "err"})
				Expect(d.Warnings()).To(BeEmpty())
			})
		})

		Describe("String", func() {
			It("Should return success message for empty collection", func() {
				Expect(d.String()).To(Equal("analysis successful"))
			})
			It("Should format diagnostic without file", func() {
				d.Add(diagnostics.Diagnostic{
					Severity: diagnostics.SeverityError,
					Line:     10,
					Column:   5,
					Message:  "undefined variable",
				})
				Expect(d.String()).To(Equal("10:5 error: undefined variable"))
			})
			It("Should format diagnostic with file", func() {
				d.Add(diagnostics.Diagnostic{
					Severity: diagnostics.SeverityWarning,
					Line:     20,
					Column:   3,
					Message:  "unused import",
					File:     "main.go",
				})
				Expect(d.String()).To(Equal("main.go:20:3 warning: unused import"))
			})
			It("Should separate multiple diagnostics with newlines", func() {
				d.Add(diagnostics.Diagnostic{Severity: diagnostics.SeverityError, Line: 1, Column: 0, Message: "first"})
				d.Add(diagnostics.Diagnostic{Severity: diagnostics.SeverityWarning, Line: 2, Column: 0, Message: "second"})
				Expect(d.String()).To(Equal("1:0 error: first\n2:0 warning: second"))
			})
		})

		Describe("Error interface", func() {
			It("Should implement error interface", func() {
				d.Add(diagnostics.Diagnostic{Severity: diagnostics.SeverityError, Line: 1, Column: 0, Message: "failed"})
				var err error = d
				Expect(err.Error()).To(Equal("1:0 error: failed"))
			})
		})
	})

	Describe("FromError", func() {
		It("Should create diagnostics from an error", func() {
			err := errors.New("something went wrong")
			d := diagnostics.FromError(err)
			Expect(d).ToNot(BeNil())
			Expect(*d).To(HaveLen(1))
			Expect((*d)[0].Severity).To(Equal(diagnostics.SeverityError))
			Expect((*d)[0].Message).To(Equal("something went wrong"))
		})
	})
})
