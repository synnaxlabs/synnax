// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Resource IDs", func() {
	It("Should build a resource ID of the ICD type", func() {
		key := uuid.New()
		Expect(icd.ResourceID(key)).To(Equal(resource.ID{
			Type: resource.TypeICD,
			Key:  key.String(),
		}))
	})

	It("Should map keys and ICDs to resource IDs", func() {
		a, b := uuid.New(), uuid.New()
		ids := []resource.ID{icd.ResourceID(a), icd.ResourceID(b)}
		Expect(icd.ResourceIDs([]icd.Key{a, b})).To(Equal(ids))
		Expect(icd.ResourceIDsFromICDs([]icd.ICD{{Key: a}, {Key: b}})).To(Equal(ids))
	})
})

var _ = Describe("Resources", func() {
	create := func(ctx SpecContext, name string) icd.ICD {
		GinkgoHelper()
		ic := icd.ICD{Name: name}
		Expect(svc.NewWriter(tx).Create(ctx, &ic)).To(Succeed())
		return ic
	}

	It("Should name each ICD by its name", func(ctx SpecContext) {
		ic := create(ctx, "Engine bus")
		Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{ic.ResourceID()})).
			To(Equal([]resource.Resource{{ID: ic.ResourceID(), Name: "Engine bus"}}))
	})

	It("Should return a validation error for a malformed key", func(ctx SpecContext) {
		Expect(svc.Resources().Retrieve(
			ctx,
			tx,
			[]resource.ID{{Type: resource.TypeICD, Key: "pump"}},
		)).Error().To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`invalid icd key "pump"`)),
		))
	})

	It("Should leave out an ID that names no ICD", func(ctx SpecContext) {
		ic := create(ctx, "Listed")
		missing := icd.ResourceID(uuid.New())
		Expect(svc.Resources().Retrieve(
			ctx,
			tx,
			[]resource.ID{missing, ic.ResourceID()},
		)).To(Equal([]resource.Resource{{ID: ic.ResourceID(), Name: "Listed"}}))
	})

	It("Should list each ICD by its name", func(ctx SpecContext) {
		ic := create(ctx, "Listed")
		Expect(svc.Resources().List(ctx, tx, 10000, nil)).To(ContainElement(
			resource.Resource{ID: ic.ResourceID(), Name: "Listed"},
		))
	})
})

var _ = Describe("Search", func() {
	create := func(ctx SpecContext, name string) resource.ID {
		GinkgoHelper()
		ic := icd.ICD{Name: name}
		Expect(svc.NewWriter(tx).Create(ctx, &ic)).To(Succeed())
		return icd.ResourceID(ic.Key)
	}

	It("Should return the matching ICDs, closest match first", func(ctx SpecContext) {
		a := create(ctx, "turbopump test")
		b := create(ctx, "turbopump")
		q := search.Query[resource.ID]{Term: "turbopump"}
		Expect(MustSucceed(svc.Resources().Search(ctx, tx, q)).Keys).
			To(Equal([]resource.ID{b, a}))
		q.Limit = 1
		Expect(MustSucceed(svc.Resources().Search(ctx, tx, q)).Keys).
			To(Equal([]resource.ID{b}))
	})

	It("Should not match a staged ICD outside its transaction", func(ctx SpecContext) {
		create(ctx, "injector")
		q := search.Query[resource.ID]{Term: "injector"}
		Expect(MustSucceed(svc.Resources().Search(ctx, db, q)).Keys).To(BeEmpty())
	})
})
