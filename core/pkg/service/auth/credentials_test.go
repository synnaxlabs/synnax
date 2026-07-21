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

var _ = Describe("SecureCredentials", func() {
	Describe("GorpKey", func() {
		It("Should return the username", func() {
			Expect(auth.SecureCredentials{Username: "root"}.GorpKey()).
				To(Equal("root"))
		})
	})

	Describe("SetOptions", func() {
		It("Should return no options", func() {
			Expect(auth.SecureCredentials{}.SetOptions()).To(BeNil())
		})
	})
})
