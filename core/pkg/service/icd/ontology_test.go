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
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Ontology IDs", func() {
	It("Should build an ontology ID of the ICD type", func() {
		key := uuid.New()
		Expect(icd.OntologyID(key)).To(Equal(ontology.ID{
			Type: ontology.ResourceTypeICD,
			Key:  key.String(),
		}))
	})

	It("Should map keys and ICDs to ontology IDs", func() {
		a, b := uuid.New(), uuid.New()
		ids := []ontology.ID{icd.OntologyID(a), icd.OntologyID(b)}
		Expect(icd.OntologyIDs([]icd.Key{a, b})).To(Equal(ids))
		Expect(icd.OntologyIDsFromICDs([]icd.ICD{{Key: a}, {Key: b}})).To(Equal(ids))
	})
})

var _ = Describe("Search", func() {
	create := func(ctx SpecContext, name string) ontology.ID {
		GinkgoHelper()
		ic := icd.ICD{Name: name}
		Expect(svc.NewWriter(tx).Create(ctx, &ic)).To(Succeed())
		return icd.OntologyID(ic.Key)
	}

	It("Should return the matching ICDs, closest match first", func(ctx SpecContext) {
		a := create(ctx, "turbopump test")
		b := create(ctx, "turbopump")
		q := search.Query[ontology.ID]{Term: "turbopump"}
		Expect(MustSucceed(svc.Search(ctx, tx, q)).Keys).
			To(Equal([]ontology.ID{b, a}))
		q.Limit = 1
		Expect(MustSucceed(svc.Search(ctx, tx, q)).Keys).
			To(Equal([]ontology.ID{b}))
	})

	It("Should not match a staged ICD outside its transaction", func(ctx SpecContext) {
		create(ctx, "injector")
		q := search.Query[ontology.ID]{Term: "injector"}
		Expect(MustSucceed(svc.Search(ctx, db, q)).Keys).To(BeEmpty())
	})
})
