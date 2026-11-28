// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	"encoding/json/v2"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/synnaxlabs/x/payload/versions/v1"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("Scalar", func() {
	DescribeTable("Should round trip through msgpack",
		func(s v1.Scalar) {
			var decoded v1.Scalar
			Expect(msgpack.Unmarshal(MustSucceed(msgpack.Marshal(s)), &decoded)).
				To(Succeed())
			Expect(decoded).To(Equal(s))
		},
		Entry("number", v1.Scalar{Variant: v1.NumberScalar{Value: 200}}),
		Entry("string", v1.Scalar{Variant: v1.StringScalar{Value: "ok"}}),
		Entry("boolean", v1.Scalar{Variant: v1.BooleanScalar{Value: true}}),
		Entry("null", v1.Scalar{Variant: v1.NullScalar{}}),
		Entry("no variant", v1.Scalar{}),
	)

	It("Should encode the same shape in msgpack and JSON", func() {
		s := v1.Scalar{Variant: v1.StringScalar{Value: "ok"}}
		var fromMsgpack, fromJSON map[string]any
		Expect(msgpack.Unmarshal(MustSucceed(msgpack.Marshal(s)), &fromMsgpack)).
			To(Succeed())
		Expect(json.Unmarshal(MustSucceed(json.Marshal(s)), &fromJSON)).To(Succeed())
		Expect(fromMsgpack).To(Equal(fromJSON))
	})

	It("Should reject an unknown type", func() {
		raw := MustSucceed(msgpack.Marshal(map[string]any{"type": "date"}))
		var decoded v1.Scalar
		Expect(msgpack.Unmarshal(raw, &decoded)).To(MatchError(validate.ErrValidation))
	})

	It("Should reject a value of the wrong type", func() {
		raw := MustSucceed(msgpack.Marshal(map[string]any{
			"type":  "number",
			"value": "x",
		}))
		var decoded v1.Scalar
		Expect(msgpack.Unmarshal(raw, &decoded)).
			To(MatchError(ContainSubstring("msgpack: ")))
	})

	It("Should reject truncated input", func() {
		var decoded v1.Scalar
		Expect(msgpack.Unmarshal([]byte{0x81}, &decoded)).To(MatchError(io.EOF))
	})
})
