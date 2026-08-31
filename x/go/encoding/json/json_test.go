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
	jsonv1 "encoding/json"
	"encoding/json/jsontext"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
)

type toEncode struct {
	Value int
}

func decoderOf(data string) *jsontext.Decoder {
	return jsontext.NewDecoder(strings.NewReader(data))
}

func encodeWith[T any](
	write func(*jsontext.Encoder, T) error,
	value T,
) string {
	var buf bytes.Buffer
	Expect(write(jsontext.NewEncoder(&buf), value)).To(Succeed())
	// A bare encoder terminates a top-level value with a newline; a marshaler writing
	// into a larger document does not.
	return strings.TrimSuffix(buf.String(), "\n")
}

type markup struct {
	Value string
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
	It("Should decode one value and leave the rest of the stream", func(
		ctx SpecContext,
	) {
		r := bytes.NewReader([]byte(`{"Value":1}{"Value":2}`))
		var d toEncode
		Expect(json.Codec.DecodeStream(ctx, r, &d)).To(Succeed())
		Expect(d).To(Equal(toEncode{1}))
	})
	It("Should add error info on encoding failure", func(ctx SpecContext) {
		Expect(json.Codec.Encode(ctx, make(chan int))).Error().To(MatchError(
			SatisfyAll(
				ContainSubstring("failed to encode value"),
				ContainSubstring("kind=chan, type=chan int"),
			),
		),
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

type wireShapes struct {
	NilSlice   []int          `json:"nil_slice"`
	NilMap     map[string]int `json:"nil_map"`
	NilBytes   []byte         `json:"nil_bytes"`
	EmptySlice []int          `json:"empty_slice"`
	Bytes      []byte         `json:"bytes"`
	ZeroInt    int            `json:"zero_int,omitempty"`
	ZeroBool   bool           `json:"zero_bool,omitempty"`
	EmptyPtr   *toEncode      `json:"empty_ptr,omitempty"`
	Duration   time.Duration  `json:"duration"`
}

var _ = Describe("Wire compatibility", func() {
	It("Should encode the same bytes as v1 where the v2 defaults differ", func(
		ctx SpecContext,
	) {
		v := wireShapes{
			EmptySlice: []int{},
			Bytes:      []byte("hi"),
			EmptyPtr:   &toEncode{},
			Duration:   time.Second,
		}
		Expect(json.Codec.Encode(ctx, v)).To(Equal(MustSucceed(jsonv1.Marshal(v))))
	})
	It("Should match object names case-insensitively", func(ctx SpecContext) {
		var d toEncode
		Expect(json.Codec.Decode(ctx, []byte(`{"value":7}`), &d)).To(Succeed())
		Expect(d).To(Equal(toEncode{7}))
	})
	It("Should take the last of a repeated object name", func(ctx SpecContext) {
		var d toEncode
		Expect(
			json.Codec.Decode(ctx, []byte(`{"Value":1,"Value":2}`), &d),
		).To(Succeed())
		Expect(d).To(Equal(toEncode{2}))
	})
})

var _ = Describe("NewCodec", func() {
	It("Should encode compactly with no options", func(ctx SpecContext) {
		b := MustSucceed(json.NewCodec().Encode(ctx, toEncode{1}))
		Expect(string(b)).To(Equal(`{"Value":1}`))
	})
	Describe("WithoutHTMLEscaping", func() {
		plain := json.NewCodec(json.WithoutHTMLEscaping())

		It("Should write <, >, and & literally", func(ctx SpecContext) {
			b := MustSucceed(plain.Encode(ctx, markup{`<a href="x">1 & 2</a>`}))
			Expect(string(b)).To(Equal(`{"Value":"<a href=\"x\">1 & 2</a>"}`))
		})

		It("Should escape them by default", func(ctx SpecContext) {
			b := MustSucceed(json.Codec.Encode(ctx, markup{"<&>"}))
			Expect(string(b)).To(Equal(`{"Value":"\u003c\u0026\u003e"}`))
		})

		It("Should still escape the line and paragraph separators", func(
			ctx SpecContext,
		) {
			b := MustSucceed(plain.Encode(ctx, markup{"a\u2028b\u2029c"}))
			Expect(string(b)).To(Equal(`{"Value":"a\u2028b\u2029c"}`))
		})

		It("Should decode to the same value as the escaping codec", func(
			ctx SpecContext,
		) {
			original := markup{`<svg viewBox="0 0 1 1"/>`}
			var escaped, literal markup
			Expect(json.Codec.Decode(
				ctx, MustSucceed(json.Codec.Encode(ctx, original)), &escaped,
			)).To(Succeed())
			Expect(plain.Decode(
				ctx, MustSucceed(plain.Encode(ctx, original)), &literal,
			)).To(Succeed())
			Expect(literal).To(Equal(escaped))
			Expect(literal).To(Equal(original))
		})

		It("Should compose with WithIndent", func(ctx SpecContext) {
			c := json.NewCodec(json.WithIndent("  "), json.WithoutHTMLEscaping())
			Expect(string(MustSucceed(c.Encode(ctx, markup{"<x>"})))).
				To(Equal("{\n  \"Value\": \"<x>\"\n}\n"))
		})
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
				),
			),
			)
		})
	})
})

var _ = Describe("Validate", func() {
	It("Should accept a well-formed document", func() {
		Expect(json.Validate([]byte(`{"a":[1,{"b":null}]}`))).To(Succeed())
	})
	It("Should reject a duplicate object name and name its location", func() {
		Expect(json.Validate([]byte(`{"a":{"b":1,"b":2}}`))).To(MatchError(
			SatisfyAll(ContainSubstring("duplicate"), ContainSubstring(`"/a"`)),
		))
	})
	It("Should reject invalid UTF-8", func() {
		Expect(json.Validate([]byte("{\"a\":\"\xff\"}"))).
			To(MatchError(ContainSubstring("invalid UTF-8")))
	})
	It("Should reject malformed syntax", func() {
		Expect(json.Validate([]byte(`{"a":1,}`))).To(HaveOccurred())
	})
})

var _ = Describe("MarshalStringInt64To", func() {
	It("Should encode an int64 value as a string", func() {
		Expect(encodeWith(json.MarshalStringInt64To, int64(12))).To(Equal(`"12"`))
		Expect(encodeWith(json.MarshalStringInt64To, int64(-1))).To(Equal(`"-1"`))
	})
})

var _ = Describe("MarshalStringUint64To", func() {
	It("Should encode a uint64 value as a string", func() {
		Expect(encodeWith(json.MarshalStringUint64To, uint64(12))).To(Equal(`"12"`))
	})
})

var _ = DescribeTable("UnmarshalStringInt64From",
	func(input string, expected int64, shouldError bool) {
		val, err := json.UnmarshalStringInt64From(decoderOf(input))
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
	Entry("boolean", `true`, int64(0), true),
)

var _ = DescribeTable("UnmarshalStringUint32From",
	func(input string, expected uint32, shouldError bool) {
		val, err := json.UnmarshalStringUint32From(decoderOf(input))
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
	Entry("null", `null`, uint32(0), true),
)

var _ = DescribeTable("UnmarshalStringUint64From",
	func(input string, expected uint64, shouldError bool) {
		val, err := json.UnmarshalStringUint64From(decoderOf(input))
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
	Entry("array", `[1]`, uint64(0), true),
)
