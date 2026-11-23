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
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	"github.com/synnaxlabs/x/encoding/orc"
	. "github.com/synnaxlabs/x/testutil"
)

// drawer is a record that draws a symbol, as a schematic does. Its table takes the name
// of the schematic table, which the suite does not open.
type drawer struct {
	Key    string
	Symbol symbol.Key
}

func (d drawer) GorpKey() string { return d.Key }

func (d drawer) EncodeOrc(w *orc.Writer) error {
	w.String(d.Key)
	w.Write(d.Symbol[:])
	return nil
}

func (d *drawer) DecodeOrc(r *orc.Reader) (err error) {
	if d.Key, err = r.String(); err != nil {
		return err
	}
	_, err = r.Read(d.Symbol[:])
	return err
}

var _ = Describe("Uses", func() {
	var (
		drawers *gorp.Table[string, drawer]
		sym     symbol.Symbol
		tx      gorp.Tx
	)
	BeforeEach(func(ctx SpecContext) {
		sym = createSymbol(ctx, createGroup(ctx, nil))
		drawers = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, drawer]{
			DB:   db,
			Name: string(resource.TypeSchematic),
			References: []gorp.Reference[string, drawer]{gorp.NewKeyReference(
				gorp.KeyReferenceConfig[string, drawer, symbol.Key]{
					Name:     "symbol",
					Table:    string(resource.TypeSchematicSymbol),
					OnDelete: gorp.Restrict,
					Kind:     gorp.Use,
					Targets: func(d *drawer) []symbol.Key {
						return []symbol.Key{d.Symbol}
					},
				},
			)},
		}))
		tx = DeferClose(db.OpenTx())
	})

	It("Should return no records for a symbol that nothing draws", func(
		ctx SpecContext,
	) {
		Expect(svc.Uses(ctx, tx, sym.Key)).To(BeEmpty())
	})

	It("Should return each record that draws the symbol", func(ctx SpecContext) {
		Expect(drawers.NewCreate().Entries(&[]drawer{
			{Key: "a", Symbol: sym.Key},
			{Key: "b", Symbol: sym.Key},
		}).Exec(ctx, tx)).To(Succeed())
		Expect(svc.Uses(ctx, tx, sym.Key)).To(ConsistOf(
			resource.ID{Type: resource.TypeSchematic, Key: "a"},
			resource.ID{Type: resource.TypeSchematic, Key: "b"},
		))
	})
})
