// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Dispatch", func() {
	type frame = actions.Scoped[symbol.Key, symbol.Action]
	var (
		db       *gorp.DB
		svc      *symbol.Service
		groupSvc *group.Service
		g        group.Group
		sym      symbol.Symbol
	)
	write := func(ctx context.Context, fn func(symbol.Writer) error) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return fn(svc.NewWriter(tx).WithDispatchKey("dk"))
		})
	}
	createGroup := func(ctx context.Context, scope resource.Type) group.Group {
		GinkgoHelper()
		created := group.Group{Name: "Symbols", Scope: scope}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).Create(ctx, &created)
		})).To(Succeed())
		return created
	}
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		groupSvc = MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
		svc = MustOpen(symbol.OpenService(ctx, symbol.ServiceConfig{
			DB:    db,
			Group: groupSvc,
			ImEx:  imex.NewService(),
		}))
		g = createGroup(ctx, resource.TypeSchematicSymbol)
		sym = symbol.Symbol{
			Name: "Before",
			Data: symbol.Spec{SVG: "<svg/>", Variant: "static", Scale: 1},
		}
		Expect(write(ctx, func(w symbol.Writer) error { return w.Create(ctx, &sym) })).
			To(Succeed())
	})
	set := func(path string, value any) symbol.Action {
		return symbol.NewSetAction(
			symbol.SetPayload{Path: []string{path}, Value: value},
		)
	}
	apply := func(ctx context.Context, acts ...symbol.Action) error {
		return write(ctx, func(w symbol.Writer) error {
			return w.Apply(ctx, sym.Key, acts...)
		})
	}
	record := func() *Recorder[symbol.Key, symbol.Action] {
		rec := &Recorder[symbol.Key, symbol.Action]{}
		DeferCleanup(svc.Actions().OnChange(rec.Record))
		return rec
	}
	retrieve := func(ctx context.Context) symbol.Symbol {
		GinkgoHelper()
		var res symbol.Symbol
		Expect(svc.NewRetrieve().
			Where(symbol.MatchKeys(sym.Key)).
			Entry(&res).
			Exec(ctx, db)).To(Succeed())
		return res
	}

	It("Should store the edited name, data, and group", func(ctx SpecContext) {
		data := symbol.Spec{
			SVG:     "<svg></svg>",
			States:  []symbol.State{},
			Variant: "actuator",
			Handles: []symbol.Handle{},
			Scale:   2,
		}
		Expect(apply(
			ctx,
			set("name", "After"),
			set("data", data),
			set("group", g.Key),
		)).To(Succeed())
		Expect(retrieve(ctx)).To(Equal(symbol.Symbol{
			Key:   sym.Key,
			Name:  "After",
			Data:  data,
			Group: &g.Key,
		}))
	})

	DescribeTable("Should refuse an edit", func(
		ctx SpecContext,
		act func(context.Context) symbol.Action,
		msg string,
	) {
		err := apply(ctx, act(ctx))
		Expect(err).To(MatchError(validate.ErrValidation))
		Expect(err).To(MatchError(ContainSubstring(msg)))
		Expect(retrieve(ctx)).To(Equal(sym))
	},
		Entry("of the key", func(context.Context) symbol.Action {
			return set("key", sym.Key)
		}, `unknown object member name "key"`),
		Entry("to a group that cannot hold a symbol", func(
			ctx context.Context,
		) symbol.Action {
			return set("group", createGroup(ctx, resource.TypeChannel).Key)
		}, "a schematic_symbol cannot sit in a channel group"),
	)

	It("Should refuse an edit that sets a zero scale", func(ctx SpecContext) {
		Expect(apply(ctx, set("data", symbol.Spec{SVG: "<svg/>", Variant: "static"}))).
			To(MatchError(ContainSubstring("data.scale: must be positive, got 0")))
		Expect(retrieve(ctx)).To(Equal(sym))
	})

	It("Should keep a stored zero scale through an unrelated edit", func(
		ctx SpecContext,
	) {
		sym.Data.Scale = 0
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return gorp.NewCreate[symbol.Key, symbol.Symbol]().Entry(&sym).Exec(ctx, tx)
		})).To(Succeed())
		Expect(apply(ctx, set("name", "After"))).To(Succeed())
		Expect(retrieve(ctx).Data.Scale).To(BeZero())
	})

	It("Should refuse an invalid data", func(ctx SpecContext) {
		// Generated validators return errors that do not wrap validate.ErrValidation.
		Expect(apply(ctx, set("data", symbol.Spec{Variant: "static"}))).
			To(MatchError(ContainSubstring("data.svg: required")))
		Expect(retrieve(ctx)).To(Equal(sym))
	})

	It("Should broadcast a create and an applied batch", func(ctx SpecContext) {
		rec := record()
		created := symbol.Symbol{
			Name: "Created",
			Data: symbol.Spec{SVG: "<svg/>", Variant: "static", Scale: 1},
		}
		Expect(write(ctx, func(w symbol.Writer) error {
			return w.Create(ctx, &created)
		})).To(Succeed())
		Expect(apply(ctx, set("name", "After"))).To(Succeed())
		frames := rec.Snapshot()
		Expect(frames).To(Equal([]frame{
			{
				Key:         created.Key,
				DispatchKey: "dk",
				Seq:         frames[0].Seq,
				Actions: []symbol.Action{
					symbol.NewCreateAction(symbol.CreatePayload{Symbol: created}),
				},
			},
			{
				Key:         sym.Key,
				DispatchKey: "dk",
				Seq:         frames[1].Seq,
				Actions:     []symbol.Action{set("name", "After")},
			},
		}))
	})

	DescribeTable("Should broadcast the set of each edit route", func(
		ctx SpecContext,
		edit func(context.Context, symbol.Writer) error,
		want func() symbol.Action,
	) {
		rec := record()
		Expect(write(ctx, func(w symbol.Writer) error { return edit(ctx, w) })).
			To(Succeed())
		frames := rec.Snapshot()
		Expect(frames).To(HaveLen(1))
		Expect(frames[0].Actions).To(Equal([]symbol.Action{want()}))
	},
		Entry("a rename", func(ctx context.Context, w symbol.Writer) error {
			return w.Rename(ctx, sym.Key, "After")
		}, func() symbol.Action { return set("name", "After") }),
		Entry("a group change", func(ctx context.Context, w symbol.Writer) error {
			return w.SetGroup(ctx, &g.Key, sym.Key)
		}, func() symbol.Action { return set("group", &g.Key) }),
	)

	It("Should broadcast one set for each symbol of a group change", func(
		ctx SpecContext,
	) {
		other := symbol.Symbol{
			Name: "Other",
			Data: symbol.Spec{SVG: "<svg/>", Variant: "static", Scale: 1},
		}
		Expect(write(ctx, func(w symbol.Writer) error {
			return w.Create(ctx, &other)
		})).To(Succeed())
		rec := record()
		Expect(write(ctx, func(w symbol.Writer) error {
			return w.SetGroup(ctx, &g.Key, sym.Key, other.Key)
		})).To(Succeed())
		frames := rec.Snapshot()
		Expect(frames).To(HaveLen(2))
		want := []symbol.Action{set("group", &g.Key)}
		Expect(frames).To(Equal([]frame{
			{Key: sym.Key, DispatchKey: "dk", Seq: frames[0].Seq, Actions: want},
			{Key: other.Key, DispatchKey: "dk", Seq: frames[1].Seq, Actions: want},
		}))
	})
})
