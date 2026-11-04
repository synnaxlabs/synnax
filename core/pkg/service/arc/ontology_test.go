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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("RetrieveResource", func() {
	DescribeTable("Should carry the Arc type in the resource data",
		func(ctx SpecContext, t arc.Type) {
			a := newArc(t, arc.Base{Name: "typed", Mode: arc.ModeText})
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
			res := MustSucceed(svc.RetrieveResource(ctx, a.Base().Key.String(), tx))
			Expect(res.Data).To(HaveKeyWithValue("type", string(t)))
		},
		Entry("automation", arc.TypeAutomation),
		Entry("module", arc.TypeModule),
	)
})

var _ = Describe("Search", func() {
	create := func(ctx SpecContext, name string) ontology.ID {
		GinkgoHelper()
		a := newAutomation(arc.Base{Name: name, Mode: arc.ModeText})
		Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
		return arc.OntologyID(a.Base().Key)
	}

	It("Should return the matching Arcs, closest match first", func(ctx SpecContext) {
		a := create(ctx, "turbopump test")
		b := create(ctx, "turbopump")
		q := search.Query[ontology.ID]{Term: "turbopump"}
		Expect(MustSucceed(svc.Search(ctx, tx, q)).Keys).
			To(Equal([]ontology.ID{b, a}))
		q.Limit = 1
		Expect(MustSucceed(svc.Search(ctx, tx, q)).Keys).
			To(Equal([]ontology.ID{b}))
	})

	It("Should not match a staged Arc outside its transaction", func(ctx SpecContext) {
		create(ctx, "injector")
		q := search.Query[ontology.ID]{Term: "injector"}
		Expect(MustSucceed(svc.Search(ctx, db, q)).Keys).To(BeEmpty())
	})
})
