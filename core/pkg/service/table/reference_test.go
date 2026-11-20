// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package table_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/color"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	xcolor "github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/defaults"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Color references", func() {
	It(
		"Should inline a deleted named color into the cells that use it",
		func(ctx SpecContext) {
			gone := color.Named{
				Name:  "Fault",
				Light: MustSucceed(xcolor.Parse("#ff0000")),
			}
			kept := color.Named{
				Name:  "Nominal",
				Light: MustSucceed(xcolor.Parse("#00ff00")),
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				w := colorSvc.NewWriter(tx)
				if err := w.Create(ctx, &gone); err != nil {
					return err
				}
				return w.Create(ctx, &kept)
			})).To(Succeed())
			named := func(n color.Named) *color.Ref {
				return &color.Ref{Variant: color.NamedRef{Key: n.Key}}
			}
			text := defaults.Fill(table.CellConfig{Variant: table.TextCellConfig{
				FillColor: named(gone),
				TextColor: named(kept),
			}})
			value := defaults.Fill(table.CellConfig{Variant: table.ValueCellConfig{
				StalenessColor: named(gone),
			}})
			t := table.Table{
				Project: &proj.Key,
				Name:    "colored",
				Rows: []table.Row{{
					Key:   "r",
					Size:  30,
					Cells: []table.Cell{cell("a", text), cell("b", value)},
				}},
				Columns: []table.Column{{Key: "c0", Size: 80}, {Key: "c1", Size: 80}},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &t)
			})).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					if err := svc.NewWriter(tx).Delete(ctx, t.Key); err != nil {
						return err
					}
					return colorSvc.NewWriter(tx).Delete(ctx, kept.Key)
				})).To(Succeed())
			})
			rec := &Recorder[table.Key, table.Action]{}
			DeferCleanup(svc.OnAction(rec.Record))
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return colorSvc.NewWriter(tx).Delete(ctx, gone.Key)
			})).To(Succeed())
			var res table.Table
			Expect(svc.NewRetrieve().
				Where(table.MatchKeys(t.Key)).
				Entry(&res).
				Exec(ctx, db)).To(Succeed())
			fixed := &color.Ref{Variant: color.FixedRef{Value: gone.Light}}
			Expect(asText(configOf(res, "a")).FillColor).To(Equal(fixed))
			Expect(asText(configOf(res, "a")).TextColor).To(Equal(named(kept)))
			v, ok := configOf(res, "b").Variant.(table.ValueCellConfig)
			Expect(MustBeOk(v, ok).StalenessColor).To(Equal(fixed))
			seen := rec.Snapshot()
			Expect(seen).To(HaveLen(1))
			Expect(seen[0].Key).To(Equal(t.Key))
			Expect(seen[0].Actions).To(HaveLen(1))
			Expect(seen[0].Actions[0].Create).ToNot(BeNil())
			Expect(seen[0].Actions[0].Create.Table).To(Equal(res))
		},
	)
})
