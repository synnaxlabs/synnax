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
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
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

// fixture is a Gorp table of E with the TableProvider for its records.
type fixture[E entry] struct {
	typ      resource.Type
	table    *gorp.Table[string, E]
	idx      *gorp.SearchIndex[string, E]
	provider *resource.TableProvider[string, E]
}

// parseKey accepts only the decimal keys that the fixtures store.
func parseKey(id resource.ID) (string, error) {
	if _, err := strconv.Atoi(id.Key); err != nil {
		return "", errors.Wrapf(validate.ErrValidation, "invalid key %q", id.Key)
	}
	return id.Key, nil
}

func openFixture[E entry](
	ctx context.Context,
	db *gorp.DB,
	typ resource.Type,
) fixture[E] {
	GinkgoHelper()
	idx := gorp.NewSearchIndex("name", func(e *E) []string {
		return []string{(*e).name()}
	})
	table := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, E]{
		DB:      db,
		Indexes: []gorp.Index[string, E]{idx},
	}))
	DeferClose(table)
	f := fixture[E]{typ: typ, table: table, idx: idx}
	f.provider = f.providerOf(typ)
	return f
}

// providerOf returns a TableProvider for the records of f that names them with t.
func (f fixture[E]) providerOf(t resource.Type) *resource.TableProvider[string, E] {
	return resource.NewTableProvider(resource.TableProviderConfig[string, E]{
		Table:  f.table,
		Search: f.idx,
		ID:     func(key string) resource.ID { return resource.ID{Type: t, Key: key} },
		Key:    parseKey,
		Name:   func(e *E) string { return (*e).name() },
	})
}

func (f fixture[E]) id(key string) resource.ID {
	return resource.ID{Type: f.typ, Key: key}
}

func (f fixture[E]) create(ctx context.Context, tx gorp.Tx, entries ...E) {
	GinkgoHelper()
	Expect(f.table.NewCreate().Entries(&entries).Exec(ctx, tx)).To(Succeed())
}

var _ = Describe("SearchIndex", func() {
	var (
		db *gorp.DB
		p  fixture[channelEntry]
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		p = openFixture[channelEntry](ctx, db, resource.TypeChannel)
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

var _ = Describe("TableProvider", func() {
	var (
		db       *gorp.DB
		channels fixture[channelEntry]
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		channels = openFixture[channelEntry](ctx, db, resource.TypeChannel)
		channels.create(
			ctx,
			db,
			channelEntry{Key: "1", Name: "pump pressure"},
			channelEntry{Key: "2", Name: "pump"},
			channelEntry{Key: "3", Name: "valve"},
		)
	})

	Describe("Search", func() {
		It("Should return the IDs of the matches, closest match first",
			func(ctx SpecContext) {
				Expect(MustSucceed(channels.provider.Search(
					ctx,
					db,
					search.Query[resource.ID]{Term: "pump"},
				)).Keys).To(Equal([]resource.ID{channels.id("2"), channels.id("1")}))
			},
		)
	})

	Describe("Retrieve", func() {
		It("Should return the names of the records that the IDs name",
			func(ctx SpecContext) {
				Expect(channels.provider.Retrieve(
					ctx,
					db,
					[]resource.ID{channels.id("3"), channels.id("1")},
				)).To(ConsistOf(
					resource.Resource{ID: channels.id("3"), Name: "valve"},
					resource.Resource{ID: channels.id("1"), Name: "pump pressure"},
				))
			},
		)

		It("Should leave out an ID that names no record", func(ctx SpecContext) {
			Expect(channels.provider.Retrieve(
				ctx,
				db,
				[]resource.ID{channels.id("99"), channels.id("2")},
			)).To(Equal([]resource.Resource{{ID: channels.id("2"), Name: "pump"}}))
		})

		It("Should return the error of Key for a malformed key", func(ctx SpecContext) {
			Expect(channels.provider.Retrieve(
				ctx,
				db,
				[]resource.ID{channels.id("pump")},
			)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(`invalid key "pump"`)),
			))
		})
	})

	Describe("List", func() {
		It("Should return at most limit records in the order of their keys",
			func(ctx SpecContext) {
				Expect(channels.provider.List(ctx, db, 2)).To(Equal([]resource.Resource{
					{ID: channels.id("1"), Name: "pump pressure"},
					{ID: channels.id("2"), Name: "pump"},
				}))
			},
		)
	})
})
