// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	graph "github.com/synnaxlabs/arc/graph/versions/v1"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp"
	v3 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v4"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

const (
	oldText = "sensor -> scale{factor=2.0} -> out"
	newText = "sensor -> scale(factor=2.0) -> out"
)

var _ = Describe("MigrateArc", func() {
	It("Should lex a stored Arc with the dashed names of its Core", func(
		ctx SpecContext,
	) {
		seed := v3.Arc{
			Key:  uuid.New(),
			Mode: v3.ModeText,
			Text: text.Text{Doc: text.Create(
				"func my-fn{n i32}() {\n    if n > 0 {\n" +
					"        for i := range(n, i64(100)) {}\n    }\n}",
			)},
		}
		cfg := parser.Config{AllowDashedNames: true}
		got := textOf(migrateFromV3(ctx, seed, cfg).Base()).Materialize().Raw
		Expect(got).To(Equal(
			"func my-fn(n i32){} {\n    if n > 0 {\n" +
				"        for i := range(i64(n), i64(100)) {}\n    }\n}",
		))
	})

	It("Should rewrite the text of a stored Arc into a fresh document", func(
		ctx SpecContext,
	) {
		seed := v3.Arc{
			Key:  uuid.New(),
			Name: "stored",
			Mode: v3.ModeText,
			Text: text.Text{Doc: text.Create(oldText)},
		}
		got := migrateFromV3(ctx, seed).Base()
		Expect(got.Key).To(Equal(seed.Key))
		Expect(got.Name).To(Equal(seed.Name))
		src := textOf(got)
		Expect(src.Materialize().Raw).To(Equal(newText))
		Expect(src.Raw).To(BeEmpty())
	})

	It("Should seed the document of an exported Arc from its raw text", func(
		ctx SpecContext,
	) {
		got := MustSucceed(v4.MigrateArc(ctx, v3.Arc{
			Name: "exported",
			Mode: v3.ModeText,
			Text: text.Text{Raw: oldText},
		})).Base()
		src := textOf(got)
		Expect(src.Materialize().Raw).To(Equal(newText))
		Expect(src.Raw).To(BeEmpty())
	})

	It("Should cast the mixed range() arguments of a stored 0.60 program", func(
		ctx SpecContext,
	) {
		seed := v3.Arc{
			Key:  uuid.New(),
			Name: "ranges",
			Mode: v3.ModeText,
			Text: text.Text{Doc: text.Create(
				"func f{n i32} () {\n    for i := range(n, i64(100)) {}\n}",
			)},
		}
		got := textOf(migrateFromV3(ctx, seed).Base()).Materialize().Raw
		Expect(got).To(Equal(
			"func f(n i32) {} {\n    for i := range(i64(n), i64(100)) {}\n}",
		))
		diag := analyze(ctx, got)
		Expect(diag.Ok()).To(BeTrue(), diag.String())
	})

	It("Should keep a stored program that does not parse", func(ctx SpecContext) {
		const broken = "func f() {\n    for i := range(n, i64(100)) {}\n"
		seed := v3.Arc{
			Key:  uuid.New(),
			Mode: v3.ModeText,
			Text: text.Text{Doc: text.Create(broken)},
		}
		got := textOf(migrateFromV3(ctx, seed).Base()).Materialize().Raw
		Expect(got).To(Equal(v4.SwapBrackets(broken)))
	})

	It("Should cast the mixed range() arguments of an imported program", func(
		ctx SpecContext,
	) {
		got := MustSucceed(v4.MigrateArc(ctx, v3.Arc{
			Name: "imported",
			Mode: v3.ModeText,
			Text: text.Text{
				Raw: "func f{n i32} () {\n    for i := range(n, i64(1)) {}\n}",
			},
		})).Base()
		Expect(textOf(got).Materialize().Raw).To(Equal(
			"func f(n i32) {} {\n    for i := range(i64(n), i64(1)) {}\n}",
		))
	})

	It("Should lift a v3 Arc as an automation", func(ctx SpecContext) {
		key := uuid.New()
		migrated := MustSucceed(v4.MigrateArc(ctx, v3.Arc{
			Key:  key,
			Name: "legacy",
			Mode: v3.ModeText,
		}))
		automation, ok := migrated.Variant.(v4.AutomationArc)
		Expect(ok).To(BeTrue())
		Expect(automation.Key).To(Equal(key))
		Expect(automation.Name).To(Equal("legacy"))
		Expect(automation.Source.Mode()).To(Equal(v4.ModeText))
	})

	It("Should leave an Arc with no text untouched", func(ctx SpecContext) {
		got := MustSucceed(
			v4.MigrateArc(ctx, v3.Arc{Name: "empty", Mode: v3.ModeText}),
		).Base()
		src := textOf(got)
		Expect(src.Doc.Inserts).To(BeEmpty())
		Expect(src.Raw).To(BeEmpty())
	})

	It("Should keep only the graph of a graph-mode Arc", func(ctx SpecContext) {
		seed := v3.Arc{
			Key:   uuid.New(),
			Name:  "graph",
			Mode:  v3.ModeGraph,
			Graph: graph.Graph{Nodes: graph.Nodes{{Key: "n1"}}},
			Text:  text.Text{Doc: text.Create(oldText)},
		}
		got := migrateFromV3(ctx, seed).Base()
		Expect(got.Source.Variant).To(Equal(v4.GraphSource{Graph: seed.Graph}))
	})

	It("Should keep only the text of a text-mode Arc", func(ctx SpecContext) {
		got := MustSucceed(v4.MigrateArc(ctx, v3.Arc{
			Name:  "text",
			Mode:  v3.ModeText,
			Graph: graph.Graph{Nodes: graph.Nodes{{Key: "n1"}}},
			Text:  text.Text{Raw: oldText},
		})).Base()
		Expect(textOf(got).Materialize().Raw).To(Equal(newText))
	})

	It("Should lift an Arc with no mode as a graph", func(ctx SpecContext) {
		seed := graph.Graph{Nodes: graph.Nodes{{Key: "n1"}}}
		got := MustSucceed(
			v4.MigrateArc(ctx, v3.Arc{Name: "modeless", Graph: seed}),
		).Base()
		Expect(got.Source.Variant).To(Equal(v4.GraphSource{Graph: seed}))
	})
})

// textOf returns the text of b, failing the spec when b is not text-mode.
func textOf(b v4.Base) text.Text {
	GinkgoHelper()
	src, ok := b.Source.Variant.(v4.TextSource)
	Expect(ok).To(BeTrue(), "expected a text source, got %T", b.Source.Variant)
	return src.Text
}

// migrateFromV3 runs the v4 migration with cfgs over a gorp-seeded v3 Arc and returns
// the migrated current Arc.
func migrateFromV3(ctx SpecContext, seed v3.Arc, cfgs ...parser.Config) v4.Arc {
	GinkgoHelper()
	db := DeferClose(gorp.Wrap(memkv.New()))
	MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[v3.Key, v3.Arc]{DB: db}))
	Expect(gorp.NewCreate[v3.Key, v3.Arc]().Entry(&seed).Exec(ctx, db)).To(Succeed())
	Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
		DB:         db,
		Namespace:  "Arc",
		Migrations: []gorp.Migration{v4.NewMigration(parser.ConfigOf(cfgs...))},
	})).To(Succeed())
	var got v4.Arc
	Expect(gorp.NewRetrieve[v4.Key, v4.Arc]().
		Where(gorp.MatchKeys[v4.Key, v4.Arc](seed.Key)).
		Entry(&got).Exec(ctx, db)).To(Succeed())
	return got
}

var _ = Describe("NewRootMigration", func() {
	It("Should put every Arc in no directory into the root", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		root, sub := uuid.New(), uuid.New()
		table := MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[v4.Key, v4.Arc]{
			DB: db,
		}))
		orphan := v4.Arc{Variant: v4.AutomationArc{
			Base: v4.Base{
				Key:    uuid.New(),
				Name:   "orphan",
				Source: v4.Source{Variant: v4.TextSource{}},
			},
		}}
		placed := v4.Arc{Variant: v4.ModuleArc{
			Base: v4.Base{
				Key:       uuid.New(),
				Name:      "placed",
				Source:    v4.Source{Variant: v4.TextSource{}},
				Directory: &sub,
			},
		}}
		Expect(table.NewCreate().Entry(&orphan).Exec(ctx, db)).To(Succeed())
		Expect(table.NewCreate().Entry(&placed).Exec(ctx, db)).To(Succeed())
		Expect(table.Close()).To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Arc",
			Migrations: []gorp.Migration{v4.NewRootMigration(root)},
		})).To(Succeed())
		directoryOf := func(key v4.Key) *uuid.UUID {
			GinkgoHelper()
			var a v4.Arc
			Expect(gorp.NewRetrieve[v4.Key, v4.Arc]().
				Where(gorp.MatchKeys[v4.Key, v4.Arc](key)).
				Entry(&a).
				Exec(ctx, db)).To(Succeed())
			return a.Base().Directory
		}
		Expect(directoryOf(orphan.Base().Key)).To(HaveValue(Equal(root)))
		Expect(directoryOf(placed.Base().Key)).To(HaveValue(Equal(sub)))
	})
})
