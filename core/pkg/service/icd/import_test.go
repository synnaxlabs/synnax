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
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

const importDBC = `VERSION ""

NS_ :

BS_:

BU_: Ecu

BO_ 256 Engine: 8 Ecu
 SG_ Rpm : 0|16@1+ (0.25,0) [0|16383.75] "rpm" Ecu
 SG_ Gear : 16|8@1+ (1,0) [0|255] "" Ecu

VAL_ 256 Gear 0 "Park" 1 "Drive" ;
`

var _ = Describe("Import", func() {
	var ic icd.ICD
	BeforeEach(func(ctx SpecContext) {
		ic = icd.ICD{
			Name: "Bus",
			Entries: []icd.Entry{
				messageEntry(canMessage("Engine", 0x100, binaryField("Rpm", 0, 16))),
			},
		}
		Expect(svc.NewWriter(tx).Create(ctx, &ic)).To(Succeed())
	})

	importICD := func(
		ctx SpecContext,
		key icd.Key,
		format icd.Format,
		data string,
	) (icd.ICD, error) {
		return svc.NewWriter(tx).Import(ctx, key, format, []byte(data))
	}

	It("Should replace the entries of the ICD", func(ctx SpecContext) {
		res := MustSucceed(importICD(ctx, ic.Key, icd.FormatDBC, importDBC))
		Expect(res.Key).To(Equal(ic.Key))
		Expect(res.Name).To(Equal("Bus"))
		Expect(res.Entries).To(HaveLen(2))
		m := messageOf(res, 0)
		Expect(m.Name).To(Equal("Engine"))
		gear := res.Entries[1].Variant.(icd.EnumEntry)
		Expect(*binaryFieldOf(m, 1).Enumeration).To(Equal(gear.Key))
		Expect(retrieveICD(ctx, ic.Key)).To(Equal(res))
	})

	It("Should keep every key when a file is imported again", func(ctx SpecContext) {
		first := MustSucceed(importICD(ctx, ic.Key, icd.FormatDBC, importDBC))
		second := MustSucceed(importICD(ctx, ic.Key, icd.FormatDBC, importDBC))
		Expect(second).To(Equal(first))
	})

	It("Should rewrite each task whose messages change", func(ctx SpecContext) {
		t := createUsingTask(ctx, "Decoder", ic, messageOf(ic, 0))
		res := MustSucceed(importICD(ctx, ic.Key, icd.FormatDBC, importDBC))
		hash := MustSucceed(icd.TaskHash(res, []icd.EntryKey{messageOf(res, 0).Key}))
		Expect(hash).ToNot(Equal(t.Config["icd_hash"]))
		config := retrieveTask(ctx, t.Key).Config
		Expect(config).To(HaveKeyWithValue("icd_hash", hash))
	})

	It("Should reject data that does not parse", func(ctx SpecContext) {
		Expect(importICD(ctx, ic.Key, icd.FormatCSV, "color\n")).Error().To(
			MatchError(`data: row 1: unknown column "color": validation error`),
		)
	})

	It("Should reject an imported ICD that is invalid", func(ctx SpecContext) {
		csv := "message,field,start_bit,bit_length\nm,a,0,8\nm,a,8,8\n"
		Expect(importICD(ctx, ic.Key, icd.FormatCSV, csv)).Error().To(MatchError(
			ContainSubstring(
				`entries.0.payload.fields.1.name: duplicate field name "a": ` +
					"validation error",
			),
		))
	})

	It("Should return not found for a missing ICD", func(ctx SpecContext) {
		Expect(importICD(ctx, uuid.New(), icd.FormatDBC, importDBC)).Error().
			To(MatchError(query.ErrNotFound))
	})
})
