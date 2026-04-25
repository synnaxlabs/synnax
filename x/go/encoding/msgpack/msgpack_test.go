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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
	vmsgpack "github.com/vmihailenco/msgpack/v5"
)

type toEncode struct {
	Value int
}

var _ = Describe("Codec", func() {
	It("Should encode and decode", func(ctx SpecContext) {
		codec := msgpack.Codec
		b := MustSucceed(codec.Encode(ctx, toEncode{1}))
		var d toEncode
		Expect(codec.Decode(ctx, b, &d)).To(Succeed())
		Expect(d.Value).To(Equal(1))
		var d2 toEncode
		Expect(codec.DecodeStream(ctx, bytes.NewReader(b), &d2)).To(Succeed())
		Expect(d2.Value).To(Equal(1))
	})
	It("Should add error info with custom type", func(ctx SpecContext) {
		codec := msgpack.Codec
		type custom struct {
			Chan  chan int
			Value int
		}
		Expect(codec.Encode(ctx, custom{Chan: make(chan int)})).Error().To(MatchError(
			SatisfyAll(
				ContainSubstring("failed to encode value"),
				ContainSubstring("kind=struct, type=msgpack_test.custom"),
			),
		))
	})
	It("Should include a stack trace on encoding errors", func(ctx SpecContext) {
		codec := msgpack.Codec
		_, err := codec.Encode(ctx, make(chan int))
		Expect(err).To(HaveOccurred())
		stack := errors.GetStackTrace(err)
		Expect(stack.String()).ToNot(BeEmpty())
		Expect(stack.String()).To(ContainSubstring(".go"))
	})
	It("Should include a stack trace on decoding errors", func(ctx SpecContext) {
		codec := msgpack.Codec
		var d toEncode
		err := codec.Decode(ctx, []byte("invalid"), &d)
		Expect(err).To(HaveOccurred())
		stack := errors.GetStackTrace(err)
		Expect(stack.String()).ToNot(BeEmpty())
		Expect(stack.String()).To(ContainSubstring(".go"))
	})
	Describe("UnmarshalUint64", func() {
		DescribeTable("Should decode various types to uint64",
			func(value any, expected uint64) {
				b := MustSucceed(vmsgpack.Marshal(value))
				dec := vmsgpack.NewDecoder(bytes.NewReader(b))
				result := MustSucceed(msgpack.UnmarshalUint64(dec))
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
			b := MustSucceed(vmsgpack.Marshal([]int{1, 2, 3}))
			dec := vmsgpack.NewDecoder(bytes.NewReader(b))
			Expect(msgpack.UnmarshalUint64(dec)).Error().To(MatchError(ContainSubstring("cannot unmarshal")))
		})
		It("Should return an error for invalid string", func() {
			b := MustSucceed(vmsgpack.Marshal("not-a-number"))
			dec := vmsgpack.NewDecoder(bytes.NewReader(b))
			Expect(msgpack.UnmarshalUint64(dec)).Error().To(MatchError(ContainSubstring("invalid")))
		})
		DescribeTable("Should return an error for negative values",
			func(value any) {
				b := MustSucceed(vmsgpack.Marshal(value))
				dec := vmsgpack.NewDecoder(bytes.NewReader(b))
				Expect(msgpack.UnmarshalUint64(dec)).Error().To(MatchError(ContainSubstring("negative")))
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
	Describe("UnmarshalUint32", func() {
		DescribeTable("Should decode various types to uint32",
			func(value any, expected uint32) {
				b := MustSucceed(vmsgpack.Marshal(value))
				dec := vmsgpack.NewDecoder(bytes.NewReader(b))
				result := MustSucceed(msgpack.UnmarshalUint32(dec))
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
			b := MustSucceed(vmsgpack.Marshal(map[string]int{"a": 1}))
			dec := vmsgpack.NewDecoder(bytes.NewReader(b))
			Expect(msgpack.UnmarshalUint32(dec)).Error().To(MatchError(ContainSubstring("cannot unmarshal")))
		})
		It("Should return an error for invalid string", func() {
			b := MustSucceed(vmsgpack.Marshal("invalid"))
			dec := vmsgpack.NewDecoder(bytes.NewReader(b))
			Expect(msgpack.UnmarshalUint32(dec)).Error().To(MatchError(ContainSubstring("invalid")))
		})
		DescribeTable("Should return an error for negative values",
			func(value any) {
				b := MustSucceed(vmsgpack.Marshal(value))
				dec := vmsgpack.NewDecoder(bytes.NewReader(b))
				Expect(msgpack.UnmarshalUint32(dec)).Error().To(MatchError(Or(ContainSubstring("negative"), ContainSubstring("out of uint32 range"))))
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
				b := MustSucceed(vmsgpack.Marshal(value))
				dec := vmsgpack.NewDecoder(bytes.NewReader(b))
				Expect(msgpack.UnmarshalUint32(dec)).Error().To(MatchError(Or(ContainSubstring("exceeds uint32 max"), ContainSubstring("out of uint32 range"))))
			},
			Entry("uint64 overflow", uint64(5000000000)),
			Entry("int64 overflow", int64(5000000000)),
			Entry("float64 overflow", float64(5000000000)),
		)
	})
	Describe("EncodedJSON", func() {
		It("Should round-trip through msgpack encoding and decoding", func() {
			original := msgpack.EncodedJSON{"key": "value", "count": int64(42)}
			b := MustSucceed(vmsgpack.Marshal(original))
			var result msgpack.EncodedJSON
			Expect(vmsgpack.Unmarshal(b, &result)).To(Succeed())
			Expect(result["key"]).To(Equal("value"))
			Expect(result["count"]).To(BeNumerically("==", 42))
		})
		It("Should decode a JSON string for backwards compatibility", func() {
			jsonStr := `{"key":"value","count":42}`
			b := MustSucceed(vmsgpack.Marshal(jsonStr))
			var result msgpack.EncodedJSON
			Expect(vmsgpack.Unmarshal(b, &result)).To(Succeed())
			Expect(result["key"]).To(Equal("value"))
			Expect(result["count"]).To(BeNumerically("==", 42))
		})
		It("Should decode a struct field from a JSON string for backwards compatibility", func() {
			type OldConfig struct {
				Name   string `msgpack:"name"`
				Schema string `msgpack:"schema"`
			}
			type NewConfig struct {
				Name   string              `msgpack:"name"`
				Schema msgpack.EncodedJSON `msgpack:"schema"`
			}
			old := OldConfig{
				Name:   "test",
				Schema: `{"field":"value"}`,
			}
			b := MustSucceed(vmsgpack.Marshal(old))
			var result NewConfig
			Expect(vmsgpack.Unmarshal(b, &result)).To(Succeed())
			Expect(result.Name).To(Equal("test"))
			Expect(result.Schema["field"]).To(Equal("value"))
		})
		It("Should decode nil to nil", func() {
			b := MustSucceed(vmsgpack.Marshal(nil))
			var result msgpack.EncodedJSON
			Expect(vmsgpack.Unmarshal(b, &result)).To(Succeed())
			Expect(result).To(BeNil())
		})
		It("Should handle map[any]any with string keys", func() {
			m := map[string]any{"key": "value"}
			b := MustSucceed(vmsgpack.Marshal(m))
			dec := vmsgpack.NewDecoder(bytes.NewReader(b))
			dec.SetMapDecoder(func(dec *vmsgpack.Decoder) (any, error) {
				return dec.DecodeUntypedMap()
			})
			var result msgpack.EncodedJSON
			Expect(result.DecodeMsgpack(dec)).To(Succeed())
			Expect(result["key"]).To(Equal("value"))
		})
		It("Should work as a struct field with map data", func() {
			type Config struct {
				Name   string              `msgpack:"name"`
				Schema msgpack.EncodedJSON `msgpack:"schema"`
			}
			original := Config{
				Name:   "test",
				Schema: msgpack.EncodedJSON{"field": "value"},
			}
			b := MustSucceed(vmsgpack.Marshal(original))
			var result Config
			Expect(vmsgpack.Unmarshal(b, &result)).To(Succeed())
			Expect(result.Name).To(Equal("test"))
			Expect(result.Schema["field"]).To(Equal("value"))
		})
		It("Should decode an empty string to an empty map", func() {
			b := MustSucceed(vmsgpack.Marshal(""))
			var result msgpack.EncodedJSON
			Expect(vmsgpack.Unmarshal(b, &result)).To(Succeed())
			Expect(result).ToNot(BeNil())
			Expect(result).To(BeEmpty())
		})
		It("Should decode an empty string struct field to an empty map", func() {
			type OldConfig struct {
				Name   string `msgpack:"name"`
				Schema string `msgpack:"schema"`
			}
			type NewConfig struct {
				Name   string              `msgpack:"name"`
				Schema msgpack.EncodedJSON `msgpack:"schema"`
			}
			old := OldConfig{Name: "test", Schema: ""}
			b := MustSucceed(vmsgpack.Marshal(old))
			var result NewConfig
			Expect(vmsgpack.Unmarshal(b, &result)).To(Succeed())
			Expect(result.Name).To(Equal("test"))
			Expect(result.Schema).ToNot(BeNil())
			Expect(result.Schema).To(BeEmpty())
		})
		It("Should return an error for an invalid JSON string", func() {
			b := MustSucceed(vmsgpack.Marshal("not valid json"))
			var result msgpack.EncodedJSON
			Expect(vmsgpack.Unmarshal(b, &result)).To(MatchError(ContainSubstring("failed to unmarshal JSON string")))
		})
		It("Should return an error for unsupported types", func() {
			b := MustSucceed(vmsgpack.Marshal(42))
			var result msgpack.EncodedJSON
			Expect(vmsgpack.Unmarshal(b, &result)).To(MatchError(ContainSubstring("unsupported type")))
		})
		It("Should return an error for non-string map keys", func() {
			m := map[int]string{1: "a"}
			b := MustSucceed(vmsgpack.Marshal(m))
			dec := vmsgpack.NewDecoder(bytes.NewReader(b))
			dec.SetMapDecoder(func(dec *vmsgpack.Decoder) (any, error) {
				return dec.DecodeUntypedMap()
			})
			var result msgpack.EncodedJSON
			Expect(result.DecodeMsgpack(dec)).To(MatchError(ContainSubstring("non-string key")))
		})
		Describe("Unmarshal", func() {
			It("Should unmarshal into a typed struct", func() {
				type Config struct {
					Name    string `json:"name"`
					Count   int    `json:"count"`
					Enabled bool   `json:"enabled"`
				}
				m := msgpack.EncodedJSON{
					"name":    "test",
					"count":   float64(42),
					"enabled": true,
				}
				var cfg Config
				Expect(m.Unmarshal(&cfg)).To(Succeed())
				Expect(cfg.Name).To(Equal("test"))
				Expect(cfg.Count).To(Equal(42))
				Expect(cfg.Enabled).To(BeTrue())
			})
			It("Should handle nil map", func() {
				var m msgpack.EncodedJSON
				type Config struct {
					Name string `json:"name"`
				}
				var cfg Config
				Expect(m.Unmarshal(&cfg)).To(Succeed())
				Expect(cfg.Name).To(Equal(""))
			})
			It("Should return an error for incompatible types", func() {
				m := msgpack.EncodedJSON{"count": "not a number"}
				type Config struct {
					Count int `json:"count"`
				}
				var cfg Config
				Expect(m.Unmarshal(&cfg)).ToNot(Succeed())
			})
		})
		It("Should work with Codec", func() {
			codec := msgpack.Codec
			ctx := context.Background()

			jsonStr := `{"name":"test","value":123}`
			b := MustSucceed(codec.Encode(ctx, jsonStr))

			var result msgpack.EncodedJSON
			Expect(codec.Decode(ctx, b, &result)).To(Succeed())
			Expect(result).To(HaveKey("name"))
			Expect(result["name"]).To(Equal("test"))
			Expect(result).To(HaveKey("value"))
			Expect(result["value"]).To(Equal(float64(123)))
		})
	})
})
