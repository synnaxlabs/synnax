// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package auth_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	vmsgpack "github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("Proof MessagePack", func() {
	It("Should round trip a password proof", func(ctx SpecContext) {
		proof := auth.Proof{Variant: auth.PasswordProof{
			Credentials: auth.Credentials{Username: "jane", Password: "secret"},
		}}
		b := MustSucceed(msgpack.Codec.Encode(ctx, proof))
		var decoded auth.Proof
		Expect(msgpack.Codec.Decode(ctx, b, &decoded)).To(Succeed())
		Expect(decoded).To(Equal(proof))
	})
	It("Should read the shape clients send", func(ctx SpecContext) {
		b := MustSucceed(msgpack.Codec.Encode(ctx, map[string]string{
			"method":   "password",
			"username": "jane",
			"password": "secret",
		}))
		var decoded auth.Proof
		Expect(msgpack.Codec.Decode(ctx, b, &decoded)).To(Succeed())
		creds := auth.Credentials{Username: "jane", Password: "secret"}
		Expect(decoded.Variant).To(Equal(auth.PasswordProof{Credentials: creds}))
	})
	It("Should round trip an API key proof", func(ctx SpecContext) {
		proof := auth.Proof{Variant: auth.APIKeyProof{
			APIKeyCredentials: auth.APIKeyCredentials{Key: "syk_abc"},
		}}
		b := MustSucceed(msgpack.Codec.Encode(ctx, proof))
		var decoded auth.Proof
		Expect(msgpack.Codec.Decode(ctx, b, &decoded)).To(Succeed())
		Expect(decoded).To(Equal(proof))
	})
	It("Should reject an API key proof whose key is not a string", func(
		ctx SpecContext,
	) {
		b := MustSucceed(msgpack.Codec.Encode(
			ctx,
			map[string]any{"method": "api_key", "key": []int{1}},
		))
		Expect(vmsgpack.Unmarshal(b, &auth.Proof{})).
			To(MatchError(ContainSubstring("decoding string/bytes")))
	})
	It("Should reject an unknown method", func(ctx SpecContext) {
		b := MustSucceed(
			msgpack.Codec.Encode(ctx, map[string]string{"method": "magic"}),
		)
		Expect(vmsgpack.Unmarshal(b, &auth.Proof{})).
			To(MatchError(validate.ErrValidation))
	})
	It("Should encode an empty proof as nil", func(ctx SpecContext) {
		b := MustSucceed(msgpack.Codec.Encode(ctx, auth.Proof{}))
		var decoded auth.Proof
		Expect(msgpack.Codec.Decode(ctx, b, &decoded)).To(Succeed())
		Expect(decoded.Variant).To(BeNil())
	})
})
