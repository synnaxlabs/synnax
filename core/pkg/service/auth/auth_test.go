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
