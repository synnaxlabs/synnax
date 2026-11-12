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
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

type toEncode struct {
	Value int
}

// rejected fails every decode with a validation error.
type rejected struct{}

func (*rejected) UnmarshalJSONFrom(*jsontext.Decoder) error {
	return errors.Wrap(validate.ErrValidation, "rejected")
}

func decoderOf(data string) *jsontext.Decoder {
	return jsontext.NewDecoder(strings.NewReader(data))
}

func encodeWith[T any](
	write func(*jsontext.Encoder, T) error,
	value T,
) string {
	GinkgoHelper()
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
		b := MustSucceed(json.Codec.Encode(toEncode{1}))
		var d toEncode
		Expect(json.Codec.Decode(b, &d)).To(Succeed())
		Expect(d).To(Equal(toEncode{1}))
		var d2 toEncode
		Expect(json.Codec.DecodeStream(bytes.NewReader(b), &d2)).To(Succeed())
		Expect(d2).To(Equal(toEncode{1}))
	})
	It("Should decode one value and leave the rest of the stream", func(
		ctx SpecContext,
	) {
		r := bytes.NewReader([]byte(`{"Value":1}{"Value":2}`))
		var d toEncode
		Expect(json.Codec.DecodeStream(r, &d)).To(Succeed())
		Expect(d).To(Equal(toEncode{1}))
	})
	It("Should add error info on encoding failure", func(ctx SpecContext) {
		Expect(json.Codec.Encode(make(chan int))).Error().To(MatchError(
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
		Expect(json.Codec.Encode(custom{Chan: make(chan int)})).
			Error().To(MatchError(SatisfyAll(
			ContainSubstring("failed to encode value"),
			ContainSubstring("kind=struct, type=json_test.custom"),
		)))
	})
	It("Should include a stack trace on encoding errors", func(ctx SpecContext) {
		_, err := json.Codec.Encode(make(chan int))
		Expect(err).To(MatchError(ContainSubstring("failed to encode value")))
		stack := errors.GetStackTrace(err)
		Expect(stack.String()).ToNot(BeEmpty())
		Expect(stack.String()).To(ContainSubstring(".go"))
	})
	It("Should include a stack trace on decoding errors", func(ctx SpecContext) {
		err := json.Codec.Decode([]byte("invalid"), &toEncode{})
		Expect(err).To(MatchError(ContainSubstring("failed to decode")))
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

var _ = Describe("Wire format", func() {
	It("Should encode a nil slice and a nil map as empty, not null", func(
		ctx SpecContext,
	) {
		type shape struct {
			Slice []int          `json:"slice"`
			Map   map[string]int `json:"map"`
		}
		Expect(MustSucceed(json.Codec.Encode(shape{}))).
			To(MatchJSON(`{"slice":[],"map":{}}`))
	})
	It("Should drop a pointer to an empty string under omitempty", func(
		ctx SpecContext,
	) {
		type shape struct {
			Ptr *string `json:"ptr,omitempty"`
			Zed *string `json:"zed,omitzero"`
		}
		empty := ""
		Expect(MustSucceed(json.Codec.Encode(shape{Ptr: &empty, Zed: &empty}))).
			To(MatchJSON(`{"zed":""}`))
	})
	It("Should keep an allocated empty collection under omitzero", func(
		ctx SpecContext,
	) {
		type shape struct {
			Nil   []int `json:"nil,omitzero"`
			Empty []int `json:"empty,omitzero"`
		}
		Expect(MustSucceed(json.Codec.Encode(shape{Empty: []int{}}))).
			To(MatchJSON(`{"empty":[]}`))
	})
	It("Should match object names case-sensitively", func(ctx SpecContext) {
		var d toEncode
		Expect(json.Codec.Decode([]byte(`{"value":7}`), &d)).To(Succeed())
		Expect(d).To(Equal(toEncode{}))
	})
	It("Should reject a repeated object name", func(ctx SpecContext) {
		var d toEncode
		Expect(json.Codec.Decode([]byte(`{"Value":1,"Value":2}`), &d)).
			To(MatchError(ContainSubstring("failed to decode")))
	})
})

var _ = Describe("NewCodec", func() {
	It("Should encode compactly with no options", func(ctx SpecContext) {
		b := MustSucceed(json.NewCodec().Encode(toEncode{1}))
		Expect(string(b)).To(Equal(`{"Value":1}`))
	})
	Describe("MatchCaseInsensitiveNames", func() {
		loose := json.NewCodec(jsonv2.MatchCaseInsensitiveNames(true))

		It("Should match an object name differing only in case", func(
			ctx SpecContext,
		) {
			var d toEncode
			Expect(loose.Decode([]byte(`{"value":7}`), &d)).To(Succeed())
			Expect(d).To(Equal(toEncode{7}))
		})

		It("Should still write the field's declared name", func(ctx SpecContext) {
			Expect(MustSucceed(loose.Encode(toEncode{7}))).
				To(MatchJSON(`{"Value":7}`))
		})
	})

	Describe("Deterministic", func() {
		stable := json.NewCodec(jsonv2.Deterministic(true))
		m := map[string]int{"z": 1, "a": 2, "m": 3, "b": 4, "q": 5}

		It("Should encode map members in sorted order", func(ctx SpecContext) {
			Expect(string(MustSucceed(stable.Encode(m)))).
				To(Equal(`{"a":2,"b":4,"m":3,"q":5,"z":1}`))
		})

		It("Should encode the same bytes on every call", func(ctx SpecContext) {
			first := MustSucceed(stable.Encode(m))
			for range 20 {
				Expect(MustSucceed(stable.Encode(m))).To(Equal(first))
			}
		})
	})

	Describe("EscapeForHTML", func() {
		It("Should write <, >, and & literally by default", func(ctx SpecContext) {
			b := MustSucceed(json.Codec.Encode(markup{`<a href="x">1 & 2</a>`}))
			Expect(string(b)).To(Equal(`{"Value":"<a href=\"x\">1 & 2</a>"}`))
		})

		It("Should escape them when asked", func(ctx SpecContext) {
			escaping := json.NewCodec(jsontext.EscapeForHTML(true))
			Expect(string(MustSucceed(escaping.Encode(markup{"<&>"})))).
				To(Equal(`{"Value":"\u003c\u0026\u003e"}`))
		})

		It("Should decode to the same value either way", func(ctx SpecContext) {
			escaping := json.NewCodec(jsontext.EscapeForHTML(true))
			original := markup{`<svg viewBox="0 0 1 1"/>`}
			var escaped, literal markup
			Expect(escaping.Decode(
				MustSucceed(escaping.Encode(original)), &escaped,
			)).To(Succeed())
			Expect(json.Codec.Decode(
				MustSucceed(json.Codec.Encode(original)), &literal,
			)).To(Succeed())
			Expect(literal).To(Equal(escaped))
			Expect(literal).To(Equal(original))
		})
	})

	Describe("WithIndent", func() {
		pretty := json.NewCodec(jsontext.WithIndent("  "))
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
			b := MustSucceed(pretty.Encode(toEncode{1}))
			Expect(string(b)).To(Equal("{\n  \"Value\": 1\n}\n"))
		})
		It("Should encode identical bytes through EncodeStream", func(ctx SpecContext) {
			b := MustSucceed(pretty.Encode(toEncode{1}))
			var buf bytes.Buffer
			Expect(pretty.EncodeStream(&buf, toEncode{1})).To(Succeed())
			Expect(buf.Bytes()).To(Equal(b))
		})
		It("Should decode its own output", func(ctx SpecContext) {
			b := MustSucceed(pretty.Encode(toEncode{1}))
			var d toEncode
			Expect(pretty.Decode(b, &d)).To(Succeed())
			Expect(d).To(Equal(toEncode{1}))
		})
		It("Should add error info on encoding failure", func(ctx SpecContext) {
			Expect(pretty.Encode(make(chan int))).Error().To(MatchError(
				SatisfyAll(
					ContainSubstring("failed to encode value"),
					ContainSubstring("kind=chan, type=chan int"),
				),
			),
			)
		})
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

var _ = DescribeTable(
	"UnmarshalStringInt64From",
	func(input string, expected int64, errMatcher types.GomegaMatcher) {
		val, err := json.UnmarshalStringInt64From(decoderOf(input))
		Expect(err).To(errMatcher)
		Expect(val).To(Equal(expected))
	},
	Entry("direct number", `123`, int64(123), Succeed()),
	Entry("string number", `"123"`, int64(123), Succeed()),
	Entry("negative number", `-123`, int64(-123), Succeed()),
	Entry("negative string", `"-123"`, int64(-123), Succeed()),
	Entry("max int64", `9223372036854775807`, int64(9223372036854775807), Succeed()),
	Entry("invalid string", `"abc"`, int64(0), MatchError(strconv.ErrSyntax)),
	Entry(
		"invalid json",
		`{invalid}`,
		int64(0),
		MatchError(ContainSubstring("cannot decode")),
	),
	Entry("boolean", `true`, int64(0), MatchError(ContainSubstring("cannot decode"))),
)

var _ = DescribeTable(
	"UnmarshalStringUint32From",
	func(input string, expected uint32, errMatcher types.GomegaMatcher) {
		val, err := json.UnmarshalStringUint32From(decoderOf(input))
		Expect(err).To(errMatcher)
		Expect(val).To(Equal(expected))
	},
	Entry("direct number", `123`, uint32(123), Succeed()),
	Entry("string number", `"123"`, uint32(123), Succeed()),
	Entry("max uint32", `4294967295`, uint32(4294967295), Succeed()),
	Entry("negative number", `-123`, uint32(0), MatchError(strconv.ErrSyntax)),
	Entry("negative string", `"-123"`, uint32(0), MatchError(strconv.ErrSyntax)),
	Entry("invalid string", `"abc"`, uint32(0), MatchError(strconv.ErrSyntax)),
	Entry(
		"invalid json",
		`{invalid}`,
		uint32(0),
		MatchError(ContainSubstring("cannot decode")),
	),
	Entry("null", `null`, uint32(0), MatchError(ContainSubstring("cannot decode"))),
)

var _ = DescribeTable(
	"UnmarshalStringUint64From",
	func(input string, expected uint64, errMatcher types.GomegaMatcher) {
		val, err := json.UnmarshalStringUint64From(decoderOf(input))
		Expect(err).To(errMatcher)
		Expect(val).To(Equal(expected))
	},
	Entry("direct number", `123`, uint64(123), Succeed()),
	Entry("string number", `"123"`, uint64(123), Succeed()),
	Entry(
		"max uint64",
		`18446744073709551615`,
		uint64(18446744073709551615),
		Succeed(),
	),
	Entry("negative number", `-123`, uint64(0), MatchError(strconv.ErrSyntax)),
	Entry("negative string", `"-123"`, uint64(0), MatchError(strconv.ErrSyntax)),
	Entry("invalid string", `"abc"`, uint64(0), MatchError(strconv.ErrSyntax)),
	Entry(
		"invalid json",
		`{invalid}`,
		uint64(0),
		MatchError(ContainSubstring("cannot decode")),
	),
	Entry("array", `[1]`, uint64(0), MatchError(ContainSubstring("cannot decode"))),
)

var _ = Describe("Decoding errors", func() {
	It("Should keep a validation error in the error chain", func(ctx SpecContext) {
		err := json.Codec.Decode([]byte(`{}`), &rejected{})
		Expect(err).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("failed to decode")),
		))
		Expect(errors.Encode(ctx, err, false).Type).To(Equal("sy.validation"))
	})
})
