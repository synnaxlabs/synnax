// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package address_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/synnaxlabs/x/address"
)

var _ = Describe("Address", func() {
	Describe("Newf", func() {
		It("Should create a new address with formatted string", func() {
			addr := address.Newf("localhost:%d", 8080)
			Expect(addr.String()).To(Equal("localhost:8080"))
		})

		It("Should handle multiple format arguments", func() {
			addr := address.Newf("%s:%d", "example.com", 9090)
			Expect(addr.String()).To(Equal("example.com:9090"))
		})
	})

	Describe("String Operations", func() {
		Context("With valid address", func() {
			var addr address.Address

			BeforeEach(func() {
				addr = address.Address("localhost:8080")
			})

			It("Should convert to string correctly", func() {
				Expect(addr.String()).To(Equal("localhost:8080"))
			})

			It("Should extract port string with colon", func() {
				Expect(addr.PortString()).To(Equal(":8080"))
			})

			It("Should extract numeric port value", func() {
				Expect(addr.Port()).To(Equal(8080))
			})

			It("Should extract host string", func() {
				Expect(addr.Host()).To(Equal("localhost"))
			})
		})

		Context("With invalid address", func() {
			It("Should handle malformed port", func() {
				addr := address.Address("localhost:invalid")
				Expect(addr.Port()).To(Equal(0))
			})

			It("Should still extract port string even if invalid", func() {
				addr := address.Address("localhost:invalid")
				Expect(addr.PortString()).To(Equal(":invalid"))
			})
		})
	})

	Describe("Rand", func() {
		It("Should generate UUID formatted addresses", func() {
			addr := address.Rand()
			// UUID v4 format: 8-4-4-4-12 characters
			Expect(addr.String()).To(HaveLen(36))
			Expect(addr.String()).To(MatchRegexp(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`))
		})

		It("Should generate unique addresses", func() {
			seen := make(map[string]bool)
			for range 1000 {
				addr := address.Rand()
				Expect(seen[addr.String()]).To(BeFalse(), "Generated duplicate address")
				seen[addr.String()] = true
			}
		})
	})

})
