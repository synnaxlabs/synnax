// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package binary_test

import (
	"bytes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/binary"
	. "github.com/synnaxlabs/x/testutil"
)

type toEncode struct {
	Value int
}

var _ = Describe("Codec", func() {
	DescribeTable("Encode + Decode", func(codec binary.Codec) {
		b, err := codec.Encode(nil, toEncode{1})
		Expect(err).ToNot(HaveOccurred())
		var d toEncode
		Expect(codec.Decode(ctx, b, &d)).To(Succeed())
		Expect(d.Value).To(Equal(1))
		var d2 toEncode
		Expect(codec.DecodeStream(nil, bytes.NewReader(b), &d2)).To(Succeed())
		Expect(d2.Value).To(Equal(1))
	},
		Entry("Gob", &binary.GobCodec{}),
		Entry("JSON", &binary.JSONCodec{}),
		Entry("MsgPack", &binary.MsgPackCodec{}),
		Entry("PassThrough", &binary.PassThroughCodec{Codec: &binary.GobCodec{}}),
	)
	Describe("PassThrough encoding and decoding", func() {
		It("Should pass through the encoding and decoding when a byte slice is provided", func() {
			codec := &binary.PassThroughCodec{Codec: &binary.GobCodec{}}
			b, err := codec.Encode(nil, []byte{1, 2, 3})
			Expect(err).ToNot(HaveOccurred())
			Expect(b).To(Equal([]byte{1, 2, 3}))
			var d []byte
			Expect(codec.Decode(nil, b, &d)).To(Succeed())
			Expect(d).To(Equal([]byte{1, 2, 3}))
		})
	})
	Describe("Additional Error Info", func() {
		DescribeTable("Standard Type", func(codec binary.Codec) {
			_, err := codec.Encode(nil, make(chan int))
			Expect(err).To(HaveOccurred())
			msg := err.Error()
			Expect(msg).To(ContainSubstring("failed to encode value"))
			Expect(msg).To(ContainSubstring("kind=chan, type=chan int"))
		},
			Entry("Gob", &binary.GobCodec{}),
			Entry("JSON", &binary.JSONCodec{}),
			Entry("MsgPack", &binary.MsgPackCodec{}),
		)
		DescribeTable("Custom Type", func(codec binary.Codec) {
			type custom struct {
				Value int
				Chan  chan int
			}
			_, err := codec.Encode(nil, custom{Chan: make(chan int)})
			Expect(err).To(HaveOccurred())
			msg := err.Error()
			Expect(msg).To(ContainSubstring("failed to encode value"))
			Expect(msg).To(ContainSubstring("kind=struct, type=binary_test.custom"))
		},
			// Explicit exclusion of Gob because it can encode arbitrary go types
			Entry("JSON", &binary.JSONCodec{}),
			Entry("MsgPack", &binary.MsgPackCodec{}),
		)
	})
	Describe("Fallback", func() {
		It("Should fallback to the next codec when the first one fails", func() {
			js := &binary.JSONCodec{}
			gb := &binary.GobCodec{}
			type abc struct {
				Value int `json:"value"`
			}
			v := abc{Value: 12}
			jsonB := MustSucceed(js.Encode(nil, v))
			gobB := MustSucceed(gb.Encode(nil, v))
			var res abc
			fbc := &binary.DecodeFallbackCodec{
				Codecs: []binary.Codec{
					&binary.GobCodec{},
					&binary.JSONCodec{},
				},
			}
			Expect(fbc.Decode(nil, jsonB, &res)).To(Succeed())
			Expect(res.Value).To(Equal(12))
			Expect(fbc.Decode(nil, gobB, &res)).To(Succeed())
			Expect(res.Value).To(Equal(12))
		})
		It("Should return the error of the last decoder if all codecs fail", func() {
			fbc := &binary.DecodeFallbackCodec{
				Codecs: []binary.Codec{
					&binary.GobCodec{},
					&binary.JSONCodec{},
				},
			}
			_, err := fbc.Encode(nil, make(chan int))
			Expect(err).To(HaveOccurred())
		})
	})
})
