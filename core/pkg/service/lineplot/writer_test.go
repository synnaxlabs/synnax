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
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/text"
)

var _ = Describe("Writer", func() {
	Describe("Create", func() {
		It("Should create a LinePlot", func(ctx SpecContext) {
			plot := lineplot.LinePlot{Name: "test"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
			Expect(plot.Key).ToNot(Equal(uuid.Nil))
		})
	})
	Describe("Update", func() {
		It("Should rename a LinePlot", func(ctx SpecContext) {
			plot := lineplot.LinePlot{Name: "test"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
			Expect(svc.NewWriter(tx).Rename(ctx, plot.Key, "test2")).To(Succeed())
			var res lineplot.LinePlot
			Expect(svc.NewRetrieve().Where(lineplot.MatchKeys(plot.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Name).To(Equal("test2"))
		})
	})
	Describe("SetData", func() {
		It("Should replace the body of a LinePlot while preserving key and name", func(ctx SpecContext) {
			plot := lineplot.LinePlot{Name: "test"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
			body := lineplot.LinePlot{
				Title:  lineplot.Title{Level: text.LevelH4, Visible: true},
				Legend: lineplot.Legend{Visible: true},
				Lines: []lineplot.Line{{
					Key:            "l1",
					Color:          new(color.MustFromHex("#abcdef")),
					StrokeWidth:    2,
					Downsample:     1,
					DownsampleMode: lineplot.DownsampleModeDecimate,
				}},
			}
			Expect(svc.NewWriter(tx).SetData(ctx, plot.Key, body)).To(Succeed())
			var res lineplot.LinePlot
			Expect(svc.NewRetrieve().Where(lineplot.MatchKeys(plot.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Key).To(Equal(plot.Key))
			Expect(res.Name).To(Equal("test"))
			Expect(res.Title.Level).To(Equal(text.LevelH4))
			Expect(res.Lines).To(HaveLen(1))
			Expect(res.Lines[0].Color).To(Equal(new(color.MustFromHex("#abcdef"))))
		})
	})
})
