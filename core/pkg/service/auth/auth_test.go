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
	"encoding/json/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Credentials", func() {
	Describe("IsZero", func() {
		It("Should return true for empty credentials", func() {
			Expect(auth.Credentials{}.IsZero()).To(BeTrue())
		})
		It("Should return false when username is set", func() {
			Expect(auth.Credentials{Username: "synnax"}.IsZero()).To(BeFalse())
		})
		It("Should return false when password is set", func() {
			Expect(auth.Credentials{Password: "seldon"}.IsZero()).To(BeFalse())
		})
		It("Should return false when both are set", func() {
			Expect(auth.Credentials{
				Username: "synnax",
				Password: "seldon",
			}.IsZero()).To(BeFalse())
		})
	})
	Describe("Validate", func() {
		It("Should return an error when the username is empty", func() {
			Expect(auth.Credentials{Password: "p"}.Validate()).To(
				MatchError(ContainSubstring("username: required")),
			)
		})
		It("Should return an error when the password is empty", func() {
			Expect(auth.Credentials{Username: "u"}.Validate()).To(
				MatchError(ContainSubstring("password: required")),
			)
		})
		It("Should succeed when both fields are set", func() {
			Expect(auth.Credentials{Username: "u", Password: "p"}.Validate()).
				To(Succeed())
		})
	})
})

var _ = Describe("Proof", func() {
	Describe("JSON", func() {
		It("Should round trip a password proof with its method", func() {
			proof := auth.Proof{Variant: auth.PasswordProof{
				Credentials: auth.Credentials{Username: "u", Password: "p"},
			}}
			data := MustSucceed(json.Marshal(proof))
			Expect(string(data)).To(ContainSubstring(`"method":"password"`))
			var decoded auth.Proof
			Expect(json.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded).To(Equal(proof))
		})
		It("Should reject an unknown method", func() {
			var decoded auth.Proof
			Expect(json.Unmarshal([]byte(`{"method":"telepathy"}`), &decoded)).
				To(MatchError(ContainSubstring("unknown method")))
		})
		It("Should encode and decode null as no variant", func() {
			data := MustSucceed(json.Marshal(auth.Proof{}))
			Expect(string(data)).To(Equal("null"))
			var decoded auth.Proof
			Expect(json.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Variant).To(BeNil())
		})
	})
})
