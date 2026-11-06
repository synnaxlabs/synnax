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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

type channelEntry struct{ Key, Name string }

func (e channelEntry) GorpKey() string { return e.Key }

func (channelEntry) SetOptions() []any { return nil }

func (e channelEntry) name() string { return e.Name }

type rackEntry struct{ Key, Name string }

func (e rackEntry) GorpKey() string { return e.Key }

func (rackEntry) SetOptions() []any { return nil }

func (e rackEntry) name() string { return e.Name }

type entry interface {
	gorp.Entry[string]
	name() string
}

// provider is a resource.Provider over a Gorp table of E.
type provider[E entry] struct {
	typ   resource.Type
	table *gorp.Table[string, E]
	idx   *gorp.SearchIndex[string, E]
}

func openProvider[E entry](
	ctx context.Context,
	db *gorp.DB,
	typ resource.Type,
) provider[E] {
	GinkgoHelper()
	idx := gorp.NewSearchIndex("name", func(e *E) []string {
		return []string{(*e).name()}
	})
	table := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, E]{
		DB:      db,
		Indexes: []gorp.Index[string, E]{idx},
	}))
	DeferClose(table)
	return provider[E]{typ: typ, table: table, idx: idx}
}

func (p provider[E]) id(key string) resource.ID {
	return resource.ID{Type: p.typ, Key: key}
}

func (p provider[E]) create(ctx context.Context, tx gorp.Tx, entries ...E) {
	GinkgoHelper()
	Expect(p.table.NewCreate().Entries(&entries).Exec(ctx, tx)).To(Succeed())
}

func (p provider[E]) Search(
	ctx context.Context,
	tx gorp.Tx,
	q search.Query[resource.ID],
) (search.Result[resource.ID], error) {
	return resource.SearchIndex(ctx, tx, p.idx, p.id, q)
}

func (p provider[E]) RetrieveResources(
	ctx context.Context,
	tx gorp.Tx,
	ids []resource.ID,
) ([]resource.Resource, error) {
	var entries []E
	if err := p.table.NewRetrieve().
		Where(gorp.MatchKeys[string, E](lo.Map(
			ids,
			func(id resource.ID, _ int) string { return id.Key },
		)...)).
		Entries(&entries).
		Exec(ctx, tx); errors.Skip(err, query.ErrNotFound) != nil {
		return nil, err
	}
	return lo.Map(entries, func(e E, _ int) resource.Resource {
		return resource.Resource{ID: p.id(e.GorpKey()), Name: e.name()}
	}), nil
}

func (p provider[E]) ListResources(
	ctx context.Context,
	tx gorp.Tx,
	limit int,
) ([]resource.Resource, error) {
	var entries []E
	if err := p.table.NewRetrieve().
		Limit(limit).
		Entries(&entries).
		Exec(ctx, tx); err != nil {
		return nil, err
	}
	return lo.Map(entries, func(e E, _ int) resource.Resource {
		return resource.Resource{ID: p.id(e.GorpKey()), Name: e.name()}
	}), nil
}

var _ = Describe("SearchIndex", func() {
	var (
		db *gorp.DB
		p  provider[channelEntry]
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		p = openProvider[channelEntry](ctx, db, resource.TypeChannel)
		p.create(
			ctx,
			db,
			channelEntry{Key: "a", Name: "turbopump inlet"},
			channelEntry{Key: "b", Name: "turbopump"},
		)
	})

	DescribeTable("Should return the IDs of at most limit matches, closest match first",
		func(ctx SpecContext, limit int, expected []string) {
			Expect(MustSucceed(resource.SearchIndex(
				ctx,
				db,
				p.idx,
				p.id,
				search.Query[resource.ID]{Term: "turbopump", Limit: limit},
			)).Keys).To(Equal(lo.Map(expected, func(k string, _ int) resource.ID {
				return p.id(k)
			})))
		},
		Entry("every match", 10, []string{"b", "a"}),
		Entry("the first match", 1, []string{"b"}),
	)

	It("Should leave out the IDs that except names", func(ctx SpecContext) {
		Expect(MustSucceed(resource.SearchIndex(
			ctx,
			db,
			p.idx,
			p.id,
			search.Query[resource.ID]{
				Term:   "turbopump inlet",
				Except: func(id resource.ID) bool { return id == p.id("a") },
			},
		)).Keys).To(Equal([]resource.ID{p.id("b")}))
	})

	It("Should return the error of the index search", func(ctx SpecContext) {
		canceled, cancel := context.WithCancel(ctx)
		cancel()
		Expect(resource.SearchIndex(
			canceled,
			db,
			gorp.NewSearchIndex("unopened", func(e *channelEntry) []string {
				return []string{e.Name}
			}),
			p.id,
			search.Query[resource.ID]{Term: "turbopump", Limit: 10},
		)).Error().To(MatchError(context.Canceled))
	})
})

var _ = Describe("Service", func() {
	var (
		db       *gorp.DB
		channels provider[channelEntry]
		racks    provider[rackEntry]
		svc      *resource.Service
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		channels = openProvider[channelEntry](ctx, db, resource.TypeChannel)
		racks = openProvider[rackEntry](ctx, db, resource.TypeRack)
		svc = MustSucceed(resource.NewService(resource.ServiceConfig{
			DB: db,
			Providers: map[resource.Type]resource.Provider{
				resource.TypeChannel: channels,
				resource.TypeRack:    racks,
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
		It("Should require a database", func() {
			Expect(resource.NewService(resource.ServiceConfig{
				Providers: map[resource.Type]resource.Provider{},
			})).Error().To(MatchError(ContainSubstring("db: must be non-nil")))
		})

		It("Should refuse a nil Provider", func() {
			Expect(resource.NewService(resource.ServiceConfig{
				DB: db,
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
				Expect(MustSucceed(svc.Search(ctx, nil, resource.SearchQuery{
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
			Expect(MustSucceed(svc.Search(ctx, nil, resource.SearchQuery{
				Term:  "pump",
				Types: []resource.Type{resource.TypeRack},
			}))).To(Equal([]resource.Resource{
				{ID: racks.id("1"), Name: "pump"},
				{ID: racks.id("2"), Name: "pump station rack"},
			}))
		})

		It("Should page through the merged matches", func(ctx SpecContext) {
			Expect(MustSucceed(svc.Search(ctx, nil, resource.SearchQuery{
				Term:   "pump",
				Limit:  2,
				Offset: 1,
			}))).To(Equal([]resource.Resource{
				{ID: racks.id("1"), Name: "pump"},
				{ID: channels.id("1"), Name: "pump pressure"},
			}))
		})

		It("Should return nothing for an offset past the last match",
			func(ctx SpecContext) {
				Expect(MustSucceed(svc.Search(ctx, nil, resource.SearchQuery{
					Term:   "pump",
					Offset: 10,
				}))).To(BeEmpty())
			},
		)

		Describe("Empty term", func() {
			It("Should list the resources of every type in order of type",
				func(ctx SpecContext) {
					Expect(MustSucceed(svc.Search(ctx, nil, resource.SearchQuery{}))).
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
				Expect(MustSucceed(svc.Search(ctx, nil, resource.SearchQuery{
					Types: []resource.Type{resource.TypeRack},
				}))).To(Equal([]resource.Resource{
					{ID: racks.id("1"), Name: "pump"},
					{ID: racks.id("2"), Name: "pump station rack"},
				}))
			})

			It("Should page across types", func(ctx SpecContext) {
				Expect(MustSucceed(svc.Search(ctx, nil, resource.SearchQuery{
					Limit:  2,
					Offset: 2,
				}))).To(Equal([]resource.Resource{
					{ID: channels.id("3"), Name: "valve"},
					{ID: racks.id("1"), Name: "pump"},
				}))
			})

			It("Should return nothing for an offset past the last resource",
				func(ctx SpecContext) {
					Expect(MustSucceed(svc.Search(ctx, nil, resource.SearchQuery{
						Offset: 10,
					}))).To(BeEmpty())
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
			Expect(svc.Search(ctx, nil, resource.SearchQuery{
				Term:  "pump",
				Types: []resource.Type{resource.TypeChannel, "turbine"},
			})).Error().To(MatchError(`types: unknown resource type "turbine"`))
		})

		It("Should refuse a negative limit or offset", func(ctx SpecContext) {
			Expect(svc.Search(ctx, nil, resource.SearchQuery{
				Term:   "pump",
				Limit:  -1,
				Offset: -1,
			})).Error().To(MatchError(
				"limit: must be greater than or equal to 0\n" +
					"offset: must be greater than or equal to 0",
			))
		})

		It("Should refuse a limit or offset past its bound", func(ctx SpecContext) {
			Expect(svc.Search(ctx, nil, resource.SearchQuery{
				Limit:  1001,
				Offset: 10001,
			})).Error().To(MatchError(
				"limit: must be less than or equal to 1000\n" +
					"offset: must be less than or equal to 10000",
			))
		})

		It("Should panic when a Provider returns an ID of a type with no Provider",
			func(ctx SpecContext) {
				users := provider[channelEntry]{
					typ:   resource.TypeUser,
					table: channels.table,
					idx:   channels.idx,
				}
				svc := MustSucceed(resource.NewService(resource.ServiceConfig{
					DB: db,
					Providers: map[resource.Type]resource.Provider{
						resource.TypeChannel: users,
					},
				}))
				Expect(func() {
					_, _ = svc.Search(ctx, nil, resource.SearchQuery{Term: "pump"})
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
					nil,
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

		It("Should return each resource once", func(ctx SpecContext) {
			Expect(MustSucceed(svc.Retrieve(
				ctx,
				nil,
				channels.id("3"),
				channels.id("3"),
			))).To(Equal([]resource.Resource{{ID: channels.id("3"), Name: "valve"}}))
		})

		It("Should leave out an ID that names no resource", func(ctx SpecContext) {
			Expect(MustSucceed(svc.Retrieve(
				ctx,
				nil,
				channels.id("99"),
				racks.id("1"),
			))).To(Equal([]resource.Resource{{ID: racks.id("1"), Name: "pump"}}))
		})

		It("Should refuse an ID of a type with no Provider", func(ctx SpecContext) {
			Expect(svc.Retrieve(
				ctx,
				nil,
				channels.id("1"),
				resource.ID{Type: resource.TypeUser, Key: "1"},
			)).Error().To(MatchError(`ids: unknown resource type "user"`))
		})
	})
})
