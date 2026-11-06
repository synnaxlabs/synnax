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
	apischematic "github.com/synnaxlabs/synnax/pkg/api/schematic"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Retrieve", func() {
	It(
		"Should return no schematics for a request with no filter",
		func(ctx SpecContext) {
			s := schematic.Schematic{Project: &proj.Key, Name: "unfiltered"}
			Expect(schematicSvc.NewWriter(nil).Create(ctx, &s)).To(Succeed())
			res := MustSucceed(apiSvc.Retrieve(
				AuthedCtx(ctx, author),
				apischematic.RetrieveRequest{},
			))
			Expect(res.Schematics).To(BeEmpty())
			Expect(res.Schematics).ToNot(BeNil())
		},
	)
})

var _ = Describe("Retrieve by snapshot and page", func() {
	var (
		u              user.User
		paged          project.Project
		editable, snap schematic.Schematic
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
		Expect(projectSvc.NewWriter(nil).Create(ctx, &paged)).To(Succeed())
		editable = schematic.Schematic{Project: &paged.Key, Name: "editable"}
		Expect(schematicSvc.NewWriter(nil).Create(ctx, &editable)).To(Succeed())
		Expect(schematicSvc.NewWriter(nil).
			Copy(ctx, editable.Key, "snap", true, nil, &snap)).To(Succeed())
	})
	retrieve := func(ctx SpecContext, req apischematic.RetrieveRequest) []string {
		GinkgoHelper()
		req.Projects = []project.Key{paged.Key}
		res := MustSucceed(apiSvc.Retrieve(AuthedCtx(ctx, u), req))
		return lo.Map(res.Schematics, func(s schematic.Schematic, _ int) string {
			return s.Name
		})
	}

	DescribeTable("Should filter on snapshot", func(
		ctx SpecContext,
		snapshot bool,
		name string,
	) {
		Expect(retrieve(ctx, apischematic.RetrieveRequest{Snapshot: &snapshot})).
			To(Equal([]string{name}))
	},
		Entry("snapshots", true, "snap"),
		Entry("editable schematics", false, "editable"),
	)

	It("Should page the matching schematics", func(ctx SpecContext) {
		first := retrieve(ctx, apischematic.RetrieveRequest{Limit: 1})
		second := retrieve(ctx, apischematic.RetrieveRequest{Limit: 1, Offset: 1})
		Expect(first).To(HaveLen(1))
		Expect(second).To(HaveLen(1))
		Expect(append(first, second...)).To(ConsistOf("editable", "snap"))
	})
})
