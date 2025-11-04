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
	"context"

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
		ctx := context.Background()
		b, err := codec.Encode(ctx, toEncode{1})
		Expect(err).ToNot(HaveOccurred())
		var d toEncode
		Expect(codec.Decode(ctx, b, &d)).To(Succeed())
		Expect(d.Value).To(Equal(1))
		var d2 toEncode
		Expect(codec.DecodeStream(ctx, bytes.NewReader(b), &d2)).To(Succeed())
		Expect(d2.Value).To(Equal(1))
	},
		Entry("Gob", &binary.GobCodec{}),
		Entry("JSON", &binary.JSONCodec{}),
		Entry("MsgPack", &binary.MsgPackCodec{}),
	)
	Describe("Additional Error Info", func() {
		DescribeTable("Standard Type", func(codec binary.Codec) {
			_, err := codec.Encode(ctx, make(chan int))
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
			_, err := codec.Encode(ctx, custom{Chan: make(chan int)})
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
			jsonB := MustSucceed(js.Encode(ctx, v))
			gobB := MustSucceed(gb.Encode(ctx, v))
			var res abc
			fbc := binary.NewDecodeFallbackCodec(&binary.GobCodec{}, &binary.JSONCodec{})
			Expect(fbc.Decode(ctx, jsonB, &res)).To(Succeed())
			Expect(res.Value).To(Equal(12))
			Expect(fbc.Decode(ctx, gobB, &res)).To(Succeed())
			Expect(res.Value).To(Equal(12))
		})
		It("Should return the error of the last decoder if all codecs fail", func() {
			fbc := binary.NewDecodeFallbackCodec(&binary.GobCodec{}, &binary.JSONCodec{})
			_, err := fbc.Encode(ctx, make(chan int))
			Expect(err).To(HaveOccurred())
		})
		It("Should handle DecodeStream fallback correctly", func() {
			js := &binary.JSONCodec{}
			type abc struct {
				Value int `json:"value"`
			}
			v := abc{Value: 12}
			jsonB := MustSucceed(js.Encode(ctx, v))

			var res abc
			fbc := binary.NewDecodeFallbackCodec(&binary.MsgPackCodec{}, &binary.JSONCodec{})

			// Create a bytes.Buffer that implements io.Reader
			buf := bytes.NewBuffer(jsonB)
			Expect(fbc.DecodeStream(ctx, buf, &res)).To(Succeed())
			Expect(res.Value).To(Equal(12))
		})

		It("Should return error when DecodeStream fails for all codecs", func() {
			fbc := binary.NewDecodeFallbackCodec(&binary.GobCodec{}, &binary.JSONCodec{})

			invalidData := []byte("completely invalid data")
			var res struct{ Value int }
			err := fbc.DecodeStream(ctx, bytes.NewReader(invalidData), &res)
			Expect(err).To(HaveOccurred())
		})
	})
	Describe("Tracing", func() {
		It("Should properly wrap encoding and decoding operations", func() {
			underlying := &binary.GobCodec{}
			codec := &binary.TracingCodec{
				Codec: underlying,
			}

			// Test encoding
			b, err := codec.Encode(ctx, toEncode{1})
			Expect(err).ToNot(HaveOccurred())

			// Test decoding
			var d toEncode
			Expect(codec.Decode(ctx, b, &d)).To(Succeed())
			Expect(d.Value).To(Equal(1))

			// Test stream decoding
			var d2 toEncode
			Expect(codec.DecodeStream(ctx, bytes.NewReader(b), &d2)).To(Succeed())
			Expect(d2.Value).To(Equal(1))
		})

		It("Should properly handle encoding errors", func() {
			underlying := &binary.JSONCodec{}
			codec := &binary.TracingCodec{
				Codec: underlying,
			}

			// Try to encode an unencodable type
			_, err := codec.Encode(ctx, make(chan int))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to encode value"))
		})

		It("Should properly handle decoding errors", func() {
			underlying := &binary.JSONCodec{}
			codec := &binary.TracingCodec{
				Codec: underlying,
			}

			invalidData := []byte("invalid json")
			var d toEncode
			err := codec.Decode(ctx, invalidData, &d)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to decode"))

			err = codec.DecodeStream(ctx, bytes.NewReader(invalidData), &d)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to decode"))
		})
	})
	Describe("String Number Unmarshaling", func() {
		DescribeTable("UnmarshalJSONStringInt64", func(input string, expected int64, shouldError bool) {
			b := []byte(input)
			val, err := binary.UnmarshalJSONStringInt64(b)
			if shouldError {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).ToNot(HaveOccurred())
				Expect(val).To(Equal(expected))
			}
		},
			Entry("direct number", `123`, int64(123), false),
			Entry("string number", `"123"`, int64(123), false),
			Entry("negative number", `-123`, int64(-123), false),
			Entry("negative string", `"-123"`, int64(-123), false),
			Entry("max int64", `9223372036854775807`, int64(9223372036854775807), false),
			Entry("invalid string", `"abc"`, int64(0), true),
			Entry("invalid json", `{invalid}`, int64(0), true),
		)

		DescribeTable("UnmarshalJSONStringUint64", func(input string, expected uint64, shouldError bool) {
			b := []byte(input)
			val, err := binary.UnmarshalJSONStringUint64(b)
			if shouldError {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).ToNot(HaveOccurred())
				Expect(val).To(Equal(expected))
			}
		},
			Entry("direct number", `123`, uint64(123), false),
			Entry("string number", `"123"`, uint64(123), false),
			Entry("max uint64", `18446744073709551615`, uint64(18446744073709551615), false),
			Entry("negative number", `-123`, uint64(0), true),
			Entry("negative string", `"-123"`, uint64(0), true),
			Entry("invalid string", `"abc"`, uint64(0), true),
			Entry("invalid json", `{invalid}`, uint64(0), true),
		)
	})
	Describe("MustEncodeJSONToString", func() {
		It("Should encode valid values to JSON string", func() {
			type testStruct struct {
				Value string `json:"value"`
			}
			str := binary.MustEncodeJSONToString(testStruct{Value: "test"})
			Expect(str).To(Equal(`{"value":"test"}`))
		})

		It("Should panic on unencodable values", func() {
			Expect(func() {
				binary.MustEncodeJSONToString(make(chan int))
			}).To(Panic())
		})
	})

	Describe("MarshalStringInt64", func() {
		It("Should encode an int64 value as a string", func() {
			Expect(binary.MarshalStringInt64(12)).To(Equal([]byte("\"12\"")))
			Expect(binary.MarshalStringInt64(-1)).To(Equal([]byte("\"-1\"")))
		})
	})

	Describe("MarshalStringUint64", func() {
		It("Should encode a uint64 value as a string", func() {
			Expect(binary.MarshalStringUint64(12)).To(Equal([]byte("\"12\"")))
		})
	})
})
