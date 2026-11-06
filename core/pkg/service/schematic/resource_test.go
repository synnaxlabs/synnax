// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schematic_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Resources", func() {
	It("Should name each schematic by its name", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		s := schematic.Schematic{Project: &proj.Key, Name: "resource-schematic"}
		Expect(svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
		Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{s.ResourceID()})).
			To(Equal([]resource.Resource{{ID: s.ResourceID(), Name: s.Name}}))
	})

	It("Should return a validation error for a malformed key", func(ctx SpecContext) {
		Expect(svc.Resources().Retrieve(
			ctx,
			nil,
			[]resource.ID{{Type: resource.TypeSchematic, Key: "pump"}},
		)).Error().To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`invalid schematic key "pump"`)),
		))
	})
})
