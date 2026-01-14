// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lineplot_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/lineplot"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/uuid"
)

var _ = Describe("Writer", func() {
	Describe("Create", func() {
		It("Should create a LinePlot", func() {
			plot := lineplot.LinePlot{
				Name: "test",
				Data: map[string]any{"key": "data"},
			}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
			Expect(plot.Key).ToNot(Equal(uuid.Nil))
		})
	})
	Describe("Update", func() {
		It("Should rename a LinePlot", func() {
			plot := lineplot.LinePlot{Name: "test", Data: map[string]any{"key": "data"}}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
			Expect(svc.NewWriter(tx).Rename(ctx, plot.Key, "test2")).To(Succeed())
			var res lineplot.LinePlot
			Expect(gorp.NewRetrieve[uuid.UUID, lineplot.LinePlot]().WhereKeys(plot.Key).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Name).To(Equal("test2"))
		})
	})
	Describe("SetData", func() {
		It("Should set the data of a LinePlot", func() {
			plot := lineplot.LinePlot{Name: "test", Data: map[string]any{"key": "data"}}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
			Expect(svc.NewWriter(tx).SetData(ctx, plot.Key, map[string]any{"key": "data2"})).To(Succeed())
			var res lineplot.LinePlot
			Expect(gorp.NewRetrieve[uuid.UUID, lineplot.LinePlot]().WhereKeys(plot.Key).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Data["key"]).To(Equal("data2"))
		})
	})
})
