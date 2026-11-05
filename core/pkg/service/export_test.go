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
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	xjson "github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Visualization exports", Ordered, func() {
	var (
		l     *service.Layer
		proj  project.Project
		g     group.Group
		rng   ranger.Range
		owned []resource.ID
		snap  schematic.Schematic
	)

	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		l = MustOpen(svctestutil.OpenLayer(ctx, testutil.NewNode(ctx)))
		proj = project.Project{Name: "Exported"}
		Expect(l.Project.NewWriter(nil).Create(ctx, &proj)).To(Succeed())
		g = group.Group{Name: "Propulsion", Parent: new(project.ResourceID(proj.Key))}
		Expect(l.Group.NewWriter(nil).Create(ctx, &g)).To(Succeed())
		sch := schematic.Schematic{Name: "Schematic", Project: &proj.Key, Group: &g.Key}
		Expect(l.Schematic.NewWriter(nil).Create(ctx, &sch)).To(Succeed())
		lp := lineplot.LinePlot{Name: "Plot", Project: &proj.Key, Group: &g.Key}
		Expect(l.LinePlot.NewWriter(nil).Create(ctx, &lp)).To(Succeed())
		lg := log.Log{Name: "Log", Project: &proj.Key, Group: &g.Key}
		Expect(l.Log.NewWriter(nil).Create(ctx, &lg)).To(Succeed())
		tb := table.Table{Name: "Table", Project: &proj.Key, Group: &g.Key}
		Expect(l.Table.NewWriter(nil).Create(ctx, &tb)).To(Succeed())
		pn := panel.Panel{Name: "Panel", Project: &proj.Key, Group: &g.Key}
		Expect(l.Panel.NewWriter(nil).Create(ctx, &pn)).To(Succeed())
		owned = []resource.ID{
			schematic.ResourceID(sch.Key),
			lineplot.ResourceID(lp.Key),
			log.ResourceID(lg.Key),
			table.ResourceID(tb.Key),
		}
		rng = ranger.Range{
			Name:      "Run",
			TimeRange: telem.SecondTS.SpanRange(telem.Second),
		}
		Expect(l.Ranger.NewWriter(nil).Create(ctx, &rng)).To(Succeed())
		Expect(l.Schematic.NewWriter(nil).
			Copy(ctx, sch.Key, "Snapshot", true, &rng.Key, &snap)).To(Succeed())
	})

	expectUnplaced := func(data []byte) {
		GinkgoHelper()
		var body map[string]any
		Expect(json.Unmarshal(data, &body)).To(Succeed())
		Expect(body).ToNot(HaveKey("group"))
		Expect(body).ToNot(HaveKey("project"))
		Expect(body).ToNot(HaveKey("range"))
	}

	It("Should leave the placement out of a single-resource export", func(
		ctx SpecContext,
	) {
		for _, id := range append(owned, schematic.ResourceID(snap.Key)) {
			env := MustSucceed(l.ImEx.Export(ctx, id))
			expectUnplaced(MustSucceed(json.Marshal(env)))
		}
	})

	It("Should leave the placement out of every member of a project bundle", func(
		ctx SpecContext,
	) {
		files, _ := MustSucceed2(l.Bundle.Export(ctx, proj.Key, xjson.Codec))
		Expect(files).To(HaveLen(6))
		for path, data := range files {
			By(path)
			expectUnplaced(data)
		}
	})
})
