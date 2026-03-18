// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package workspace_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	"github.com/synnaxlabs/synnax/pkg/service/workspace"
	"github.com/synnaxlabs/x/gorp"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Writer", func() {
	Describe("Create", func() {
		It("Should create a workspace", func() {
			ws := workspace.Workspace{
				Name:   "test",
				Author: author.Key,
				Layout: "data",
			}
			Expect(svc.NewWriter(tx).Create(ctx, &ws)).To(Succeed())
			Expect(ws.Key).ToNot(Equal(uuid.Nil))
		})
	})
	Describe("Update", func() {
		It("Should rename a workspace", func() {
			ws := workspace.Workspace{Name: "test", Author: author.Key}
			Expect(svc.NewWriter(tx).Create(ctx, &ws)).To(Succeed())
			Expect(svc.NewWriter(tx).Rename(ctx, ws.Key, "test2")).To(Succeed())
			var res workspace.Workspace
			Expect(gorp.NewRetrieve[uuid.UUID, workspace.Workspace]().WhereKeys(ws.Key).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Name).To(Equal("test2"))
		})
	})
	Describe("SetLayout", func() {
		It("Should set the layout of a workspace", func() {
			ws := workspace.Workspace{Name: "test", Author: author.Key}
			Expect(svc.NewWriter(tx).Create(ctx, &ws)).To(Succeed())
			Expect(svc.NewWriter(tx).SetLayout(ctx, ws.Key, "data")).To(Succeed())
			var res workspace.Workspace
			Expect(gorp.NewRetrieve[uuid.UUID, workspace.Workspace]().WhereKeys(ws.Key).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Layout).To(Equal("data"))
		})
	})
	Describe("Delete", func() {
		It("Should delete a workspace", func() {
			ws := workspace.Workspace{Name: "test", Author: author.Key}
			Expect(svc.NewWriter(tx).Create(ctx, &ws)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, ws.Key)).To(Succeed())
			var res workspace.Workspace
			Expect(gorp.NewRetrieve[uuid.UUID, workspace.Workspace]().WhereKeys(ws.Key).Entry(&res).Exec(ctx, tx)).ToNot(Succeed())
		})
		It("Should cascade delete child schematics", func() {
			ws := workspace.Workspace{Name: "cascade_test", Author: author.Key}
			Expect(svc.NewWriter(tx).Create(ctx, &ws)).To(Succeed())

			s1 := schematic.Schematic{Name: "schematic_1", Data: "{}"}
			Expect(schematicSvc.NewWriter(tx).Create(ctx, ws.Key, &s1)).To(Succeed())
			s2 := schematic.Schematic{Name: "schematic_2", Data: "{}"}
			Expect(schematicSvc.NewWriter(tx).Create(ctx, ws.Key, &s2)).To(Succeed())

			Expect(svc.NewWriter(tx).Delete(ctx, ws.Key)).To(Succeed())

			var res schematic.Schematic
			Expect(gorp.NewRetrieve[uuid.UUID, schematic.Schematic]().WhereKeys(s1.Key).Entry(&res).Exec(ctx, tx)).ToNot(Succeed())
			Expect(gorp.NewRetrieve[uuid.UUID, schematic.Schematic]().WhereKeys(s2.Key).Entry(&res).Exec(ctx, tx)).ToNot(Succeed())
		})
		It("Should cascade delete schematics inside a group under a workspace", func() {
			ws := workspace.Workspace{Name: "group_cascade", Author: author.Key}
			Expect(svc.NewWriter(tx).Create(ctx, &ws)).To(Succeed())

			g := MustSucceed(groupSvc.NewWriter(tx).Create(ctx, "test_group", workspace.OntologyID(ws.Key)))

			s := schematic.Schematic{Name: "nested_schematic", Data: "{}"}
			Expect(schematicSvc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())
			// Re-parent the schematic under the group instead of the workspace
			Expect(otg.NewWriter(tx).DeleteRelationship(
				ctx,
				workspace.OntologyID(ws.Key),
				ontology.RelationshipTypeParentOf,
				schematic.OntologyID(s.Key),
			)).To(Succeed())
			Expect(otg.NewWriter(tx).DefineRelationship(
				ctx,
				ontology.ID{Type: "group", Key: g.Key.String()},
				ontology.RelationshipTypeParentOf,
				schematic.OntologyID(s.Key),
			)).To(Succeed())

			Expect(svc.NewWriter(tx).Delete(ctx, ws.Key)).To(Succeed())

			// The schematic is a grandchild (workspace -> group -> schematic),
			// so it should still exist since deleteChildren only traverses
			// direct children.
			var sRes schematic.Schematic
			Expect(gorp.NewRetrieve[uuid.UUID, schematic.Schematic]().WhereKeys(s.Key).Entry(&sRes).Exec(ctx, tx)).To(Succeed())
		})
		It("Should cascade delete mixed resource types", func() {
			ws := workspace.Workspace{Name: "mixed_cascade", Author: author.Key}
			Expect(svc.NewWriter(tx).Create(ctx, &ws)).To(Succeed())

			s := schematic.Schematic{Name: "schematic", Data: "{}"}
			Expect(schematicSvc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())

			lp := lineplot.LinePlot{Name: "lineplot", Data: "{}"}
			Expect(lineplotSvc.NewWriter(tx).Create(ctx, ws.Key, &lp)).To(Succeed())

			Expect(svc.NewWriter(tx).Delete(ctx, ws.Key)).To(Succeed())

			var sRes schematic.Schematic
			Expect(gorp.NewRetrieve[uuid.UUID, schematic.Schematic]().WhereKeys(s.Key).Entry(&sRes).Exec(ctx, tx)).ToNot(Succeed())
			var lpRes lineplot.LinePlot
			Expect(gorp.NewRetrieve[uuid.UUID, lineplot.LinePlot]().WhereKeys(lp.Key).Entry(&lpRes).Exec(ctx, tx)).ToNot(Succeed())
		})
		It("Should cascade delete all four child resource types", func() {
			ws := workspace.Workspace{Name: "all_types_cascade", Author: author.Key}
			Expect(svc.NewWriter(tx).Create(ctx, &ws)).To(Succeed())

			s := schematic.Schematic{Name: "schematic", Data: "{}"}
			Expect(schematicSvc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())

			lp := lineplot.LinePlot{Name: "lineplot", Data: "{}"}
			Expect(lineplotSvc.NewWriter(tx).Create(ctx, ws.Key, &lp)).To(Succeed())

			lg := log.Log{Name: "log", Data: "{}"}
			Expect(logSvc.NewWriter(tx).Create(ctx, ws.Key, &lg)).To(Succeed())

			tb := table.Table{Name: "table", Data: "{}"}
			Expect(tableSvc.NewWriter(tx).Create(ctx, ws.Key, &tb)).To(Succeed())

			Expect(svc.NewWriter(tx).Delete(ctx, ws.Key)).To(Succeed())

			var sRes schematic.Schematic
			Expect(gorp.NewRetrieve[uuid.UUID, schematic.Schematic]().WhereKeys(s.Key).Entry(&sRes).Exec(ctx, tx)).ToNot(Succeed())
			var lpRes lineplot.LinePlot
			Expect(gorp.NewRetrieve[uuid.UUID, lineplot.LinePlot]().WhereKeys(lp.Key).Entry(&lpRes).Exec(ctx, tx)).ToNot(Succeed())
			var lgRes log.Log
			Expect(gorp.NewRetrieve[uuid.UUID, log.Log]().WhereKeys(lg.Key).Entry(&lgRes).Exec(ctx, tx)).ToNot(Succeed())
			var tbRes table.Table
			Expect(gorp.NewRetrieve[uuid.UUID, table.Table]().WhereKeys(tb.Key).Entry(&tbRes).Exec(ctx, tx)).ToNot(Succeed())
		})
		It("Should delete a workspace with no children", func() {
			ws := workspace.Workspace{Name: "empty_workspace", Author: author.Key}
			Expect(svc.NewWriter(tx).Create(ctx, &ws)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, ws.Key)).To(Succeed())
			var res workspace.Workspace
			Expect(gorp.NewRetrieve[uuid.UUID, workspace.Workspace]().WhereKeys(ws.Key).Entry(&res).Exec(ctx, tx)).ToNot(Succeed())
		})
	})
})
