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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xuuid "github.com/synnaxlabs/x/uuid"
)

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
