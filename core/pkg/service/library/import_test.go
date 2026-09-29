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
	"github.com/synnaxlabs/synnax/pkg/service/library/icd"
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

var _ = Describe("ImportICD", func() {
	var lib library.Library
	BeforeEach(func(ctx SpecContext) {
		lib = library.Library{
			Name: "Bus",
			Entries: []library.Entry{
				messageEntry(canMessage("Old", 0x200, binaryField("a", 0, 8))),
			},
		}
		Expect(svc.NewWriter(tx).Create(ctx, &lib)).To(Succeed())
	})

	importICD := func(
		ctx SpecContext,
		key library.Key,
		format icd.Format,
		data string,
	) (library.Library, error) {
		return svc.NewWriter(tx).ImportICD(ctx, key, format, []byte(data))
	}

	It("Should replace the entries of the library", func(ctx SpecContext) {
		res := MustSucceed(importICD(ctx, lib.Key, icd.FormatDBC, importDBC))
		Expect(res.Key).To(Equal(lib.Key))
		Expect(res.Name).To(Equal("Bus"))
		Expect(res.Entries).To(HaveLen(2))
		m := messageOf(res, 0)
		Expect(m.Name).To(Equal("Engine"))
		gear := res.Entries[1].Variant.(library.EnumEntry)
		Expect(*binaryFieldOf(m, 1).Enumeration).To(Equal(gear.Key))
		var stored library.Library
		Expect(svc.NewRetrieve().
			Where(library.MatchKeys(lib.Key)).
			Entry(&stored).
			Exec(ctx, tx)).To(Succeed())
		Expect(stored).To(Equal(res))
	})

	It("Should keep every key when a file is imported again", func(ctx SpecContext) {
		first := MustSucceed(importICD(ctx, lib.Key, icd.FormatDBC, importDBC))
		second := MustSucceed(importICD(ctx, lib.Key, icd.FormatDBC, importDBC))
		Expect(second).To(Equal(first))
	})

	It("Should re-stamp every task that uses the library", func(ctx SpecContext) {
		t := createUsingTask(ctx, lib.Key)
		res := MustSucceed(importICD(ctx, lib.Key, icd.FormatDBC, importDBC))
		hash := MustSucceed(library.Hash(res))
		Expect(hash).ToNot(Equal(t.Config["library_hash"]))
		config := retrieveTask(ctx, t.Key).Config
		Expect(config).To(HaveKeyWithValue("library_hash", hash))
	})

	It("Should reject data that does not parse", func(ctx SpecContext) {
		Expect(importICD(ctx, lib.Key, icd.FormatCSV, "color\n")).Error().To(
			MatchError(`data: row 1: unknown column "color": validation error`),
		)
	})

	It("Should reject an imported library that is invalid", func(ctx SpecContext) {
		csv := "message,field,start_bit,bit_length\nm,a,0,8\nm,a,8,8\n"
		Expect(importICD(ctx, lib.Key, icd.FormatCSV, csv)).Error().To(MatchError(
			ContainSubstring(`entries.0.fields.1.name: duplicate field name "a"`),
		))
	})

	It("Should return not found for a missing library", func(ctx SpecContext) {
		Expect(importICD(ctx, uuid.New(), icd.FormatDBC, importDBC)).Error().
			To(MatchError(query.ErrNotFound))
	})
})
