// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package color_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/color"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	xcolor "github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Resources", func() {
	var (
		db  *gorp.DB
		svc *color.Service
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		groups := MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
		svc = MustOpen(
			color.OpenService(ctx, color.ServiceConfig{DB: db, Group: groups}),
		)
	})
	create := func(ctx SpecContext, name string) resource.Resource {
		GinkgoHelper()
		n := color.Named{Name: name, Light: MustSucceed(xcolor.Parse("#ff0000"))}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &n)
		})).To(Succeed())
		return resource.Resource{ID: color.ResourceID(n.Key), Name: name}
	}

	It("Should name each named color by its name", func(ctx SpecContext) {
		r := create(ctx, "Breaker")
		Expect(svc.Resources().Retrieve(ctx, db, []resource.ID{r.ID})).
			To(Equal([]resource.Resource{r}))
	})

	It("Should return a validation error for a malformed key", func(ctx SpecContext) {
		Expect(svc.Resources().Retrieve(
			ctx,
			db,
			[]resource.ID{{Type: resource.TypeColor, Key: "red"}},
		)).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`invalid color key "red"`)),
		))
	})

	It("Should leave out an ID that names no named color", func(ctx SpecContext) {
		r := create(ctx, "Breaker")
		missing := color.ResourceID(uuid.New())
		Expect(svc.Resources().Retrieve(ctx, db, []resource.ID{missing, r.ID})).
			To(Equal([]resource.Resource{r}))
	})

	It("Should list each named color by its name", func(ctx SpecContext) {
		r := create(ctx, "Breaker")
		Expect(svc.Resources().List(ctx, db, 100, nil)).To(ContainElement(r))
	})

	It("Should search named colors by name, closest first", func(ctx SpecContext) {
		a := create(ctx, "Breaker closed")
		b := create(ctx, "Breaker")
		create(ctx, "Valve")
		q := search.Query[resource.ID]{Term: "breaker"}
		Expect(MustSucceed(svc.Resources().Search(ctx, db, q)).Keys).
			To(Equal([]resource.ID{b.ID, a.ID}))
		q.Limit = 1
		Expect(MustSucceed(svc.Resources().Search(ctx, db, q)).Keys).
			To(Equal([]resource.ID{b.ID}))
	})
})
