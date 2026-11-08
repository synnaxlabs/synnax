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
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	"github.com/synnaxlabs/x/defaults"
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Resources", func() {
	It("Should name each symbol by its name", func(ctx SpecContext) {
		sym := createSymbol(ctx, createGroup(ctx, nil))
		Expect(svc.Resources().Retrieve(ctx, db, []resource.ID{sym.ResourceID()})).
			To(Equal([]resource.Resource{{ID: sym.ResourceID(), Name: sym.Name}}))
	})

	It("Should return a validation error for a malformed key", func(ctx SpecContext) {
		Expect(svc.Resources().Retrieve(
			ctx,
			db,
			[]resource.ID{{Type: resource.TypeSchematicSymbol, Key: "pump"}},
		)).Error().To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`invalid schematic_symbol key "pump"`)),
		))
	})

	It("Should leave out an ID that names no symbol", func(ctx SpecContext) {
		sym := createSymbol(ctx, createGroup(ctx, nil))
		missing := resource.ID{
			Type: resource.TypeSchematicSymbol,
			Key:  uuid.New().String(),
		}
		Expect(svc.Resources().Retrieve(
			ctx,
			db,
			[]resource.ID{missing, sym.ResourceID()},
		)).To(Equal([]resource.Resource{{ID: sym.ResourceID(), Name: sym.Name}}))
	})

	It("Should list each symbol by its name", func(ctx SpecContext) {
		sym := createSymbol(ctx, createGroup(ctx, nil))
		Expect(svc.Resources().List(ctx, db, 10000, nil)).To(ContainElement(
			resource.Resource{ID: sym.ResourceID(), Name: sym.Name},
		))
	})
})

var _ = Describe("Search", func() {
	var tx gorp.Tx
	BeforeEach(func() { tx = DeferClose(db.OpenTx()) })
	create := func(ctx SpecContext, name string) resource.ID {
		GinkgoHelper()
		s := defaults.Fill(symbol.Symbol{
			Name: name,
			Data: symbol.Spec{SVG: "<svg></svg>", Variant: "valve"},
		})
		Expect(svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
		return symbol.ResourceID(s.Key)
	}

	It(
		"Should return the matching symbols, closest match first",
		func(ctx SpecContext) {
			a := create(ctx, "turbopump test")
			b := create(ctx, "turbopump")
			q := search.Query[resource.ID]{Term: "turbopump"}
			Expect(MustSucceed(svc.Resources().Search(ctx, tx, q)).Keys).
				To(Equal([]resource.ID{b, a}))
			q.Limit = 1
			Expect(MustSucceed(svc.Resources().Search(ctx, tx, q)).Keys).
				To(Equal([]resource.ID{b}))
		},
	)

	It(
		"Should not match a staged symbol outside its transaction",
		func(ctx SpecContext) {
			create(ctx, "injector")
			q := search.Query[resource.ID]{Term: "injector"}
			Expect(MustSucceed(svc.Resources().Search(ctx, db, q)).Keys).To(BeEmpty())
		},
	)
})
