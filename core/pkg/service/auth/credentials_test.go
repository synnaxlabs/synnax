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

var _ = Describe("InsecureCredentials", func() {
	Describe("IsZero", func() {
		It("Should return true for empty credentials", func() {
			Expect(auth.InsecureCredentials{}.IsZero()).To(BeTrue())
		})
		It("Should return false when username is set", func() {
			Expect(auth.InsecureCredentials{Username: "synnax"}.IsZero()).To(BeFalse())
		})
		It("Should return false when password is set", func() {
			Expect(auth.InsecureCredentials{Password: "seldon"}.IsZero()).To(BeFalse())
		})
		It("Should return false when both are set", func() {
			Expect(auth.InsecureCredentials{
				Username: "synnax",
				Password: "seldon",
			}.IsZero()).To(BeFalse())
		})
	})
	Describe("Validate", func() {
		It("Should return an error when the username is empty", func() {
			Expect(auth.InsecureCredentials{Password: "p"}.Validate()).To(
				MatchError(ContainSubstring("username: required")),
			)
		})
		It("Should return an error when the password is empty", func() {
			Expect(auth.InsecureCredentials{Username: "u"}.Validate()).To(
				MatchError(ContainSubstring("password: required")),
			)
		})
		It("Should succeed when both fields are set", func() {
			Expect(auth.InsecureCredentials{Username: "u", Password: "p"}.Validate()).
				To(Succeed())
		})
	})
})

var _ = Describe("SecureCredentials", func() {
	Describe("GorpKey", func() {
		It("Should return the username", func() {
			Expect(auth.SecureCredentials{Username: "synnax"}.GorpKey()).To(Equal("synnax"))
		})
	})
	Describe("SetOptions", func() {
		It("Should return an empty slice", func() {
			Expect(auth.SecureCredentials{Username: "synnax"}.SetOptions()).To(BeEmpty())
		})
	})
})
