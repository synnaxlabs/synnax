// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package core_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Frame", func() {
	Describe("NewFrame", func() {
		It("Should correctly create a new frame", func() {
			f := core.NewFrame(
				[]core.ChannelKey{1, 2, 3},
				[]telem.Series{
					telem.NewSeriesV[int64](1, 2, 3, 4),
					telem.NewSeriesV[uint32](1, 2, 3, 4),
					telem.NewSecondsTSV(1, 2, 3, 4),
				})
			Expect(f.Keys).To(Equal([]core.ChannelKey{1, 2, 3}))
			Expect(f.Series).To(HaveLen(3))
			Expect(f.Series[0].Data).To(Equal(telem.NewSeriesV[int64](1, 2, 3, 4).Data))
			Expect(f.Series[1].Data).To(Equal(telem.NewSeriesV[uint32](1, 2, 3, 4).Data))
			Expect(f.Series[2].Data).To(Equal(telem.NewSecondsTSV(1, 2, 3, 4).Data))
		})

		It("Should panic when creating a new frame with not same keys and series", func() {
			Expect(func() { core.NewFrame([]core.ChannelKey{1, 2}, []telem.Series{telem.NewSeriesV(1, 2, 3, 4)}) }).To(Panic())
		})
	})

	Describe("Len", func() {
		It("Should correctly return the length of a frame", func() {
			f := core.NewFrame(
				[]core.ChannelKey{1, 2, 3},
				[]telem.Series{
					telem.NewSeriesV[int64](1, 2, 3, 4),
					telem.NewSeriesV[uint32](1, 2, 3, 4),
					telem.NewSecondsTSV(1, 2, 3, 4),
				})
			Expect(f.Len()).To(Equal(int64(4)))
		})

		It("Should panic when the frame is uneven", func() {
			f := core.NewFrame(
				[]core.ChannelKey{1, 2, 3},
				[]telem.Series{
					telem.NewSeriesV[int64](1, 2, 3, 4),
					telem.NewSeriesV[uint32](1, 2, 3, 4),
					telem.NewSecondsTSV(1, 2, 3, 4, 5),
				})
			Expect(func() { f.Len() }).To(Panic())
		})
	})
})
