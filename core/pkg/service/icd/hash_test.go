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

var _ = Describe("TaskHash", func() {
	var (
		ic                   icd.ICD
		engine, brake, other icd.Message
	)
	BeforeEach(func() {
		engine = canMessage("Engine", 0x100, binaryField("gear", 0, 8))
		engine.Key = uuid.New()
		brake = canMessage("Brake", 0x200, binaryField("pressure", 0, 16))
		brake.Key = uuid.New()
		other = canMessage("Other", 0x300, binaryField("x", 0, 8))
		other.Key = uuid.New()
		ic = icd.ICD{
			Name:     "Bus",
			Messages: []icd.Message{engine, brake, other},
		}
	})

	hash := func(ic icd.ICD, messages ...icd.MessageKey) string {
		GinkgoHelper()
		return MustSucceed(icd.TaskHash(ic, messages))
	}

	It("Should not change when an unused message changes", func() {
		before := hash(ic, engine.Key)
		ic = setScale(ic, 1, 0.5)
		Expect(hash(ic, engine.Key)).To(Equal(before))
	})

	It("Should not change when the ICD is renamed", func() {
		before := hash(ic, engine.Key)
		ic.Name = "Renamed"
		Expect(hash(ic, engine.Key)).To(Equal(before))
	})

	It("Should change when a used message changes", func() {
		before := hash(ic, engine.Key)
		ic = setScale(ic, 0, 0.5)
		Expect(hash(ic, engine.Key)).ToNot(Equal(before))
	})

	It("Should not change when the messages are reordered", func() {
		before := hash(ic, engine.Key, brake.Key)
		ic.Messages[0], ic.Messages[1] = ic.Messages[1], ic.Messages[0]
		Expect(hash(ic, brake.Key, engine.Key)).To(Equal(before))
	})

	It("Should ignore messages the ICD does not hold", func() {
		Expect(hash(ic, engine.Key, uuid.New())).To(Equal(hash(ic, engine.Key)))
	})

	It("Should differ for tasks that use different messages", func() {
		Expect(hash(ic, engine.Key)).ToNot(Equal(hash(ic, brake.Key)))
	})
})
