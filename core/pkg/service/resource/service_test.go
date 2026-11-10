// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package resource_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Service", func() {
	var (
		db       *gorp.DB
		channels fixture[channelEntry]
		racks    fixture[rackEntry]
		svc      *resource.Service
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		channels = openFixture[channelEntry](ctx, db, resource.TypeChannel)
		racks = openFixture[rackEntry](ctx, db, resource.TypeRack)
		svc = MustSucceed(resource.NewService(resource.ServiceConfig{
			Providers: map[resource.Type]resource.Provider{
				resource.TypeChannel: channels.provider,
				resource.TypeRack:    racks.provider,
			},
		}))
		channels.create(
			ctx,
			db,
			channelEntry{Key: "1", Name: "pump pressure"},
			channelEntry{Key: "2", Name: "pump"},
			channelEntry{Key: "3", Name: "valve"},
		)
		racks.create(
			ctx,
			db,
			rackEntry{Key: "1", Name: "pump"},
			rackEntry{Key: "2", Name: "pump station rack"},
		)
	})

	Describe("NewService", func() {
		It("Should require providers", func() {
			Expect(resource.NewService(resource.ServiceConfig{})).Error().
				To(MatchError(ContainSubstring("providers: must be non-nil")))
		})

		It("Should refuse a nil Provider", func() {
			Expect(resource.NewService(resource.ServiceConfig{
				Providers: map[resource.Type]resource.Provider{
					resource.TypeChannel: nil,
				},
			})).Error().
				To(MatchError(ContainSubstring("providers.channel: must be non-nil")))
		})
	})

	Describe("Search", func() {
		It("Should merge the matches of every type, closest match first",
			func(ctx SpecContext) {
				Expect(MustSucceed(svc.Search(ctx, db, resource.SearchQuery{
					Term: "pump",
				}))).To(Equal([]resource.Resource{
					{ID: channels.id("2"), Name: "pump"},
					{ID: racks.id("1"), Name: "pump"},
					{ID: channels.id("1"), Name: "pump pressure"},
					{ID: racks.id("2"), Name: "pump station rack"},
				}))
			},
		)

		It("Should search only the given types", func(ctx SpecContext) {
			Expect(MustSucceed(svc.Search(ctx, db, resource.SearchQuery{
				Term:  "pump",
				Types: []resource.Type{resource.TypeRack},
			}))).To(Equal([]resource.Resource{
				{ID: racks.id("1"), Name: "pump"},
				{ID: racks.id("2"), Name: "pump station rack"},
			}))
		})

		It("Should keep the closest merged matches within the limit",
			func(ctx SpecContext) {
				Expect(MustSucceed(svc.Search(ctx, db, resource.SearchQuery{
					Term:  "pump",
					Limit: 2,
				}))).To(Equal([]resource.Resource{
					{ID: channels.id("2"), Name: "pump"},
					{ID: racks.id("1"), Name: "pump"},
				}))
			},
		)

		It("Should fill the page with the matches that Except keeps",
			func(ctx SpecContext) {
				Expect(MustSucceed(svc.Search(ctx, db, resource.SearchQuery{
					Term:   "pump",
					Limit:  2,
					Except: func(id resource.ID) bool { return id.Type == resource.TypeRack },
				}))).To(Equal([]resource.Resource{
					{ID: channels.id("2"), Name: "pump"},
					{ID: channels.id("1"), Name: "pump pressure"},
				}))
			},
		)

		Describe("Empty term", func() {
			It("Should list the resources of every type in order of type",
				func(ctx SpecContext) {
					Expect(MustSucceed(svc.Search(ctx, db, resource.SearchQuery{}))).
						To(Equal([]resource.Resource{
							{ID: channels.id("1"), Name: "pump pressure"},
							{ID: channels.id("2"), Name: "pump"},
							{ID: channels.id("3"), Name: "valve"},
							{ID: racks.id("1"), Name: "pump"},
							{ID: racks.id("2"), Name: "pump station rack"},
						}))
				},
			)

			It("Should list only the given types", func(ctx SpecContext) {
				Expect(MustSucceed(svc.Search(ctx, db, resource.SearchQuery{
					Types: []resource.Type{resource.TypeRack},
				}))).To(Equal([]resource.Resource{
					{ID: racks.id("1"), Name: "pump"},
					{ID: racks.id("2"), Name: "pump station rack"},
				}))
			})

			It("Should fill the limit across types", func(ctx SpecContext) {
				Expect(MustSucceed(svc.Search(ctx, db, resource.SearchQuery{
					Limit: 4,
				}))).To(Equal([]resource.Resource{
					{ID: channels.id("1"), Name: "pump pressure"},
					{ID: channels.id("2"), Name: "pump"},
					{ID: channels.id("3"), Name: "valve"},
					{ID: racks.id("1"), Name: "pump"},
				}))
			})

			It("Should fill the page with the resources that Except keeps",
				func(ctx SpecContext) {
					Expect(MustSucceed(svc.Search(ctx, db, resource.SearchQuery{
						Limit:  3,
						Except: func(id resource.ID) bool { return id == channels.id("2") },
					}))).To(Equal([]resource.Resource{
						{ID: channels.id("1"), Name: "pump pressure"},
						{ID: channels.id("3"), Name: "valve"},
						{ID: racks.id("1"), Name: "pump"},
					}))
				},
			)

			It("Should read the changes staged against the transaction",
				func(ctx SpecContext) {
					tx := db.OpenTx()
					defer func() { Expect(tx.Close()).To(Succeed()) }()
					racks.create(ctx, tx, rackEntry{Key: "3", Name: "staged"})
					Expect(MustSucceed(svc.Search(ctx, tx, resource.SearchQuery{
						Types: []resource.Type{resource.TypeRack},
					}))).To(ContainElement(resource.Resource{
						ID:   racks.id("3"),
						Name: "staged",
					}))
				},
			)
		})

		It("Should read the changes staged against the transaction",
			func(ctx SpecContext) {
				tx := db.OpenTx()
				defer func() { Expect(tx.Close()).To(Succeed()) }()
				channels.create(ctx, tx, channelEntry{Key: "4", Name: "staged"})
				Expect(MustSucceed(svc.Search(ctx, tx, resource.SearchQuery{
					Term: "staged",
				}))).To(Equal([]resource.Resource{
					{ID: channels.id("4"), Name: "staged"},
				}))
			},
		)

		It("Should refuse a type with no Provider", func(ctx SpecContext) {
			Expect(svc.Search(ctx, db, resource.SearchQuery{
				Term:  "pump",
				Types: []resource.Type{resource.TypeChannel, "turbine"},
			})).Error().To(MatchError(`types: unknown resource type "turbine"`))
		})

		It("Should refuse a negative limit", func(ctx SpecContext) {
			Expect(svc.Search(ctx, db, resource.SearchQuery{
				Term:  "pump",
				Limit: -1,
			})).Error().To(MatchError("limit: must be greater than or equal to 0"))
		})

		It("Should refuse a limit past its bound", func(ctx SpecContext) {
			Expect(svc.Search(ctx, db, resource.SearchQuery{
				Limit: 1001,
			})).Error().To(MatchError("limit: must be less than or equal to 1000"))
		})

		It("Should panic when a Provider returns an ID of a type with no Provider",
			func(ctx SpecContext) {
				svc := MustSucceed(resource.NewService(resource.ServiceConfig{
					Providers: map[resource.Type]resource.Provider{
						resource.TypeChannel: channels.providerOf(resource.TypeUser),
					},
				}))
				Expect(func() {
					MustSucceed(
						svc.Search(ctx, db, resource.SearchQuery{Term: "pump"}),
					)
				}).To(PanicWith(ContainSubstring("no provider for type user")))
			},
		)
	})

	Describe("Retrieve", func() {
		It(
			"Should return the resources in the order of the IDs",
			func(ctx SpecContext) {
				Expect(MustSucceed(svc.Retrieve(
					ctx,
					db,
					racks.id("2"),
					channels.id("3"),
					racks.id("1"),
				))).To(Equal([]resource.Resource{
					{ID: racks.id("2"), Name: "pump station rack"},
					{ID: channels.id("3"), Name: "valve"},
					{ID: racks.id("1"), Name: "pump"},
				}))
			},
		)

		It("Should retrieve and search resources through a snapshot", func(
			ctx SpecContext,
		) {
			var retrieved, found []resource.Resource
			Expect(db.WithSnapshot(ctx, func(snap gorp.Snapshot) (err error) {
				if retrieved, err = svc.Retrieve(ctx, snap, racks.id("1")); err != nil {
					return err
				}
				found, err = svc.Search(ctx, snap, resource.SearchQuery{
					Term:  "pump",
					Types: []resource.Type{resource.TypeRack},
				})
				return err
			})).To(Succeed())
			Expect(retrieved).To(Equal([]resource.Resource{
				{ID: racks.id("1"), Name: "pump"},
			}))
			Expect(found).To(Equal([]resource.Resource{
				{ID: racks.id("1"), Name: "pump"},
				{ID: racks.id("2"), Name: "pump station rack"},
			}))
		})

		It("Should return each resource once", func(ctx SpecContext) {
			Expect(MustSucceed(svc.Retrieve(
				ctx,
				db,
				channels.id("3"),
				channels.id("3"),
			))).To(Equal([]resource.Resource{{ID: channels.id("3"), Name: "valve"}}))
		})

		It("Should leave out an ID that names no resource", func(ctx SpecContext) {
			Expect(MustSucceed(svc.Retrieve(
				ctx,
				db,
				channels.id("99"),
				racks.id("1"),
			))).To(Equal([]resource.Resource{{ID: racks.id("1"), Name: "pump"}}))
		})

		It("Should refuse an ID of a type with no Provider", func(ctx SpecContext) {
			Expect(svc.Retrieve(
				ctx,
				db,
				channels.id("1"),
				resource.ID{Type: resource.TypeUser, Key: "1"},
			)).Error().To(MatchError(`ids: unknown resource type "user"`))
		})
	})
})
