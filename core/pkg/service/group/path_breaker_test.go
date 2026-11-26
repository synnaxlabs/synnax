// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package group_test

import (
	"context"
	"fmt"
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Paths breaker", Ordered, func() {
	var (
		db  *gorp.DB
		svc *group.Service
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		db = DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		svc = MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
	})
	// chain creates depth nested channel folders through the writer, one per level, and
	// returns the key of the deepest.
	chain := func(ctx context.Context, depth int) group.Key {
		GinkgoHelper()
		var parent *resource.ID
		var last group.Key
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			w := svc.NewWriter(tx)
			for i := range depth {
				g := group.Group{
					Name:   fmt.Sprintf("d%d_%d", depth, i),
					Scope:  resource.TypeChannel,
					Parent: parent,
				}
				if err := w.Create(ctx, &g); err != nil {
					return err
				}
				parent, last = new(g.ResourceID()), g.Key
			}
			return nil
		})).To(Succeed())
		return last
	}
	// allocated returns the fewest bytes that one group.Paths call for key allocates
	// over three runs.
	allocated := func(ctx context.Context, key group.Key, depth int) uint64 {
		GinkgoHelper()
		least := ^uint64(0)
		for range 3 {
			var before, after runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&before)
			paths := MustSucceed(group.Paths(ctx, db, key))
			runtime.ReadMemStats(&after)
			Expect(paths[key]).To(HaveLen(depth))
			least = min(least, after.TotalAlloc-before.TotalAlloc)
		}
		return least
	}

	It("Should allocate in proportion to the depth of a path", func(ctx SpecContext) {
		const depth = 500
		shallow, deep := chain(ctx, depth), chain(ctx, 2*depth)
		a, b := allocated(ctx, shallow, depth), allocated(ctx, deep, 2*depth)
		GinkgoWriter.Printf(
			"depth %d: %d bytes, depth %d: %d bytes, ratio %.2f\n",
			depth, a, 2*depth, b, float64(b)/float64(a),
		)
		// A path of twice the depth holds twice the groups and takes twice the reads.
		Expect(float64(b) / float64(a)).To(BeNumerically("<", 3))
	})
})
