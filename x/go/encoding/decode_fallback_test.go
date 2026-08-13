// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package encoding_test

import (
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/encoding/gob"
	"github.com/synnaxlabs/x/encoding/json"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("NewDecodeFallbackCodec", func() {
	var codec encoding.Codec
	BeforeEach(func() {
		codec = encoding.NewDecodeFallbackCodec(gob.Codec, json.Codec)
	})
	Describe("Decoding", func() {
		It(
			"Should fallback to the next codec when the first one fails",
			func(ctx SpecContext) {
				type abc struct {
					Value int `json:"value"`
				}
				v := abc{Value: 12}
				jsonB := MustSucceed(json.Codec.Encode(ctx, v))
				gobB := MustSucceed(gob.Codec.Encode(ctx, v))
				var res abc
				Expect(codec.Decode(ctx, jsonB, &res)).To(Succeed())
				Expect(res).To(Equal(v))
				Expect(codec.Decode(ctx, gobB, &res)).To(Succeed())
				Expect(res).To(Equal(v))
			},
		)
		It(
			"Should return the error of the last encoder if all codecs fail to encode",
			func(ctx SpecContext) {
				Expect(codec.Encode(ctx, make(chan int))).Error().To(HaveOccurred())
			},
		)
		It(
			"Should return an error when all codecs fail to decode",
			func(ctx SpecContext) {
				invalidData := []byte("completely invalid data")
				Expect(
					codec.Decode(ctx, invalidData, &struct{ Value int }{}),
				).To(MatchError(ContainSubstring("all codecs failed to decode")))
			},
		)
		It("Should handle DecodeStream fallback correctly", func(ctx SpecContext) {
			type abc struct {
				Value int `json:"value"`
			}
			v := abc{Value: 123}
			jsonB := MustSucceed(json.Codec.Encode(ctx, v))
			var res abc
			buf := bytes.NewBuffer(jsonB)
			Expect(codec.DecodeStream(ctx, buf, &res)).To(Succeed())
			Expect(res).To(Equal(v))
		})
		It(
			"Should return error when DecodeStream fails for all codecs",
			func(ctx SpecContext) {
				invalidData := []byte("completely invalid data")
				var res struct{ Value int }
				Expect(
					codec.DecodeStream(ctx, bytes.NewReader(invalidData), &res),
				).To(MatchError(ContainSubstring("all codecs failed to decode")))
			},
		)
	})
})
