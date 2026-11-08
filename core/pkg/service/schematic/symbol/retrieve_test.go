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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	"github.com/synnaxlabs/x/defaults"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Retrieve", func() {
	var (
		tx               gorp.Tx
		sym1, sym2, sym3 symbol.Symbol
	)

	createSymbols := func(ctx SpecContext) {
		tx = DeferClose(db.OpenTx())
		sym1 = defaults.Fill(symbol.Symbol{
			Name: "symbol-1",
			Data: symbol.Spec{
				SVG:     "<svg>1</svg>",
				Variant: "valve",
				States:  []symbol.State{{Key: "default", Name: "default"}},
			},
		})
		sym2 = defaults.Fill(symbol.Symbol{
			Name: "symbol-2",
			Data: symbol.Spec{
				SVG:     "<svg>2</svg>",
				Variant: "valve",
				States: []symbol.State{
					{Key: "default", Name: "default"},
					{Key: "active", Name: "active"},
				},
			},
		})
		sym3 = defaults.Fill(symbol.Symbol{
			Name: "symbol-3",
			Data: symbol.Spec{
				SVG:     "<svg>3</svg>",
				Variant: "valve",
				States: []symbol.State{
					{Key: "default", Name: "default"},
					{Key: "active", Name: "active"},
					{Key: "error", Name: "error"},
				},
			},
		})
		Expect(svc.NewWriter(tx).Create(ctx, &sym1)).To(Succeed())
		Expect(svc.NewWriter(tx).Create(ctx, &sym2)).To(Succeed())
		Expect(svc.NewWriter(tx).Create(ctx, &sym3)).To(Succeed())
	}

	Describe("WhereKeys", func() {
		BeforeEach(createSymbols)
		It("Should retrieve a single symbol by key", func(ctx SpecContext) {
			var retrieved symbol.Symbol
			Expect(svc.NewRetrieve().
				Where(symbol.MatchKeys(sym1.Key)).
				Entry(&retrieved).
				Exec(ctx, tx)).To(Succeed())
			Expect(retrieved.Key).To(Equal(sym1.Key))
			Expect(retrieved.Name).To(Equal(sym1.Name))
			Expect(retrieved.Data.SVG).To(Equal(sym1.Data.SVG))
		})

		It("Should retrieve multiple symbols by keys", func(ctx SpecContext) {
			var retrieved []symbol.Symbol
			Expect(svc.NewRetrieve().
				Where(symbol.MatchKeys(sym1.Key, sym3.Key)).
				Entries(&retrieved).
				Exec(ctx, tx)).To(Succeed())
			Expect(retrieved).To(HaveLen(2))

			keys := []uuid.UUID{retrieved[0].Key, retrieved[1].Key}
			Expect(keys).To(ContainElements(sym1.Key, sym3.Key))
		})

		It("Should return error when symbol not found", func(ctx SpecContext) {
			var retrieved symbol.Symbol
			Expect(svc.NewRetrieve().
				Where(symbol.MatchKeys(uuid.New())).
				Entry(&retrieved).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Exec", func() {
		It("Should execute with provided transaction", func(ctx SpecContext) {
			createSymbols(ctx)
			var retrieved symbol.Symbol
			Expect(svc.NewRetrieve().
				Where(symbol.MatchKeys(sym1.Key)).
				Entry(&retrieved).
				Exec(ctx, tx)).To(Succeed())
			Expect(retrieved.Key).To(Equal(sym1.Key))
		})

		It("Should execute without transaction", func(ctx SpecContext) {
			symNoTx := defaults.Fill(symbol.Symbol{
				Name: "no-tx-symbol",
				Data: symbol.Spec{SVG: "<svg>no-tx</svg>", Variant: "valve"},
			})
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &symNoTx)
			})).To(Succeed())

			var retrieved symbol.Symbol
			Expect(svc.NewRetrieve().
				Where(symbol.MatchKeys(symNoTx.Key)).
				Entry(&retrieved).
				Exec(ctx, db)).To(Succeed())
			Expect(retrieved.Key).To(Equal(symNoTx.Key))

			// Clean up
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, symNoTx.Key)
			})).To(Succeed())
		})
	})

	Describe("Complex Queries", func() {
		BeforeEach(createSymbols)
		It("Should retrieve all symbols created", func(ctx SpecContext) {
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

		It("Should handle large data correctly", func(ctx SpecContext) {
			largeData := symbol.Spec{
				SVG:     "<svg>" + string(make([]byte, 10000)) + "</svg>",
				Variant: "valve",
			}
			for i := range 100 {
				largeData.States = append(largeData.States, symbol.State{
					Key:  "state" + string(rune(i)),
					Name: "state" + string(rune(i)),
					Regions: []symbol.Region{
						{Key: "region" + string(rune(i)), Name: "input"},
					},
				})
			}

			largeSym := defaults.Fill(symbol.Symbol{
				Name: "large-symbol",
				Data: largeData,
			})
			Expect(
				svc.NewWriter(tx).Create(ctx, &largeSym),
			).To(Succeed())

			var retrieved symbol.Symbol
			Expect(svc.NewRetrieve().
				Where(symbol.MatchKeys(largeSym.Key)).
				Entry(&retrieved).
				Exec(ctx, tx)).To(Succeed())
			Expect(retrieved.Data.SVG).To(Equal(largeData.SVG))
		})
	})
})
