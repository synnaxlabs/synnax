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
	"context"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
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
	ambiguousPT := func() channel.AmbiguousNameError {
		return channel.AmbiguousNameError{
			Name:    "pt",
			Holders: channel.Keys{inStand.Key(), inRig.Key()},
		}
	}
	matchAmbiguousPT := func() OmegaMatcher {
		return SatisfyAll(
			MatchError(query.ErrQuery),
			WithTransform(func(err error) channel.Keys {
				ambiguous, _ := errors.AsType[channel.AmbiguousNameError](err)
				return ambiguous.Holders
			}, ConsistOf(inStand.Key(), inRig.Key())),
		)
	}
	// access returns an access read that lets the subject retrieve every record but
	// hidden.
	access := func(hidden ...resource.ID) func(
		context.Context,
	) (func(resource.ID) bool, error) {
		return func(context.Context) (func(resource.ID) bool, error) {
			return func(id resource.ID) bool { return !slices.Contains(hidden, id) }, nil
		}
	}

	Describe("AmbiguousNameError", func() {
		It("Should name the holders by key and no path", func() {
			Expect(ambiguousPT().Error()).To(Equal(
				"2 channels are named 'pt' (keys " + inStand.Key().String() + ", " +
					inRig.Key().String() + ")",
			))
		})
	})

	Describe("KeysByNames", func() {
		It("Should return the key of each name that one channel holds", func(
			ctx SpecContext,
		) {
			Expect(svc.KeysByNames(ctx, db, []string{"solo", "missing"})).
				To(Equal(channel.Keys{solo.Key()}))
		})

		It("Should refuse a name that several channels hold", func(ctx SpecContext) {
			Expect(svc.KeysByNames(ctx, db, []string{"solo", "pt"})).Error().
				To(matchAmbiguousPT())
		})

		It("Should match each name exactly, never as a pattern", func(ctx SpecContext) {
			Expect(svc.KeysByNames(ctx, db, []string{"solo", "s.*", "p."})).
				To(Equal(channel.Keys{solo.Key()}))
		})
	})

	Describe("DescribeAmbiguity", func() {
		It("Should quote the path of each holder that the subject can see", func(
			ctx SpecContext,
		) {
			Expect(svc.DescribeAmbiguity(ctx, db, ambiguousPT(), access())).
				To(SatisfyAll(
					MatchError("2 channels are named 'pt': 'rig.pt', 'stand.pt'"),
					matchAmbiguousPT(),
				))
		})

		It("Should count the holders in a folder that the subject cannot see", func(
			ctx SpecContext,
		) {
			Expect(svc.DescribeAmbiguity(
				ctx,
				db,
				ambiguousPT(),
				access(rig.ResourceID()),
			)).To(MatchError(
				"2 channels are named 'pt': 'stand.pt', and 1 more you cannot see",
			))
		})

		It("Should count a holder that the subject cannot retrieve", func(
			ctx SpecContext,
		) {
			Expect(svc.DescribeAmbiguity(
				ctx,
				db,
				ambiguousPT(),
				access(inStand.ResourceID(), inRig.ResourceID()),
			)).To(MatchError("2 channels are named 'pt': 2 you cannot see"))
		})

		It("Should return another error unchanged", func(ctx SpecContext) {
			err := errors.New("other")
			Expect(svc.DescribeAmbiguity(ctx, db, err, access())).To(BeIdenticalTo(err))
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

		It("Should not take the channel of the name in another folder", func(
			ctx SpecContext,
		) {
			ch := channel.Channel{
				Name:     "solo",
				DataType: telem.Float32T,
				Virtual:  true,
				Group:    &stand.Key,
			}
			Expect(create(ctx, &ch)).To(MatchError(
				ContainSubstring("channel with name 'solo' already exists"),
			))
		})

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
			Expect(create(ctx, &ch)).To(matchAmbiguousPT())
		})
	})

	Describe("A calculation", func() {
		It("Should report a name that several channels hold as a diagnostic", func(
			ctx SpecContext,
		) {
			c := channel.Channel{
				Name:       "calc",
				DataType:   telem.Float32T,
				Virtual:    true,
				Expression: "return pt",
			}
			err := db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &c)
			})
			diags, ok := errors.AsType[*diagnostics.Diagnostics](err)
			Expect(ok).To(BeTrue(), "expected diagnostics, got %v", err)
			Expect(diags.Error()).To(ContainSubstring("2 channels are named 'pt'"))
		})
	})

	Describe("The Arc resolver", func() {
		It("Should refuse a name that several channels hold", func(ctx SpecContext) {
			Expect(svc.NewArcSymbolResolver(db).Resolve(ctx, "pt")).Error().
				To(matchAmbiguousPT())
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
			Expect(MustSucceed(r.ResolveKey(ctx, uint32(solo.Key()))).ID).
				To(Equal(int(solo.Key())))
		})

		It("Should find nothing at a path that no channel is at", func(
			ctx SpecContext,
		) {
			Expect(svc.NewArcSymbolResolver(db).Resolve(ctx, "stand.missing")).Error().
				To(MatchError(query.ErrNotFound))
		})
	})

	Describe("The subject resolver", func() {
		It(
			"Should find nothing at a path through a folder the subject cannot see",
			func(
				ctx SpecContext,
			) {
				r := svc.NewSubjectResolver(db, access(rig.ResourceID()))
				Expect(
					r.Resolve(ctx, "rig.pt"),
				).Error().
					To(MatchError(query.ErrNotFound))
				Expect(MustSucceed(r.Resolve(ctx, "stand.pt")).ID).
					To(Equal(int(inStand.Key())))
			},
		)

		It("Should name only the paths that the subject can see", func(
			ctx SpecContext,
		) {
			r := svc.NewSubjectResolver(db, access(rig.ResourceID()))
			Expect(r.Resolve(ctx, "pt")).Error().To(SatisfyAll(
				MatchError(
					"2 channels are named 'pt': 'stand.pt', and 1 more you cannot see",
				),
				matchAmbiguousPT(),
			))
		})

		It("Should count a holder that the subject cannot retrieve", func(
			ctx SpecContext,
		) {
			r := svc.NewSubjectResolver(db, access(inRig.ResourceID()))
			Expect(r.Resolve(ctx, "pt")).Error().To(SatisfyAll(
				MatchError(
					"2 channels are named 'pt': 'stand.pt', and 1 more you cannot see",
				),
				matchAmbiguousPT(),
			))
		})

		It("Should resolve nothing for the one holder of a name that the subject "+
			"cannot retrieve", func(ctx SpecContext) {
			r := svc.NewSubjectResolver(db, access(solo.ResourceID()))
			Expect(r.Resolve(ctx, "solo")).Error().To(MatchError(query.ErrNotFound))
		})
	})
})
