// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	stderrors "errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	"github.com/synnaxlabs/x/errors"
	_ "github.com/synnaxlabs/x/testutil" // registers the custom formatter
)

var _ = Describe("Format", func() {
	Describe("formatErrorWithStack", func() {
		It("should include stack trace when formatting errors created with errors.New", func() {
			err := errors.New("test error")
			formatted := format.Object(err, 1)
			Expect(formatted).To(ContainSubstring("test error"))
			Expect(formatted).To(ContainSubstring("Error Origin Stack Trace:"))
			Expect(formatted).To(ContainSubstring("format_test.go"))
		})

		It("should include stack trace when formatting wrapped errors", func() {
			baseErr := errors.New("base error")
			wrappedErr := errors.Wrap(baseErr, "wrapped")
			formatted := format.Object(wrappedErr, 1)
			Expect(formatted).To(ContainSubstring("wrapped: base error"))
			Expect(formatted).To(ContainSubstring("Error Origin Stack Trace:"))
		})

		It("should handle errors without stack traces gracefully", func() {
			err := stderrors.New("standard error")
			formatted := format.Object(err, 1)
			Expect(formatted).To(ContainSubstring("standard error"))
			// Should not contain stack trace header since std errors don't have stack traces
		})

		It("should not affect non-error values", func() {
			value := "just a string"
			formatted := format.Object(value, 1)
			Expect(formatted).To(ContainSubstring("just a string"))
			Expect(formatted).ToNot(ContainSubstring("Error Origin Stack Trace:"))
		})
	})
})
