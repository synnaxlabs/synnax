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
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp/internal/index"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Reused changes", func() {
	var l *index.Lookup[*scope, int, string]
	BeforeEach(func() {
		l = index.NewLookup[*scope, int, string]()
		_, finish := l.Populate()
		finish(nil)
	})

	It("Should show a scope none of the changes of the scope before it", func() {
		first := &scope{}
		l.Stage(first, 1, "a")
		l.StageDelete(first, 2)
		first.end(false)
		next := &scope{}
		Expect(l.Get(next, "a")).To(BeEmpty())
		l.Stage(next, 3, "b")
		Expect(l.Get(next, "b")).To(Equal([]int{3}))
		Expect(l.Get(next, "a")).To(BeEmpty())
		next.end(true)
		Expect(l.Get(nil, "a", "b")).To(Equal([]int{3}))
	})

	It("Should apply the changes of a large scope and of the one after it", func() {
		large := &scope{}
		for i := range 200 {
			l.Stage(large, i, "v"+strconv.Itoa(i))
		}
		large.end(true)
		next := &scope{}
		l.Stage(next, 500, "v0")
		next.end(true)
		Expect(l.Get(nil, "v0")).To(ConsistOf(0, 500))
		Expect(l.Get(nil, "v199")).To(Equal([]int{199}))
	})
})

var _ = Describe("Lookup holders of one value", func() {
	var l *index.Lookup[*scope, int, string]
	BeforeEach(func() {
		l = index.NewLookup[*scope, int, string]()
		insert, finish := l.Populate()
		for _, k := range []int{1, 2, 3} {
			insert(k, "a")
		}
		finish(nil)
	})

	It("Should keep the other holders when the first one leaves", func() {
		l.Delete(1)
		Expect(l.Get(nil, "a")).To(ConsistOf(2, 3))
		l.Delete(3)
		Expect(l.Get(nil, "a")).To(Equal([]int{2}))
		_, found := MustSucceed2(l.Holder(nil, "a", nil))
		Expect(found).To(BeTrue())
		l.Delete(2)
		Expect(l.Get(nil, "a")).To(BeEmpty())
		_, found = MustSucceed2(l.Holder(nil, "a", nil))
		Expect(found).To(BeFalse())
	})

	It("Should move a holder to another value", func() {
		l.Set(1, "b")
		Expect(l.Get(nil, "a")).To(ConsistOf(2, 3))
		Expect(l.Get(nil, "b")).To(Equal([]int{1}))
	})
})
