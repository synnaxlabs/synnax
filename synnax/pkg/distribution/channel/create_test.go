// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/core/mock"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

var _ = Describe("TypedWriter", Ordered, func() {
	var (
		services map[core.NodeID]channel.Service
		builder  *mock.CoreBuilder
		log      *zap.Logger
	)
	BeforeAll(func() {
		log = zap.NewNop()
		builder, services = provisionServices(log)
	})
	AfterAll(func() {
		Expect(builder.Close()).To(Succeed())
		Expect(builder.Cleanup()).To(Succeed())
	})
	Context("Single channel", func() {
		var (
			channelLeaseNodeID aspen.NodeID
			ch                 channel.Channel
		)
		JustBeforeEach(func() {
			var err error
			ch = channel.Channel{
				Rate:     5 * telem.Hz,
				Name:     "SG01",
				DataType: telem.Float64T,
				NodeID:   channelLeaseNodeID,
			}
			err = services[1].Create(&ch)
			Expect(err).ToNot(HaveOccurred())
		})
		Context("Node is local", func() {
			BeforeEach(func() { channelLeaseNodeID = 1 })
			It("Should create the channel without error", func() {
				Expect(ch.Key().NodeID()).To(Equal(aspen.NodeID(1)))
				Expect(ch.Key().LocalKey()).To(Equal(storage.ChannelKey(1)))
			})
			It("Should create the channel in the cesium gorpDB", func() {
				channels, err := builder.Cores[1].Storage.TS.RetrieveChannels(ch.Key().String())
				Expect(err).ToNot(HaveOccurred())
				Expect(channels).To(HaveLen(1))
				cesiumCH := channels[0]
				Expect(cesiumCH.Key).To(Equal(ch.Key().String()))
				Expect(cesiumCH.DataType).To(Equal(telem.Float64T))
				Expect(cesiumCH.Rate).To(Equal(5 * telem.Hz))
			})
		})
		Context("Node is remote", func() {
			BeforeEach(func() { channelLeaseNodeID = 2 })
			It("Should create the channel without error", func() {
				Expect(ch.Key().NodeID()).To(Equal(aspen.NodeID(2)))
				Expect(ch.Key().LocalKey()).To(Equal(storage.ChannelKey(1)))
			})
			It("Should create the channel in the cesium gorpDB", func() {
				channels, err := builder.Cores[2].Storage.TS.RetrieveChannels(ch.Key().String())
				Expect(err).ToNot(HaveOccurred())
				Expect(channels).To(HaveLen(1))
				cesiumCH := channels[0]
				Expect(cesiumCH.Key).To(Equal(ch.Key().String()))
				Expect(cesiumCH.DataType).To(Equal(telem.Float64T))
				Expect(cesiumCH.Rate).To(Equal(5 * telem.Hz))
			})
			It("Should not create the channel on another nodes DB", func() {
				channels, err := builder.Cores[1].Storage.TS.RetrieveChannels(ch.Key().String())
				Expect(err).To(HaveOccurred())
				Expect(channels).To(HaveLen(0))
			})
			It("Should assign a sequential key to the channels on each node",
				func() {
					ch2 := &channel.Channel{
						Rate:     5 * telem.Hz,
						Name:     "SG01",
						DataType: telem.Float64T,
						NodeID:   1,
					}
					err := services[1].Create(ch2)
					Expect(err).To(BeNil())
					Expect(ch2.Key().NodeID()).To(Equal(aspen.NodeID(1)))
					Expect(ch2.Key().LocalKey()).To(Equal(storage.ChannelKey(3)))
				})
		})
	})
})
