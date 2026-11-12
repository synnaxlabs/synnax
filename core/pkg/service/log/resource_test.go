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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/defaults"
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Resources", func() {
	It("Should name each log by its name", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		l := defaults.Fill(log.Log{Project: &proj.Key, Name: "resource-log"})
		Expect(svc.NewWriter(tx).Create(ctx, &l)).To(Succeed())
		Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{l.ResourceID()})).
			To(Equal([]resource.Resource{{ID: l.ResourceID(), Name: l.Name}}))
	})

	It("Should return a validation error for a malformed key", func(ctx SpecContext) {
		Expect(svc.Resources().Retrieve(
			ctx,
			db,
			[]resource.ID{{Type: resource.TypeLog, Key: "pump"}},
		)).Error().To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`invalid log key "pump"`)),
		))
	})

	It("Should leave out an ID that names no log", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		l := defaults.Fill(log.Log{Project: &proj.Key, Name: "listed-log"})
		Expect(svc.NewWriter(tx).Create(ctx, &l)).To(Succeed())
		missing := resource.ID{Type: resource.TypeLog, Key: uuid.New().String()}
		Expect(svc.Resources().Retrieve(
			ctx,
			tx,
			[]resource.ID{missing, l.ResourceID()},
		)).To(Equal([]resource.Resource{{ID: l.ResourceID(), Name: l.Name}}))
	})

	It("Should list each log by its name", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		l := defaults.Fill(log.Log{Project: &proj.Key, Name: "listed-log"})
		Expect(svc.NewWriter(tx).Create(ctx, &l)).To(Succeed())
		Expect(svc.Resources().List(ctx, tx, 10000, nil)).To(ContainElement(
			resource.Resource{ID: l.ResourceID(), Name: l.Name},
		))
	})
})

var _ = Describe("Search", func() {
	var tx gorp.Tx
	BeforeEach(func() { tx = DeferClose(db.OpenTx()) })
	create := func(ctx SpecContext, name string) resource.ID {
		GinkgoHelper()
		l := defaults.Fill(log.Log{Project: &proj.Key, Name: name})
		Expect(svc.NewWriter(tx).Create(ctx, &l)).To(Succeed())
		return log.ResourceID(l.Key)
	}

	It("Should return the matching logs, closest match first", func(ctx SpecContext) {
		a := create(ctx, "turbopump test")
		b := create(ctx, "turbopump")
		q := search.Query[resource.ID]{Term: "turbopump"}
		Expect(MustSucceed(svc.Resources().Search(ctx, tx, q)).Keys).
			To(Equal([]resource.ID{b, a}))
		q.Limit = 1
		Expect(MustSucceed(svc.Resources().Search(ctx, tx, q)).Keys).
			To(Equal([]resource.ID{b}))
	})

	It("Should not match a staged log outside its transaction", func(ctx SpecContext) {
		create(ctx, "injector")
		q := search.Query[resource.ID]{Term: "injector"}
		Expect(MustSucceed(svc.Resources().Search(ctx, db, q)).Keys).To(BeEmpty())
	})
})
