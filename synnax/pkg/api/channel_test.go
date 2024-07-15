// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package api_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("ChannelReader", Ordered, func() {
	var (
		builder        *mock.Builder
		prov           api.Provider
		svc            *api.ChannelService
		createdChannel api.Channel
	)
	BeforeAll(func() {
		builder = mock.Open()
		prov = builder.New(ctx)
		svc = api.NewChannelService(prov)
		res, err := svc.Create(context.TODO(), api.ChannelCreateRequest{
			Channels: []api.Channel{{
				Name:        "test",
				Leaseholder: 1,
				DataType:    telem.Float64T,
				Rate:        25 * telem.Hz,
			}},
		})
		Expect(err).To(BeNil())
		Expect(res.Channels).To(HaveLen(1))
		createdChannel = res.Channels[0]
	})
	AfterAll(func() {
		Expect(builder.Close()).To(Succeed())
		Expect(builder.Cleanup()).To(Succeed())
	})
	Describe("Create", func() {
		DescribeTable("Validation Errors", func(
			ch api.Channel,
			field string,
			message string,
		) {
			_, err := svc.Create(context.TODO(), api.ChannelCreateRequest{
				Channels: []api.Channel{ch},
			})
			Expect(err).To(HaveOccurred())
			//Expect(err.Err).To(HaveOccurred())
			//flds, ok := err.Err.(errors.Fields)
			//Expect(ok).To(BeTrue())
			//Expect(flds[0].Field).To(Equal(field))
			//Expect(flds[0].Message).To(Equal(message))
			//Expect(len(res.Channels)).To(Equal(0))
		},
			Entry("No Data Variant", api.Channel{
				Name:        "test",
				Leaseholder: 1,
				Rate:        25 * telem.Hz,
			}, "channels[0].data_type", "required"),
		)
	})
	Describe("RetrieveP", func() {
		It("Should retrieve all created channels", func() {
			res, err := svc.Retrieve(context.TODO(), api.ChannelRetrieveRequest{})
			Expect(err).To(BeNil())
			Expect(len(res.Channels)).To(BeNumerically(">", 0))
		})
		It("Should retrieve a Channel by its key", func() {
			res, err := svc.Retrieve(context.TODO(), api.ChannelRetrieveRequest{
				Keys: channel.Keys{createdChannel.Key},
			})
			Expect(err).To(BeNil())
			Expect(res.Channels).To(HaveLen(1))
		})
		It("Should retrieve channels by their node Name", func() {
			res, err := svc.Retrieve(context.TODO(), api.ChannelRetrieveRequest{
				NodeKey: 1,
			})
			Expect(err).To(BeNil())
			Expect(res.Channels).To(HaveLen(1))
		})
		It("Should retrieve channels by their name", func() {
			res, err := svc.Retrieve(context.TODO(), api.ChannelRetrieveRequest{
				Names: []string{"test"},
			})
			Expect(err).To(BeNil())
			Expect(res.Channels).To(HaveLen(1))
			for _, ch := range res.Channels {
				Expect(ch.Name).To(Equal("test"))
			}
		})
	})
	Describe("Delete", func() {
		var createdChannels []api.Channel
		BeforeEach(func() {
			res, err := svc.Create(context.TODO(), api.ChannelCreateRequest{
				Channels: []api.Channel{{
					Name:        "test_channel_1",
					Leaseholder: 1,
					DataType:    telem.Float64T,
					Rate:        25 * telem.Hz,
				}, {
					Name:        "test_channel_2",
					Leaseholder: 1,
					DataType:    telem.Float64T,
					Rate:        25 * telem.Hz,
				}, {
					Name:        "test_channel_3",
					Leaseholder: 1,
					DataType:    telem.Float64T,
					Rate:        25 * telem.Hz,
				},
				}})
			Expect(err).To(BeNil())
			Expect(res.Channels).To(HaveLen(3))
			createdChannels = res.Channels
		})
		It("Should delete a channel by its key", func() {
			null, err := svc.Delete(context.TODO(), api.ChannelDeleteRequest{
				Keys: channel.Keys{createdChannels[0].Key},
			})
			Expect(err).To(BeNil())
			Expect(null.Type()).To(BeNil())
			res, err := svc.Retrieve(context.TODO(), api.ChannelRetrieveRequest{
				Keys: channel.Keys{createdChannels[0].Key},
			})
			Expect(err).To(HaveOccurred())
			Expect(res.Channels).To(HaveLen(0))
		})
		It("Should delete a channel by its name", func() {
			null, err := svc.Delete(context.TODO(), api.ChannelDeleteRequest{
				Names: []string{createdChannels[1].Name},
			})
			Expect(err).To(BeNil())
			Expect(null.Type()).To(BeNil())
			res, err := svc.Retrieve(context.TODO(), api.ChannelRetrieveRequest{
				Names: []string{createdChannels[1].Name},
			})
			Expect(err).To(BeNil())
			Expect(res.Channels).To(HaveLen(0))
		})
		It("Should delete a channel that doesn't exist by key with an error", func() {
			null, err := svc.Delete(context.TODO(), api.ChannelDeleteRequest{
				Keys: channel.Keys{createdChannels[0].Key},
			})
			Expect(err).To(BeNil())
			Expect(null.Type()).To(BeNil())
			null, err = svc.Delete(context.TODO(), api.ChannelDeleteRequest{
				Keys: channel.Keys{createdChannels[0].Key},
			})
			Expect(err).To(HaveOccurred())
			Expect(null.Type()).To(BeNil())
		})
		It("Should delete a channel that doesn't exist by name without error", func() {
			for range 2 {
				null, err := svc.Delete(context.TODO(), api.ChannelDeleteRequest{
					Names: []string{"test"},
				})
				Expect(err).To(BeNil())
				Expect(null.Type()).To(BeNil())
			}
		})
		It("Should delete multiple channels by a list of keys and names", func() {
			null, err := svc.Delete(context.TODO(), api.ChannelDeleteRequest{
				Keys:  channel.Keys{createdChannels[0].Key},
				Names: []string{createdChannels[1].Name},
			})
			Expect(err).To(BeNil())
			Expect(null.Type()).To(BeNil())
			res, err := svc.Retrieve(context.TODO(), api.ChannelRetrieveRequest{
				Keys: channel.Keys{createdChannels[0].Key},
			})
			Expect(err).To(HaveOccurred())
			Expect(res.Channels).To(HaveLen(0))
			res, err = svc.Retrieve(context.TODO(), api.ChannelRetrieveRequest{
				Names: []string{createdChannels[1].Name},
			})
			Expect(err).To(BeNil())
			Expect(res.Channels).To(HaveLen(0))
		})
		It("Should give an error when trying to delete an internal channel", func() {
			res, err := svc.Retrieve(context.TODO(), api.ChannelRetrieveRequest{
				Names: []string{
					"sy_ontology_resource_set",
					"sy_ontology_resource_delete",
					"sy_ontology_relationship_set",
					"sy_ontology_relationship_delete",
				},
			})
			Expect(err).To(BeNil())
			Expect(res.Channels).To(HaveLen(4))
			for _, ch := range res.Channels {
				Expect(ch.Internal).To(BeTrue())
			}

			null, err := svc.Delete(context.TODO(), api.ChannelDeleteRequest{
				Keys: channel.Keys{res.Channels[0].Key},
			})
			Expect(err).To(HaveOccurred())
			Expect(null.Type()).To(BeNil())

			null, err = svc.Delete(context.TODO(), api.ChannelDeleteRequest{
				Names: []string{res.Channels[1].Name},
			})
			Expect(err).To(HaveOccurred())
			Expect(null.Type()).To(BeNil())

			null, err = svc.Delete(context.TODO(), api.ChannelDeleteRequest{
				Keys:  channel.Keys{res.Channels[2].Key},
				Names: []string{res.Channels[3].Name},
			})
			Expect(err).To(HaveOccurred())
			Expect(null.Type()).To(BeNil())

			res, err = svc.Retrieve(context.TODO(), api.ChannelRetrieveRequest{
				Keys: channel.Keys{
					res.Channels[0].Key,
					res.Channels[1].Key,
					res.Channels[2].Key,
					res.Channels[3].Key,
				},
			})
			Expect(err).To(BeNil())
			Expect(res.Channels).To(HaveLen(4))
			for _, ch := range res.Channels {
				Expect(ch.Internal).To(BeTrue())
			}
		})
	})
})
