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
	"github.com/synnaxlabs/x/telem"
)

func pointerfy(tr telem.TimeRange) skl.Pointer {
	return skl.Pointer{
		TimeRange: tr,
	}
}

var _ = Describe("Skip List", func() {
	It("Should work", func() {
		l := skl.NewSkipList()
		Expect(l.Insert(pointerfy((10 * telem.SecondTS).Range(20 * telem.SecondTS)))).To(Succeed())
		Expect(l.Insert(pointerfy((30 * telem.SecondTS).Range(40 * telem.SecondTS)))).To(Succeed())
		Expect(l.Insert(pointerfy((20 * telem.SecondTS).Range(25 * telem.SecondTS)))).To(Succeed())
		Expect(l.Insert(pointerfy((35 * telem.SecondTS).Range(45 * telem.SecondTS)))).ToNot(Succeed())

		node, ok := l.SearchGE(15 * telem.SecondTS)
		Expect(ok).To(BeTrue())
		Expect(node.Ptr.TimeRange).To(Equal((10 * telem.SecondTS).Range(20 * telem.SecondTS)))

		node, ok = l.SearchGE(27 * telem.SecondTS)
		Expect(ok).To(BeFalse())
		Expect(node.Ptr.TimeRange).To(Equal((30 * telem.SecondTS).Range(40 * telem.SecondTS)))
	})

	It("Should be concurrent-safe", func() {
		//l := skl.NewSkipList()
	})
})
