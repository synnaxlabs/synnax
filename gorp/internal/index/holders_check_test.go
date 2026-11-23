// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package index_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp/internal/index"
)

var _ = Describe("Lookup holders across sizes", func() {
	var l *index.Lookup[*scope, int, string]
	BeforeEach(func() {
		l = index.NewLookup[*scope, int, string]()
		_, finish := l.Populate()
		finish(nil)
	})

	It("Should keep every holder of a value as it grows and shrinks", func() {
		Expect(l.Get(nil, "v")).To(BeEmpty())
		l.Set(1, "v")
		l.Set(1, "v")
		Expect(l.Get(nil, "v")).To(ConsistOf(1))
		l.Set(2, "v")
		Expect(l.Get(nil, "v")).To(ConsistOf(1, 2))
		l.Delete(1)
		Expect(l.Get(nil, "v")).To(ConsistOf(2))
		l.Delete(2)
		Expect(l.Get(nil, "v")).To(BeEmpty())
		l.Set(3, "v")
		Expect(l.Get(nil, "v")).To(ConsistOf(3))
		var all []int
		for k := 3; k <= 40; k++ {
			l.Set(k, "v")
			all = append(all, k)
		}
		Expect(l.Get(nil, "v")).To(ConsistOf(all))
		for k := 4; k <= 40; k++ {
			l.Delete(k)
		}
		Expect(l.Get(nil, "v")).To(ConsistOf(3))
	})

	It("Should keep every staged holder of a value as it grows and shrinks", func() {
		txn := &scope{}
		l.Stage(txn, 1, "v")
		l.Stage(txn, 1, "v")
		Expect(l.Get(txn, "v")).To(ConsistOf(1))
		l.Stage(txn, 2, "v")
		Expect(l.Get(txn, "v")).To(ConsistOf(1, 2))
		l.StageDelete(txn, 1)
		Expect(l.Get(txn, "v")).To(ConsistOf(2))
		var all []int
		for k := 2; k <= 40; k++ {
			l.Stage(txn, k, "v")
			all = append(all, k)
		}
		Expect(l.Get(txn, "v")).To(ConsistOf(all))
		for k := 3; k <= 40; k++ {
			l.Stage(txn, k, "w")
		}
		Expect(l.Get(txn, "v")).To(ConsistOf(2))
		Expect(l.Get(txn, "w")).To(HaveLen(38))
		txn.end(true)
		Expect(l.Get(nil, "v")).To(ConsistOf(2))
		Expect(l.Get(nil, "w")).To(HaveLen(38))
	})
})
