// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package library_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/library"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Hash", func() {
	base := func() library.Library {
		return library.Library{
			Key:  uuid.New(),
			Name: "A",
			Entries: []library.Entry{
				enumEntry("State", library.EnumValue{Value: 1, Name: "On"}),
			},
		}
	}

	It("Should return 16 lowercase hex characters", func() {
		Expect(MustSucceed(library.Hash(base()))).To(MatchRegexp("^[0-9a-f]{16}$"))
	})

	It("Should hash equal entries equally regardless of key and name", func() {
		a, b := base(), base()
		b.Name = "B"
		Expect(MustSucceed(library.Hash(a))).To(Equal(MustSucceed(library.Hash(b))))
	})

	It("Should change when an entry changes", func() {
		a, b := base(), base()
		b.Entries = []library.Entry{
			enumEntry("State", library.EnumValue{Value: 2, Name: "On"}),
		}
		Expect(MustSucceed(library.Hash(a))).ToNot(Equal(MustSucceed(library.Hash(b))))
	})
})
