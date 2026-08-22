// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer_test

import (
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/codec"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	svcchannel "github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	svcframer "github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/transport/http/framer"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// readBounds spans every timestamp the specs write.
var readBounds = telem.TimeRange{Start: 0, End: telem.SecondTS * 100}

// createIndexed creates an index channel and one channel of each given data type
// indexed by it.
func createIndexed(
	ctx SpecContext,
	dataTypes ...telem.DataType,
) (svcchannel.Channel, []svcchannel.Channel) {
	GinkgoHelper()
	index := svcchannel.Channel{
		Name:     UniqueChannelName(),
		DataType: telem.TimeStampT,
		IsIndex:  true,
	}
	Expect(channelWriter.Create(ctx, &index)).To(Succeed())
	channels := make([]svcchannel.Channel, len(dataTypes))
	for i, dt := range dataTypes {
		channels[i] = svcchannel.Channel{
			Name:       UniqueChannelName(),
			DataType:   dt,
			LocalIndex: index.LocalKey,
		}
		Expect(channelWriter.Create(ctx, &channels[i])).To(Succeed())
	}
	return index, channels
}

// write commits timestamps to index and one series to each of the given channels.
func write(
	ctx SpecContext,
	index svcchannel.Channel,
	timestamps telem.Series,
	channels []svcchannel.Channel,
	series ...telem.Series,
) {
	GinkgoHelper()
	keys := append(
		svcchannel.Keys{index.Key()},
		svcchannel.KeysFromChannels(channels)...,
	)
	series = append([]telem.Series{timestamps}, series...)
	w := MustSucceed(framerSvc.OpenWriter(ctx, svcframer.WriterConfig{
		Start: telem.ValueAt[telem.TimeStamp](timestamps, 0),
		Keys:  keys,
	}))
	Expect(w.Write(frame.NewMulti(keys, series))).To(BeTrue())
	Expect(w.Close()).To(Succeed())
}

// read opens a response over every sample in the given channels, indexed by index.
func read(
	ctx SpecContext,
	index svcchannel.Channel,
	channels ...svcchannel.Channel,
) framer.ReadResponse {
	GinkgoHelper()
	iter := MustSucceed(framerSvc.OpenIterator(ctx, svcframer.IteratorConfig{
		Keys:   svcchannel.KeysFromChannels(append(channels, index)),
		Bounds: readBounds,
	}))
	return framer.ReadResponse{
		Iterator: iter,
		Channels: channels,
		Indexes:  []svcchannel.Channel{index},
	}
}

// splitRecords splits a read body into the payloads of its data records and the
// payload of its terminator, failing if the body ends without one.
func splitRecords(body []byte) (data [][]byte, terminator []byte) {
	GinkgoHelper()
	for len(body) > 0 {
		Expect(len(body)).To(BeNumerically(">=", 5))
		size := int(telem.ByteOrder.Uint32(body[1:5]))
		kind, payload := body[0], body[5:5+size]
		body = body[5+size:]
		if kind == 1 {
			Expect(body).To(BeEmpty())
			return data, payload
		}
		data = append(data, payload)
	}
	Fail("body ended without a terminator record")
	return data, terminator
}

var _ = Describe("Read", func() {
	Describe("FrameEncoder", func() {
		It("Should emit the frame media type", func() {
			Expect(framer.FrameEncoder.ContentType()).
				To(Equal("application/vnd.synnax.frame"))
		})

		It("Should encode the read as data records and an empty terminator", func(
			ctx SpecContext,
		) {
			index, channels := createIndexed(ctx, telem.Float32T)
			write(
				ctx,
				index,
				telem.NewSeriesSecondsTSV(1, 2, 3),
				channels,
				telem.NewSeriesV[float32](1, 2, 3),
			)
			b := MustSucceed(
				framer.FrameEncoder.Encode(ctx, read(ctx, index, channels...)),
			)
			data, terminator := splitRecords(b)
			Expect(terminator).To(BeEmpty())
			Expect(data).To(HaveLen(1))
			c := codec.NewStatic(
				svcchannel.KeysFromChannels(channels),
				[]telem.DataType{telem.Float32T},
			)
			fr := MustSucceed(c.Decode(data[0]))
			Expect(fr.Count()).To(Equal(1))
			Expect(fr.Get(channels[0].Key()).Series[0]).
				To(telem.MatchWrittenSeries(telem.NewSeriesV[float32](1, 2, 3)))
		})

		It("Should encode a read with no data as a single terminator", func(
			ctx SpecContext,
		) {
			index, channels := createIndexed(ctx, telem.Float32T)
			b := MustSucceed(
				framer.FrameEncoder.Encode(ctx, read(ctx, index, channels...)),
			)
			data, terminator := splitRecords(b)
			Expect(data).To(BeEmpty())
			Expect(terminator).To(BeEmpty())
		})

		It("Should return an error when the value is not a read response", func(
			ctx SpecContext,
		) {
			var buf bytes.Buffer
			Expect(framer.FrameEncoder.EncodeStream(ctx, &buf, 42)).
				To(MatchError(ContainSubstring("cannot encode a value of type int")))
		})
	})
})
