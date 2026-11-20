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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Path segment breakers", func() {
	It("Should sanitize a long run of combining marks to a legal name", func() {
		s := "a" + strings.Repeat("\u0301", 31)
		out := validate.SanitizePathSegment(s)
		Expect(out).ToNot(ContainSubstring("\u034F"))
		Expect(validate.IsPathSegment(out)).To(BeTrue())
		Expect(validate.IsNormalName(out)).To(BeTrue())
	})

	It("Should turn each invalid byte into U+FFFD", func() {
		Expect(validate.SanitizePathSegment("a\xff\xfeb")).
			To(Equal("a\uFFFD\uFFFDb"))
	})
})
