// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Resources", Ordered, func() {
	var s stack
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		s = openStack(ctx, "Resource Rack")
	})

	create := func(ctx SpecContext, tx gorp.Tx, name string, internal bool) task.Task {
		t := task.Task{Type: testType, Rack: s.rack.Key, Name: name, Internal: internal}
		Expect(s.task.NewWriter(tx).Create(ctx, &t)).To(Succeed())
		return t
	}

	It("Should name each task by its name", func(ctx SpecContext) {
		tx := DeferClose(s.db.OpenTx())
		t := create(ctx, tx, "Page on-call", false)
		Expect(s.task.Resources().Retrieve(ctx, tx, []resource.ID{t.ResourceID()})).
			To(Equal([]resource.Resource{{ID: t.ResourceID(), Name: "Page on-call"}}))
	})

	It("Should return a validation error for a malformed key", func(ctx SpecContext) {
		Expect(s.task.Resources().Retrieve(
			ctx,
			s.db,
			[]resource.ID{{Type: resource.TypeTask, Key: "pump"}},
		)).Error().To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`invalid task key "pump"`)),
		))
	})

	It("Should leave out an ID that names no task", func(ctx SpecContext) {
		tx := DeferClose(s.db.OpenTx())
		t := create(ctx, tx, "Page on-call", false)
		missing := resource.ID{Type: resource.TypeTask, Key: uuid.New().String()}
		Expect(s.task.Resources().Retrieve(
			ctx,
			tx,
			[]resource.ID{missing, t.ResourceID()},
		)).To(Equal([]resource.Resource{{ID: t.ResourceID(), Name: "Page on-call"}}))
	})

	It("Should leave internal tasks out of a search", func(ctx SpecContext) {
		tx := DeferClose(s.db.OpenTx())
		create(ctx, tx, "Page", true)
		visible := create(ctx, tx, "Page on-call", false)
		Expect(MustSucceed(s.task.Resources().Search(
			ctx,
			tx,
			search.Query[resource.ID]{Term: "Page", Limit: 1},
		)).Keys).To(Equal([]resource.ID{visible.ResourceID()}))
	})

	It("Should leave internal tasks out of a list", func(ctx SpecContext) {
		tx := DeferClose(s.db.OpenTx())
		create(ctx, tx, "Page", true)
		visible := create(ctx, tx, "Page on-call", false)
		Expect(s.task.Resources().List(ctx, tx, 10, nil)).To(Equal(
			[]resource.Resource{{ID: visible.ResourceID(), Name: "Page on-call"}},
		))
	})
})
