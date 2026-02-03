// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/freighter/fhttp"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/codec"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("FramerCodec", Ordered, func() {
	var (
		ctx         context.Context
		mockCluster *mock.Cluster
		dist        *distribution.Layer
	)
	BeforeAll(func() {
		ctx = context.Background()
		mockCluster = mock.ProvisionCluster(ctx, 1)
		dist = mockCluster.Nodes[1].Layer
	})
	AfterAll(func() {
		Expect(dist.Close()).To(Succeed())
		Expect(mockCluster.Close()).To(Succeed())
	})
	Describe("Frame Write Request", func() {
		It("Should encode and decode single channel int32", func() {
			keys := channel.Keys{1}
			v := api.WSFramerCodec{
				Codec:          codec.NewStatic(keys, []telem.DataType{"int32"}),
				LowerPerfCodec: &binary.JSONCodec{},
			}
			req := api.FrameWriterRequest{
				Command: writer.CommandWrite,
				Frame:   frame.NewMulti(keys, []telem.Series{telem.NewSeriesV[int32](1, 2, 3)}),
			}
			msg := fhttp.WSMessage[api.FrameWriterRequest]{Type: "data", Payload: req}
			encoded := MustSucceed(v.Encode(ctx, msg))
			Expect(encoded[0]).To(Equal(uint8(255)))
			var resMsg fhttp.WSMessage[api.FrameWriterRequest]
			Expect(v.Decode(ctx, encoded, &resMsg)).To(Succeed())
			Expect(resMsg.Type).To(Equal(fhttp.WSMessageTypeData))
			Expect(resMsg.Payload.Command).To(Equal(writer.CommandWrite))
			Expect(resMsg.Payload.Frame.KeysSlice()).To(Equal([]channel.Key{1}))
			Expect(resMsg.Payload.Frame.Count()).To(Equal(1))
			Expect(resMsg.Payload.Frame.SeriesAt(0)).To(telem.MatchSeriesData(telem.NewSeriesV[int32](1, 2, 3)))
		})

		It("Should encode and decode multiple channels", func() {
			keys := channel.Keys{1, 2, 3}
			v := api.WSFramerCodec{
				Codec:          codec.NewStatic(keys, []telem.DataType{"int32", "float32", "uint64"}),
				LowerPerfCodec: &binary.JSONCodec{},
			}
			req := api.FrameWriterRequest{
				Command: writer.CommandWrite,
				Frame: frame.NewMulti(keys, []telem.Series{
					telem.NewSeriesV[int32](1, 2),
					telem.NewSeriesV[float32](1.1, 2.2),
					telem.NewSeriesV[uint64](100, 200),
				}),
			}
			msg := fhttp.WSMessage[api.FrameWriterRequest]{Type: "data", Payload: req}
			encoded := MustSucceed(v.Encode(ctx, msg))
			Expect(encoded[0]).To(Equal(uint8(255)))
			var resMsg fhttp.WSMessage[api.FrameWriterRequest]
			Expect(v.Decode(ctx, encoded, &resMsg)).To(Succeed())
			Expect(resMsg.Payload.Frame.KeysSlice()).To(Equal([]channel.Key{1, 2, 3}))
			Expect(resMsg.Payload.Frame.Count()).To(Equal(3))
			Expect(resMsg.Payload.Frame.SeriesAt(0)).To(telem.MatchSeriesData(telem.NewSeriesV[int32](1, 2)))
			Expect(resMsg.Payload.Frame.SeriesAt(1)).To(telem.MatchSeriesData(telem.NewSeriesV[float32](1.1, 2.2)))
			Expect(resMsg.Payload.Frame.SeriesAt(2)).To(telem.MatchSeriesData(telem.NewSeriesV[uint64](100, 200)))
		})

		It("Should encode and decode open command", func() {
			channels := []channel.Channel{
				{
					Name:     channel.NewRandomName(),
					DataType: telem.Int64T,
					Virtual:  true,
				},
				{
					Name:     channel.NewRandomName(),
					DataType: telem.Int64T,
					Virtual:  true,
				},
			}
			Expect(dist.Channel.CreateMany(ctx, &channels)).To(Succeed())
			keys := channel.KeysFromChannels(channels)
			cdec := codec.NewDynamic(dist.Channel)
			v := api.WSFramerCodec{
				Codec:          cdec,
				LowerPerfCodec: &binary.JSONCodec{},
			}
			req := api.FrameWriterRequest{Command: writer.CommandOpen, Config: api.FrameWriterConfig{Keys: keys}}
			msg := fhttp.WSMessage[api.FrameWriterRequest]{Type: "data", Payload: req}
			encoded := MustSucceed(v.Encode(ctx, msg))
			var resMsg fhttp.WSMessage[api.FrameWriterRequest]
			Expect(v.Decode(ctx, encoded, &resMsg)).To(Succeed())
			Expect(cdec.Initialized()).To(BeTrue())
			Expect(resMsg.Payload.Command).To(Equal(writer.CommandOpen))
			Expect(resMsg.Payload.Config.Keys).To(Equal(keys))
		})

		It("Should encode and decode open message", func() {
			v := api.WSFramerCodec{
				Codec:          codec.NewStatic(channel.Keys{1}, []telem.DataType{"int32"}),
				LowerPerfCodec: &binary.JSONCodec{},
			}
			msg := fhttp.WSMessage[api.FrameWriterRequest]{Type: fhttp.WSMessageTypeOpen}
			encoded := MustSucceed(v.Encode(ctx, msg))
			var resMsg fhttp.WSMessage[api.FrameWriterRequest]
			Expect(v.Decode(ctx, encoded, &resMsg)).To(Succeed())
			Expect(resMsg.Type).To(Equal(fhttp.WSMessageTypeOpen))
		})

		It("Should encode and decode close message", func() {
			v := api.WSFramerCodec{
				Codec:          codec.NewStatic(channel.Keys{1}, []telem.DataType{"int32"}),
				LowerPerfCodec: &binary.JSONCodec{},
			}
			msg := fhttp.WSMessage[api.FrameWriterRequest]{Type: fhttp.WSMessageTypeClose}
			encoded := MustSucceed(v.Encode(ctx, msg))
			var resMsg fhttp.WSMessage[api.FrameWriterRequest]
			Expect(v.Decode(ctx, encoded, &resMsg)).To(Succeed())
			Expect(resMsg.Type).To(Equal(fhttp.WSMessageTypeClose))
		})
	})

	Describe("Frame Writer Response", func() {
		It("Should encode and decode response", func() {
			v := api.WSFramerCodec{
				Codec:          codec.NewStatic(channel.Keys{1}, []telem.DataType{"int32"}),
				LowerPerfCodec: &binary.JSONCodec{},
			}
			res := api.FrameWriterResponse{Command: writer.CommandWrite, Authorized: true}
			msg := fhttp.WSMessage[api.FrameWriterResponse]{Type: fhttp.WSMessageTypeData, Payload: res}
			encoded := MustSucceed(v.Encode(ctx, msg))
			var resMsg fhttp.WSMessage[api.FrameWriterResponse]
			Expect(v.Decode(ctx, encoded, &resMsg)).To(Succeed())
			Expect(resMsg.Type).To(Equal(fhttp.WSMessageTypeData))
			Expect(resMsg.Payload.Command).To(Equal(writer.CommandWrite))
			Expect(resMsg.Payload.Authorized).To(BeTrue())
		})
	})

	Describe("Frame Streamer Request", func() {
		It("Should encode and decode request", func() {
			channels := []channel.Channel{
				{
					Name:     channel.NewRandomName(),
					DataType: telem.Int64T,
					Virtual:  true,
				},
				{
					Name:     channel.NewRandomName(),
					DataType: telem.Int64T,
					Virtual:  true,
				},
			}
			Expect(dist.Channel.CreateMany(ctx, &channels)).To(Succeed())
			keys := channel.KeysFromChannels(channels)
			cdec := codec.NewDynamic(dist.Channel)
			v := api.WSFramerCodec{
				Codec:          cdec,
				LowerPerfCodec: &binary.JSONCodec{},
			}
			req := api.FrameStreamerRequest{Keys: keys}
			msg := fhttp.WSMessage[api.FrameStreamerRequest]{Type: "data", Payload: req}
			encoded := MustSucceed(v.Encode(ctx, msg))
			var resMsg fhttp.WSMessage[api.FrameStreamerRequest]
			Expect(v.Decode(ctx, encoded, &resMsg)).To(Succeed())
			Expect(resMsg.Type).To(Equal(fhttp.WSMessageTypeData))
			Expect(resMsg.Payload.Keys).To(Equal(keys))
		})
	})

	Describe("Frame Stream Response", func() {
		It("Should encode and decode single channel int32", func() {
			keys := channel.Keys{1}
			v := api.WSFramerCodec{
				Codec:          codec.NewStatic(keys, []telem.DataType{"int32"}),
				LowerPerfCodec: &binary.JSONCodec{},
			}
			res := api.FrameStreamerResponse{
				Frame: frame.NewMulti(keys, []telem.Series{telem.NewSeriesV[int32](1, 2, 3)}),
			}
			msg := fhttp.WSMessage[api.FrameStreamerResponse]{Type: "data", Payload: res}
			encoded := MustSucceed(v.Encode(ctx, msg))
			Expect(encoded[0]).To(Equal(uint8(255)))
			var resMsg fhttp.WSMessage[api.FrameStreamerResponse]
			Expect(v.Decode(ctx, encoded, &resMsg)).To(Succeed())
			Expect(resMsg.Type).To(Equal(fhttp.WSMessageTypeData))
			Expect(resMsg.Payload.Frame.KeysSlice()).To(Equal([]channel.Key{1}))
			Expect(resMsg.Payload.Frame.Count()).To(Equal(1))
			Expect(resMsg.Payload.Frame.SeriesAt(0)).To(telem.MatchSeriesData(telem.NewSeriesV[int32](1, 2, 3)))
		})

		It("Should encode and decode multiple channels", func() {
			keys := channel.Keys{1, 2}
			v := api.WSFramerCodec{
				Codec:          codec.NewStatic(keys, []telem.DataType{"float64", "int64"}),
				LowerPerfCodec: &binary.JSONCodec{},
			}
			res := api.FrameStreamerResponse{
				Frame: frame.NewMulti(keys, []telem.Series{
					telem.NewSeriesV(1.5, 2.5, 3.5),
					telem.NewSeriesV[int64](1000, 2000, 3000),
				}),
			}
			msg := fhttp.WSMessage[api.FrameStreamerResponse]{Type: "data", Payload: res}
			encoded := MustSucceed(v.Encode(ctx, msg))
			var resMsg fhttp.WSMessage[api.FrameStreamerResponse]
			Expect(v.Decode(ctx, encoded, &resMsg)).To(Succeed())
			Expect(resMsg.Payload.Frame.KeysSlice()).To(Equal([]channel.Key{1, 2}))
			Expect(resMsg.Payload.Frame.Count()).To(Equal(2))
			Expect(resMsg.Payload.Frame.SeriesAt(0)).To(telem.MatchSeriesData(telem.NewSeriesV(1.5, 2.5, 3.5)))
			Expect(resMsg.Payload.Frame.SeriesAt(1)).To(telem.MatchSeriesData(telem.NewSeriesV[int64](1000, 2000, 3000)))
		})
		It("Should encode and decode empty frame", func() {
			v := api.WSFramerCodec{
				Codec:          codec.NewStatic(channel.Keys{1}, []telem.DataType{"int32"}),
				LowerPerfCodec: &binary.JSONCodec{},
			}
			res := api.FrameStreamerResponse{Frame: frame.Frame{}}
			msg := fhttp.WSMessage[api.FrameStreamerResponse]{Type: "data", Payload: res}
			encoded := MustSucceed(v.Encode(ctx, msg))
			var resMsg fhttp.WSMessage[api.FrameStreamerResponse]
			Expect(v.Decode(ctx, encoded, &resMsg)).To(Succeed())
			Expect(resMsg.Type).To(Equal(fhttp.WSMessageTypeData))
			Expect(resMsg.Payload.Frame.Empty()).To(BeTrue())
		})

		It("Should encode and decode open message", func() {
			v := api.WSFramerCodec{
				Codec:          codec.NewStatic(channel.Keys{1}, []telem.DataType{"int32"}),
				LowerPerfCodec: &binary.JSONCodec{},
			}
			msg := fhttp.WSMessage[api.FrameStreamerResponse]{Type: fhttp.WSMessageTypeOpen}
			encoded := MustSucceed(v.Encode(ctx, msg))
			var resMsg fhttp.WSMessage[api.FrameStreamerResponse]
			Expect(v.Decode(ctx, encoded, &resMsg)).To(Succeed())
			Expect(resMsg.Type).To(Equal(fhttp.WSMessageTypeOpen))
		})

		It("Should encode and decode close message", func() {
			v := api.WSFramerCodec{
				Codec:          codec.NewStatic(channel.Keys{1}, []telem.DataType{"int32"}),
				LowerPerfCodec: &binary.JSONCodec{},
			}
			msg := fhttp.WSMessage[api.FrameStreamerResponse]{Type: fhttp.WSMessageTypeClose}
			encoded := MustSucceed(v.Encode(ctx, msg))
			var resMsg fhttp.WSMessage[api.FrameStreamerResponse]
			Expect(v.Decode(ctx, encoded, &resMsg)).To(Succeed())
			Expect(resMsg.Type).To(Equal(fhttp.WSMessageTypeClose))
		})
	})
})
