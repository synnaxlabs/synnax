// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	"encoding/json/v2"
	"io/fs"
	"os"
	"strings"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	v0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v1"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// v0Fixtures holds the valid Check fixtures as v0 wrote them.
var v0Fixtures = os.DirFS("testdata/v0")

func v0FixtureEntries() []TableEntry {
	GinkgoHelper()
	paths := MustSucceed(fs.Glob(v0Fixtures, "*.json"))
	return lo.Map(paths, func(path string, _ int) TableEntry {
		return Entry(strings.TrimSuffix(path, ".json"), path)
	})
}

func v0Binary(kind payload.Kind) v0.BinaryField {
	return v0.BinaryField{
		Key:             uuid.New(),
		Name:            "speed",
		Scale:           1,
		StartBit:        4,
		BitLength:       16,
		ByteOrder:       v0.ByteOrderBigEndian,
		Kind:            kind,
		Multiplexor:     new(uuid.New()),
		MultiplexValues: []int32{2},
	}
}

func v1BinaryBase(old v0.BinaryField) v1.BinaryBaseField {
	return v1.BinaryBaseField{
		BaseField:       old.BaseField,
		StartBit:        old.StartBit,
		BitLength:       old.BitLength,
		ByteOrder:       old.ByteOrder,
		Multiplexor:     old.Multiplexor,
		MultiplexValues: old.MultiplexValues,
	}
}

func binaryMessage(fields ...v0.BinaryField) v0.Message {
	return v0.Message{
		Key:  uuid.New(),
		Name: "status",
		Payload: v0.Payload{Variant: v0.BinaryPayload{
			Length: new(uint16(4)),
			Fields: fields,
		}},
	}
}

var jsonBase = v0.JSONBaseField{
	Key:     uuid.New(),
	Name:    "value",
	Scale:   1,
	Pointer: "/value",
}

var _ = Describe("Migration", func() {
	It("Should lift each stored ICD", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v0.Key, v0.ICD]{DB: db}))
		field := v0Binary(payload.KindUnsigned)
		old := v0.ICD{
			Key:      uuid.New(),
			Name:     "engine",
			Messages: []v0.Message{binaryMessage(field)},
		}
		Expect(gorp.NewCreate[v0.Key, v0.ICD]().Entry(&old).Exec(ctx, db)).
			To(Succeed())
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v1.Key, v1.ICD]{
			DB: db, Migrations: []gorp.Migration{v1.Migration},
		}))
		var got v1.ICD
		Expect(gorp.NewRetrieve[v1.Key, v1.ICD]().
			Where(gorp.MatchKeys[v1.Key, v1.ICD](old.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got).To(Equal(v1.ICD{
			Key:  old.Key,
			Name: "engine",
			Messages: []v1.Message{{
				Key:  old.Messages[0].Key,
				Name: "status",
				Payload: v1.Payload{Variant: v1.BinaryPayload{
					Length: new(uint16(4)),
					Fields: []v1.BinaryField{{Variant: v1.UnsignedBinaryField{
						BinaryBaseField: v1BinaryBase(field),
					}}},
				}},
			}},
		}))
	})
})

var _ = Describe("MigrateICD", func() {
	It("Should keep the key, the name, and each message", func(ctx SpecContext) {
		old := v0.ICD{
			Key:      uuid.New(),
			Name:     "engine",
			Messages: []v0.Message{{Key: uuid.New(), Name: "status"}},
		}
		Expect(MustSucceed(v1.MigrateICD(ctx, old))).To(Equal(v1.ICD{
			Key:      old.Key,
			Name:     "engine",
			Messages: []v1.Message{{Key: old.Messages[0].Key, Name: "status"}},
		}))
	})

	DescribeTable(
		"Should lift a valid v0 fixture to an ICD that passes Check",
		func(ctx SpecContext, path string) {
			var f struct {
				ICD v0.ICD `json:"icd"`
			}
			Expect(json.Unmarshal(MustSucceed(fs.ReadFile(v0Fixtures, path)), &f)).
				To(Succeed())
			Expect(icd.Check(MustSucceed(v1.MigrateICD(ctx, f.ICD)))).To(BeEmpty())
		},
		v0FixtureEntries(),
	)
})

var _ = Describe("MigrateMessage", func() {
	It("Should keep each part and give no cycle time", func(ctx SpecContext) {
		old := v0.Message{
			Key:  uuid.New(),
			Name: "status",
			Payload: v0.Payload{Variant: v0.TextPayload{
				Delimiter: ",",
				Fields:    []v0.TextField{},
			}},
		}
		Expect(MustSucceed(v1.MigrateMessage(ctx, old))).To(Equal(v1.Message{
			Key:  old.Key,
			Name: "status",
			Payload: v1.Payload{Variant: v1.TextPayload{
				Delimiter: ",",
				Fields:    []v1.TextField{},
			}},
		}))
	})

	DescribeTable(
		"Should lift a binary field to the variant of its kind, with no default",
		func(ctx SpecContext, kind payload.Kind, want v1.BinaryField) {
			field := v0Binary(kind)
			want.SetBase(v1BinaryBase(field))
			got := MustSucceed(v1.MigrateMessage(ctx, binaryMessage(field)))
			Expect(got.Payload).To(Equal(v1.Payload{Variant: v1.BinaryPayload{
				Length: new(uint16(4)),
				Fields: []v1.BinaryField{want},
			}}))
		},
		Entry("float", payload.KindFloat, v1.BinaryField{
			Variant: v1.FloatBinaryField{},
		}),
		Entry("integer", payload.KindInteger, v1.BinaryField{
			Variant: v1.IntegerBinaryField{},
		}),
		Entry("unsigned", payload.KindUnsigned, v1.BinaryField{
			Variant: v1.UnsignedBinaryField{},
		}),
	)

	It("Should fail on a binary field that is not a number", func(ctx SpecContext) {
		old := binaryMessage(v0Binary(payload.KindString))
		Expect(v1.MigrateMessage(ctx, old)).Error().To(MatchError(
			`binary field "speed" has kind "string", not float, integer, or unsigned`,
		))
	})
})

var _ = Describe("MigrateTextField", func() {
	It("Should keep the base of a text field, with no default", func(ctx SpecContext) {
		base := v0.BaseField{Key: uuid.New(), Name: "temp", Scale: 2, Units: "C"}
		old := v0.TextField{Variant: v0.TaggedTextField{BaseField: base, Tag: "T="}}
		Expect(MustSucceed(v1.MigrateTextField(ctx, old))).To(Equal(v1.TextField{
			Variant: v1.TaggedTextField{
				TextBaseField: v1.TextBaseField{BaseField: base},
				Tag:           "T=",
			},
		}))
	})
})

var _ = Describe("MigrateJSONField", func() {
	It("Should make a timestamp default a constant default", func(ctx SpecContext) {
		old := v0.JSONField{Variant: v0.TimestampJSONField{
			JSONBaseField: jsonBase,
			TimeFormat:    payload.TimeFormatUnixMs,
			Default:       new(telem.TimeStamp(5)),
		}}
		Expect(MustSucceed(v1.MigrateJSONField(ctx, old))).To(Equal(v1.JSONField{
			Variant: v1.TimestampJSONField{
				JSONBaseField: jsonBase,
				TimeFormat:    payload.TimeFormatUnixMs,
				Default: &v1.TimestampDefault{
					Variant: v1.ConstantTimestampDefault{Value: 5},
				},
			},
		}))
	})

	It("Should give a timestamp field with no default none", func(ctx SpecContext) {
		old := v0.JSONField{Variant: v0.TimestampJSONField{
			JSONBaseField: jsonBase,
			TimeFormat:    payload.TimeFormatUnixMs,
		}}
		Expect(MustSucceed(v1.MigrateJSONField(ctx, old))).To(Equal(v1.JSONField{
			Variant: v1.TimestampJSONField{
				JSONBaseField: jsonBase,
				TimeFormat:    payload.TimeFormatUnixMs,
			},
		}))
	})

	DescribeTable(
		"Should give a field of a new default none",
		func(ctx SpecContext, old v0.JSONField, want v1.JSONField) {
			Expect(MustSucceed(v1.MigrateJSONField(ctx, old))).To(Equal(want))
		},
		Entry("json",
			v0.JSONField{Variant: v0.JSONJSONField{JSONBaseField: jsonBase}},
			v1.JSONField{Variant: v1.JSONJSONField{JSONBaseField: jsonBase}},
		),
		Entry("uuid",
			v0.JSONField{Variant: v0.UuidJSONField{JSONBaseField: jsonBase}},
			v1.JSONField{Variant: v1.UuidJSONField{JSONBaseField: jsonBase}},
		),
		Entry("bytes",
			v0.JSONField{Variant: v0.BytesJSONField{JSONBaseField: jsonBase}},
			v1.JSONField{Variant: v1.BytesJSONField{JSONBaseField: jsonBase}},
		),
	)
})
