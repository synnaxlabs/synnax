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
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/workspace"
	"github.com/synnaxlabs/x/gorp"
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
	})
})
