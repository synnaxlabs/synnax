// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/defaults"
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Search", func() {
	create := func(ctx SpecContext, name string) ontology.ID {
		GinkgoHelper()
		l := defaults.Fill(log.Log{Name: name})
		Expect(svc.NewWriter(tx).Create(ctx, proj.Key, &l)).To(Succeed())
		return log.OntologyID(l.Key)
	}

	It("Should return the matching logs, closest match first", func(ctx SpecContext) {
		a := create(ctx, "turbopump test")
		b := create(ctx, "turbopump")
		q := search.Query[ontology.ID]{Term: "turbopump"}
		Expect(MustSucceed(svc.Search(ctx, tx, q)).Keys).
			To(Equal([]ontology.ID{b, a}))
		q.Limit = 1
		Expect(MustSucceed(svc.Search(ctx, tx, q)).Keys).
			To(Equal([]ontology.ID{b}))
	})

	It("Should not match a staged log outside its transaction", func(ctx SpecContext) {
		create(ctx, "injector")
		q := search.Query[ontology.ID]{Term: "injector"}
		Expect(MustSucceed(svc.Search(ctx, db, q)).Keys).To(BeEmpty())
	})
})
