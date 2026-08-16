// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package msgpack_test

import (
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xmsgpack "github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/vmihailenco/msgpack/v5"
)

type toEncode struct {
	Value int
}

var _ = Describe("Codec", func() {
	Describe("ContentType", func() {
		It("Should report the MessagePack content type", func() {
			Expect(xmsgpack.Codec.ContentType()).To(Equal("application/msgpack"))
		})
	})
	It("Should encode and decode", func(ctx SpecContext) {
		b := MustSucceed(xmsgpack.Codec.Encode(ctx, toEncode{1}))
		var d toEncode
		Expect(xmsgpack.Codec.Decode(ctx, b, &d)).To(Succeed())
		Expect(d).To(Equal(toEncode{1}))
		var d2 toEncode
		Expect(xmsgpack.Codec.DecodeStream(ctx, bytes.NewReader(b), &d2)).To(Succeed())
		Expect(d2).To(Equal(toEncode{1}))
	})
	It("Should add error info with custom type", func(ctx SpecContext) {
		type custom struct {
			Chan  chan int
			Value int
		}
		Expect(xmsgpack.Codec.Encode(ctx, custom{Chan: make(chan int)})).Error().
			To(MatchError(
				SatisfyAll(
					ContainSubstring("failed to encode value"),
					ContainSubstring("kind=struct, type=msgpack_test.custom"),
				),
			))
	})
	It("Should include a stack trace on encoding errors", func(ctx SpecContext) {
		_, err := xmsgpack.Codec.Encode(ctx, make(chan int))
		Expect(err).To(HaveOccurred())
		stack := errors.GetStackTrace(err)
		Expect(stack.String()).ToNot(BeEmpty())
		Expect(stack.String()).To(ContainSubstring(".go"))
	})
	It("Should include a stack trace on decoding errors", func(ctx SpecContext) {
		var d toEncode
		err := xmsgpack.Codec.Decode(ctx, []byte("invalid"), &d)
		Expect(err).To(HaveOccurred())
		stack := errors.GetStackTrace(err)
		Expect(stack.String()).ToNot(BeEmpty())
		Expect(stack.String()).To(ContainSubstring(".go"))
	})
})

var _ = Describe("EncodedJSON", func() {
	It("Should round-trip through msgpack encoding and decoding", func() {
		original := xmsgpack.EncodedJSON{"key": "value", "count": int64(42)}
		b := MustSucceed(msgpack.Marshal(original))
		var result xmsgpack.EncodedJSON
		Expect(msgpack.Unmarshal(b, &result)).To(Succeed())
		Expect(result).To(Equal(original))
	})
	It("Should decode a JSON string for backwards compatibility", func() {
		jsonStr := `{"key":"value","count":42}`
		b := MustSucceed(msgpack.Marshal(jsonStr))
		var result xmsgpack.EncodedJSON
		Expect(msgpack.Unmarshal(b, &result)).To(Succeed())
		Expect(result).To(Equal(xmsgpack.EncodedJSON{"key": "value", "count": 42.0}))
	})
	It(
		"Should decode a struct field from a JSON string for backwards compatibility",
		func() {
			type oldConfig struct {
				Name   string `msgpack:"name"`
				Schema string `msgpack:"schema"`
			}
			type newConfig struct {
				Name   string               `msgpack:"name"`
				Schema xmsgpack.EncodedJSON `msgpack:"schema"`
			}
			old := oldConfig{
				Name:   "test",
				Schema: `{"field":"value"}`,
			}
			b := MustSucceed(msgpack.Marshal(old))
			var result newConfig
			Expect(msgpack.Unmarshal(b, &result)).To(Succeed())
			Expect(result).To(Equal(newConfig{
				Name:   "test",
				Schema: xmsgpack.EncodedJSON{"field": "value"},
			}))
		},
	)
	It("Should decode nil to nil", func() {
		b := MustSucceed(msgpack.Marshal(nil))
		var result xmsgpack.EncodedJSON
		Expect(msgpack.Unmarshal(b, &result)).To(Succeed())
		Expect(result).To(BeNil())
	})
	It("Should handle map[any]any with string keys", func() {
		m := map[string]any{"key": "value"}
		b := MustSucceed(msgpack.Marshal(m))
		dec := msgpack.NewDecoder(bytes.NewReader(b))
		dec.SetMapDecoder(func(dec *msgpack.Decoder) (any, error) {
			return dec.DecodeUntypedMap()
		})
		var result xmsgpack.EncodedJSON
		Expect(result.DecodeMsgpack(dec)).To(Succeed())
		Expect(result["key"]).To(Equal("value"))
	})
	It("Should work as a struct field with map data", func() {
		type config struct {
			Name   string               `msgpack:"name"`
			Schema xmsgpack.EncodedJSON `msgpack:"schema"`
		}
		original := config{
			Name:   "test",
			Schema: xmsgpack.EncodedJSON{"field": "value"},
		}
		b := MustSucceed(msgpack.Marshal(original))
		var result config
		Expect(msgpack.Unmarshal(b, &result)).To(Succeed())
		Expect(result).To(Equal(original))
	})
	It("Should decode an empty string to an empty map", func() {
		b := MustSucceed(msgpack.Marshal(""))
		var result xmsgpack.EncodedJSON
		Expect(msgpack.Unmarshal(b, &result)).To(Succeed())
		Expect(result).To(BeEmpty())
	})
	It("Should decode an empty string struct field to an empty map", func() {
		type oldConfig struct {
			Name   string `msgpack:"name"`
			Schema string `msgpack:"schema"`
		}
		type newConfig struct {
			Name   string               `msgpack:"name"`
			Schema xmsgpack.EncodedJSON `msgpack:"schema"`
		}
		old := oldConfig{Name: "test", Schema: ""}
		b := MustSucceed(msgpack.Marshal(old))
		var result newConfig
		Expect(msgpack.Unmarshal(b, &result)).To(Succeed())
		Expect(result).To(Equal(newConfig{
			Name:   "test",
			Schema: xmsgpack.EncodedJSON{},
		}))
	})
	It("Should return an error for an invalid JSON string", func() {
		b := MustSucceed(msgpack.Marshal("not valid json"))
		var result xmsgpack.EncodedJSON
		Expect(
			msgpack.Unmarshal(b, &result),
		).To(MatchError(ContainSubstring("failed to unmarshal JSON string")))
	})
	It("Should return an error for unsupported types", func() {
		b := MustSucceed(msgpack.Marshal(42))
		var result xmsgpack.EncodedJSON
		Expect(
			msgpack.Unmarshal(b, &result),
		).To(MatchError(ContainSubstring("unsupported type")))
	})
	It("Should return an error for non-string map keys", func() {
		m := map[int]string{1: "a"}
		b := MustSucceed(msgpack.Marshal(m))
		dec := msgpack.NewDecoder(bytes.NewReader(b))
		dec.SetMapDecoder(func(dec *msgpack.Decoder) (any, error) {
			return dec.DecodeUntypedMap()
		})
		var result xmsgpack.EncodedJSON
		Expect(
			result.DecodeMsgpack(dec),
		).To(MatchError(ContainSubstring("non-string key")))
	})
	Describe("Unmarshal", func() {
		It("Should unmarshal into a typed struct", func() {
			type config struct {
				Name    string `json:"name"`
				Count   int    `json:"count"`
				Enabled bool   `json:"enabled"`
			}
			m := xmsgpack.EncodedJSON{
				"name":    "test",
				"count":   float64(42),
				"enabled": true,
			}
			var cfg config
			Expect(m.Unmarshal(&cfg)).To(Succeed())
			Expect(cfg).To(Equal(config{
				Name:    "test",
				Count:   42,
				Enabled: true,
			}))
		})
		It("Should handle nil map", func() {
			var m xmsgpack.EncodedJSON
			type config struct {
				Name string `json:"name"`
			}
			var cfg config
			Expect(m.Unmarshal(&cfg)).To(Succeed())
			Expect(cfg).To(Equal(config{}))
		})
		It("Should return an error for incompatible types", func() {
			m := xmsgpack.EncodedJSON{"count": "not a number"}
			type config struct {
				Count int `json:"count"`
			}
			Expect(m.Unmarshal(&config{})).ToNot(Succeed())
		})
	})
	It("Should work with Codec", func(ctx SpecContext) {
		jsonStr := `{"name":"test","value":123}`
		b := MustSucceed(xmsgpack.Codec.Encode(ctx, jsonStr))
		var result xmsgpack.EncodedJSON
		Expect(xmsgpack.Codec.Decode(ctx, b, &result)).To(Succeed())
		Expect(result).To(Equal(xmsgpack.EncodedJSON{"name": "test", "value": 123.0}))
	})
})

var _ = Describe("UnmarshalUint32", func() {
	DescribeTable("Should decode various types to uint32",
		func(value any, expected uint32) {
			b := MustSucceed(msgpack.Marshal(value))
			dec := msgpack.NewDecoder(bytes.NewReader(b))
			result := MustSucceed(xmsgpack.UnmarshalUint32(dec))
			Expect(result).To(Equal(expected))
		},
		Entry("uint64", uint64(123456), uint32(123456)),
		Entry("uint32", uint32(123456), uint32(123456)),
		Entry("uint16", uint16(1234), uint32(1234)),
		Entry("uint8", uint8(123), uint32(123)),
		Entry("int64", int64(123456), uint32(123456)),
		Entry("int32", int32(123456), uint32(123456)),
		Entry("int16", int16(1234), uint32(1234)),
		Entry("int8", int8(123), uint32(123)),
		Entry("int", int(123456), uint32(123456)),
		Entry("float64", float64(65536), uint32(65536)),
		Entry("float32", float32(1234), uint32(1234)),
		Entry("string", "65537", uint32(65537)),
	)
	It("Should return an error for unsupported types", func() {
		b := MustSucceed(msgpack.Marshal(map[string]int{"a": 1}))
		dec := msgpack.NewDecoder(bytes.NewReader(b))
		Expect(
			xmsgpack.UnmarshalUint32(dec),
		).Error().
			To(MatchError(ContainSubstring("cannot unmarshal")))
	})
	It("Should return an error for invalid string", func() {
		b := MustSucceed(msgpack.Marshal("invalid"))
		dec := msgpack.NewDecoder(bytes.NewReader(b))
		Expect(
			xmsgpack.UnmarshalUint32(dec),
		).Error().
			To(MatchError(ContainSubstring("invalid")))
	})
	DescribeTable("Should return an error for negative values",
		func(value any) {
			b := MustSucceed(msgpack.Marshal(value))
			dec := msgpack.NewDecoder(bytes.NewReader(b))
			Expect(
				xmsgpack.UnmarshalUint32(dec),
			).Error().
				To(MatchError(Or(ContainSubstring("negative"), ContainSubstring("out of uint32 range"))))
		},
		Entry("negative int64", int64(-1)),
		Entry("negative int32", int32(-1)),
		Entry("negative int16", int16(-1)),
		Entry("negative int8", int8(-1)),
		Entry("negative int", int(-1)),
		Entry("negative float64", float64(-1.5)),
		Entry("negative float32", float32(-1.5)),
	)
	DescribeTable("Should return an error for overflow values",
		func(value any) {
			b := MustSucceed(msgpack.Marshal(value))
			dec := msgpack.NewDecoder(bytes.NewReader(b))
			Expect(
				xmsgpack.UnmarshalUint32(dec),
			).Error().
				To(MatchError(Or(ContainSubstring("exceeds uint32 max"), ContainSubstring("out of uint32 range"))))
		},
		Entry("uint64 overflow", uint64(5000000000)),
		Entry("int64 overflow", int64(5000000000)),
		Entry("float64 overflow", float64(5000000000)),
	)
})

var _ = Describe("UnmarshalUint64", func() {
	DescribeTable("Should decode various types to uint64",
		func(value any, expected uint64) {
			b := MustSucceed(msgpack.Marshal(value))
			dec := msgpack.NewDecoder(bytes.NewReader(b))
			result := MustSucceed(xmsgpack.UnmarshalUint64(dec))
			Expect(result).To(Equal(expected))
		},
		Entry("uint64", uint64(12345678901234), uint64(12345678901234)),
		Entry("uint32", uint32(123456), uint64(123456)),
		Entry("uint16", uint16(1234), uint64(1234)),
		Entry("uint8", uint8(123), uint64(123)),
		Entry("int64", int64(12345678901234), uint64(12345678901234)),
		Entry("int32", int32(123456), uint64(123456)),
		Entry("int16", int16(1234), uint64(1234)),
		Entry("int8", int8(123), uint64(123)),
		Entry("int", int(123456789), uint64(123456789)),
		Entry("float64", float64(123456), uint64(123456)),
		Entry("float32", float32(1234), uint64(1234)),
		Entry("string", "281543696187399", uint64(281543696187399)),
	)
	It("Should return an error for unsupported types", func() {
		b := MustSucceed(msgpack.Marshal([]int{1, 2, 3}))
		dec := msgpack.NewDecoder(bytes.NewReader(b))
		Expect(
			xmsgpack.UnmarshalUint64(dec),
		).Error().
			To(MatchError(ContainSubstring("cannot unmarshal")))
	})
	It("Should return an error for invalid string", func() {
		b := MustSucceed(msgpack.Marshal("not-a-number"))
		dec := msgpack.NewDecoder(bytes.NewReader(b))
		Expect(
			xmsgpack.UnmarshalUint64(dec),
		).Error().
			To(MatchError(ContainSubstring("invalid")))
	})
	DescribeTable("Should return an error for negative values",
		func(value any) {
			b := MustSucceed(msgpack.Marshal(value))
			dec := msgpack.NewDecoder(bytes.NewReader(b))
			Expect(
				xmsgpack.UnmarshalUint64(dec),
			).Error().
				To(MatchError(ContainSubstring("negative")))
		},
		Entry("negative int64", int64(-1)),
		Entry("negative int32", int32(-1)),
		Entry("negative int16", int16(-1)),
		Entry("negative int8", int8(-1)),
		Entry("negative int", int(-1)),
		Entry("negative float64", float64(-1.5)),
		Entry("negative float32", float32(-1.5)),
	)
})
