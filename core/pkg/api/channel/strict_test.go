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
	"fmt"
	"reflect"
	"slices"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Strict retrieve", func() {
	var (
		u             user.User
		shown, hidden group.Group
		name, solo    string
		inShown       channel.Channel
		inHidden      channel.Channel
		alone         channel.Channel
		top           channel.Channel
	)
	stored := func(n string, key channel.LocalKey, g *group.Key) channel.Channel {
		return channel.Channel{
			Name:        n,
			Leaseholder: node.KeyFree,
			LocalKey:    key,
			DataType:    telem.Float32T,
			Virtual:     true,
			Group:       g,
		}
	}
	BeforeEach(func(ctx SpecContext) {
		u = MustSucceed(rbactestutil.CreateUser(
			ctx, db, rbacSvc, userSvc, user.User{Name: uuid.New().String()},
		))
		shown = group.Group{Name: UniqueChannelName(), Scope: resource.TypeChannel}
		hidden = group.Group{Name: UniqueChannelName(), Scope: resource.TypeChannel}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			w := groupSvc.NewWriter(tx)
			for _, g := range []*group.Group{&shown, &hidden} {
				if err := w.Create(ctx, g); err != nil {
					return err
				}
			}
			return nil
		})).To(Succeed())
		name, solo = UniqueChannelName(), UniqueChannelName()
		inShown = stored(name, 1_000_301, &shown.Key)
		inHidden = stored(name, 1_000_302, &hidden.Key)
		alone = stored(solo, 1_000_303, &shown.Key)
		top = stored(UniqueChannelName(), 1_000_304, nil)
		Expect(gorp.WrapWriter[channel.Key, channel.Channel](db).
			Set(ctx, inShown, inHidden, alone, top)).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(gorp.WrapWriter[channel.Key, channel.Channel](db).
				Delete(ctx, inShown.Key(), inHidden.Key(), alone.Key(), top.Key())).
				To(Succeed())
		})
		grantOn(
			ctx,
			u.ResourceID(),
			[]access.Action{access.ActionRetrieve},
			resource.ID{Type: resource.TypeChannel},
			shown.ResourceID(),
		)
	})
	retrieve := func(
		ctx SpecContext,
		where channel.Where,
		strict bool,
	) (apichannel.RetrieveResponse, error) {
		return retrieveChannels(AuthedCtx(ctx, u), apichannel.RetrieveRequest{
			Where:  &where,
			Strict: strict,
		})
	}
	byName := func(names ...string) channel.Where {
		return channel.Where{Name: &filter.String{In: names}}
	}

	It("Should refuse a name that no channel at the top level holds, naming only the "+
		"paths of the name that the subject can see", func(ctx SpecContext) {
		Expect(retrieve(ctx, byName(name), true)).Error().To(SatisfyAll(
			MatchError(query.ErrNotFound),
			MatchError(fmt.Sprintf(
				"%s does not exist. Did you mean %s.%s?",
				name,
				shown.Name,
				name,
			)),
		))
	})

	It("Should return every holder of a name when the retrieve is not strict", func(
		ctx SpecContext,
	) {
		res := MustSucceed(retrieve(ctx, byName(name), false))
		Expect(res.Channels).To(ConsistOf(
			HaveField("Key", inShown.Key()),
			HaveField("Key", inHidden.Key()),
		))
	})

	It("Should return the channel of a name at the top level", func(ctx SpecContext) {
		res := MustSucceed(retrieve(ctx, byName(top.Name), true))
		Expect(res.Channels).To(ConsistOf(HaveField("Key", top.Key())))
	})

	It("Should take keys, names, and paths as the branches of one or", func(
		ctx SpecContext,
	) {
		path := shown.Name + "." + solo
		res := MustSucceed(retrieve(ctx, channel.Where{Or: []channel.Where{
			{Key: &filter.Set[channel.Key]{In: []channel.Key{inShown.Key()}}},
			{Name: &filter.String{In: []string{top.Name}}},
			{Path: &filter.Set[string]{In: []string{path}}},
		}}, true))
		Expect(res.Channels).To(ConsistOf(
			HaveField("Key", inShown.Key()),
			HaveField("Key", top.Key()),
			HaveField("Key", alone.Key()),
		))
	})

	It("Should refuse a name in a branch of the or that no channel at the top level "+
		"holds", func(ctx SpecContext) {
		Expect(retrieve(ctx, channel.Where{Or: []channel.Where{
			{Name: &filter.String{In: []string{name}}},
		}}, true)).Error().To(MatchError(query.ErrNotFound))
	})

	DescribeTable("Should refuse a strict where with any other match", func(
		ctx SpecContext,
		where channel.Where,
	) {
		Expect(retrieve(ctx, where, true)).Error().To(MatchError(ContainSubstring(
			"where: a strict retrieve takes only key, name, and path matches",
		)))
	},
		Entry("a data type", channel.Where{
			Name:     &filter.String{In: []string{"x"}},
			DataType: &filter.Set[telem.DataType]{In: []telem.DataType{telem.Float32T}},
		}),
		Entry("a name prefix", channel.Where{
			Name: &filter.String{StartsWith: new("x")},
		}),
		Entry("a nested and", channel.Where{Or: []channel.Where{
			{And: []channel.Where{{Name: &filter.String{In: []string{"x"}}}}},
		}}),
		Entry("an or beside another match", channel.Where{
			Or:   []channel.Where{{Name: &filter.String{In: []string{"x"}}}},
			Name: &filter.String{In: []string{"y"}},
		}),
	)

	DescribeTable("Should refuse each other field of a where and of its lookup filters",
		func(ctx SpecContext, w channel.Where) {
			Expect(retrieve(ctx, w, true)).Error().To(MatchError(ContainSubstring(
				"where: a strict retrieve takes only key, name, and path matches",
			)))
		},
		otherWhereFields(),
	)

	DescribeTable("Should refuse a strict retrieve that pages or searches",
		func(ctx SpecContext, req apichannel.RetrieveRequest, msg string) {
			where := byName(name)
			req.Where, req.Strict = &where, true
			Expect(retrieveChannels(AuthedCtx(ctx, u), req)).Error().
				To(MatchError(msg))
		},
		Entry(
			"a search term",
			apichannel.RetrieveRequest{SearchTerm: "x"},
			"search_term: a strict retrieve takes no search term",
		),
		Entry(
			"an order",
			apichannel.RetrieveRequest{
				Order: &apichannel.Order{Field: apichannel.OrderFieldName},
			},
			"order: a strict retrieve takes no order",
		),
		Entry(
			"a cursor",
			apichannel.RetrieveRequest{After: &apichannel.Cursor{}},
			"after: a strict retrieve takes no cursor",
		),
		Entry(
			"a limit",
			apichannel.RetrieveRequest{Limit: new(uint32(1))},
			"limit: a strict retrieve takes no limit",
		),
	)
})

// otherWhereFields returns an entry with a where for each field of a where, and of its
// key, name, and path filters, that a strict retrieve refuses.
func otherWhereFields() []TableEntry {
	set := func(v reflect.Value) {
		switch v.Kind() {
		case reflect.Pointer:
			v.Set(reflect.New(v.Type().Elem()))
		case reflect.Slice:
			v.Set(reflect.MakeSlice(v.Type(), 1, 1))
		case reflect.Bool:
			v.SetBool(true)
		default:
			panic("no non-zero value for kind " + v.Kind().String())
		}
	}
	var entries []TableEntry
	fields := reflect.TypeFor[channel.Where]()
	for i := range fields.NumField() {
		field := fields.Field(i).Name
		if slices.Contains([]string{"Key", "Name", "Path", "Or"}, field) {
			continue
		}
		var w channel.Where
		set(reflect.ValueOf(&w).Elem().Field(i))
		entries = append(entries, Entry(field, w))
	}
	lookups := []struct {
		name  string
		fresh func() (channel.Where, reflect.Value)
	}{
		{"name", func() (channel.Where, reflect.Value) {
			w := channel.Where{Name: &filter.String{}}
			return w, reflect.ValueOf(w.Name).Elem()
		}},
		{"key", func() (channel.Where, reflect.Value) {
			w := channel.Where{Key: &filter.Set[channel.Key]{Eq: new(channel.Key(1))}}
			return w, reflect.ValueOf(w.Key).Elem()
		}},
		{"path", func() (channel.Where, reflect.Value) {
			w := channel.Where{Path: &filter.Set[string]{Eq: new("x")}}
			return w, reflect.ValueOf(w.Path).Elem()
		}},
	}
	for _, l := range lookups {
		_, v := l.fresh()
		for i := range v.NumField() {
			field := v.Type().Field(i).Name
			if field == "Eq" || field == "In" {
				continue
			}
			w, lookup := l.fresh()
			set(lookup.Field(i))
			entries = append(entries, Entry(l.name+" "+field, w))
		}
	}
	return entries
}
