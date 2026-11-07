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
	"github.com/synnaxlabs/synnax/pkg/service/resource"
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
			nil,
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
