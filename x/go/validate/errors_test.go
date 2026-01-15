// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package validate_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Errors", func() {
	Describe("PathedError", func() {
		It("Should add a path to an error", func() {
			base := errors.New("cat")
			pathed := validate.PathedError(base, "field")
			Expect(pathed).To(MatchError(ContainSubstring("field: cat")))
		})
		It("Should append nested paths correctly", func() {
			base := errors.New("cat")
			first := validate.PathedError(base, "first")
			parent := validate.PathedError(first, "parent")
			Expect(parent).To(MatchError(ContainSubstring("parent.first: cat")))
		})

		Describe("Encoding + Decoding", func() {
			It("Should correctly encode and decode", func() {
				base := errors.New("cat")
				pathed := validate.PathedError(base, "field")
				encoded := errors.Encode(ctx, pathed, false)
				decoded := errors.Decode(ctx, encoded)
				Expect(decoded).To(MatchError(ContainSubstring("field: cat")))
			})

			It("Should have the correct encoded representation", func() {
				base := errors.New("cat")
				pathed := validate.PathedError(base, "field")
				encoded := errors.Encode(ctx, pathed, false)
				Expect(encoded.Type).To(Equal("sy.validation.path"))
				Expect(encoded.Data).To(Equal("{\"error\":{\"type\":\"unknown\",\"data\":\"cat\"},\"path\":[\"field\"]}"))
			})

			It("Should correctly encode and decode nested paths", func() {
				base := errors.New("cat")
				first := validate.PathedError(base, "first")
				parent := validate.PathedError(first, "parent")
				encoded := errors.Encode(ctx, parent, false)
				decoded := errors.Decode(ctx, encoded)
				Expect(decoded).To(MatchError(ContainSubstring("parent.first: cat")))
			})
		})
	})

	Describe("InvalidTypeError", func() {
		It("Should format the error message correctly", func() {
			err := validate.NewInvalidTypeError("cat", "dog")
			Expect(err).To(HaveOccurredAs(validate.InvalidTypeError))
			Expect(err).To(MatchError(ContainSubstring("expected cat but received dog")))
		})
	})
})
