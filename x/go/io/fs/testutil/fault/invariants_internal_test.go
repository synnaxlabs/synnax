// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build invariants || race

package fault

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Invariants", func() {
	DescribeTable("Should panic on synced ranges that are not sorted and apart",
		func(synced []Span) {
			Expect(func() { checkSynced(synced) }).To(PanicWith(MatchError(
				ContainSubstring("are not sorted and apart"),
			)))
		},
		Entry("out of order", []Span{{From: 10, To: 20}, {From: 0, To: 5}}),
		Entry("touching", []Span{{From: 0, To: 5}, {From: 5, To: 9}}),
		Entry("empty", []Span{{From: 3, To: 3}}),
	)

	It("Should pass synced ranges that are sorted and apart", func() {
		synced := []Span{{From: 0, To: 5}, {From: 6, To: 9}}
		Expect(func() { checkSynced(synced) }).ToNot(Panic())
	})
})
