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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// brNumPlot is a plot with an integer key, so its key has more than one text form.
type brNumPlot struct {
	Key uint32
	Ys  []uint32
}

func (p brNumPlot) GorpKey() uint32 { return p.Key }

var _ = Describe("Retarget breaker", func() {
	var (
		db        *gorp.DB
		channels  *gorp.Table[uint32, rtChannel]
		owners    *gorp.Table[string, rtOwner]
		numPlots  *gorp.Table[uint32, brNumPlot]
		numWrites int
	)
	BeforeEach(func(ctx SpecContext) {
		numWrites = 0
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		StartRun(db.Run)
		channels = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, rtChannel]{
			DB:     db,
			Name:   "channel",
			Issued: func(k uint32) bool { return k <= 10 },
		}))
		owners = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, rtOwner]{
			DB:   db,
			Name: "owner",
			References: []gorp.Reference[string, rtOwner]{
				gorp.NewKeyReference(gorp.KeyReferenceConfig[string, rtOwner, uint32]{
					Name:     "channel",
					Table:    "channel",
					OnDelete: gorp.Restrict,
					Kind:     gorp.Use,
					Unique:   true,
					Targets:  func(o *rtOwner) []uint32 { return []uint32{o.Channel} },
					Retarget: func(o *rtOwner, to map[uint32]uint32) {
						o.Channel = rtMove(to, o.Channel)
					},
				}),
			},
		}))
		numPlots = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, brNumPlot]{
			DB:   db,
			Name: "num_plot",
			References: []gorp.Reference[uint32, brNumPlot]{
				gorp.NewKeyReference(gorp.KeyReferenceConfig[uint32, brNumPlot, uint32]{
					Name:     "ys",
					Table:    "channel",
					OnDelete: gorp.Weak,
					Kind:     gorp.Use,
					Targets:  func(p *brNumPlot) []uint32 { return p.Ys },
					Retarget: func(p *brNumPlot, to map[uint32]uint32) {
						for i, k := range p.Ys {
							p.Ys[i] = rtMove(to, k)
						}
					},
				}),
			},
			OnRewrite: func(context.Context, gorp.Tx, brNumPlot) error {
				numWrites++
				return nil
			},
		}))
		for k := uint32(1); k <= 4; k++ {
			Expect(channels.NewCreate().Entry(&rtChannel{Key: k}).Exec(ctx, db)).
				To(Succeed())
		}
	})

	dependents := func(ctx SpecContext, keys ...uint32) []gorp.Dependent {
		GinkgoHelper()
		var out []gorp.Dependent
		for _, deps := range MustSucceed(channels.DependentsEach(ctx, db, keys)) {
			out = append(out, deps...)
		}
		return out
	}
	retarget := func(
		ctx SpecContext,
		to map[uint32]uint32,
		holders []gorp.Dependent,
	) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return channels.Retarget(ctx, tx, to, holders)
		})
	}
	retrieveOwner := func(ctx SpecContext, k string) rtOwner {
		GinkgoHelper()
		var o rtOwner
		Expect(owners.NewRetrieve().
			Where(gorp.MatchKeys[string, rtOwner](k)).
			Entry(&o).
			Exec(ctx, db)).To(Succeed())
		return o
	}
	createOwners := func(ctx SpecContext) {
		GinkgoHelper()
		for _, o := range []rtOwner{{Key: "a", Channel: 1}, {Key: "b", Channel: 2}} {
			Expect(owners.NewCreate().Entry(&o).Exec(ctx, db)).To(Succeed())
		}
	}

	DescribeTable(
		"Should refuse to swap or chain the targets of two holders of a Unique reference",
		func(ctx SpecContext, to map[uint32]uint32) {
			createOwners(ctx)
			Expect(retarget(ctx, to, dependents(ctx, 1, 2))).
				To(MatchError(ContainSubstring("owner b already references channel 2")))
			Expect(retrieveOwner(ctx, "a").Channel).To(Equal(uint32(1)))
			Expect(retrieveOwner(ctx, "b").Channel).To(Equal(uint32(2)))
		},
		Entry("a swap", map[uint32]uint32{1: 2, 2: 1}),
		Entry("a chain", map[uint32]uint32{1: 2, 2: 3}),
	)

	It("Should apply the map once to a holder named in two text forms of its key", func(
		ctx SpecContext,
	) {
		Expect(numPlots.NewCreate().
			Entry(&brNumPlot{Key: 5, Ys: []uint32{1, 2}}).
			Exec(ctx, db)).To(Succeed())
		holders := []gorp.Dependent{
			{Holder: gorp.Target{Table: "num_plot", Key: "5"}, Reference: "ys"},
			{Holder: gorp.Target{Table: "num_plot", Key: "05"}, Reference: "ys"},
		}
		Expect(retarget(ctx, map[uint32]uint32{1: 2, 2: 1}, holders)).To(Succeed())
		var p brNumPlot
		Expect(numPlots.NewRetrieve().
			Where(gorp.MatchKeys[uint32, brNumPlot](5)).
			Entry(&p).
			Exec(ctx, db)).To(Succeed())
		Expect(p.Ys).To(Equal([]uint32{2, 1}))
		Expect(numWrites).To(Equal(1))
	})
})
