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
	It("Should encode and decode", func(ctx SpecContext) {
		codec := json.Codec
		b := MustSucceed(codec.Encode(ctx, toEncode{1}))
		var d toEncode
		Expect(codec.Decode(ctx, b, &d)).To(Succeed())
		Expect(d.Value).To(Equal(1))
		var d2 toEncode
		Expect(codec.DecodeStream(ctx, bytes.NewReader(b), &d2)).To(Succeed())
		Expect(d2.Value).To(Equal(1))
	})
	It("Should add error info on encoding failure", func(ctx SpecContext) {
		codec := json.Codec
		Expect(codec.Encode(ctx, make(chan int))).Error().To(MatchError(
			SatisfyAll(
				ContainSubstring("failed to encode value"),
				ContainSubstring("kind=chan, type=chan int"),
			),
		))
	})
	It("Should add error info with custom type", func(ctx SpecContext) {
		codec := json.Codec
		type custom struct {
			Chan  chan int
			Value int
		}
		Expect(codec.Encode(ctx, custom{Chan: make(chan int)})).Error().To(MatchError(
			SatisfyAll(
				ContainSubstring("failed to encode value"),
				ContainSubstring("kind=struct, type=json_test.custom"),
			),
		))
	})
	It("Should include a stack trace on encoding errors", func(ctx SpecContext) {
		codec := json.Codec
		_, err := codec.Encode(ctx, make(chan int))
		Expect(err).To(HaveOccurred())
		stack := errors.GetStackTrace(err)
		Expect(stack.String()).ToNot(BeEmpty())
		Expect(stack.String()).To(ContainSubstring(".go"))
	})
	It("Should include a stack trace on decoding errors", func(ctx SpecContext) {
		codec := json.Codec
		var d toEncode
		err := codec.Decode(ctx, []byte("invalid"), &d)
		Expect(err).To(HaveOccurred())
		stack := errors.GetStackTrace(err)
		Expect(stack.String()).ToNot(BeEmpty())
		Expect(stack.String()).To(ContainSubstring(".go"))
	})
	Describe("String Number Unmarshaling", func() {
		DescribeTable("UnmarshalStringInt64", func(input string, expected int64, shouldError bool) {
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
			Entry("max int64", `9223372036854775807`, int64(9223372036854775807), false),
			Entry("invalid string", `"abc"`, int64(0), true),
			Entry("invalid json", `{invalid}`, int64(0), true),
		)
		DescribeTable("UnmarshalStringUint64", func(input string, expected uint64, shouldError bool) {
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
			Entry("max uint64", `18446744073709551615`, uint64(18446744073709551615), false),
			Entry("negative number", `-123`, uint64(0), true),
			Entry("negative string", `"-123"`, uint64(0), true),
			Entry("invalid string", `"abc"`, uint64(0), true),
			Entry("invalid json", `{invalid}`, uint64(0), true),
		)
	})
	Describe("MarshalStringInt64", func() {
		It("Should encode an int64 value as a string", func() {
			Expect(json.MarshalStringInt64(12)).To(Equal([]byte("\"12\"")))
			Expect(json.MarshalStringInt64(-1)).To(Equal([]byte("\"-1\"")))
		})
	})
	Describe("MarshalStringUint64", func() {
		It("Should encode a uint64 value as a string", func() {
			Expect(json.MarshalStringUint64(12)).To(Equal([]byte("\"12\"")))
		})
	})
})
