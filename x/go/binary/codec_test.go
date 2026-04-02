// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/x/encoding/gob"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Codec", func() {
	Describe("Fallback", func() {
		It("Should fallback to the next codec when the first one fails", func(ctx SpecContext) {
			js := &json.Codec{}
			gb := &gob.Codec{}
			type abc struct {
				Value int `json:"value"`
			}
			v := abc{Value: 12}
			jsonB := MustSucceed(js.Encode(ctx, v))
			gobB := MustSucceed(gb.Encode(ctx, v))
			var res abc
			fbc := binary.NewDecodeFallbackCodec(&gob.Codec{}, &json.Codec{})
			Expect(fbc.Decode(ctx, jsonB, &res)).To(Succeed())
			Expect(res.Value).To(Equal(12))
			Expect(fbc.Decode(ctx, gobB, &res)).To(Succeed())
			Expect(res.Value).To(Equal(12))
		})
		It("Should return the error of the last encoder if all codecs fail to encode", func(ctx SpecContext) {
			fbc := binary.NewDecodeFallbackCodec(&gob.Codec{}, &json.Codec{})
			Expect(fbc.Encode(ctx, make(chan int))).Error().To(HaveOccurred())
		})
		It("Should return an error when all codecs fail to decode", func(ctx SpecContext) {
			fbc := binary.NewDecodeFallbackCodec(&gob.Codec{}, &json.Codec{})
			invalidData := []byte("completely invalid data")
			var res struct{ Value int }
			Expect(fbc.Decode(ctx, invalidData, &res)).To(MatchError(ContainSubstring("all codecs failed to decode")))
		})
		It("Should handle DecodeStream fallback correctly", func(ctx SpecContext) {
			js := &json.Codec{}
			type abc struct {
				Value int `json:"value"`
			}
			v := abc{Value: 12}
			jsonB := MustSucceed(js.Encode(ctx, v))

			var res abc
			fbc := binary.NewDecodeFallbackCodec(&msgpack.Codec{}, &json.Codec{})

			buf := bytes.NewBuffer(jsonB)
			Expect(fbc.DecodeStream(ctx, buf, &res)).To(Succeed())
			Expect(res.Value).To(Equal(12))
		})
		It("Should return error when DecodeStream fails for all codecs", func(ctx SpecContext) {
			fbc := binary.NewDecodeFallbackCodec(&gob.Codec{}, &json.Codec{})
			invalidData := []byte("completely invalid data")
			var res struct{ Value int }
			Expect(fbc.DecodeStream(ctx, bytes.NewReader(invalidData), &res)).To(MatchError(ContainSubstring("all codecs failed to decode")))
		})
	})
})
