// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package project_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Resources", func() {
	It("Should name each project by its name", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		p := project.Project{Name: "resource-project"}
		Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
		Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{p.ResourceID()})).
			To(Equal([]resource.Resource{{ID: p.ResourceID(), Name: p.Name}}))
	})

	It("Should return a validation error for a malformed key", func(ctx SpecContext) {
		Expect(svc.Resources().Retrieve(
			ctx,
			nil,
			[]resource.ID{{Type: resource.TypeProject, Key: "pump"}},
		)).Error().To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`invalid project key "pump"`)),
		))
	})

	It("Should leave out an ID that names no project", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		p := project.Project{Name: "listed-project"}
		Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
		missing := resource.ID{Type: resource.TypeProject, Key: uuid.New().String()}
		Expect(svc.Resources().Retrieve(
			ctx,
			tx,
			[]resource.ID{missing, p.ResourceID()},
		)).To(Equal([]resource.Resource{{ID: p.ResourceID(), Name: p.Name}}))
	})

	It("Should list each project by its name", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		p := project.Project{Name: "listed-project"}
		Expect(svc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
		Expect(svc.Resources().List(ctx, tx, 10000, nil)).To(ContainElement(
			resource.Resource{ID: p.ResourceID(), Name: p.Name},
		))
	})
})
