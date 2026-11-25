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
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
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

var _ = Describe("Clusters", func() {
	// at returns a fault in the file id, from a read of the bytes from, to.
	at := func(id fault.ID, from, to int64) fault.Fault {
		return fault.Fault{
			File:  id,
			Block: from / 4096,
			Read:  fault.Span{From: from, To: to},
		}
	}

	DescribeTable("Should group the faults whose reads overlap in one file",
		func(faults []fault.Fault, sizes []int, most int) {
			got := make([]int, 0, len(sizes))
			for _, cluster := range clusters(faults) {
				got = append(got, len(cluster))
			}
			Expect(got).To(Equal(sizes))
			Expect(largest(faults)).To(Equal(most))
		},
		Entry("no fault", []fault.Fault(nil), []int{}, 0),
		Entry("two reads of the same bytes",
			[]fault.Fault{at(1, 0, 100), at(1, 0, 100)}, []int{2}, 2),
		Entry("two reads side by side",
			[]fault.Fault{at(1, 0, 100), at(1, 100, 200)}, []int{1, 1}, 1),
		Entry("the same bytes of two files",
			[]fault.Fault{at(1, 0, 100), at(2, 0, 100)}, []int{1, 1}, 1),
		Entry(
			"a pair and a fault apart",
			[]fault.Fault{
				at(1, 0, 100),
				at(1, 0, 100),
				at(1, 500, 600),
			},
			[]int{2, 1},
			2,
		),
		Entry("a chain of overlaps",
			[]fault.Fault{at(1, 0, 100), at(1, 200, 300), at(1, 50, 250)}, []int{3}, 3),
	)
})
