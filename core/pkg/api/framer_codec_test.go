// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/codec"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("FramerCodec", func() {
	Describe("Frame Write Request", func() {
		It("Should encode and decode the request correctly", func() {
			dataTypes := []telem.DataType{"int32"}
			keys := channel.Keys{1}
			v := api.WSFramerCodec{
				Codec:          codec.NewStatic(keys, dataTypes),
				LowerPerfCodec: &binary.JSONCodec{},
			}
			req := api.FrameWriterRequest{
				Command: writer.Write,
				Frame: core.MultiFrame(
					keys,
					[]telem.Series{telem.NewSeriesV[int32](1, 2, 3)},
				),
			}
			msg := fhttp.WSMessage[api.FrameWriterRequest]{Type: "data", Payload: req}
			ctx := context.Background()
			encoded := MustSucceed(v.Encode(ctx, msg))
			Expect(encoded[0]).To(Equal(uint8(255)))
			var resMsg fhttp.WSMessage[api.FrameWriterRequest]
			Expect(v.Decode(ctx, encoded, &resMsg)).To(Succeed())
			frm := resMsg.Payload.Frame
			Expect(resMsg.Type).To(Equal(fhttp.WSMessageTypeData))
			Expect(frm.KeysSlice()).To(Equal([]channel.Key{1}))
			Expect(frm.Count()).To(Equal(1))
			Expect(frm.SeriesAt(0)).To(
				telem.MatchSeriesData(telem.NewSeriesV[int32](1, 2, 3)),
			)
		})
	})
	Describe("Frame Stream Response", func() {
		It("Should encode and decode the response correctly", func() {
			dataTypes := []telem.DataType{"int32"}
			keys := channel.Keys{1}
			v := api.WSFramerCodec{
				Codec:          codec.NewStatic(keys, dataTypes),
				LowerPerfCodec: &binary.JSONCodec{},
			}
			res := api.FrameStreamerResponse{
				Frame: core.MultiFrame(
					keys,
					[]telem.Series{telem.NewSeriesV[int32](1, 2, 3)},
				),
			}
			msg := fhttp.WSMessage[api.FrameStreamerResponse]{Type: "data", Payload: res}
			ctx := context.Background()
			encoded := MustSucceed(v.Encode(ctx, msg))
			Expect(encoded[0]).To(Equal(uint8(255)))
			var resMsg fhttp.WSMessage[api.FrameStreamerResponse]
			Expect(v.Decode(ctx, encoded, &resMsg)).To(Succeed())
			frm := resMsg.Payload.Frame
			Expect(resMsg.Type).To(Equal(fhttp.WSMessageTypeData))
			Expect(frm.KeysSlice()).To(Equal([]channel.Key{1}))
			Expect(frm.Count()).To(Equal(1))
			Expect(frm.SeriesAt(0)).To(telem.MatchSeriesData(telem.NewSeriesV[int32](1, 2, 3)))
		})
	})
})
