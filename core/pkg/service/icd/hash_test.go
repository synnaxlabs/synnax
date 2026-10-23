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
			Entries: []icd.Entry{
				enumEntry("State", icd.EnumValue{Value: 1, Name: "On"}),
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

	It("Should change when an entry changes", func() {
		a, b := base(), base()
		b.Entries = []icd.Entry{enumEntry("State", icd.EnumValue{Value: 2, Name: "On"})}
		Expect(MustSucceed(icd.Hash(a))).ToNot(Equal(MustSucceed(icd.Hash(b))))
	})
})

var _ = Describe("TaskHash", func() {
	var (
		ic                     icd.ICD
		gear, mode             icd.EnumEntry
		engine, brake, unnamed icd.MessageEntry
	)
	BeforeEach(func() {
		gear = icd.EnumEntry{
			Key: uuid.New(), Name: "Gear",
			Values: []icd.EnumValue{{Value: 0, Name: "Park"}},
		}
		mode = icd.EnumEntry{
			Key: uuid.New(), Name: "Mode",
			Values: []icd.EnumValue{{Value: 0, Name: "Idle"}},
		}
		geared := binaryField("gear", 0, 8)
		geared.Enumeration = new(gear.Key)
		engine = canMessage("Engine", 0x100, geared)
		engine.Key = uuid.New()
		brake = canMessage("Brake", 0x200, binaryField("pressure", 0, 16))
		brake.Key = uuid.New()
		unnamed = canMessage("Other", 0x300, binaryField("x", 0, 8))
		unnamed.Key = uuid.New()
		ic = icd.ICD{
			Name: "Bus",
			Entries: []icd.Entry{
				{Variant: gear},
				{Variant: mode},
				messageEntry(engine),
				messageEntry(brake),
				messageEntry(unnamed),
			},
		}
	})

	hash := func(ic icd.ICD, messages ...icd.EntryKey) string {
		GinkgoHelper()
		return MustSucceed(icd.TaskHash(ic, messages))
	}

	It("Should not change when an unused message changes", func() {
		before := hash(ic, engine.Key)
		ic = setScale(ic, 3, 0.5)
		Expect(hash(ic, engine.Key)).To(Equal(before))
	})

	It("Should not change when an unused enum changes", func() {
		before := hash(ic, engine.Key)
		mode.Values = append(mode.Values, icd.EnumValue{Value: 1, Name: "Run"})
		ic.Entries[1] = icd.Entry{Variant: mode}
		Expect(hash(ic, engine.Key)).To(Equal(before))
	})

	It("Should not change when the ICD is renamed", func() {
		before := hash(ic, engine.Key)
		ic.Name = "Renamed"
		Expect(hash(ic, engine.Key)).To(Equal(before))
	})

	It("Should change when a used message changes", func() {
		before := hash(ic, engine.Key)
		ic = setScale(ic, 2, 0.5)
		Expect(hash(ic, engine.Key)).ToNot(Equal(before))
	})

	It("Should change when an enum a used field names changes", func() {
		before := hash(ic, engine.Key)
		gear.Values = append(gear.Values, icd.EnumValue{Value: 1, Name: "Drive"})
		ic.Entries[0] = icd.Entry{Variant: gear}
		Expect(hash(ic, engine.Key)).ToNot(Equal(before))
	})

	It("Should not change when the entries are reordered", func() {
		before := hash(ic, engine.Key, brake.Key)
		ic.Entries[2], ic.Entries[3] = ic.Entries[3], ic.Entries[2]
		ic.Entries[0], ic.Entries[1] = ic.Entries[1], ic.Entries[0]
		Expect(hash(ic, brake.Key, engine.Key)).To(Equal(before))
	})

	It("Should ignore messages the ICD does not hold", func() {
		Expect(hash(ic, engine.Key, uuid.New())).To(Equal(hash(ic, engine.Key)))
	})

	It("Should differ for tasks that use different messages", func() {
		Expect(hash(ic, engine.Key)).ToNot(Equal(hash(ic, brake.Key)))
	})
})
