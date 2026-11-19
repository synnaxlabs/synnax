// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build invariants || race

package crash

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Invariants", func() {
	It("Should panic on an attempt out of commit order", func() {
		attempts := []attempt{{record: Record{ID: 1}}, {record: Record{ID: 3}}}
		Expect(func() { checkOrder(attempts) }).To(PanicWith(MatchError(
			"crash: attempt 2 holds commit 3",
		)))
	})

	It("Should pass attempts in commit order", func() {
		attempts := []attempt{{record: Record{ID: 1}}, {record: Record{ID: 2}}}
		Expect(func() { checkOrder(attempts) }).ToNot(Panic())
	})
})
