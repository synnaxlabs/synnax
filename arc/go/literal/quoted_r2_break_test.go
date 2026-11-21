// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package literal_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/literal"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe(
	"Format string placeholders beside quoted names (round 2 break)",
	func() {
		It("R2-L1 closes a placeholder that holds a string with an apostrophe", func() {
			Expect(MustSucceed(literal.FmtStrParse(`{"it's"}`))).To(Equal(
				[]literal.FmtStrSegment{{
					Text:          `"it's"`,
					IsPlaceholder: true,
					Start:         0,
					End:           8,
					SpecOffset:    -1,
				}},
			))
		})

		It("R2-L2 splits the spec after a string that holds one bracket", func() {
			Expect(MustSucceed(literal.FmtStrParse(`{"[":s}`))).To(Equal(
				[]literal.FmtStrSegment{{
					Text:          `"["`,
					Spec:          "s",
					IsPlaceholder: true,
					Start:         0,
					End:           7,
					SpecOffset:    4,
				}},
			))
		})
	},
)
