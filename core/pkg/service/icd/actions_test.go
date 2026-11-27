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
	"github.com/synnaxlabs/x/patch"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Actions", func() {
	base := icd.ICD{Key: uuid.New(), Name: "Bus", Messages: []icd.Message{}}
	set := func(path []string, value any) icd.Action {
		return icd.NewSetAction(icd.SetPayload{Path: path, Value: value})
	}

	DescribeTable("Should reject", func(act icd.Action) {
		Expect(icd.Reduce(base, act)).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("decoding document")),
		))
	},
		Entry("a path to the key", set([]string{"key"}, uuid.New().String())),
		Entry("a value the name cannot hold", set([]string{"name"}, 1)),
		Entry(
			"a list the ICD does not declare",
			icd.NewInsertAction(icd.InsertPayload{
				Path:  []string{"tags"},
				Value: map[string]any{"key": "a"},
			}),
		),
	)

	It("Should reject a create", func() {
		Expect(icd.Reduce(
			base,
			icd.NewCreateAction(icd.CreatePayload{ICD: base}),
		)).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("only the Core dispatches an ICD create")),
		))
	})

	It("Should reject the removal of the name", func() {
		Expect(icd.Reduce(
			base,
			icd.NewRemoveAction(icd.RemovePayload{Key: "name"}),
		)).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("[name]: the document requires a value")),
		))
	})

	It("Should set the name and keep the key", func() {
		Expect(icd.Reduce(base, set([]string{"name"}, "Vehicle"))).To(Equal(icd.ICD{
			Key:      base.Key,
			Name:     "Vehicle",
			Messages: []icd.Message{},
		}))
	})

	Describe("Framing", func() {
		fixed := func(length uint32) *icd.Framing {
			return &icd.Framing{Variant: icd.FixedFraming{Length: length}}
		}

		It("Should set the framing of an ICD with none", func() {
			next := MustSucceed(
				icd.Reduce(base, set([]string{"framing"}, map[string]any{
					"type":              "sync",
					"sync":              "qlU=",
					"length_offset":     2,
					"length_size":       "two",
					"byte_order":        "big_endian",
					"length_adjustment": 0,
					"checksum": map[string]any{
						"algorithm":  "crc32",
						"byte_order": "little_endian",
					},
					"max_length": 512,
				})),
			)
			Expect(next.Framing).To(Equal(&icd.Framing{Variant: icd.SyncFraming{
				Sync:         []byte{0xAA, 0x55},
				LengthOffset: 2,
				LengthSize:   icd.LengthSizeTwo,
				ByteOrder:    icd.ByteOrderBigEndian,
				Checksum: &icd.Checksum{
					Algorithm: icd.ChecksumAlgorithmCrc32,
					ByteOrder: icd.ByteOrderLittleEndian,
				},
				MaxLength: 512,
			}}))
		})

		It("Should set one value of the framing", func() {
			framed := base
			framed.Framing = fixed(8)
			Expect(icd.Reduce(framed, icd.NewSetAction(icd.SetPayload{
				Path:  []string{"framing", "length"},
				Value: 64,
				Guards: []patch.Guard{
					{Path: []string{"framing", "type"}, Value: "fixed"},
				},
			}))).To(HaveField("Framing", Equal(fixed(64))))
		})

		It("Should remove the framing", func() {
			framed := base
			framed.Framing = fixed(8)
			Expect(icd.Reduce(
				framed,
				icd.NewRemoveAction(icd.RemovePayload{Key: "framing"}),
			)).To(HaveField("Framing", BeNil()))
		})

		It("Should remove the framing on a set that holds no value", func() {
			framed := base
			framed.Framing = fixed(8)
			Expect(icd.Reduce(framed, set([]string{"framing"}, nil))).
				To(HaveField("Framing", BeNil()))
		})
	})
})
