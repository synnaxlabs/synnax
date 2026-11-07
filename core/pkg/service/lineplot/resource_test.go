// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lineplot_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Resources", func() {
	It("Should name each line plot by its name", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		lp := lineplot.LinePlot{Project: &proj.Key, Name: "resource-lineplot"}
		Expect(svc.NewWriter(tx).Create(ctx, &lp)).To(Succeed())
		Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{lp.ResourceID()})).
			To(Equal([]resource.Resource{{ID: lp.ResourceID(), Name: lp.Name}}))
	})

	It("Should return a validation error for a malformed key", func(ctx SpecContext) {
		Expect(svc.Resources().Retrieve(
			ctx,
			nil,
			[]resource.ID{{Type: resource.TypeLineplot, Key: "pump"}},
		)).Error().To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`invalid lineplot key "pump"`)),
		))
	})

	It("Should leave out an ID that names no line plot", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		lp := lineplot.LinePlot{Project: &proj.Key, Name: "listed-lineplot"}
		Expect(svc.NewWriter(tx).Create(ctx, &lp)).To(Succeed())
		missing := resource.ID{Type: resource.TypeLineplot, Key: uuid.New().String()}
		Expect(svc.Resources().Retrieve(
			ctx,
			tx,
			[]resource.ID{missing, lp.ResourceID()},
		)).To(Equal([]resource.Resource{{ID: lp.ResourceID(), Name: lp.Name}}))
	})

	It("Should list each line plot by its name", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		lp := lineplot.LinePlot{Project: &proj.Key, Name: "listed-lineplot"}
		Expect(svc.NewWriter(tx).Create(ctx, &lp)).To(Succeed())
		Expect(svc.Resources().List(ctx, tx, 10000, nil)).To(ContainElement(
			resource.Resource{ID: lp.ResourceID(), Name: lp.Name},
		))
	})
})
