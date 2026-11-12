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
	"encoding/json/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	"github.com/synnaxlabs/x/defaults"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Writer", func() {
	var tx gorp.Tx
	BeforeEach(func() { tx = DeferClose(db.OpenTx()) })
	Describe("Create", func() {
		It("Should create a Symbol", func(ctx SpecContext) {
			sym := defaults.Fill(symbol.Symbol{
				Name: "test-symbol",
				Data: symbol.Spec{
					SVG:     "<svg>...</svg>",
					Variant: "valve",
					States: []symbol.State{
						{Key: "default", Name: "default"},
						{Key: "active", Name: "active"},
					},
				},
			})
			Expect(svc.NewWriter(tx).Create(ctx, &sym)).To(Succeed())
			Expect(sym.Key).ToNot(Equal(uuid.Nil()))
		})

		DescribeTable("Should reject a symbol that fails schema validation",
			func(ctx SpecContext, sym symbol.Symbol, msg string) {
				Expect(svc.NewWriter(tx).Create(ctx, &sym)).
					To(MatchError(ContainSubstring(msg)))
			},
			Entry("missing name", defaults.Fill(symbol.Symbol{
				Data: symbol.Spec{SVG: "<svg/>", Variant: "valve"},
			}), "name: required"),
			Entry("missing svg", defaults.Fill(symbol.Symbol{
				Name: "no-svg",
				Data: symbol.Spec{Variant: "valve"},
			}), "svg: required"),
			Entry("missing variant", defaults.Fill(symbol.Symbol{
				Name: "no-variant",
				Data: symbol.Spec{SVG: "<svg/>"},
			}), "variant: required"),
			Entry("invalid handle orientation", defaults.Fill(symbol.Symbol{
				Name: "bad-handle",
				Data: symbol.Spec{
					SVG:     "<svg/>",
					Variant: "valve",
					Handles: []symbol.Handle{{Key: "h1", Orientation: "diagonal"}},
				},
			}), "invalid orientation"),
		)

		It(
			"Should default the scale that a decoded symbol omits",
			func(ctx SpecContext) {
				var sym symbol.Symbol
				Expect(json.Unmarshal(
					[]byte(
						`{"name":"scaled","data":{"svg":"<svg/>","variant":"valve"}}`,
					),
					&sym,
				)).To(Succeed())
				Expect(svc.NewWriter(tx).Create(ctx, &sym)).To(Succeed())
				var res symbol.Symbol
				Expect(svc.NewRetrieve().
					Where(symbol.MatchKeys(sym.Key)).
					Entry(&res).
					Exec(ctx, tx)).To(Succeed())
				Expect(res.Data.Scale).To(Equal(1.0))
			},
		)

		It("Should reject a decoded symbol with a zero scale", func(ctx SpecContext) {
			var sym symbol.Symbol
			Expect(json.Unmarshal([]byte(
				`{"name":"scaled","data":{"svg":"<svg/>","variant":"valve","scale":0}}`,
			), &sym)).To(Succeed())
			Expect(svc.NewWriter(tx).Create(ctx, &sym)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("data.scale: must be positive")),
			))
		})

		It("Should create a Symbol with a predefined key", func(ctx SpecContext) {
			key := uuid.New()
			sym := defaults.Fill(symbol.Symbol{
				Key:  key,
				Name: "predefined-key-symbol",
				Data: symbol.Spec{SVG: "<svg>...</svg>", Variant: "valve"},
			})
			Expect(svc.NewWriter(tx).Create(ctx, &sym)).To(Succeed())
			Expect(sym.Key).To(Equal(key))
		})

		It("Should refuse a symbol whose key exists", func(ctx SpecContext) {
			g := group.Group{Name: "Stored", Scope: resource.TypeSchematicSymbol}
			Expect(groupSvc.NewWriter(tx).Create(ctx, &g)).To(Succeed())
			stored := defaults.Fill(symbol.Symbol{
				Name:  "original-name",
				Data:  symbol.Spec{SVG: "<svg>original</svg>", Variant: "valve"},
				Group: &g.Key,
			})
			Expect(svc.NewWriter(tx).Create(ctx, &stored)).To(Succeed())
			again := defaults.Fill(symbol.Symbol{
				Key:  stored.Key,
				Name: "updated-name",
				Data: symbol.Spec{SVG: "<svg>updated</svg>", Variant: "valve"},
			})
			Expect(svc.NewWriter(tx).Create(ctx, &again)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					"schematic_symbol %s already exists",
					stored.Key,
				)),
			))
			var res symbol.Symbol
			Expect(svc.NewRetrieve().
				Where(symbol.MatchKeys(stored.Key)).
				Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(Equal(stored))
		})

		It("Should create a symbol in a group", func(ctx SpecContext) {
			g := group.Group{Name: "Valves", Scope: resource.TypeSchematicSymbol}
			Expect(groupSvc.NewWriter(tx).Create(ctx, &g)).To(Succeed())
			sym := defaults.Fill(symbol.Symbol{
				Name:  "group-test",
				Data:  symbol.Spec{SVG: "<svg>...</svg>", Variant: "valve"},
				Group: &g.Key,
			})
			Expect(svc.NewWriter(tx).Create(ctx, &sym)).To(Succeed())
			var res []symbol.Symbol
			Expect(svc.NewRetrieve().
				Where(symbol.MatchGroups(g.Key)).
				Entries(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(HaveLen(1))
			Expect(res[0].Key).To(Equal(sym.Key))
		})
	})

	Describe("SetGroup", func() {
		It("Should move symbols into a group and back out", func(ctx SpecContext) {
			g := group.Group{Name: "Moved", Scope: resource.TypeSchematicSymbol}
			Expect(groupSvc.NewWriter(tx).Create(ctx, &g)).To(Succeed())
			sym := defaults.Fill(symbol.Symbol{
				Name: "moved",
				Data: symbol.Spec{SVG: "<svg/>", Variant: "valve"},
			})
			w := svc.NewWriter(tx)
			Expect(w.Create(ctx, &sym)).To(Succeed())
			Expect(w.SetGroup(ctx, &g.Key, sym.Key)).To(Succeed())
			retrieve := func() symbol.Symbol {
				var res symbol.Symbol
				Expect(svc.NewRetrieve().
					Where(symbol.MatchKeys(sym.Key)).
					Entry(&res).
					Exec(ctx, tx)).To(Succeed())
				return res
			}
			Expect(retrieve().Group).To(Equal(&g.Key))
			Expect(w.SetGroup(ctx, nil, sym.Key)).To(Succeed())
			Expect(retrieve().Group).To(BeNil())
		})
	})

	Describe("CreateMany", func() {
		It("Should create multiple Symbols", func(ctx SpecContext) {
			symbols := defaults.Fill([]symbol.Symbol{
				{
					Name: "symbol-1",
					Data: symbol.Spec{SVG: "<svg>1</svg>", Variant: "valve"},
				},
				{
					Name: "symbol-2",
					Data: symbol.Spec{SVG: "<svg>2</svg>", Variant: "valve"},
				},
			})
			Expect(
				svc.NewWriter(tx).CreateMany(ctx, &symbols),
			).To(Succeed())

			var retrieved []symbol.Symbol
			Expect(svc.NewRetrieve().Where(symbol.MatchKeys(
				symbols[0].Key,
				symbols[1].Key,
			)).Entries(&retrieved).Exec(ctx, tx)).To(Succeed())
			Expect(retrieved).To(HaveLen(2))
		})
	})

	Describe("Rename", func() {
		It("Should rename a Symbol", func(ctx SpecContext) {
			sym := defaults.Fill(symbol.Symbol{
				Name: "original-name",
				Data: symbol.Spec{SVG: "<svg>...</svg>", Variant: "valve"},
			})
			Expect(svc.NewWriter(tx).Create(ctx, &sym)).To(Succeed())
			Expect(svc.NewWriter(tx).Rename(ctx, sym.Key, "new-name")).To(Succeed())

			var res symbol.Symbol
			Expect(svc.NewRetrieve().
				Where(symbol.MatchKeys(sym.Key)).
				Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res.Name).To(Equal("new-name"))
		})

		It("Should not affect data when renaming", func(ctx SpecContext) {
			originalData := symbol.Spec{
				SVG:     "<svg>complex</svg>",
				Variant: "valve",
				States: []symbol.State{
					{Key: "default", Name: "default"},
					{Key: "active", Name: "active"},
					{Key: "error", Name: "error"},
				},
			}
			sym := defaults.Fill(symbol.Symbol{
				Name: "data-preservation-test",
				Data: originalData,
			})
			Expect(svc.NewWriter(tx).Create(ctx, &sym)).To(Succeed())
			Expect(svc.NewWriter(tx).Rename(ctx, sym.Key, "renamed")).To(Succeed())

			var res symbol.Symbol
			Expect(
				svc.NewRetrieve().
					Where(symbol.MatchKeys(sym.Key)).
					Entry(&res).
					Exec(ctx, tx),
			).To(Succeed())
			Expect(res.Data.SVG).To(Equal(originalData.SVG))
		})
	})

	Describe("Delete", func() {
		It("Should delete a single Symbol", func(ctx SpecContext) {
			sym := defaults.Fill(symbol.Symbol{
				Name: "to-delete",
				Data: symbol.Spec{SVG: "<svg>...</svg>", Variant: "valve"},
			})
			Expect(svc.NewWriter(tx).Create(ctx, &sym)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, sym.Key)).To(Succeed())

			var res symbol.Symbol
			Expect(svc.NewRetrieve().
				Where(symbol.MatchKeys(sym.Key)).
				Entry(&res).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})

		It("Should delete multiple Symbols", func(ctx SpecContext) {
			sym1 := defaults.Fill(symbol.Symbol{
				Name: "to-delete-1",
				Data: symbol.Spec{SVG: "<svg>1</svg>", Variant: "valve"},
			})
			sym2 := defaults.Fill(symbol.Symbol{
				Name: "to-delete-2",
				Data: symbol.Spec{SVG: "<svg>2</svg>", Variant: "valve"},
			})
			sym3 := defaults.Fill(symbol.Symbol{
				Name: "to-keep",
				Data: symbol.Spec{SVG: "<svg>3</svg>", Variant: "valve"},
			})

			Expect(
				svc.NewWriter(tx).Create(ctx, &sym1),
			).To(Succeed())
			Expect(
				svc.NewWriter(tx).Create(ctx, &sym2),
			).To(Succeed())
			Expect(
				svc.NewWriter(tx).Create(ctx, &sym3),
			).To(Succeed())

			Expect(svc.NewWriter(tx).Delete(ctx, sym1.Key, sym2.Key)).To(Succeed())

			var res []symbol.Symbol
			Expect(svc.NewRetrieve().
				Where(symbol.MatchKeys(sym1.Key, sym2.Key, sym3.Key)).
				Entries(&res).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
			Expect(res).To(HaveLen(1))
			Expect(res[0].Key).To(Equal(sym3.Key))
		})

		It("Should leave a deleted symbol out of its resources", func(ctx SpecContext) {
			sym := defaults.Fill(symbol.Symbol{
				Name: "resource-delete-test",
				Data: symbol.Spec{SVG: "<svg>...</svg>", Variant: "valve"},
			})
			Expect(svc.NewWriter(tx).Create(ctx, &sym)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, sym.Key)).To(Succeed())
			Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{sym.ResourceID()})).
				To(BeEmpty())
		})
	})
})
