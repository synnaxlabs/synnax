// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package skl_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/skl"
)

var _ = Describe("Skip List", func() {
	It("Should work", func() {
		l := skl.NewSkipList()
		l.Insert(3)
		l.Insert(5)
		l.Insert(10)
		l.Insert(2)
		l.Insert(7)
		v, ok := l.SearchLE(7)
		Expect(v).To(Equal(7))
		Expect(ok).To(BeTrue())

		v, ok = l.SearchLE(4)
		Expect(v).To(Equal(3))
		Expect(ok).To(BeTrue())

		v, ok = l.SearchLE(1)
		Expect(ok).To(BeFalse())

		l.Delete(4, 10)
		Expect(l.Len()).To(Equal(3))

		l.Delete(3, 100)
		Expect(l.Len()).To(Equal(1))

		v, ok = l.SearchLE(20)
		Expect(v).To(Equal(2))
		Expect(ok).To(BeTrue())
	})

	It("Should be concurrent-safe", func() {
		l := skl.NewSkipList()
		l.Insert(3)
		l.Insert(5)
		l.Insert(10)
		l.Insert(2)
		l.Insert(7)
		v, ok := l.SearchLE(7)
		Expect(v).To(Equal(7))
		Expect(ok).To(BeTrue())

		v, ok = l.SearchLE(4)
		Expect(v).To(Equal(3))
		Expect(ok).To(BeTrue())

		v, ok = l.SearchLE(1)
		Expect(ok).To(BeFalse())

		l.Delete(4, 10)
		Expect(l.Len()).To(Equal(3))

		l.Delete(3, 100)
		Expect(l.Len()).To(Equal(1))

		v, ok = l.SearchLE(20)
		Expect(v).To(Equal(2))
		Expect(ok).To(BeTrue())
	})
})
