// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
)

type testAction struct {
	Type string `json:"type" msgpack:"type"`
}

func scoped(dispatchKey string, seq uint64) actions.Scoped[string, testAction] {
	return actions.Scoped[string, testAction]{DispatchKey: dispatchKey, Seq: seq}
}

var _ = Describe("Recorder", func() {
	It(
		"Should return recorded actions in the order they were received",
		func(ctx SpecContext) {
			rec := &Recorder[string, testAction]{}
			rec.Record(ctx, scoped("a", 0))
			rec.Record(ctx, scoped("b", 1))
			rec.Record(ctx, scoped("c", 2))
			seen := rec.Snapshot()
			Expect(seen).To(HaveLen(3))
			Expect(seen[0].DispatchKey).To(Equal("a"))
			Expect(seen[1].DispatchKey).To(Equal("b"))
			Expect(seen[2].DispatchKey).To(Equal("c"))
		},
	)

	It("Should return an empty snapshot before anything is recorded", func() {
		rec := &Recorder[string, testAction]{}
		Expect(rec.Snapshot()).To(BeEmpty())
	})

	It(
		"Should return a copy that does not alias the internal slice",
		func(ctx SpecContext) {
			rec := &Recorder[string, testAction]{}
			rec.Record(ctx, scoped("a", 0))
			first := rec.Snapshot()
			first[0].DispatchKey = "mutated"
			rec.Record(ctx, scoped("b", 1))
			second := rec.Snapshot()
			Expect(second).To(HaveLen(2))
			Expect(second[0].DispatchKey).To(Equal("a"))
			Expect(second[1].DispatchKey).To(Equal("b"))
		},
	)

	It("Should record concurrently without data races", func(ctx SpecContext) {
		rec := &Recorder[string, testAction]{}
		const goroutines, perGoroutine = 8, 50
		var wg sync.WaitGroup
		wg.Add(goroutines)
		for range goroutines {
			go func() {
				defer wg.Done()
				for range perGoroutine {
					rec.Record(ctx, scoped("x", 0))
				}
			}()
		}
		wg.Wait()
		Expect(rec.Snapshot()).To(HaveLen(goroutines * perGoroutine))
	})
})
