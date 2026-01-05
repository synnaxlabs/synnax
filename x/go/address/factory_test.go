// Copyright 2026 Synnax Labs, Inc.
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

var _ = Describe("Factory", func() {
	Describe("NewLocalFactory", func() {
		It("Should create a factory with localhost and specified port", func() {
			f := address.NewLocalFactory(8080)
			addr := f.Next()
			Expect(addr.Host()).To(Equal("localhost"))
			Expect(addr.Port()).To(Equal(8080))
		})
	})

	Describe("Next", func() {
		It("Should increment port number for each call", func() {
			f := address.NewLocalFactory(8080)
			addr1 := f.Next()
			addr2 := f.Next()
			addr3 := f.Next()

			Expect(addr1.Port()).To(Equal(8080))
			Expect(addr2.Port()).To(Equal(8081))
			Expect(addr3.Port()).To(Equal(8082))
		})

		It("Should maintain the same host across calls", func() {
			f := address.NewLocalFactory(8080)
			addr1 := f.Next()
			addr2 := f.Next()

			Expect(addr1.Host()).To(Equal("localhost"))
			Expect(addr2.Host()).To(Equal("localhost"))
		})
	})

	Describe("Generated", func() {
		It("Should return a slice of all generated addresses", func() {
			f := address.NewLocalFactory(8080)
			addr1 := f.Next()
			addr2 := f.Next()
			Expect(f.Generated()).To(ConsistOf(addr1, addr2))
		})
	})

	Describe("NextN", func() {
		It("Should generate the requested number of addresses", func() {
			f := address.NewLocalFactory(8080)
			addrs := f.NextN(3)
			Expect(addrs).To(HaveLen(3))
		})

		It("Should generate sequential port numbers", func() {
			f := address.NewLocalFactory(8080)
			addrs := f.NextN(3)
			Expect(addrs[0].Port()).To(Equal(8080))
			Expect(addrs[1].Port()).To(Equal(8081))
			Expect(addrs[2].Port()).To(Equal(8082))
		})

		It("Should maintain correct sequence after multiple NextN calls", func() {
			f := address.NewLocalFactory(8080)
			first := f.NextN(2)
			second := f.NextN(2)

			Expect(first[0].Port()).To(Equal(8080))
			Expect(first[1].Port()).To(Equal(8081))
			Expect(second[0].Port()).To(Equal(8082))
			Expect(second[1].Port()).To(Equal(8083))
		})

		It("Should handle zero addresses request", func() {
			f := address.NewLocalFactory(8080)
			addrs := f.NextN(0)
			Expect(addrs).To(BeEmpty())
		})

		It("Should interleave correctly with Next calls", func() {
			f := address.NewLocalFactory(8080)
			single := f.Next()
			batch := f.NextN(2)
			lastSingle := f.Next()

			Expect(single.Port()).To(Equal(8080))
			Expect(batch[0].Port()).To(Equal(8081))
			Expect(batch[1].Port()).To(Equal(8082))
			Expect(lastSingle.Port()).To(Equal(8083))
		})
	})
})
