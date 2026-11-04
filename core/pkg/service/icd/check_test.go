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
	"math"
	"os"
	"path/filepath"
	"strings"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
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

func checkFixtureEntries(kind string) []TableEntry {
	paths := MustSucceed(
		filepath.Glob(filepath.Join("testdata", "fixtures", kind, "*.json")),
	)
	return lo.Map(paths, func(path string, _ int) TableEntry {
		return Entry(strings.TrimSuffix(filepath.Base(path), ".json"), path)
	})
}

func loadCheckFixture(path string) checkFixture {
	GinkgoHelper()
	var f checkFixture
	Expect(json.Unmarshal(MustSucceed(os.ReadFile(path)), &f)).To(Succeed())
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
		f := loadCheckFixture(filepath.Join(
			"testdata", "fixtures", "valid", "can_multiplexed.json",
		))
		Expect(svc.NewWriter(tx).Create(ctx, &f.ICD)).To(Succeed())
	})

	It("Should return an empty list, not nil, for a valid ICD", func() {
		problems := icd.Check(icd.ICD{Name: "Empty"})
		Expect(problems).ToNot(BeNil())
		Expect(problems).To(BeEmpty())
	})

	It("Should find every problem in an ICD, not only the first", func() {
		mux := binaryField("mux", 0, 4)
		mux.Key = uuid.New()
		sig := binaryField("sig", 8, 8)
		sig.Scale = 0
		sig.Multiplexor = new(mux.Key)
		sig.MultiplexValues = []int32{16}
		ic := icd.ICD{Messages: []icd.Message{
			canMessage("M", 0x800, mux, sig),
			binaryMessage("M"),
		}}
		Expect(icd.Check(ic)).To(ConsistOf(
			validate.Problem{Path: "name", Message: "required"},
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

	It("Should join every problem into the error a save returns", func() {
		problems := icd.Check(icd.ICD{Messages: []icd.Message{binaryMessage("")}})
		Expect(problems.Err()).To(MatchError(
			"name: required: validation error\n" +
				"messages.0.name: required: validation error",
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
			"messages.0.payload", "format is required",
		),
		Entry("text field without an encoding",
			icd.ICD{Name: "L", Messages: []icd.Message{
				textMessage("M", icd.TextField{}),
			}},
			"messages.0.payload.fields.0", "encoding is required",
		),
		Entry("JSON field without a type",
			icd.ICD{Name: "L", Messages: []icd.Message{{
				Name: "M",
				Payload: icd.Payload{Variant: icd.JSONPayload{
					Fields: []icd.JSONField{{}},
				}},
			}}},
			"messages.0.payload.fields.0", "type is required",
		),
		Entry("identifier without a type",
			icd.ICD{Name: "L", Messages: []icd.Message{updateBinary(
				binaryMessage("M", binaryField("a", 0, 8)),
				func(p *icd.BinaryPayload) { p.Identifier = &icd.Identifier{} },
			)}},
			"messages.0.payload.identifier", "type is required",
		),
	)

	DescribeTable("Should require finite numbers, which JSON cannot carry",
		func(update func(*icd.BinaryField), path, message string) {
			f := binaryField("a", 0, 8)
			update(&f)
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
			func(f *icd.BinaryField) { f.Scale = math.NaN() },
			"scale", "scale must be finite and non-zero",
		),
		Entry("infinite scale",
			func(f *icd.BinaryField) { f.Scale = math.Inf(1) },
			"scale", "scale must be finite and non-zero",
		),
		Entry("infinite offset",
			func(f *icd.BinaryField) { f.Offset = math.Inf(-1) },
			"offset", "offset must be finite",
		),
		Entry("NaN minimum",
			func(f *icd.BinaryField) { f.Min = new(math.NaN()) },
			"min", "min must be finite",
		),
	)

	It("Should not count messages and fields without keys as duplicates", func() {
		ic := icd.ICD{Name: "L", Messages: []icd.Message{
			canMessage("A", 1, binaryField("a", 0, 8), binaryField("b", 8, 8)),
			canMessage("B", 2, binaryField("a", 0, 8)),
		}}
		Expect(icd.Check(ic)).To(BeEmpty())
	})
})
