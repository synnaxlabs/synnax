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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
)

var _ = Describe("Password", func() {
	Describe("Hash", func() {
		It("Should hash a password without error", func() {
			raw := password.Raw("password")
			_, err := raw.Hash()
			Expect(err).ToNot(HaveOccurred())
		})
	})
	Describe("Compare", func() {
		It("Should return a nil error for a valid password", func() {
			raw := password.Raw("password")
			hashed, err := raw.Hash()
			Expect(err).ToNot(HaveOccurred())
			err = hashed.Validate(raw)
			Expect(err).ToNot(HaveOccurred())
		})
		It("Should return a password.Invalid error for an invalid password", func() {
			raw := password.Raw("password")
			hashed, err := raw.Hash()
			Expect(err).ToNot(HaveOccurred())
			err = hashed.Validate("wrong")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(password.ErrInvalid))
		})
	})
})
