// Copyright 2024 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/core/mock"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Delete", Ordered, func() {
	var (
		services map[core.NodeKey]channel.Service
		builder  *mock.CoreBuilder
	)
	BeforeAll(func() { builder, services = provisionServices() })
	AfterAll(func() {
		Expect(builder.Close()).To(Succeed())
		Expect(builder.Cleanup()).To(Succeed())
	})
	Describe("Channel Deletion", func() {
		Context("Single Channel", func() {
			var (
				idxCh, ch channel.Channel
			)
			JustBeforeEach(func() {
				idxCh.Name = "SG01_time"
				idxCh.DataType = telem.TimeStampT
				idxCh.IsIndex = true
				Expect(services[1].Create(ctx, &idxCh)).To(Succeed())
				ch.Name = "SG01"
				ch.DataType = telem.Float64T
				ch.LocalIndex = idxCh.LocalKey
				Expect(services[1].Create(ctx, &ch)).To(Succeed())
			})
			Context("Node is local", func() {
				BeforeEach(func() {
					idxCh.Leaseholder = 1
					ch.Leaseholder = 1
				})
				It("Should not allow deletion of index channel with dependent channels", func() {
					Expect(services[1].Delete(ctx, idxCh.Key(), true)).ToNot(Succeed())
				})
				It("Should delete the channel without error", func() {
					Expect(services[1].DeleteMany(ctx, channel.Keys{idxCh.Key(), ch.Key()}, true)).To(Succeed())
				})
				It("Should not be able to retrieve the channel after deletion", func() {
					Expect(services[1].Delete(ctx, ch.Key(), true)).To(Succeed())
					exists, err := services[1].NewRetrieve().WhereKeys(ch.Key()).Exists(ctx, nil)
					Expect(err).ToNot(HaveOccurred())
					Expect(exists).To(BeFalse())
				})
				It("Should not be able to retrieve the channel from the storage DB", func() {
					Expect(services[1].Delete(ctx, ch.Key(), true)).To(Succeed())
					channels, err := builder.Cores[1].Storage.TS.RetrieveChannels(ctx, ch.Key().StorageKey())
					Expect(err).To(MatchError(cesium.ErrChannelNotFound))
					Expect(channels).To(BeEmpty())
				})
			})

			Context("Node is remote", func() {
				BeforeEach(func() {
					idxCh.Leaseholder = 2
					ch.Leaseholder = 2
				})
				It("Should not allow deletion of index channel with dependent channels", func() {
					Expect(services[1].Delete(ctx, idxCh.Key(), true)).ToNot(Succeed())
				})
				It("Should delete the channel without error", func() {
					Expect(services[1].DeleteMany(ctx, []channel.Key{idxCh.Key(), ch.Key()}, true)).To(Succeed())
				})
				It("Should not be able to retrieve the channel after deletion", func() {
					Expect(services[1].Delete(ctx, ch.Key(), true)).To(Succeed())
					exists, err := services[2].NewRetrieve().WhereKeys(ch.Key()).Exists(ctx, nil)
					Expect(err).ToNot(HaveOccurred())
					Expect(exists).To(BeFalse())
					Eventually(func(g Gomega) {
						exists, err = services[1].NewRetrieve().WhereKeys(ch.Key()).Exists(ctx, nil)
						g.Expect(err).ToNot(HaveOccurred())
						g.Expect(exists).To(BeFalse())
					}).Should(Succeed())
				})
				It("Should not be able to retrieve the channel from the storage DB", func() {
					Expect(services[1].Delete(ctx, ch.Key(), true)).To(Succeed())
					channels, err := builder.Cores[2].Storage.TS.RetrieveChannels(ctx, ch.Key().StorageKey())
					Expect(err).To(MatchError(cesium.ErrChannelNotFound))
					Expect(channels).To(BeEmpty())
				})
			})
		})
	})
})
