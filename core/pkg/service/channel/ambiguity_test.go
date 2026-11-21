// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// Names stay unique across the Core on write, so only names stored before that rule,
// in different folders, are held by several channels.
var _ = Describe("Ambiguous names", func() {
	var (
		db                   *gorp.DB
		svc                  *channel.Service
		stand, rig           group.Group
		inStand, inRig, solo channel.Channel
	)
	const ambiguous = "2 channels are named 'pt': rig.pt, stand.pt. Name one by its path"
	BeforeEach(func(ctx SpecContext) {
		fresh := testutil.NewNode(ctx)
		db = fresh.DB
		cfg, groupSvc := serviceConfig(ctx, fresh, fresh.DB)
		stand = group.Group{Name: "stand", Scope: resource.TypeChannel}
		rig = group.Group{Name: "rig", Scope: resource.TypeChannel}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			for _, g := range []*group.Group{&stand, &rig} {
				if err := groupSvc.NewWriter(tx).Create(ctx, g); err != nil {
					return err
				}
			}
			return nil
		})).To(Succeed())
		stored := func(name string, key channel.LocalKey, g *group.Key) channel.Channel {
			return channel.Channel{
				Name:        name,
				Leaseholder: node.KeyFree,
				LocalKey:    key,
				DataType:    telem.Float32T,
				Virtual:     true,
				Group:       g,
			}
		}
		inStand = stored("pt", 7, &stand.Key)
		inRig = stored("pt", 8, &rig.Key)
		solo = stored("solo", 9, nil)
		createStored(ctx, db, inStand, inRig, solo)
		svc = MustOpen(channel.OpenService(ctx, cfg))
	})
	exists := func(ctx SpecContext, k channel.Key) bool {
		GinkgoHelper()
		return MustSucceed(svc.NewRetrieve().
			Where(channel.MatchKeys(k)).
			Exists(ctx, db))
	}

	Describe("DeleteManyByNames", func() {
		It("Should refuse a name that several channels hold and delete nothing", func(
			ctx SpecContext,
		) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).DeleteManyByNames(
					ctx,
					[]string{"solo", "pt"},
					false,
				)
			})).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(ambiguous)),
			))
			Expect(exists(ctx, solo.Key())).To(BeTrue())
			Expect(exists(ctx, inStand.Key())).To(BeTrue())
		})

		It("Should delete a channel whose name one channel holds", func(
			ctx SpecContext,
		) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).DeleteManyByNames(ctx, []string{"solo"}, false)
			})).To(Succeed())
			Expect(exists(ctx, solo.Key())).To(BeFalse())
		})
	})

	Describe("RetrieveIfNameExists", func() {
		create := func(ctx SpecContext, ch *channel.Channel) error {
			return db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, ch, channel.RetrieveIfNameExists())
			})
		}

		It("Should match the channel of the name in the folder of the request", func(
			ctx SpecContext,
		) {
			ch := channel.Channel{
				Name:     "pt",
				DataType: telem.Float32T,
				Virtual:  true,
				Group:    &rig.Key,
			}
			Expect(create(ctx, &ch)).To(Succeed())
			Expect(ch.Key()).To(Equal(inRig.Key()))
		})

		It(
			"Should match a name anywhere when the folder of the request holds none",
			func(
				ctx SpecContext,
			) {
				ch := channel.Channel{
					Name:     "solo",
					DataType: telem.Float32T,
					Virtual:  true,
					Group:    &stand.Key,
				}
				Expect(create(ctx, &ch)).To(Succeed())
				Expect(ch.Key()).To(Equal(solo.Key()))
			},
		)

		It("Should match a name anywhere when the request names no folder", func(
			ctx SpecContext,
		) {
			ch := channel.Channel{Name: "solo", DataType: telem.Float32T, Virtual: true}
			Expect(create(ctx, &ch)).To(Succeed())
			Expect(ch.Key()).To(Equal(solo.Key()))
		})

		It("Should refuse a name that several channels hold when the request names no "+
			"folder", func(ctx SpecContext) {
			ch := channel.Channel{Name: "pt", DataType: telem.Float32T, Virtual: true}
			Expect(create(ctx, &ch)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(ambiguous)),
			))
		})
	})

	Describe("The Arc resolver", func() {
		It("Should refuse a name that several channels hold, naming each path", func(
			ctx SpecContext,
		) {
			Expect(svc.NewArcSymbolResolver(db).Resolve(ctx, "pt")).Error().
				To(MatchError(ContainSubstring(ambiguous)))
		})

		It("Should resolve a path to its channel, with the path as the name", func(
			ctx SpecContext,
		) {
			sym := MustSucceed(svc.NewArcSymbolResolver(db).Resolve(ctx, "stand.pt"))
			Expect(sym.ID).To(Equal(int(inStand.Key())))
			Expect(sym.Name).To(Equal("stand.pt"))
		})

		It("Should resolve a name that one channel holds, and a key", func(
			ctx SpecContext,
		) {
			r := svc.NewArcSymbolResolver(db)
			Expect(MustSucceed(r.Resolve(ctx, "solo")).ID).To(Equal(int(solo.Key())))
			key := solo.Key().String()
			Expect(MustSucceed(r.Resolve(ctx, key)).ID).To(Equal(int(solo.Key())))
		})

		It(
			"Should find nothing at a path that no channel is at",
			func(ctx SpecContext) {
				Expect(
					svc.NewArcSymbolResolver(db).Resolve(ctx, "stand.missing"),
				).Error().
					To(MatchError(query.ErrNotFound))
			},
		)
	})
})
