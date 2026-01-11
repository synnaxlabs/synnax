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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/schematic/symbol"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/uuid"
)

var _ = Describe("Retrieve", func() {
	var (
		sym1, sym2, sym3 symbol.Symbol
	)

	BeforeEach(func() {
		sym1 = symbol.Symbol{
			Name: "symbol-1",
			Data: map[string]any{
				"svg":    "<svg>1</svg>",
				"states": []string{"default"},
			},
		}
		sym2 = symbol.Symbol{
			Name: "symbol-2",
			Data: map[string]any{
				"svg":    "<svg>2</svg>",
				"states": []string{"default", "active"},
			},
		}
		sym3 = symbol.Symbol{
			Name: "symbol-3",
			Data: map[string]any{
				"svg":    "<svg>3</svg>",
				"states": []string{"default", "active", "error"},
			},
		}
		Expect(svc.NewWriter(tx).Create(ctx, &sym1, ws.OntologyID())).To(Succeed())
		Expect(svc.NewWriter(tx).Create(ctx, &sym2, ws.OntologyID())).To(Succeed())
		Expect(svc.NewWriter(tx).Create(ctx, &sym3, ws.OntologyID())).To(Succeed())
	})

	Describe("WhereKeys", func() {
		It("Should retrieve a single symbol by key", func() {
			var retrieved symbol.Symbol
			Expect(svc.NewRetrieve().
				WhereKeys(sym1.Key).
				Entry(&retrieved).
				Exec(ctx, tx)).To(Succeed())
			Expect(retrieved.Key).To(Equal(sym1.Key))
			Expect(retrieved.Name).To(Equal(sym1.Name))
			Expect(retrieved.Data["svg"]).To(Equal(sym1.Data["svg"]))
		})

		It("Should retrieve multiple symbols by keys", func() {
			var retrieved []symbol.Symbol
			Expect(svc.NewRetrieve().
				WhereKeys(sym1.Key, sym3.Key).
				Entries(&retrieved).
				Exec(ctx, tx)).To(Succeed())
			Expect(retrieved).To(HaveLen(2))

			keys := []uuid.UUID{retrieved[0].Key, retrieved[1].Key}
			Expect(keys).To(ContainElements(sym1.Key, sym3.Key))
		})

		It("Should return error when symbol not found", func() {
			var retrieved symbol.Symbol
			err := svc.NewRetrieve().
				WhereKeys(uuid.New()).
				Entry(&retrieved).
				Exec(ctx, tx)
			Expect(err).To(MatchError(query.NotFound))
		})
	})

	Describe("Exec", func() {
		It("Should execute with provided transaction", func() {
			var retrieved symbol.Symbol
			Expect(svc.NewRetrieve().
				WhereKeys(sym1.Key).
				Entry(&retrieved).
				Exec(ctx, tx)).To(Succeed())
			Expect(retrieved.Key).To(Equal(sym1.Key))
		})

		It("Should execute without transaction", func() {
			// Create a symbol without transaction
			symNoTx := symbol.Symbol{
				Name: "no-tx-symbol",
				Data: map[string]any{"svg": "<svg>no-tx</svg>"},
			}
			Expect(svc.NewWriter(nil).Create(ctx, &symNoTx, ws.OntologyID())).To(Succeed())

			var retrieved symbol.Symbol
			Expect(svc.NewRetrieve().
				WhereKeys(symNoTx.Key).
				Entry(&retrieved).
				Exec(ctx, nil)).To(Succeed())
			Expect(retrieved.Key).To(Equal(symNoTx.Key))

			// Clean up
			Expect(svc.NewWriter(nil).Delete(ctx, symNoTx.Key)).To(Succeed())
		})
	})

	Describe("Complex Queries", func() {
		It("Should retrieve all symbols created", func() {
			var allSymbols []symbol.Symbol
			Expect(svc.NewRetrieve().
				Entries(&allSymbols).
				Exec(ctx, tx)).To(Succeed())
			Expect(allSymbols).To(HaveLen(3))

			names := []string{}
			for _, s := range allSymbols {
				names = append(names, s.Name)
			}
			Expect(names).To(ContainElements("symbol-1", "symbol-2", "symbol-3"))
		})

		It("Should handle large data correctly", func() {
			largeData := map[string]any{
				"svg":     "<svg>" + string(make([]byte, 10000)) + "</svg>",
				"states":  []string{},
				"regions": []map[string]any{},
			}
			for i := 0; i < 100; i++ {
				largeData["states"] = append(largeData["states"].([]string), "state"+string(rune(i)))
				largeData["regions"] = append(largeData["regions"].([]map[string]any), map[string]any{
					"id":   "region" + string(rune(i)),
					"type": "input",
				})
			}

			largeSym := symbol.Symbol{
				Name: "large-symbol",
				Data: largeData,
			}
			Expect(svc.NewWriter(tx).Create(ctx, &largeSym, ws.OntologyID())).To(Succeed())

			var retrieved symbol.Symbol
			Expect(svc.NewRetrieve().
				WhereKeys(largeSym.Key).
				Entry(&retrieved).
				Exec(ctx, tx)).To(Succeed())
			Expect(retrieved.Data["svg"]).To(Equal(largeData["svg"]))
		})
	})
})
