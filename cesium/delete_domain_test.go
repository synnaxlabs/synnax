// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium_test

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Delete", Ordered, func() {
	var db *cesium.DB
	var (
		basic1 cesium.ChannelKey = 1
	)
	BeforeAll(func() {
		db = openMemDB()
	})
	AfterAll(func() { Expect(db.Close()).To(Succeed()) })
	Describe("Deleting a domain as a whole from a channel", func() {
		It("Should delete chunks of a channel", func() {
			By("Creating a channel")
			Expect(db.CreateChannel(
				ctx,
				cesium.Channel{Key: basic1, DataType: telem.Int64T, Rate: 1},
			)).To(Succeed())
			w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
				Channels: []cesium.ChannelKey{basic1},
				Start:    10 * telem.SecondTS,
			}))

			By("Writing data to the channel")
			ok := w.Write(cesium.NewFrame(
				[]cesium.ChannelKey{basic1},
				[]telem.Series{
					telem.NewSeriesV[int64](10, 11, 12, 13, 14, 15, 16, 17, 18),
				}),
			)
			Expect(ok).To(BeTrue())
			_, ok = w.Commit()
			Expect(ok).To(BeTrue())
			Expect(w.Close()).To(Succeed())

			//frame, err := db.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 19 * telem.SecondTS}, basic1)
			//Expect(err).To(BeNil())
			//fmt.Println("frame before deletion: ", frame)

			By("Deleting channel data")
			Expect(db.DeleteTimeRange(ctx, basic1, telem.TimeRange{
				Start: 12 * telem.SecondTS,
				End:   15 * telem.SecondTS,
			})).To(Succeed())

			frame, err := db.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 19 * telem.SecondTS}, basic1)
			Expect(err).To(BeNil())
			fmt.Println("frame after deletion: ", frame)
		})
	})
})
