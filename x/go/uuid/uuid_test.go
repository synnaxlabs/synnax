// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package uuid_test

import (
	"encoding/json"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/testutil"
	xuuid "github.com/synnaxlabs/x/uuid"
)

var _ = Describe("Encoding", func() {
	text := "01020304-0506-0708-090a-0b0c0d0e0f10"
	key := uuid.UUID{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d,
		0x0e, 0x0f, 0x10,
	}

	It("Should encode a UUID as its lowercase 8-4-4-4-12 text", func() {
		Expect(key.String()).To(Equal(text))
		Expect(string(MustSucceed(key.MarshalText()))).To(Equal(text))
		Expect(string(MustSucceed(json.Marshal(key)))).To(Equal(`"` + text + `"`))
	})

	It("Should decode a UUID from its JSON text", func() {
		var decoded uuid.UUID
		Expect(json.Unmarshal([]byte(`"`+text+`"`), &decoded)).To(Succeed())
		Expect(decoded).To(Equal(key))
	})

	DescribeTable("Should parse the text forms clients send", func(s string) {
		Expect(MustSucceed(uuid.Parse(s))).To(Equal(key))
	},
		Entry("uppercase", "01020304-0506-0708-090A-0B0C0D0E0F10"),
		Entry("braces", "{"+text+"}"),
		Entry("URN", "urn:uuid:"+text),
		Entry("no hyphens", "0102030405060708090a0b0c0d0e0f10"),
	)

	It("Should encode the nil UUID as all zeros", func() {
		Expect(uuid.Nil().String()).To(Equal("00000000-0000-0000-0000-000000000000"))
	})
})

var _ = Describe("NewV5", func() {
	dns := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")

	It("Should derive the RFC 9562 example UUID", func() {
		Expect(xuuid.NewV5(dns, "www.example.com").String()).
			To(Equal("2ed6657d-e927-568b-95e1-2665a8aea6a2"))
	})

	It("Should derive a different UUID for a different name", func() {
		Expect(xuuid.NewV5(dns, "example.com")).
			ToNot(Equal(xuuid.NewV5(dns, "www.example.com")))
	})
})
