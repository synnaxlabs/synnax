// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"slices"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/arc/versions"
	v3 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v3"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("OpenTable", func() {
	It("Should open a table that stores Arcs", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		MustOpen(arc.OpenTable(ctx, arc.TableConfig{DB: db}))
		a := newAutomation(arc.Base{
			Key:    uuid.New(),
			Name:   "stored",
			Source: arc.Source{Variant: arc.TextSource{}},
		})
		Expect(gorp.NewCreate[arc.Key, arc.Arc]().Entry(&a).Exec(ctx, db)).
			To(Succeed())
		var stored arc.Arc
		Expect(gorp.NewRetrieve[arc.Key, arc.Arc]().
			Where(gorp.MatchKeys[arc.Key, arc.Arc](a.Base().Key)).
			Entry(&stored).
			Exec(ctx, db)).To(Succeed())
		Expect(stored.Base().Name).To(Equal("stored"))
	})

	It("Should migrate stored v3 Arcs with its parser config", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		chain := versions.NewMigrations(uuid.New(), parser.Config{})
		swap := slices.IndexFunc(chain, func(m gorp.Migration) bool {
			return m.Key() == "v61_swap_brackets"
		})
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Arc",
			Migrations: chain[:swap],
		})).To(Succeed())
		MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[v3.Key, v3.Arc]{DB: db}))
		seed := v3.Arc{
			Key:  uuid.New(),
			Mode: v3.ModeText,
			Text: text.Text{Doc: text.Create("func my-fn{k f64}() {\n    out = k\n}")},
		}
		Expect(gorp.NewCreate[v3.Key, v3.Arc]().Entry(&seed).Exec(ctx, db)).
			To(Succeed())
		MustOpen(arc.OpenTable(ctx, arc.TableConfig{
			DB:     db,
			Parser: parser.Config{AllowDashedNames: true},
		}))
		var stored arc.Arc
		Expect(gorp.NewRetrieve[arc.Key, arc.Arc]().
			Where(gorp.MatchKeys[arc.Key, arc.Arc](seed.Key)).
			Entry(&stored).
			Exec(ctx, db)).To(Succeed())
		src, ok := stored.Base().Source.Variant.(arc.TextSource)
		Expect(ok).To(BeTrue())
		Expect(src.Text.Materialize().Raw).
			To(Equal("func my-fn(k f64){} {\n    out = k\n}"))
	})

	It("Should reject a config with no DB", func(ctx SpecContext) {
		Expect(arc.OpenTable(ctx, arc.TableConfig{})).Error().
			To(MatchError(ContainSubstring("db: must be non-nil")))
	})
})

var _ = Describe("TableConfig", func() {
	dashed := parser.Config{AllowDashedNames: true}

	It("Should take a non-zero parser config", func() {
		Expect(arc.TableConfig{}.Override(arc.TableConfig{Parser: dashed}).Parser).
			To(Equal(dashed))
	})

	It("Should keep its parser config over a zero one", func() {
		Expect(arc.TableConfig{Parser: dashed}.Override(arc.TableConfig{}).Parser).
			To(Equal(dashed))
	})
})
