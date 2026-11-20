// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package signals_test

import (
	"runtime"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
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

var _ = Describe("Reference feed memory", func() {
	It("Should not hold the memory of a bulk commit once small commits follow", func() {
		appendCommit := signals.NewReferenceFeedAppend(func(t gorp.Target) resource.ID {
			return resource.ID{Type: resource.Type(t.Table), Key: t.Key}
		})
		Expect(appendCommit(commitOf(1, "warm")).Empty()).To(BeFalse())
		before := heapAfterGC()
		Expect(appendCommit(commitOf(500_000, "bulk")).Empty()).To(BeFalse())
		for range 10 {
			Expect(appendCommit(commitOf(1, "small")).Empty()).To(BeFalse())
		}
		after := heapAfterGC()
		held := int64(after) - int64(before)
		GinkgoWriter.Printf("held after the bulk commit: %d bytes\n", held)
		runtime.KeepAlive(appendCommit)
		appendCommit = nil
		GinkgoWriter.Printf(
			"held once the feed is dropped: %d bytes\n",
			int64(heapAfterGC())-int64(before),
		)
		Expect(held).To(BeNumerically("<", 8<<20))
	})
})
