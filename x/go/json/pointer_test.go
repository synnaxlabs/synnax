// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package json_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xjson "github.com/synnaxlabs/x/json"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Pointer", func() {
	Describe("ParsePointer", func() {
		It("Should reject a pointer that does not start with a slash", func() {
			Expect(xjson.ParsePointer("temperature")).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(`must be empty or start with "/"`)),
			))
		})
		It("Should reject a tilde that does not form an escape", func() {
			Expect(xjson.ParsePointer("/a~2b")).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(`"~" must be followed by "0" or "1"`)),
			))
		})
		It("Should reject a trailing tilde", func() {
			Expect(xjson.ParsePointer("/a~")).Error().To(
				MatchError(validate.ErrValidation),
			)
		})
	})

	Describe("String", func() {
		DescribeTable("Should round trip the source text",
			func(text string) {
				Expect(MustSucceed(xjson.ParsePointer(text)).String()).To(Equal(text))
			},
			Entry("root", ""),
			Entry("key", "/temperature"),
			Entry("escapes", "/nested/a~1b/m~0n"),
			Entry("empty token", "/nested/"),
		)
	})
})
