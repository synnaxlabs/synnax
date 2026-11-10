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
	"encoding/base64"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Token", func() {
	It("Should round trip a key and the hash of its secret", func() {
		key := uuid.New()
		token, hash := MustSucceed2(auth.NewToken("syk_", key))
		Expect(token).To(HavePrefix("syk_"))
		parsedKey, parsedHash := MustSucceed2(auth.ParseToken("syk_", token))
		Expect(parsedKey).To(Equal(key))
		Expect(parsedHash).To(Equal(hash))
	})
	It("Should make a different secret every time", func() {
		key := uuid.New()
		a, aHash := MustSucceed2(auth.NewToken("sys_", key))
		b, bHash := MustSucceed2(auth.NewToken("sys_", key))
		Expect(a).ToNot(Equal(b))
		Expect(aHash).ToNot(Equal(bHash))
	})
	DescribeTable("Should reject a malformed token", func(token string) {
		Expect(auth.ParseToken("syk_", token)).Error().
			To(MatchError(auth.ErrInvalidToken))
	},
		Entry("wrong prefix", func() string {
			token, _ := MustSucceed2(auth.NewToken("sys_", uuid.New()))
			return token
		}()),
		Entry("invalid encoding", "syk_!!!"),
		Entry(
			"wrong decoded length",
			"syk_"+base64.RawURLEncoding.EncodeToString([]byte("short")),
		),
		Entry("empty", ""),
		Entry("prefix only", "syk_"),
	)
})
