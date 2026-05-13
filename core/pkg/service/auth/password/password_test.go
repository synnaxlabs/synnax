// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package password_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Password", func() {
	Describe("Hash", func() {
		It("Should hash a password without error", func() {
			raw := password.Raw("password")
			Expect(raw.Hash()).ToNot(BeEmpty())
		})
		It("Should return a password.ErrInvalid error for a password that is too long", func() {
			raw := password.Raw(strings.Repeat("a", 73))
			Expect(raw.Hash()).Error().To(MatchError(password.ErrInvalidHash))
		})
	})
	Describe("Compare", func() {
		var (
			raw    password.Raw
			hashed password.Hashed
		)
		BeforeEach(func() {
			raw = password.Raw("password")
			hashed = MustSucceed(raw.Hash())
		})
		It("Should return a nil error for a valid password", func() {
			Expect(hashed.Validate(raw)).To(Succeed())
		})
		It("Should return a password.ErrInvalid for an invalid password", func() {
			Expect(hashed.Validate("wrong")).To(MatchError(password.ErrInvalid))
		})
	})
})
