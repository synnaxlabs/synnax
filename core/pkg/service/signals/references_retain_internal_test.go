// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package signals

// The buffers of a reference feed have no exported reader, so this spec is internal.

import (
	"context"
	"runtime"
	"strconv"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// heapAfterGC returns the bytes of live heap objects after a collection.
func heapAfterGC() uint64 {
	runtime.GC()
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.HeapAlloc
}

// commitOf returns a commit in which n schematics each start to read one channel.
func commitOf(n int, prefix string) gorp.Commit {
	changes := make(gorp.Changes, n)
	link := gorp.Link{
		Target: gorp.Target{Table: string(resource.TypeChannel), Key: "1"},
		Kind:   gorp.Use,
		Tag:    1,
	}
	for i := range changes {
		changes[i] = gorp.Change{
			Record: gorp.Target{
				Table: string(resource.TypeSchematic),
				Key:   prefix + strconv.Itoa(i) + "-0123456789abcdef0123456789",
			},
			Kind:  gorp.ChangeCreate,
			Added: []gorp.Link{link},
			Links: []gorp.Link{link},
		}
	}
	return gorp.Commit{Seq: 1, Changes: changes}
}

var _ = ginkgo.Describe("Reference feed buffers", func() {
	ginkgo.It("Should free the buffers of a bulk commit after small commits", func() {
		f := newReferenceFeed(resourceOf)
		disconnect := ReferenceChanges{feed: f}.OnChange(
			func(context.Context, []ReferenceChange) {},
		)
		defer disconnect()
		gomega.Expect(f.append(commitOf(1, "warm"), framer.Frame{}).Empty()).
			To(gomega.BeFalse())
		before := heapAfterGC()
		gomega.Expect(f.append(commitOf(500_000, "bulk"), framer.Frame{}).Empty()).
			To(gomega.BeFalse())
		for range 10 {
			gomega.Expect(f.append(commitOf(1, "small"), framer.Frame{}).Empty()).
				To(gomega.BeFalse())
		}
		gomega.Expect(cap(f.targets)).To(gomega.BeNumerically("<=", referenceRetained))
		gomega.Expect(cap(f.links)).To(gomega.BeNumerically("<=", referenceRetained))
		held := int64(heapAfterGC()) - int64(before)
		gomega.Expect(held).To(gomega.BeNumerically("<", 8<<20))
		runtime.KeepAlive(f)
	})
})
