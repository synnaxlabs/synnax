// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"slices"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
)

var _ = ginkgo.Describe("builtInSet, checker cases", func() {
	free := func(l LocalKey) Key { return distchannel.NewKey(node.KeyFree, l) }
	leased := func(n node.Key, l LocalKey) Key { return distchannel.NewKey(n, l) }
	const maxLocal = LocalKey(1<<20 - 1)

	ginkgo.It(
		"Should hold keys at word edges, late keys, and leased keys only",
		func() {
			in := []Key{
				free(1), free(63), free(64), free(127), free(128), free(maxLocal),
				leased(1, 5), leased(2, 1), leased(1, maxLocal),
			}
			out := []Key{
				free(2), free(62), free(65), free(126), free(129), free(maxLocal - 1),
				leased(1, 4), leased(1, 6), leased(3, 5), leased(2, 2), leased(1, 1),
				leased(node.KeyFree-1, 1), free(0),
			}
			b := &builtInSet{}
			for _, k := range in {
				b = b.with(k)
				b = b.with(k)
			}
			for _, k := range in {
				gomega.Expect(b.contains(k)).To(gomega.BeTrue(), "key %d", k)
			}
			for _, k := range out {
				gomega.Expect(b.contains(k)).To(gomega.BeFalse(), "key %d", k)
			}
			keys := b.keys()
			gomega.Expect(keys).To(gomega.ConsistOf(in))
			gomega.Expect(slices.IsSorted(keys)).To(gomega.BeTrue())
		},
	)

	ginkgo.It("Should leave an earlier set unchanged", func() {
		first := (&builtInSet{}).with(free(3)).with(leased(1, 3))
		second := first.with(free(4)).with(leased(1, 2)).with(free(200))
		gomega.Expect(first.contains(free(4))).To(gomega.BeFalse())
		gomega.Expect(first.contains(leased(1, 2))).To(gomega.BeFalse())
		gomega.Expect(first.contains(free(200))).To(gomega.BeFalse())
		gomega.Expect(first.keys()).To(gomega.ConsistOf(free(3), leased(1, 3)))
		gomega.Expect(second.keys()).To(gomega.HaveLen(5))
	})

	ginkgo.It("Should hold nothing when empty", func() {
		b := &builtInSet{}
		gomega.Expect(b.contains(free(1))).To(gomega.BeFalse())
		gomega.Expect(b.contains(leased(1, 1))).To(gomega.BeFalse())
		gomega.Expect(b.keys()).To(gomega.BeEmpty())
	})
})
