package api_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter/fhttp"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/codec"
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
			cd := codec.NewCodec(dataTypes, keys)
			v := api.WSFramerCodec{BaseCodec: &cd, LowerPerfCodec: &binary.JSONCodec{}}
			req := api.FrameWriterRequest{
				Command: writer.Write,
				Frame: api.Frame{
					Keys:   keys,
					Series: []telem.Series{telem.NewSeriesV[int32](1, 2, 3)},
				},
			}
			msg := fhttp.WSMessage[api.FrameWriterRequest]{Type: "data", Payload: req}
			encoded := MustSucceed(v.Encode(ctx, msg))
			Expect(encoded[0]).To(Equal(uint8(255)))
			var resMsg fhttp.WSMessage[api.FrameWriterRequest]
			Expect(v.Decode(ctx, encoded, &resMsg)).To(Succeed())
			Expect(resMsg.Type).To(Equal(fhttp.WSMsgTypeData))
			Expect(resMsg.Payload.Frame.Keys).To(Equal(channel.Keys{1}))
			Expect(len(resMsg.Payload.Frame.Series)).To(Equal(1))
			Expect(resMsg.Payload.Frame.Series[0].Data).To(
				Equal(telem.NewSeriesV[int32](1, 2, 3).Data),
			)
		})
	})
	Describe("Frame Stream Response", func() {
		It("Should encode and decode the response correctly", func() {
			dataTypes := []telem.DataType{"int32"}
			keys := channel.Keys{1}
			cd := codec.NewCodec(dataTypes, keys)
			v := api.WSFramerCodec{BaseCodec: &cd, LowerPerfCodec: &binary.JSONCodec{}}
			res := api.FrameStreamerResponse{
				Frame: api.Frame{
					Keys:   keys,
					Series: []telem.Series{telem.NewSeriesV[int32](1, 2, 3)},
				},
			}
			msg := fhttp.WSMessage[api.FrameStreamerResponse]{Type: "data", Payload: res}
			encoded := MustSucceed(v.Encode(ctx, msg))
			Expect(encoded[0]).To(Equal(uint8(255)))
			var resMsg fhttp.WSMessage[api.FrameStreamerResponse]
			Expect(v.Decode(ctx, encoded, &resMsg)).To(Succeed())
			Expect(resMsg.Type).To(Equal(fhttp.WSMsgTypeData))
			Expect(resMsg.Payload.Frame.Keys).To(Equal(channel.Keys{1}))
			Expect(len(resMsg.Payload.Frame.Series)).To(Equal(1))
			Expect(resMsg.Payload.Frame.Series[0].Data).To(
				Equal(telem.NewSeriesV[int32](1, 2, 3).Data),
			)
		})
	})
})
