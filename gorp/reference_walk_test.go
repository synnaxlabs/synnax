// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp_test

import (
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Target walks", func() {
	var (
		db      *gorp.DB
		holders *gorp.Table[int32, refBenchHolder]
		ref     *gorp.KeyReference[int32, refBenchHolder, int32]
		walks   atomic.Int32
	)
	write := func(ctx SpecContext, targets ...int32) {
		GinkgoHelper()
		h := refBenchHolder{Key: 1, Targets: targets}
		Expect(holders.NewCreate().Entry(&h).Exec(ctx, db)).To(Succeed())
	}
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		StartRun(db.Run)
		targets := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[int32, benchEntry]{
			DB:   db,
			Name: "ref_target",
		}))
		Expect(targets.NewCreate().
			Entries(&[]benchEntry{{ID: 0}, {ID: 1}, {ID: 2}}).
			Exec(ctx, db)).To(Succeed())
		ref = gorp.NewKeyReference(
			gorp.KeyReferenceConfig[int32, refBenchHolder, int32]{
				Name:     "targets",
				Table:    "ref_target",
				OnDelete: gorp.Restrict,
				Kind:     gorp.Use,
				Targets: func(h *refBenchHolder) []int32 {
					walks.Add(1)
					return h.Targets
				},
			},
		)
		holders = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[int32, refBenchHolder]{
			DB:         db,
			Name:       "ref_holder",
			References: []gorp.Reference[int32, refBenchHolder]{ref},
		}))
		write(ctx, 0, 1)
		Expect(ref.Get(ctx, db, 0)).To(ConsistOf(int32(1)))
		walks.Store(0)
	})

	It("Should walk the targets of a write that keeps them once", func(
		ctx SpecContext,
	) {
		write(ctx, 0, 1)
		Expect(walks.Load()).To(BeEquivalentTo(1))
	})

	It("Should walk the targets of a write that changes them twice", func(
		ctx SpecContext,
	) {
		write(ctx, 0, 2)
		Expect(walks.Load()).To(BeEquivalentTo(2))
		Expect(ref.Get(ctx, db, 2)).To(ConsistOf(int32(1)))
		Expect(ref.Get(ctx, db, 1)).To(BeEmpty())
	})
})
