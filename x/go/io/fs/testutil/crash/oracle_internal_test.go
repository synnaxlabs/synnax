// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package crash

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Second recovery", func() {
	attempts := []attempt{
		{record: Record{ID: 1}, ack: Durable, ackedAt: 2},
		{record: Record{ID: 2}, ack: Visible, ackedAt: 4},
	}
	first := []Record{{ID: 1}, {ID: 2}}
	relaxed := rules{exemptFrom: 0, refusalAllowed: true, unreadableAllowed: true}
	changed := rules{exemptFrom: strictRules.exemptFrom, unreadableAllowed: true}

	DescribeTable(
		"Should judge what the second recovery kept",
		func(second []Record, r rules, faulted bool, rule Rule, lost bool) {
			b, gotLost := checkSecond(attempts, first, second, r, faulted)
			if rule == "" {
				Expect(b).To(BeNil())
			} else {
				Expect(b).ToNot(BeNil())
				Expect(b.rule).To(Equal(rule))
			}
			Expect(gotLost).To(Equal(lost))
		},
		Entry("all that the first held", first, strictRules, false, Rule(""), false),
		Entry("a missing commit with no fault",
			first[:1], strictRules, false, R1, false),
		Entry(
			"a missing commit after faults",
			first[:1],
			relaxed,
			true,
			Rule(""),
			true,
		),
		Entry("a missing durable commit after faults",
			[]Record(nil), relaxed, true, Rule(""), true),
		Entry("an unreadable commit that the first read, with no fault",
			[]Record{{ID: 1}, {ID: 2, Unreadable: true}}, changed, false, R1, false),
		Entry(
			"an unreadable commit that the first read, after faults",
			[]Record{
				{ID: 1},
				{ID: 2, Unreadable: true},
			},
			relaxed,
			true,
			Rule(""),
			false,
		),
	)
})
