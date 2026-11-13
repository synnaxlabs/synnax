// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package service_test

import (
	"encoding/json/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	xjson "github.com/synnaxlabs/x/encoding/json"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Visualization exports", Ordered, func() {
	var (
		l    *service.Layer
		db   *gorp.DB
		proj project.Project
		g    group.Group
		sch  schematic.Schematic
		lp   lineplot.LinePlot
		lg   log.Log
		tb   table.Table
	)

	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := testutil.NewNode(ctx)
		db = node.DB
		l = MustOpen(svctestutil.OpenLayer(ctx, node))
		proj = project.Project{Name: "Exported"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return l.Project.NewWriter(tx).Create(ctx, &proj)
		})).To(Succeed())
		g = group.Group{Name: "Propulsion", Parent: new(project.ResourceID(proj.Key))}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return l.Group.NewWriter(tx).Create(ctx, &g)
		})).To(Succeed())
		sch = schematic.Schematic{Name: "Schematic", Project: &proj.Key, Group: &g.Key}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return l.Schematic.NewWriter(tx).Create(ctx, &sch)
		})).To(Succeed())
		lp = lineplot.LinePlot{Name: "Plot", Project: &proj.Key, Group: &g.Key}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return l.LinePlot.NewWriter(tx).Create(ctx, &lp)
		})).To(Succeed())
		lg = log.Log{Name: "Log", Project: &proj.Key, Group: &g.Key}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return l.Log.NewWriter(tx).Create(ctx, &lg)
		})).To(Succeed())
		tb = table.Table{Name: "Table", Project: &proj.Key, Group: &g.Key}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return l.Table.NewWriter(tx).Create(ctx, &tb)
		})).To(Succeed())
		pn := panel.Panel{Name: "Panel", Project: &proj.Key, Group: &g.Key}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return l.Panel.NewWriter(tx).Create(ctx, &pn)
		})).To(Succeed())
	})

	expectUnplaced := func(data []byte) {
		GinkgoHelper()
		var body map[string]any
		Expect(json.Unmarshal(data, &body)).To(Succeed())
		Expect(body).ToNot(HaveKey("group"))
		Expect(body).ToNot(HaveKey("project"))
	}

	DescribeTable("Should leave the placement out of a single-resource export",
		func(ctx SpecContext, id func() resource.ID) {
			env := MustSucceed(l.ImEx.Export(ctx, db, id()))
			expectUnplaced(MustSucceed(json.Marshal(env)))
		},
		Entry("schematic", func() resource.ID { return sch.ResourceID() }),
		Entry("line plot", func() resource.ID { return lp.ResourceID() }),
		Entry("log", func() resource.ID { return lg.ResourceID() }),
		Entry("table", func() resource.ID { return tb.ResourceID() }),
	)

	It("Should leave the placement out of every member of a project bundle", func(
		ctx SpecContext,
	) {
		files, _ := MustSucceed2(l.Bundle.Export(ctx, db, proj.Key, xjson.Codec))
		Expect(files).To(HaveLen(6))
		for path, data := range files {
			By(path)
			expectUnplaced(data)
		}
	})
})
