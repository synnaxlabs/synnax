// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Hash", func() {
	base := func() icd.ICD {
		return icd.ICD{
			Key:  uuid.New(),
			Name: "A",
			Messages: []icd.Message{
				canMessage("State", 0x100, binaryField("on", 0, 1)),
			},
		}
	}

	It("Should return 16 lowercase hex characters", func() {
		Expect(MustSucceed(icd.Hash(base()))).To(MatchRegexp("^[0-9a-f]{16}$"))
	})

	It("Should ignore the key and the stored hash", func() {
		a, b := base(), base()
		b.Hash = "0123456789abcdef"
		Expect(MustSucceed(icd.Hash(a))).To(Equal(MustSucceed(icd.Hash(b))))
	})

	It("Should not change when the name changes", func() {
		a, b := base(), base()
		b.Name = "B"
		Expect(MustSucceed(icd.Hash(a))).To(Equal(MustSucceed(icd.Hash(b))))
	})

	It("Should change when a message changes", func() {
		a := base()
		b := setScale(a, 0, 0.5)
		Expect(MustSucceed(icd.Hash(a))).ToNot(Equal(MustSucceed(icd.Hash(b))))
	})
})
