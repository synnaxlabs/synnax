// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package identifier_test

import (
	"strings"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/identifier"
)

var _ = Describe("IsKey", func() {
	Describe("UUID inputs", func() {
		It("Should accept a freshly generated UUID", func() {
			Expect(identifier.IsKey(uuid.NewString())).To(BeTrue())
		})

		It("Should accept the zero UUID", func() {
			Expect(identifier.IsKey("00000000-0000-0000-0000-000000000000")).To(BeTrue())
		})

		It("Should accept an upper-cased UUID (uuid.Parse is case-insensitive)", func() {
			Expect(identifier.IsKey(strings.ToUpper(uuid.NewString()))).To(BeTrue())
		})
	})

	Describe("Non-UUID inputs", func() {
		It("Should reject a snake_case name", func() {
			Expect(identifier.IsKey("ox_alarm")).To(BeFalse())
		})

		It("Should reject an empty string", func() {
			Expect(identifier.IsKey("")).To(BeFalse())
		})

		It("Should reject a clearly non-UUID string", func() {
			Expect(identifier.IsKey("not-a-uuid")).To(BeFalse())
		})

		It("Should reject a UUID-shaped string with non-hex characters", func() {
			Expect(identifier.IsKey("xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx")).To(BeFalse())
		})
	})

	Describe("Whitespace and surrounding-character edges", func() {
		It("Should reject a UUID with a trailing space (no trim)", func() {
			Expect(identifier.IsKey(uuid.NewString() + " ")).To(BeFalse())
		})

		It("Should reject a UUID with a leading space", func() {
			Expect(identifier.IsKey(" " + uuid.NewString())).To(BeFalse())
		})

		It("Should reject a UUID with trailing junk characters", func() {
			Expect(identifier.IsKey("12345678-1234-1234-1234-123456789012extra")).To(BeFalse())
		})
	})
})
