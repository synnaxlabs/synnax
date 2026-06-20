// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package crdt_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/crdt"
)

var _ = Describe("CRDT", func() {
	Describe("Local editing", func() {
		It("Should materialize forward insertions", func() {
			t := crdt.New(1)
			t.Insert(0, "hello")
			Expect(t.String()).To(Equal("hello"))
			Expect(t.Len()).To(Equal(5))
		})

		It("Should materialize backward insertions", func() {
			t := crdt.New(1)
			t.Insert(0, "o")
			t.Insert(0, "l")
			t.Insert(0, "l")
			t.Insert(0, "e")
			t.Insert(0, "h")
			Expect(t.String()).To(Equal("hello"))
		})

		It("Should insert into the middle", func() {
			t := crdt.New(1)
			t.Insert(0, "ad")
			t.Insert(1, "bc")
			Expect(t.String()).To(Equal("abcd"))
		})

		It("Should append at the end", func() {
			t := crdt.New(1)
			t.Insert(0, "ab")
			t.Insert(t.Len(), "cd")
			Expect(t.String()).To(Equal("abcd"))
		})

		It("Should delete a range", func() {
			t := crdt.New(1)
			t.Insert(0, "hello world")
			t.Delete(5, 6)
			Expect(t.String()).To(Equal("hello"))
		})

		It("Should append after deleting the tail", func() {
			t := crdt.New(1)
			t.Insert(0, "ab")
			t.Delete(1, 1)
			t.Insert(t.Len(), "c")
			Expect(t.String()).To(Equal("ac"))
		})

		It("Should support multi-byte code points", func() {
			t := crdt.New(1)
			t.Insert(0, "héllo→世界")
			Expect(t.String()).To(Equal("héllo→世界"))
			Expect(t.Len()).To(Equal(8))
		})

		It("Should report the id at a live index", func() {
			t := crdt.New(1)
			ops := t.Insert(0, "ab")
			id, ok := t.IndexToID(1)
			Expect(ok).To(BeTrue())
			Expect(id).To(Equal(ops[1].ID))
			_, ok = t.IndexToID(2)
			Expect(ok).To(BeFalse())
		})
	})

	Describe("Replication", func() {
		It("Should converge two replicas editing the same gap", func() {
			a, b := crdt.New(1), crdt.New(2)
			opsA := a.Insert(0, "abc")
			opsB := b.Insert(0, "xyz")
			b.ApplyInsert(opsA...)
			a.ApplyInsert(opsB...)
			Expect(a.String()).To(Equal(b.String()))
		})

		It("Should not interleave concurrent same-position runs", func() {
			a, b := crdt.New(1), crdt.New(2)
			opsA := a.Insert(0, "abc")
			opsB := b.Insert(0, "xyz")
			b.ApplyInsert(opsA...)
			a.ApplyInsert(opsB...)
			Expect(a.String()).To(SatisfyAny(Equal("abcxyz"), Equal("xyzabc")))
		})

		It("Should not interleave concurrent backward typing", func() {
			a, b := crdt.New(1), crdt.New(2)
			var opsA, opsB []crdt.Insert
			for _, r := range []string{"c", "b", "a"} {
				opsA = append(opsA, a.Insert(0, r)...)
			}
			for _, r := range []string{"z", "y", "x"} {
				opsB = append(opsB, b.Insert(0, r)...)
			}
			b.ApplyInsert(opsA...)
			a.ApplyInsert(opsB...)
			Expect(a.String()).To(Equal(b.String()))
			Expect(a.String()).To(SatisfyAny(Equal("abcxyz"), Equal("xyzabc")))
		})

		It("Should not interleave concurrent inserts before a shared character", func() {
			a, b := crdt.New(1), crdt.New(2)
			seed := a.Insert(0, "o")
			b.ApplyInsert(seed...)
			opsA := a.Insert(0, "ab")
			opsB := b.Insert(0, "xy")
			b.ApplyInsert(opsA...)
			a.ApplyInsert(opsB...)
			Expect(a.String()).To(Equal(b.String()))
			Expect(a.String()).To(SatisfyAny(Equal("abxyo"), Equal("xyabo")))
		})

		It("Should integrate inserts delivered before their origin", func() {
			a, b := crdt.New(1), crdt.New(2)
			ops := a.Insert(0, "abc")
			b.ApplyInsert(ops[2], ops[0], ops[1])
			Expect(b.String()).To(Equal("abc"))
		})

		It("Should delete a character inserted concurrently", func() {
			a, b := crdt.New(1), crdt.New(2)
			opsA := a.Insert(0, "abc")
			b.ApplyInsert(opsA...)
			delOps := b.Delete(1, 1)
			a.ApplyDelete(delOps...)
			Expect(a.String()).To(Equal("ac"))
			Expect(b.String()).To(Equal("ac"))
		})

		It("Should apply a delete that arrives before the insert", func() {
			a, b := crdt.New(1), crdt.New(2)
			insertOps := a.Insert(0, "abc")
			delOps := a.Delete(1, 1)
			b.ApplyDelete(delOps...)
			b.ApplyInsert(insertOps...)
			Expect(b.String()).To(Equal("ac"))
		})

		It("Should ignore duplicate operations", func() {
			a, b := crdt.New(1), crdt.New(2)
			ops := a.Insert(0, "abc")
			b.ApplyInsert(ops...)
			b.ApplyInsert(ops...)
			Expect(b.String()).To(Equal("abc"))
		})
	})
})
