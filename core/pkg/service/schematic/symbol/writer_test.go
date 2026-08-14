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
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	"github.com/synnaxlabs/x/query"
)

var _ = Describe("Writer", func() {
	Describe("Create", func() {
		It("Should create a Symbol", func(ctx SpecContext) {
			sym := symbol.Symbol{
				Name: "test-symbol",
				Data: symbol.Spec{
					SVG:     "<svg>...</svg>",
					Variant: "valve",
					States: []symbol.State{
						{Key: "default", Name: "default"},
						{Key: "active", Name: "active"},
					},
				},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &sym, proj.OntologyID())).To(Succeed())
			Expect(sym.Key).ToNot(Equal(uuid.Nil))
		})

		DescribeTable("Should reject a symbol that fails schema validation",
			func(ctx SpecContext, sym symbol.Symbol, msg string) {
				Expect(svc.NewWriter(tx).Create(ctx, &sym, proj.OntologyID())).
					To(MatchError(ContainSubstring(msg)))
			},
			Entry("missing name", symbol.Symbol{
				Data: symbol.Spec{SVG: "<svg/>", Variant: "valve"},
			}, "name: required"),
			Entry("missing svg", symbol.Symbol{
				Name: "no-svg",
				Data: symbol.Spec{Variant: "valve"},
			}, "svg: required"),
			Entry("missing variant", symbol.Symbol{
				Name: "no-variant",
				Data: symbol.Spec{SVG: "<svg/>"},
			}, "variant: required"),
			Entry("invalid handle orientation", symbol.Symbol{
				Name: "bad-handle",
				Data: symbol.Spec{
					SVG:     "<svg/>",
					Variant: "valve",
					Handles: []symbol.Handle{{Key: "h1", Orientation: "diagonal"}},
				},
			}, "invalid orientation"),
		)

		It("Should apply schema defaults for omitted fields", func(ctx SpecContext) {
			sym := symbol.Symbol{
				Name: "defaults",
				Data: symbol.Spec{SVG: "<svg/>", Variant: "valve"},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &sym, proj.OntologyID())).To(Succeed())
			// Defaults are stamped on the caller's struct.
			Expect(sym.Data.Scale).To(Equal(1.0))

			var res symbol.Symbol
			Expect(svc.NewRetrieve().
				Where(symbol.MatchKeys(sym.Key)).
				Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res.Data.Scale).To(Equal(1.0))
		})

		It("Should create a Symbol with a predefined key", func(ctx SpecContext) {
			key := uuid.New()
			sym := symbol.Symbol{
				Key:  key,
				Name: "predefined-key-symbol",
				Data: symbol.Spec{SVG: "<svg>...</svg>", Variant: "valve"},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &sym, proj.OntologyID())).To(Succeed())
			Expect(sym.Key).To(Equal(key))
		})

		It(
			"Should update an existing Symbol if key already exists",
			func(ctx SpecContext) {
				key := uuid.New()
				sym1 := symbol.Symbol{
					Key:  key,
					Name: "original-name",
					Data: symbol.Spec{SVG: "<svg>original</svg>", Variant: "valve"},
				}
				Expect(
					svc.NewWriter(tx).Create(ctx, &sym1, proj.OntologyID()),
				).To(Succeed())

				sym2 := symbol.Symbol{
					Key:  key,
					Name: "updated-name",
					Data: symbol.Spec{SVG: "<svg>updated</svg>", Variant: "valve"},
				}
				Expect(
					svc.NewWriter(tx).Create(ctx, &sym2, proj.OntologyID()),
				).To(Succeed())

				var retrieved symbol.Symbol
				Expect(
					svc.NewRetrieve().
						Where(symbol.MatchKeys(key)).
						Entry(&retrieved).
						Exec(ctx, tx),
				).To(Succeed())
				Expect(retrieved.Name).To(Equal("updated-name"))
				Expect(retrieved.Data.SVG).To(Equal("<svg>updated</svg>"))
			},
		)

		It("Should properly set ontology relationships", func(ctx SpecContext) {
			sym := symbol.Symbol{
				Name: "ontology-test",
				Data: symbol.Spec{SVG: "<svg>...</svg>", Variant: "valve"},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &sym, proj.OntologyID())).To(Succeed())

			var res []ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(proj.OntologyID()).
				TraverseTo(ontology.ChildrenTraverser).
				Entries(&res).
				Exec(ctx, tx)).To(Succeed())

			keys := lo.Map(
				res,
				func(r ontology.Resource, _ int) string { return r.ID.Key },
			)
			Expect(keys).To(ContainElement(sym.Key.String()))
		})

		It(
			"Should create a Symbol under the permanent symbols group if provided",
			func(ctx SpecContext) {
				sym := symbol.Symbol{
					Name: "group-test",
					Data: symbol.Spec{SVG: "<svg>...</svg>", Variant: "valve"},
				}
				groupOntologyID := ontology.ID{
					Type: "group",
					Key:  svc.Group().Key.String(),
				}
				Expect(
					svc.NewWriter(tx).Create(ctx, &sym, groupOntologyID),
				).To(Succeed())

				var res []ontology.Resource
				Expect(otg.NewRetrieve().
					WhereIDs(groupOntologyID).
					TraverseTo(ontology.ChildrenTraverser).
					Entries(&res).
					Exec(ctx, tx)).To(Succeed())

				keys := lo.Map(
					res,
					func(r ontology.Resource, _ int) string { return r.ID.Key },
				)
				Expect(keys).To(ContainElement(sym.Key.String()))
			},
		)
	})

	Describe("CreateMany", func() {
		It("Should create multiple Symbols", func(ctx SpecContext) {
			symbols := []symbol.Symbol{
				{
					Name: "symbol-1",
					Data: symbol.Spec{SVG: "<svg>1</svg>", Variant: "valve"},
				},
				{
					Name: "symbol-2",
					Data: symbol.Spec{SVG: "<svg>2</svg>", Variant: "valve"},
				},
			}
			Expect(
				svc.NewWriter(tx).CreateMany(ctx, &symbols, proj.OntologyID()),
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
			sym := symbol.Symbol{
				Name: "original-name",
				Data: symbol.Spec{SVG: "<svg>...</svg>", Variant: "valve"},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &sym, proj.OntologyID())).To(Succeed())
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
			sym := symbol.Symbol{
				Name: "data-preservation-test",
				Data: originalData,
			}
			Expect(svc.NewWriter(tx).Create(ctx, &sym, proj.OntologyID())).To(Succeed())
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
			sym := symbol.Symbol{
				Name: "to-delete",
				Data: symbol.Spec{SVG: "<svg>...</svg>", Variant: "valve"},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &sym, proj.OntologyID())).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, sym.Key)).To(Succeed())

			var res symbol.Symbol
			err := svc.NewRetrieve().
				Where(symbol.MatchKeys(sym.Key)).
				Entry(&res).
				Exec(ctx, tx)
			Expect(err).To(MatchError(query.ErrNotFound))
		})

		It("Should delete multiple Symbols", func(ctx SpecContext) {
			sym1 := symbol.Symbol{
				Name: "to-delete-1",
				Data: symbol.Spec{SVG: "<svg>1</svg>", Variant: "valve"},
			}
			sym2 := symbol.Symbol{
				Name: "to-delete-2",
				Data: symbol.Spec{SVG: "<svg>2</svg>", Variant: "valve"},
			}
			sym3 := symbol.Symbol{
				Name: "to-keep",
				Data: symbol.Spec{SVG: "<svg>3</svg>", Variant: "valve"},
			}

			Expect(
				svc.NewWriter(tx).Create(ctx, &sym1, proj.OntologyID()),
			).To(Succeed())
			Expect(
				svc.NewWriter(tx).Create(ctx, &sym2, proj.OntologyID()),
			).To(Succeed())
			Expect(
				svc.NewWriter(tx).Create(ctx, &sym3, proj.OntologyID()),
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

		It("Should remove ontology relationships when deleting", func(ctx SpecContext) {
			sym := symbol.Symbol{
				Name: "ontology-delete-test",
				Data: symbol.Spec{SVG: "<svg>...</svg>", Variant: "valve"},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &sym, proj.OntologyID())).To(Succeed())

			// Verify it exists in ontology
			var resBefore []ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(proj.OntologyID()).
				TraverseTo(ontology.ChildrenTraverser).
				Entries(&resBefore).
				Exec(ctx, tx)).To(Succeed())
			keysBefore := lo.Map(
				resBefore,
				func(r ontology.Resource, _ int) string { return r.ID.Key },
			)
			Expect(keysBefore).To(ContainElement(sym.Key.String()))

			// Delete the symbol
			Expect(svc.NewWriter(tx).Delete(ctx, sym.Key)).To(Succeed())

			// Verify it's removed from ontology
			var resAfter []ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(proj.OntologyID()).
				TraverseTo(ontology.ChildrenTraverser).
				Entries(&resAfter).
				Exec(ctx, tx)).To(Succeed())
			keysAfter := lo.Map(
				resAfter,
				func(r ontology.Resource, _ int) string { return r.ID.Key },
			)
			Expect(keysAfter).ToNot(ContainElement(sym.Key.String()))
		})
	})
})
