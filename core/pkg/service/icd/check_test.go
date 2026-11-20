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
	"encoding/json/v2"
	"fmt"
	"io/fs"
	"math"
	"os"
	"strings"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/x/payload"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// checkFixture is one ICD from testdata/fixtures. An invalid fixture lists the exact
// problems Check must find, and says whether the Driver's codec also rejects it.
type checkFixture struct {
	Description  string            `json:"description"`
	CodecRejects bool              `json:"codec_rejects"`
	Problems     validate.Problems `json:"problems"`
	ICD          icd.ICD           `json:"icd"`
}

var checkFixtures = os.DirFS("testdata/fixtures")

func checkFixtureEntries(kind string) []TableEntry {
	GinkgoHelper()
	paths := MustSucceed(fs.Glob(checkFixtures, kind+"/*.json"))
	return lo.Map(paths, func(path string, _ int) TableEntry {
		name := strings.TrimSuffix(strings.TrimPrefix(path, kind+"/"), ".json")
		return Entry(name, path)
	})
}

func loadCheckFixture(path string) checkFixture {
	GinkgoHelper()
	var f checkFixture
	Expect(json.Unmarshal(MustSucceed(fs.ReadFile(checkFixtures, path)), &f)).
		To(Succeed())
	Expect(f.Description).ToNot(BeEmpty())
	return f
}

var _ = Describe("Check", func() {
	DescribeTable("Should find no problems in a valid fixture", func(path string) {
		f := loadCheckFixture(path)
		Expect(f.Problems).To(BeEmpty())
		Expect(icd.Check(f.ICD)).To(BeEmpty())
	}, checkFixtureEntries("valid"))

	DescribeTable(
		"Should find exactly the problems an invalid fixture names",
		func(path string) {
			f := loadCheckFixture(path)
			Expect(f.Problems).ToNot(BeEmpty())
			Expect(icd.Check(f.ICD)).To(ConsistOf(f.Problems))
		},
		checkFixtureEntries("invalid"),
	)

	It("Should store a valid fixture", func(ctx SpecContext) {
		f := loadCheckFixture("valid/can_multiplexed.json")
		Expect(svc.NewWriter(tx).Create(ctx, &f.ICD)).To(Succeed())
	})

	It("Should return an empty list, not nil, for a valid ICD", func() {
		problems := icd.Check(icd.ICD{Name: "Empty"})
		Expect(problems).ToNot(BeNil())
		Expect(problems).To(BeEmpty())
	})

	It("Should find every problem in an ICD, not only the first", func() {
		muxKey := uuid.New()
		mux := updateBase(binaryField("mux", 0, 4), func(b *icd.BinaryBaseField) {
			b.Key = muxKey
		})
		sig := updateBase(binaryField("sig", 8, 8), func(b *icd.BinaryBaseField) {
			b.Scale = 0
			b.Multiplexor = new(muxKey)
			b.MultiplexValues = []int32{16}
		})
		ic := icd.ICD{Messages: []icd.Message{
			canMessage("M", 0x800, mux, sig),
			binaryMessage("M"),
		}}
		Expect(icd.Check(ic)).To(ConsistOf(
			validate.Problem{Path: "name", Message: "name is required"},
			validate.Problem{
				Path:    "messages.0.payload.fields.1.scale",
				Message: "scale must be finite and non-zero",
			},
			validate.Problem{
				Path:    "messages.0.payload.fields.1.multiplex_values.0",
				Message: `value 16 does not fit the multiplexor "mux"`,
			},
			validate.Problem{
				Path:    "messages.0.payload.identifier.id",
				Message: "id 0x800 exceeds the maximum 0x7FF",
			},
			validate.Problem{
				Path:    "messages.1.name",
				Message: `duplicate message name "M"`,
			},
		))
	})

	DescribeTable("Should require a format, encoding, or type",
		func(ic icd.ICD, path, message string) {
			Expect(icd.Check(ic)).To(ConsistOf(
				validate.Problem{Path: path, Message: message},
			))
		},
		Entry("message without a format",
			icd.ICD{Name: "L", Messages: []icd.Message{{Name: "M"}}},
			"messages.0.payload.format", "format is required",
		),
		Entry("text field without an encoding",
			icd.ICD{Name: "L", Messages: []icd.Message{
				textMessage("M", icd.TextField{}),
			}},
			"messages.0.payload.fields.0.encoding", "encoding is required",
		),
		Entry("tagged field without a tag",
			icd.ICD{Name: "L", Messages: []icd.Message{
				textMessage("M", icd.TextField{Variant: icd.TaggedTextField{
					BaseField: icd.BaseField{Name: "temp", Scale: 1},
				}}),
			}},
			"messages.0.payload.fields.0.tag", "tag is required",
		),
		Entry("JSON field without a type",
			icd.ICD{Name: "L", Messages: []icd.Message{{
				Name: "M",
				Payload: icd.Payload{Variant: icd.JSONPayload{
					Fields: []icd.JSONField{{}},
				}},
			}}},
			"messages.0.payload.fields.0.type", "type is required",
		),
		Entry("JSON match without a type",
			icd.ICD{Name: "L", Messages: []icd.Message{{
				Name: "M",
				Payload: icd.Payload{Variant: icd.JSONPayload{
					Match: &icd.JSONMatch{},
				}},
			}}},
			"messages.0.payload.match.type", "type is required",
		),
		Entry("binary field without a kind",
			icd.ICD{Name: "L", Messages: []icd.Message{
				binaryMessage("M", icd.BinaryField{}),
			}},
			"messages.0.payload.fields.0.kind", "kind is required",
		),
		Entry("identifier without a type",
			icd.ICD{Name: "L", Messages: []icd.Message{updateBinary(
				binaryMessage("M", binaryField("a", 0, 8)),
				func(p *icd.BinaryPayload) { p.Identifier = &icd.Identifier{} },
			)}},
			"messages.0.payload.identifier.type", "type is required",
		),
	)

	DescribeTable("Should require finite numbers, which JSON cannot carry",
		func(update func(*icd.BinaryBaseField), path, message string) {
			f := updateBase(binaryField("a", 0, 8), update)
			ic := icd.ICD{Name: "L", Messages: []icd.Message{
				canMessage("M", 1, f),
			}}
			Expect(icd.Check(ic)).To(ConsistOf(
				validate.Problem{
					Path:    "messages.0.payload.fields.0." + path,
					Message: message,
				},
			))
		},
		Entry("NaN scale",
			func(f *icd.BinaryBaseField) { f.Scale = math.NaN() },
			"scale", "scale must be finite and non-zero",
		),
		Entry("infinite scale",
			func(f *icd.BinaryBaseField) { f.Scale = math.Inf(1) },
			"scale", "scale must be finite and non-zero",
		),
		Entry("infinite offset",
			func(f *icd.BinaryBaseField) { f.Offset = math.Inf(-1) },
			"offset", "offset must be finite",
		),
		Entry("NaN minimum",
			func(f *icd.BinaryBaseField) { f.Min = new(math.NaN()) },
			"min", "min must be finite",
		),
	)

	DescribeTable("Should accept every digit of a 64-bit unsigned default",
		func(d payload.Unsigned) {
			f := icd.BinaryField{Variant: icd.UnsignedBinaryField{
				BinaryBaseField: binaryField("a", 0, 64).Base(),
				Default:         new(d),
			}}
			ic := icd.ICD{Name: "L", Messages: []icd.Message{canMessage("M", 1, f)}}
			Expect(icd.Check(ic)).To(BeEmpty())
		},
		Entry("the largest unsigned integer", payload.Unsigned(math.MaxUint64)),
		Entry("an integer past a float64", payload.Unsigned(9007199254740993)),
	)

	It("Should not count messages and fields without keys as duplicates", func() {
		ic := icd.ICD{Name: "L", Messages: []icd.Message{
			canMessage("A", 1, binaryField("a", 0, 8), binaryField("b", 8, 8)),
			canMessage("B", 2, binaryField("a", 0, 8)),
		}}
		Expect(icd.Check(ic)).To(BeEmpty())
	})

	Describe("Overlapping fields", func() {
		check := func(fields ...icd.BinaryField) validate.Problems {
			return icd.Check(icd.ICD{Name: "L", Messages: []icd.Message{
				canMessage("M", 1, fields...),
			}})
		}
		overlapping := func(i int) validate.Problem {
			return validate.Problem{
				Path:    fmt.Sprintf("messages.0.payload.fields.%d.start_bit", i),
				Message: `field overlaps the bits of field "a"`,
			}
		}
		bigEndian := func(f icd.BinaryField) icd.BinaryField {
			return updateBase(f, func(b *icd.BinaryBaseField) {
				b.ByteOrder = icd.ByteOrderBigEndian
			})
		}
		multiplexed := func(
			f icd.BinaryField,
			mux icd.BinaryField,
			values ...int32,
		) icd.BinaryField {
			return updateBase(f, func(b *icd.BinaryBaseField) {
				b.Multiplexor = new(mux.Base().Key)
				b.MultiplexValues = values
			})
		}
		withKey := func(f icd.BinaryField) icd.BinaryField {
			return updateBase(f, func(b *icd.BinaryBaseField) { b.Key = uuid.New() })
		}
		var mux icd.BinaryField
		BeforeEach(func() {
			mux = withKey(binaryField("mux", 0, 4))
		})

		It("Should find two fields that read the same bits", func() {
			Expect(check(binaryField("a", 0, 8), binaryField("b", 4, 8))).
				To(ConsistOf(overlapping(1)))
		})

		It("Should report a field against the first field it overlaps", func() {
			Expect(check(
				binaryField("a", 0, 8),
				binaryField("b", 8, 8),
				binaryField("c", 4, 8),
			)).To(ConsistOf(overlapping(2)))
		})

		It("Should not find fields that only touch", func() {
			Expect(check(binaryField("a", 0, 8), binaryField("b", 8, 8))).To(BeEmpty())
		})

		It("Should compare fields with different byte orders", func() {
			Expect(check(binaryField("a", 0, 8), bigEndian(binaryField("b", 7, 8)))).
				To(ConsistOf(overlapping(1)))
			Expect(check(bigEndian(binaryField("a", 7, 16)), binaryField("b", 16, 8))).
				To(BeEmpty())
		})

		It(
			"Should find a multiplexed field over a field that is always present",
			func() {
				Expect(check(
					mux,
					binaryField("a", 8, 8),
					multiplexed(binaryField("b", 8, 8), mux, 0),
				)).To(ConsistOf(overlapping(2)))
			},
		)

		It(
			"Should find a field that is always present over a multiplexed field",
			func() {
				Expect(check(
					mux,
					multiplexed(binaryField("a", 8, 8), mux, 0),
					binaryField("b", 8, 8),
				)).To(ConsistOf(overlapping(2)))
			},
		)

		It("Should find multiplexed fields that share a multiplex value", func() {
			Expect(check(
				mux,
				multiplexed(binaryField("a", 8, 8), mux, 0, 1),
				multiplexed(binaryField("b", 8, 8), mux, 1, 2),
			)).To(ConsistOf(overlapping(2)))
		})

		It(
			"Should not find multiplexed fields that are never present together",
			func() {
				Expect(check(
					mux,
					multiplexed(binaryField("a", 8, 8), mux, 0),
					multiplexed(binaryField("b", 8, 8), mux, 1),
				)).To(BeEmpty())
			},
		)

		It("Should not compare fields under different multiplexors", func() {
			other := withKey(binaryField("other", 4, 4))
			Expect(check(
				mux,
				other,
				multiplexed(binaryField("a", 8, 8), mux, 0),
				multiplexed(binaryField("b", 8, 8), other, 0),
			)).To(BeEmpty())
		})
	})
})
