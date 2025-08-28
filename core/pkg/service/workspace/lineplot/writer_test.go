// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lineplot_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/schematic"
	"github.com/synnaxlabs/x/gorp"
)

var _ = Describe("Writer", func() {
	Describe("Create", func() {
		It("Should create a LinePlot", func() {
			schematic := schematic.Schematic{
				Name: "test",
				Data: "data",
			}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &schematic)).To(Succeed())
			Expect(schematic.Key).ToNot(Equal(uuid.Nil))
		})
	})
	Describe("Update", func() {
		It("Should rename a LinePlot", func() {
			p := schematic.Schematic{Name: "test", Data: "data"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &p)).To(Succeed())
			Expect(svc.NewWriter(tx).Rename(ctx, p.Key, "test2")).To(Succeed())
			var res schematic.Schematic
			Expect(gorp.NewRetrieve[uuid.UUID, schematic.Schematic]().WhereKeys(p.Key).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Name).To(Equal("test2"))
		})
	})
	Describe("SetData", func() {
		It("Should set the data of a LinePlot", func() {
			p := schematic.Schematic{Name: "test", Data: "data"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &p)).To(Succeed())
			Expect(svc.NewWriter(tx).SetData(ctx, p.Key, "data2")).To(Succeed())
			var res schematic.Schematic
			Expect(gorp.NewRetrieve[uuid.UUID, schematic.Schematic]().WhereKeys(p.Key).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Data).To(Equal("data2"))
		})
	})
})
