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
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Resources", func() {
	It("Should name each table by its name", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		t := table.Table{Project: &proj.Key, Name: "resource-table"}
		Expect(svc.NewWriter(tx).Create(ctx, &t)).To(Succeed())
		Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{t.ResourceID()})).
			To(Equal([]resource.Resource{{ID: t.ResourceID(), Name: t.Name}}))
	})

	It("Should return a validation error for a malformed key", func(ctx SpecContext) {
		Expect(svc.Resources().Retrieve(
			ctx,
			nil,
			[]resource.ID{{Type: resource.TypeTable, Key: "pump"}},
		)).Error().To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`invalid table key "pump"`)),
		))
	})
})
