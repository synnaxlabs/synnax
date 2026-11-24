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

// problem is a problem that Check finds, with its path dotted.
type problem struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// problemsOf returns the problems that icd.Check finds in ic. Each must wrap
// validate.ErrValidation.
func problemsOf(ic icd.ICD) []problem {
	GinkgoHelper()
	return lo.Map(icd.Check(ic), func(p validate.PathError, _ int) problem {
		Expect(p).To(MatchError(validate.ErrValidation))
		return problem{Path: strings.Join(p.Path, "."), Message: p.Err.Error()}
	})
}

// checkFixture is one ICD from testdata/fixtures. An invalid fixture lists the exact
// problems Check must find, and says whether the Driver's codec also rejects it.
type checkFixture struct {
	Description  string    `json:"description"`
	CodecRejects bool      `json:"codec_rejects"`
	Problems     []problem `json:"problems"`
	ICD          icd.ICD   `json:"icd"`
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
		Expect(problemsOf(f.ICD)).To(BeEmpty())
	}, checkFixtureEntries("valid"))

	DescribeTable(
		"Should find exactly the problems an invalid fixture names",
		func(path string) {
			f := loadCheckFixture(path)
			Expect(f.Problems).ToNot(BeEmpty())
			Expect(problemsOf(f.ICD)).To(ConsistOf(f.Problems))
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
		Expect(problemsOf(ic)).To(ConsistOf(
			problem{Path: "name", Message: "name is required"},
			problem{
				Path:    "messages.0.payload.fields.1.scale",
				Message: "scale must be finite and non-zero",
			},
			problem{
				Path:    "messages.0.payload.fields.1.multiplex_values.0",
				Message: `value 16 does not fit the multiplexor "mux"`,
			},
			problem{
				Path:    "messages.0.payload.identifier.id",
				Message: "id 0x800 exceeds the maximum 0x7FF",
			},
			problem{
				Path:    "messages.1.name",
				Message: `duplicate message name "M"`,
			},
		))
	})

	DescribeTable(
		"Should refuse a tag that the Driver cannot match",
		func(tag, message string) {
			ic := icd.ICD{Name: "L", Messages: []icd.Message{
				textMessage("M", icd.TextField{Variant: icd.TaggedTextField{
					BaseField: icd.BaseField{Name: "temp", Scale: 1},
					Tag:       tag,
				}}),
			}}
			Expect(problemsOf(ic)).To(ConsistOf(problem{
				Path:    "messages.0.payload.fields.0.tag",
				Message: message,
			}))
		},
		Entry(
			"a tag that holds the delimiter",
			"T,1",
			`the tag contains the delimiter ","`,
		),
		Entry("a tag that starts with a space", " T", "the tag starts with whitespace"),
		Entry("a tag that starts with a tab", "\tT", "the tag starts with whitespace"),
		Entry(
			"a tag that starts with a carriage return",
			"\rT",
			"the tag starts with whitespace",
		),
	)

	DescribeTable(
		"Should refuse a tag that the Driver cannot tell from another tag",
		func(first, second string, field int, message string) {
			ic := icd.ICD{Name: "L", Messages: []icd.Message{textMessage(
				"M",
				icd.TextField{Variant: icd.TaggedTextField{
					BaseField: icd.BaseField{Name: "temp", Scale: 1},
					Tag:       first,
				}},
				icd.TextField{Variant: icd.TaggedTextField{
					BaseField: icd.BaseField{Name: "hum", Scale: 1},
					Tag:       second,
				}},
			)}}
			Expect(problemsOf(ic)).To(ConsistOf(problem{
				Path:    fmt.Sprintf("messages.0.payload.fields.%d.tag", field),
				Message: message,
			}))
		},
		Entry("a tag and a digit", "T", "T1", 1,
			`a value of field temp after its tag "T" reads as this tag`),
		Entry("a tag that ends in a digit and another digit", "CH1", "CH10", 1,
			`a value of field temp after its tag "CH1" reads as this tag`),
		Entry("a tag and a minus sign", "T", "T-", 1,
			`a value of field temp after its tag "T" reads as this tag`),
		Entry("a tag and a plus sign", "T", "T+", 1,
			`a value of field temp after its tag "T" reads as this tag`),
		Entry("a tag and a point", "T", "T.", 1,
			`a value of field temp after its tag "T" reads as this tag`),
		Entry("a tag, a space, and a digit", "T", "T 5", 1,
			`a value of field temp after its tag "T" reads as this tag`),
		Entry("a tag and a space", "T", "T ", 1,
			`a value of field temp after its tag "T" reads as this tag`),
		Entry("the longer tag first", "T=1", "T=", 0,
			`a value of field hum after its tag "T=" reads as this tag`),
		Entry(
			"two equal tags",
			"T",
			"T",
			1,
			`the tag "T" is also the tag of field temp`,
		),
	)

	DescribeTable(
		"Should accept a tag that starts another tag with text no value starts with",
		func(first, second string) {
			ic := icd.ICD{Name: "L", Messages: []icd.Message{textMessage(
				"M",
				icd.TextField{Variant: icd.TaggedTextField{
					BaseField: icd.BaseField{Name: "temp", Scale: 1},
					Tag:       first,
				}},
				icd.TextField{Variant: icd.TaggedTextField{
					BaseField: icd.BaseField{Name: "hum", Scale: 1},
					Tag:       second,
				}},
			)}}
			Expect(problemsOf(ic)).To(BeEmpty())
		},
		Entry("a tag and a letter", "T", "TE"),
		Entry("a tag and an equals sign", "T", "T="),
		Entry("the longer tag first", "TE", "T"),
	)

	It("Should refuse an empty delimiter but not each tag in its message", func() {
		ic := icd.ICD{Name: "L", Messages: []icd.Message{{
			Name: "M",
			Payload: icd.Payload{Variant: icd.TextPayload{
				Fields: []icd.TextField{{Variant: icd.TaggedTextField{
					BaseField: icd.BaseField{Name: "temp", Scale: 1},
					Tag:       "T",
				}}},
			}},
		}}}
		Expect(problemsOf(ic)).To(ConsistOf(problem{
			Path:    "messages.0.payload.delimiter",
			Message: "delimiter is required",
		}))
	})

	DescribeTable("Should require a format, encoding, or type",
		func(ic icd.ICD, path, message string) {
			Expect(problemsOf(ic)).To(ConsistOf(
				problem{Path: path, Message: message},
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
			Expect(problemsOf(ic)).To(ConsistOf(
				problem{
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

	DescribeTable("Should refuse a value outside the min and max of its field",
		func(m icd.Message, path, message string) {
			ic := icd.ICD{Name: "L", Messages: []icd.Message{m}}
			Expect(problemsOf(ic)).To(ConsistOf(problem{
				Path:    "messages.0.payload.fields.0." + path,
				Message: message,
			}))
		},
		Entry("a JSON default above the max",
			jsonMessage(icd.JSONField{Variant: icd.FloatJSONField{
				Name: "v", Scale: 1, Pointer: "/v",
				Max: new(100.0), Default: new(payload.Float(120)),
			}}),
			"default", "default 120 is above the max 100",
		),
		Entry("a scaled JSON default that decodes above the max",
			jsonMessage(icd.JSONField{Variant: icd.IntegerJSONField{
				Name: "v", Scale: 10, Pointer: "/v",
				Max: new(100.0), Default: new(payload.Integer(12)),
			}}),
			"default", "default 12 decodes to 120, which is above the max 100",
		),
		Entry("a JSON enum value above the max",
			jsonMessage(icd.JSONField{Variant: icd.UnsignedJSONField{
				Name: "v", Scale: 1, Pointer: "/v", Max: new(10.0),
				EnumValues: []payload.UnsignedEntry{{Label: "BOOST", Value: 20}},
			}}),
			"enum_values.0.value", "enum value 20 is above the max 10",
		),
		Entry("a JSON enum value below the min, which takes no scale",
			jsonMessage(icd.JSONField{Variant: icd.FloatJSONField{
				Name: "v", Scale: 10, Pointer: "/v", Min: new(0.0),
				EnumValues: []payload.FloatEntry{{Label: "OFF", Value: -1}},
			}}),
			"enum_values.0.value", "enum value -1 is below the min 0",
		),
		Entry("a binary default above the max",
			canMessage("M", 1, icd.BinaryField{Variant: icd.UnsignedBinaryField{
				Name: "a", Scale: 1, BitLength: 8, ByteOrder: icd.ByteOrderLittleEndian,
				Max: new(100.0), Default: new(payload.Unsigned(200)),
			}}),
			"default", "default 200 is above the max 100",
		),
		Entry("a text default below the min",
			textMessage("M", icd.TextField{Variant: icd.TaggedTextField{
				BaseField: icd.BaseField{Name: "temp", Scale: 1, Min: new(0.0)},
				Tag:       "T",
				Default:   new(payload.Float(-1)),
			}}),
			"default", "default -1 is below the min 0",
		),
		Entry("a default that is not finite, which reports only that",
			jsonMessage(icd.JSONField{Variant: icd.FloatJSONField{
				Name: "v", Scale: 1, Pointer: "/v",
				Max: new(100.0), Default: new(payload.Float(math.Inf(1))),
			}}),
			"default", "default must be finite",
		),
	)

	DescribeTable("Should accept a value at a limit of its field",
		func(m icd.Message) {
			ic := icd.ICD{Name: "L", Messages: []icd.Message{m}}
			Expect(problemsOf(ic)).To(BeEmpty())
		},
		Entry("a scaled JSON default that float error puts above the max",
			jsonMessage(icd.JSONField{Variant: icd.FloatJSONField{
				Name: "v", Scale: 0.1, Pointer: "/v",
				Max: new(0.3), Default: new(payload.Float(3)),
			}}),
		),
		Entry("a JSON enum value at the max",
			jsonMessage(icd.JSONField{Variant: icd.IntegerJSONField{
				Name: "v", Scale: 1, Pointer: "/v", Max: new(10.0),
				EnumValues: []payload.IntegerEntry{{Label: "HIGH", Value: 10}},
			}}),
		),
		Entry("a binary default at the min",
			canMessage("M", 1, icd.BinaryField{Variant: icd.IntegerBinaryField{
				Name: "a", Scale: 1, BitLength: 8, ByteOrder: icd.ByteOrderLittleEndian,
				Min: new(-5.0), Default: new(payload.Integer(-5)),
			}}),
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
		Expect(problemsOf(ic)).To(BeEmpty())
	})

	Describe("Overlapping fields", func() {
		check := func(fields ...icd.BinaryField) []problem {
			return problemsOf(icd.ICD{Name: "L", Messages: []icd.Message{
				canMessage("M", 1, fields...),
			}})
		}
		overlapping := func(i int) problem {
			return problem{
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
