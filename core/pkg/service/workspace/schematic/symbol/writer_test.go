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
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/schematic/symbol"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Writer", func() {
	Describe("Create", func() {
		It("Should create a Symbol", func() {
			sym := symbol.Symbol{
				Name: "test-symbol",
				Data: map[string]any{
					"svg":     "<svg>...</svg>",
					"states":  []string{"default", "active"},
					"regions": []map[string]any{},
				},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &sym, ws.OntologyID())).To(Succeed())
			Expect(sym.Key).ToNot(Equal(uuid.Nil))
		})

		It("Should create a Symbol with a predefined key", func() {
			key := uuid.New()
			sym := symbol.Symbol{
				Key:  key,
				Name: "predefined-key-symbol",
				Data: map[string]any{
					"svg": "<svg>...</svg>",
				},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &sym, ws.OntologyID())).To(Succeed())
			Expect(sym.Key).To(Equal(key))
		})

		It("Should update an existing Symbol if key already exists", func() {
			key := uuid.New()
			sym1 := symbol.Symbol{
				Key:  key,
				Name: "original-name",
				Data: map[string]any{
					"svg": "<svg>original</svg>",
				},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &sym1, ws.OntologyID())).To(Succeed())

			sym2 := symbol.Symbol{
				Key:  key,
				Name: "updated-name",
				Data: map[string]any{
					"svg": "<svg>updated</svg>",
				},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &sym2, ws.OntologyID())).To(Succeed())

			var retrieved symbol.Symbol
			Expect(svc.NewRetrieve().WhereKeys(key).Entry(&retrieved).Exec(ctx, tx)).To(Succeed())
			Expect(retrieved.Name).To(Equal("updated-name"))
			Expect(retrieved.Data["svg"]).To(Equal("<svg>updated</svg>"))
		})

		It("Should properly set ontology relationships", func() {
			sym := symbol.Symbol{
				Name: "ontology-test",
				Data: map[string]any{
					"svg": "<svg>...</svg>",
				},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &sym, ws.OntologyID())).To(Succeed())

			var res []ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(ws.OntologyID()).
				TraverseTo(ontology.ChildrenTraverser).
				Entries(&res).
				Exec(ctx, tx)).To(Succeed())

			keys := lo.Map(res, func(r ontology.Resource, _ int) string { return r.ID.Key })
			Expect(keys).To(ContainElement(sym.Key.String()))
		})

		It("Should create a Symbol under the permanent symbols group if provided", func() {
			sym := symbol.Symbol{
				Name: "group-test",
				Data: map[string]any{
					"svg": "<svg>...</svg>",
				},
			}
			groupOntologyID := ontology.ID{Type: "group", Key: svc.Group().Key.String()}
			Expect(svc.NewWriter(tx).Create(ctx, &sym, groupOntologyID)).To(Succeed())

			var res []ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(groupOntologyID).
				TraverseTo(ontology.ChildrenTraverser).
				Entries(&res).
				Exec(ctx, tx)).To(Succeed())

			keys := lo.Map(res, func(r ontology.Resource, _ int) string { return r.ID.Key })
			Expect(keys).To(ContainElement(sym.Key.String()))
		})
	})

	Describe("Rename", func() {
		It("Should rename a Symbol", func() {
			sym := symbol.Symbol{
				Name: "original-name",
				Data: map[string]any{
					"svg": "<svg>...</svg>",
				},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &sym, ws.OntologyID())).To(Succeed())
			Expect(svc.NewWriter(tx).Rename(ctx, sym.Key, "new-name")).To(Succeed())

			var res symbol.Symbol
			Expect(gorp.NewRetrieve[uuid.UUID, symbol.Symbol]().
				WhereKeys(sym.Key).
				Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res.Name).To(Equal("new-name"))
		})

		It("Should not affect data when renaming", func() {
			originalData := map[string]any{
				"svg":     "<svg>complex</svg>",
				"states":  []string{"default", "active", "error"},
				"regions": []map[string]any{{"id": "region1"}},
			}
			sym := symbol.Symbol{
				Name: "data-preservation-test",
				Data: originalData,
			}
			Expect(svc.NewWriter(tx).Create(ctx, &sym, ws.OntologyID())).To(Succeed())
			Expect(svc.NewWriter(tx).Rename(ctx, sym.Key, "renamed")).To(Succeed())

			var res symbol.Symbol
			Expect(svc.NewRetrieve().WhereKeys(sym.Key).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Data["svg"]).To(Equal(originalData["svg"]))
		})
	})

	Describe("Delete", func() {
		It("Should delete a single Symbol", func() {
			sym := symbol.Symbol{
				Name: "to-delete",
				Data: map[string]any{
					"svg": "<svg>...</svg>",
				},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &sym, ws.OntologyID())).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, sym.Key)).To(Succeed())

			var res symbol.Symbol
			err := svc.NewRetrieve().WhereKeys(sym.Key).Entry(&res).Exec(ctx, tx)
			Expect(err).To(MatchError(query.ErrNotFound))
		})

		It("Should delete multiple Symbols", func() {
			sym1 := symbol.Symbol{
				Name: "to-delete-1",
				Data: map[string]any{"svg": "<svg>1</svg>"},
			}
			sym2 := symbol.Symbol{
				Name: "to-delete-2",
				Data: map[string]any{"svg": "<svg>2</svg>"},
			}
			sym3 := symbol.Symbol{
				Name: "to-keep",
				Data: map[string]any{"svg": "<svg>3</svg>"},
			}

			Expect(svc.NewWriter(tx).Create(ctx, &sym1, ws.OntologyID())).To(Succeed())
			Expect(svc.NewWriter(tx).Create(ctx, &sym2, ws.OntologyID())).To(Succeed())
			Expect(svc.NewWriter(tx).Create(ctx, &sym3, ws.OntologyID())).To(Succeed())

			Expect(svc.NewWriter(tx).Delete(ctx, sym1.Key, sym2.Key)).To(Succeed())

			var res []symbol.Symbol
			Expect(svc.NewRetrieve().
				WhereKeys(sym1.Key, sym2.Key, sym3.Key).
				Entries(&res).
				Exec(ctx, tx)).To(HaveOccurredAs(query.ErrNotFound))
			Expect(res).To(HaveLen(1))
			Expect(res[0].Key).To(Equal(sym3.Key))
		})

		It("Should remove ontology relationships when deleting", func() {
			sym := symbol.Symbol{
				Name: "ontology-delete-test",
				Data: map[string]any{"svg": "<svg>...</svg>"},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &sym, ws.OntologyID())).To(Succeed())

			// Verify it exists in ontology
			var resBefore []ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(ws.OntologyID()).
				TraverseTo(ontology.ChildrenTraverser).
				Entries(&resBefore).
				Exec(ctx, tx)).To(Succeed())
			keysBefore := lo.Map(resBefore, func(r ontology.Resource, _ int) string { return r.ID.Key })
			Expect(keysBefore).To(ContainElement(sym.Key.String()))

			// Delete the symbol
			Expect(svc.NewWriter(tx).Delete(ctx, sym.Key)).To(Succeed())

			// Verify it's removed from ontology
			var resAfter []ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(ws.OntologyID()).
				TraverseTo(ontology.ChildrenTraverser).
				Entries(&resAfter).
				Exec(ctx, tx)).To(Succeed())
			keysAfter := lo.Map(resAfter, func(r ontology.Resource, _ int) string { return r.ID.Key })
			Expect(keysAfter).ToNot(ContainElement(sym.Key.String()))
		})
	})
})
