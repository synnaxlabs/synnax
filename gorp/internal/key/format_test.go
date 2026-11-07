// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package key_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp/internal/key"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Format", func() {
	It("Should write a string key as itself", func() {
		Expect(key.Format(name("cat"))).To(Equal("cat"))
		Expect(MustSucceed(key.Parse[name]("cat"))).To(Equal(name("cat")))
	})

	It("Should write an integer key in base 10", func() {
		Expect(key.Format(int32(-12))).To(Equal("-12"))
		Expect(key.Format(uint64(1) << 40)).To(Equal("1099511627776"))
		Expect(MustSucceed(key.Parse[uint32]("42"))).To(Equal(uint32(42)))
	})

	It("Should write a 16-byte key in the UUID form", func() {
		u := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
		Expect(key.Format(u)).To(Equal(u.String()))
		Expect(MustSucceed(key.Parse[uuid.UUID](u.String()))).To(Equal(u))
	})

	It("Should reject text that is not a key", func() {
		Expect(key.Parse[uint8]("300")).Error().
			To(MatchError(ContainSubstring(`invalid key "300"`)))
		Expect(key.Parse[uuid.UUID]("abcd")).Error().
			To(MatchError(ContainSubstring("expected 16 bytes")))
	})
})
