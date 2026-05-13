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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Password", func() {
	Describe("Hash", func() {
		It("Should hash a password without error", func() {
			Expect(auth.RawPassword("password").Hash()).ToNot(BeEmpty())
		})
		It("Should return ErrInvalidHash for a password longer than bcrypt allows", func() {
			Expect(auth.RawPassword(strings.Repeat("a", 73)).Hash()).Error().
				To(MatchError(auth.ErrInvalidHash))
		})
	})
	Describe("Validate", func() {
		var (
			raw    auth.RawPassword
			hashed auth.HashedPassword
		)
		BeforeEach(func() {
			raw = auth.RawPassword("password")
			hashed = MustSucceed(raw.Hash())
		})
		It("Should return a nil error for a valid password", func() {
			Expect(hashed.Validate(raw)).To(Succeed())
		})
		It("Should return ErrInvalidCredentials for an invalid password", func() {
			Expect(hashed.Validate("wrong")).To(MatchError(auth.ErrInvalidCredentials))
		})
	})
})
