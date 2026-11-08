// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Resources", func() {
	It("Should name each Arc by its name", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		a := newAutomation(arc.Base{
			Name:   "resource-arc",
			Source: arc.Source{Variant: arc.TextSource{}},
		})
		Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
		Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{a.ResourceID()})).
			To(Equal([]resource.Resource{{ID: a.ResourceID(), Name: a.Base().Name}}))
	})

	It("Should name each Arc directory by its name", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		d := directory.Directory{Name: "resource_dir"}
		Expect(svc.NewWriter(tx).CreateDirectory(ctx, &d, rootDirectory(ctx, tx).Key)).
			To(Succeed())
		Expect(svc.DirectoryResources().Retrieve(
			ctx,
			tx,
			[]resource.ID{d.ResourceID()},
		)).To(Equal([]resource.Resource{{ID: d.ResourceID(), Name: d.Name}}))
	})

	It("Should return a validation error for a malformed key", func(ctx SpecContext) {
		Expect(svc.Resources().Retrieve(
			ctx,
			db,
			[]resource.ID{{Type: resource.TypeArc, Key: "pump"}},
		)).Error().To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`invalid arc key "pump"`)),
		))
	})

	It("Should leave out an ID that names no Arc", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		a := newAutomation(arc.Base{
			Name:   "listed-arc",
			Source: arc.Source{Variant: arc.TextSource{}},
		})
		Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
		missing := resource.ID{Type: resource.TypeArc, Key: uuid.New().String()}
		Expect(svc.Resources().Retrieve(
			ctx,
			tx,
			[]resource.ID{missing, a.ResourceID()},
		)).To(Equal([]resource.Resource{{ID: a.ResourceID(), Name: a.Base().Name}}))
	})

	It("Should list each Arc by its name", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		a := newAutomation(arc.Base{
			Name:   "listed-arc",
			Source: arc.Source{Variant: arc.TextSource{}},
		})
		Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
		Expect(svc.Resources().List(ctx, tx, 10000, nil)).To(ContainElement(
			resource.Resource{ID: a.ResourceID(), Name: a.Base().Name},
		))
	})
})

var _ = Describe("Search", func() {
	var tx gorp.Tx
	BeforeEach(func() { tx = DeferClose(db.OpenTx()) })
	create := func(ctx SpecContext, name string) resource.ID {
		GinkgoHelper()
		a := newAutomation(
			arc.Base{Name: name, Source: arc.Source{Variant: arc.TextSource{}}},
		)
		Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
		return a.ResourceID()
	}

	It("Should return the matching Arcs, closest match first", func(ctx SpecContext) {
		a := create(ctx, "turbopump test")
		b := create(ctx, "turbopump")
		q := search.Query[resource.ID]{Term: "turbopump"}
		Expect(MustSucceed(svc.Resources().Search(ctx, tx, q)).Keys).
			To(Equal([]resource.ID{b, a}))
		q.Limit = 1
		Expect(MustSucceed(svc.Resources().Search(ctx, tx, q)).Keys).
			To(Equal([]resource.ID{b}))
	})

	It("Should not match a staged Arc outside its transaction", func(ctx SpecContext) {
		create(ctx, "injector")
		q := search.Query[resource.ID]{Term: "injector"}
		Expect(MustSucceed(svc.Resources().Search(ctx, db, q)).Keys).To(BeEmpty())
	})
})
