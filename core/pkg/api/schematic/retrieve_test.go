// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schematic_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	apischematic "github.com/synnaxlabs/synnax/pkg/api/schematic"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Retrieve by page", func() {
	var (
		u             user.User
		paged         project.Project
		first, second schematic.Schematic
	)
	BeforeEach(func(ctx SpecContext) {
		u = newUser(ctx)
		grantOn(
			ctx,
			u.ResourceID(),
			access.ActionRetrieve,
			resource.ID{Type: resource.TypeSchematic},
		)
		paged = project.Project{Name: "paged"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return projectSvc.NewWriter(tx).Create(ctx, &paged)
		})).To(Succeed())
		first = schematic.Schematic{Project: &paged.Key, Name: "first"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return schematicSvc.NewWriter(tx).Create(ctx, &first)
		})).To(Succeed())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return schematicSvc.NewWriter(tx).Copy(ctx, first.Key, "second", &second)
		})).To(Succeed())
	})
	retrieve := func(ctx SpecContext, req apischematic.RetrieveRequest) []string {
		GinkgoHelper()
		if req.Where == nil {
			req.Where = &schematic.Where{}
		}
		req.Where.Projects = []project.Key{paged.Key}
		res := MustSucceed(retrieve(AuthedCtx(ctx, u), req))
		return lo.Map(res.Schematics, func(s schematic.Schematic, _ int) string {
			return s.Name
		})
	}

	It("Should page the matching schematics", func(ctx SpecContext) {
		order := &apischematic.Order{Field: apischematic.OrderFieldName}
		Expect(retrieve(ctx, apischematic.RetrieveRequest{
			Order: order,
			Limit: new(uint32(1)),
		})).To(Equal([]string{"first"}))
		Expect(retrieve(ctx, apischematic.RetrieveRequest{
			Order: order,
			After: &apischematic.Cursor{Key: first.Key, Name: new("first")},
			Limit: new(uint32(1)),
		})).To(Equal([]string{"second"}))
	})
})

var _ = Describe("Retrieve access", func() {
	It("Should leave out the schematics that the subject cannot retrieve", func(
		ctx SpecContext,
	) {
		u := newUser(ctx)
		granted := createSchematic(ctx, "leave-out-granted")
		createSchematic(ctx, "leave-out-denied")
		grantOn(ctx, u.ResourceID(), access.ActionRetrieve, granted.ResourceID())
		res := MustSucceed(retrieve(AuthedCtx(ctx, u), apischematic.RetrieveRequest{}))
		Expect(res.Schematics).To(HaveExactElements(HaveField("Key", granted.Key)))
	})
})
