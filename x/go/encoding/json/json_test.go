// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package json_test

import (
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
)

type toEncode struct {
	Value int
}

var _ = Describe("Codec", func() {
	Describe("ContentType", func() {
		It("Should report the JSON content type", func() {
			Expect(json.Codec.ContentType()).To(Equal("application/json"))
		})
	})
	It("Should encode and decode", func(ctx SpecContext) {
		b := MustSucceed(json.Codec.Encode(ctx, toEncode{1}))
		var d toEncode
		Expect(json.Codec.Decode(ctx, b, &d)).To(Succeed())
		Expect(d).To(Equal(toEncode{1}))
		var d2 toEncode
		Expect(json.Codec.DecodeStream(ctx, bytes.NewReader(b), &d2)).To(Succeed())
		Expect(d2).To(Equal(toEncode{1}))
	})
	It("Should add error info on encoding failure", func(ctx SpecContext) {
		Expect(json.Codec.Encode(ctx, make(chan int))).Error().To(MatchError(
			SatisfyAll(
				ContainSubstring("failed to encode value"),
				ContainSubstring("kind=chan, type=chan int"),
			)),
		)
	})
	It("Should add error info with custom type", func(ctx SpecContext) {
		type custom struct {
			Chan  chan int
			Value int
		}
		Expect(json.Codec.Encode(ctx, custom{Chan: make(chan int)})).
			Error().To(MatchError(SatisfyAll(
			ContainSubstring("failed to encode value"),
			ContainSubstring("kind=struct, type=json_test.custom"),
		)))
	})
	It("Should include a stack trace on encoding errors", func(ctx SpecContext) {
		_, err := json.Codec.Encode(ctx, make(chan int))
		Expect(err).To(HaveOccurred())
		stack := errors.GetStackTrace(err)
		Expect(stack.String()).ToNot(BeEmpty())
		Expect(stack.String()).To(ContainSubstring(".go"))
	})
	It("Should include a stack trace on decoding errors", func(ctx SpecContext) {
		err := json.Codec.Decode(ctx, []byte("invalid"), &toEncode{})
		Expect(err).To(HaveOccurred())
		stack := errors.GetStackTrace(err)
		Expect(stack.String()).ToNot(BeEmpty())
		Expect(stack.String()).To(ContainSubstring(".go"))
	})
	Describe("Extension", func() {
		It("Should report the JSON file extension", func() {
			Expect(json.Codec.Extension()).To(Equal(".json"))
		})
	})
})

var _ = Describe("NewCodec", func() {
	It("Should encode compactly with no options", func(ctx SpecContext) {
		b := MustSucceed(json.NewCodec().Encode(ctx, toEncode{1}))
		Expect(string(b)).To(Equal(`{"Value":1}`))
	})
	Describe("WithIndent", func() {
		pretty := json.NewCodec(json.WithIndent("  "))
		Describe("ContentType", func() {
			It("Should report the JSON content type", func() {
				Expect(pretty.ContentType()).To(Equal("application/json"))
			})
		})
		Describe("Extension", func() {
			It("Should report the JSON file extension", func() {
				Expect(pretty.Extension()).To(Equal(".json"))
			})
		})
		It("Should encode with the indentation and a trailing newline", func(
			ctx SpecContext,
		) {
			b := MustSucceed(pretty.Encode(ctx, toEncode{1}))
			Expect(string(b)).To(Equal("{\n  \"Value\": 1\n}\n"))
		})
		It("Should encode identical bytes through EncodeStream", func(ctx SpecContext) {
			b := MustSucceed(pretty.Encode(ctx, toEncode{1}))
			var buf bytes.Buffer
			Expect(pretty.EncodeStream(ctx, &buf, toEncode{1})).To(Succeed())
			Expect(buf.Bytes()).To(Equal(b))
		})
		It("Should decode its own output", func(ctx SpecContext) {
			b := MustSucceed(pretty.Encode(ctx, toEncode{1}))
			var d toEncode
			Expect(pretty.Decode(ctx, b, &d)).To(Succeed())
			Expect(d).To(Equal(toEncode{1}))
		})
		It("Should add error info on encoding failure", func(ctx SpecContext) {
			Expect(pretty.Encode(ctx, make(chan int))).Error().To(MatchError(
				SatisfyAll(
					ContainSubstring("failed to encode value"),
					ContainSubstring("kind=chan, type=chan int"),
				)),
			)
		})
	})
})

var _ = Describe("MarshalStringInt64", func() {
	It("Should encode an int64 value as a string", func() {
		Expect(json.MarshalStringInt64(12)).To(Equal([]byte("\"12\"")))
		Expect(json.MarshalStringInt64(-1)).To(Equal([]byte("\"-1\"")))
	})
})

var _ = Describe("MarshalStringUint64", func() {
	It("Should encode a uint64 value as a string", func() {
		Expect(json.MarshalStringUint64(12)).To(Equal([]byte("\"12\"")))
	})
})

var _ = DescribeTable("UnmarshalStringInt64",
	func(input string, expected int64, shouldError bool) {
		b := []byte(input)
		val, err := json.UnmarshalStringInt64(b)
		if shouldError {
			Expect(err).To(MatchError(ContainSubstring("invalid")))
		} else {
			Expect(err).ToNot(HaveOccurred())
			Expect(val).To(Equal(expected))
		}
	},
	Entry("direct number", `123`, int64(123), false),
	Entry("string number", `"123"`, int64(123), false),
	Entry("negative number", `-123`, int64(-123), false),
	Entry("negative string", `"-123"`, int64(-123), false),
	Entry(
		"max int64",
		`9223372036854775807`,
		int64(9223372036854775807),
		false,
	),
	Entry("invalid string", `"abc"`, int64(0), true),
	Entry("invalid json", `{invalid}`, int64(0), true),
)

var _ = DescribeTable("UnmarshalStringUint32",
	func(input string, expected uint32, shouldError bool) {
		b := []byte(input)
		val, err := json.UnmarshalStringUint32(b)
		if shouldError {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(err).ToNot(HaveOccurred())
			Expect(val).To(Equal(expected))
		}
	},
	Entry("direct number", `123`, uint32(123), false),
	Entry("string number", `"123"`, uint32(123), false),
	Entry("max uint32", `4294967295`, uint32(4294967295), false),
	Entry("negative number", `-123`, uint32(0), true),
	Entry("negative string", `"-123"`, uint32(0), true),
	Entry("invalid string", `"abc"`, uint32(0), true),
	Entry("invalid json", `{invalid}`, uint32(0), true),
)

var _ = DescribeTable("UnmarshalStringUint64",
	func(input string, expected uint64, shouldError bool) {
		b := []byte(input)
		val, err := json.UnmarshalStringUint64(b)
		if shouldError {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(err).ToNot(HaveOccurred())
			Expect(val).To(Equal(expected))
		}
	},
	Entry("direct number", `123`, uint64(123), false),
	Entry("string number", `"123"`, uint64(123), false),
	Entry(
		"max uint64",
		`18446744073709551615`,
		uint64(18446744073709551615),
		false,
	),
	Entry("negative number", `-123`, uint64(0), true),
	Entry("negative string", `"-123"`, uint64(0), true),
	Entry("invalid string", `"abc"`, uint64(0), true),
	Entry("invalid json", `{invalid}`, uint64(0), true),
)
