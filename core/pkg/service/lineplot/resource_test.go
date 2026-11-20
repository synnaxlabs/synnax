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
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/defaults"
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Resources", func() {
	It("Should name each line plot by its name", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		lp := defaults.Fill(
			lineplot.LinePlot{Project: &proj.Key, Name: "resource-lineplot"},
		)
		Expect(svc.NewWriter(tx).Create(ctx, &lp)).To(Succeed())
		Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{lp.ResourceID()})).
			To(Equal([]resource.Resource{{ID: lp.ResourceID(), Name: lp.Name}}))
	})

	It("Should return a validation error for a malformed key", func(ctx SpecContext) {
		Expect(svc.Resources().Retrieve(
			ctx,
			db,
			[]resource.ID{{Type: resource.TypeLineplot, Key: "pump"}},
		)).Error().To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`invalid lineplot key "pump"`)),
		))
	})

	It("Should leave out an ID that names no line plot", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		lp := defaults.Fill(
			lineplot.LinePlot{Project: &proj.Key, Name: "listed-lineplot"},
		)
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
		lp := defaults.Fill(
			lineplot.LinePlot{Project: &proj.Key, Name: "listed-lineplot"},
		)
		Expect(svc.NewWriter(tx).Create(ctx, &lp)).To(Succeed())
		Expect(svc.Resources().List(ctx, tx, 10000, nil)).To(ContainElement(
			resource.Resource{ID: lp.ResourceID(), Name: lp.Name},
		))
	})
})

var _ = Describe("Search", func() {
	var tx gorp.Tx
	BeforeEach(func() { tx = DeferClose(db.OpenTx()) })
	create := func(ctx SpecContext, name string) resource.ID {
		GinkgoHelper()
		p := defaults.Fill(lineplot.LinePlot{Project: &proj.Key, Name: name})
		Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
		return lineplot.ResourceID(p.Key)
	}

	It(
		"Should return the matching line plots, closest match first",
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
		"Should not match a staged line plot outside its transaction",
		func(ctx SpecContext) {
			create(ctx, "injector")
			q := search.Query[resource.ID]{Term: "injector"}
			Expect(MustSucceed(svc.Resources().Search(ctx, db, q)).Keys).To(BeEmpty())
		},
	)
})
