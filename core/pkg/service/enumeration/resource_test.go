// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package enumeration_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/enumeration"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Resources", func() {
	var (
		db  *gorp.DB
		svc *enumeration.Service
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		svc = MustOpen(enumeration.OpenService(ctx, enumeration.ServiceConfig{DB: db}))
	})
	create := func(ctx SpecContext, name string) resource.Resource {
		GinkgoHelper()
		e := enumeration.Enumeration{Name: name}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return createOne(ctx, svc.NewWriter(tx), &e)
		})).To(Succeed())
		return resource.Resource{ID: enumeration.ResourceID(e.Key), Name: name}
	}

	It("Should name each enumeration by its name", func(ctx SpecContext) {
		r := create(ctx, "Breaker")
		Expect(svc.Resources().Retrieve(ctx, db, []resource.ID{r.ID})).
			To(Equal([]resource.Resource{r}))
	})

	It("Should return a validation error for a malformed key", func(ctx SpecContext) {
		Expect(svc.Resources().Retrieve(
			ctx,
			db,
			[]resource.ID{{Type: resource.TypeEnumeration, Key: "valve"}},
		)).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`invalid enumeration key "valve"`)),
		))
	})

	It("Should leave out an ID that names no enumeration", func(ctx SpecContext) {
		r := create(ctx, "Breaker")
		missing := enumeration.ResourceID(uuid.New())
		Expect(svc.Resources().Retrieve(ctx, db, []resource.ID{missing, r.ID})).
			To(Equal([]resource.Resource{r}))
	})

	It("Should list each enumeration by its name", func(ctx SpecContext) {
		r := create(ctx, "Breaker")
		Expect(svc.Resources().List(ctx, db, 100, nil)).To(ContainElement(r))
	})

	It("Should search enumerations by name, closest first", func(ctx SpecContext) {
		a := create(ctx, "Breaker states")
		b := create(ctx, "Breaker")
		q := search.Query[resource.ID]{Term: "breaker"}
		Expect(MustSucceed(svc.Resources().Search(ctx, db, q)).Keys).
			To(Equal([]resource.ID{b.ID, a.ID}))
		q.Limit = 1
		Expect(MustSucceed(svc.Resources().Search(ctx, db, q)).Keys).
			To(Equal([]resource.ID{b.ID}))
	})
})
