// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v5_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp"
	v4 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v4"
	v5 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v5"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migration", func() {
	It("Should lift each Arc with no channel lists and the rest unchanged", func(
		ctx SpecContext,
	) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v4.Key, v4.Arc]{DB: db}))
		dir := uuid.New()
		textSource := func(raw string) v4.Source {
			return v4.Source{
				Variant: v4.TextSource{Text: text.Text{Doc: text.Create(raw)}},
			}
		}
		automation := v4.Base{
			Key:       uuid.New(),
			Name:      "automation",
			Directory: &dir,
			Source:    textSource("a -> b"),
		}
		module := v4.Base{
			Key:    uuid.New(),
			Name:   "module",
			Source: textSource("func f() {}"),
		}
		arcs := []v4.Arc{
			{Variant: v4.AutomationArc{Base: automation}},
			{Variant: v4.ModuleArc{Base: module}},
		}
		Expect(gorp.NewCreate[v4.Key, v4.Arc]().Entries(&arcs).Exec(ctx, db)).
			To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Arc",
			Migrations: []gorp.Migration{v5.Migration},
		})).To(Succeed())
		var out []v5.Arc
		Expect(gorp.NewRetrieve[v5.Key, v5.Arc]().Entries(&out).Exec(ctx, db)).
			To(Succeed())
		Expect(out).To(HaveLen(2))
		for _, a := range out {
			b := a.Base()
			Expect(b.Reads).To(BeEmpty())
			Expect(b.Writes).To(BeEmpty())
			src, ok := b.Source.Variant.(v5.TextSource)
			Expect(ok).To(BeTrue())
			raw := src.Text.Materialize().Raw
			switch b.Key {
			case automation.Key:
				Expect(a.Variant).To(BeAssignableToTypeOf(v5.AutomationArc{}))
				Expect(b.Name).To(Equal("automation"))
				Expect(b.Directory).To(Equal(&dir))
				Expect(raw).To(Equal("a -> b"))
			case module.Key:
				Expect(a.Variant).To(BeAssignableToTypeOf(v5.ModuleArc{}))
				Expect(b.Name).To(Equal("module"))
				Expect(raw).To(Equal("func f() {}"))
			default:
				Fail("unexpected Arc " + b.Key.String())
			}
		}
	})
})
