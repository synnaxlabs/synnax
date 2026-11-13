// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v5_test

import (
	"bytes"
	"context"
	"math"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	v3 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v4"
	v5 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v5"
	xkv "github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// lift migrates f through an endpoint that f does not index.
func lift(f v4.ReadField) v5.ReadField {
	GinkgoHelper()
	ep := MustSucceed(v5.MigrateReadEndpoint(
		context.Background(),
		v4.ReadEndpoint{Fields: []v4.ReadField{f}},
	))
	return ep.Fields[0]
}

func storedChannels(found map[channel.Key]v5.Channel) v5.Channels {
	return func(
		context.Context,
		gorp.Snapshot,
		[]channel.Key,
	) (map[channel.Key]v5.Channel, error) {
		return found, nil
	}
}

func untypedField(dt telem.DataType, entries ...payload.EnumEntry) v4.ReadField {
	return v4.ReadField{
		Key:        "f",
		Channel:    7,
		Pointer:    "/v",
		DataType:   dt,
		EnumValues: entries,
	}
}

func optionalField(
	dt telem.DataType,
	fill any,
	entries ...payload.EnumEntry,
) v4.ReadField {
	f := untypedField(dt, entries...)
	f.Optional = true
	if fill != nil {
		f.FillValue = &fill
	}
	return f
}

// fillOf returns the fill of f, or nil when f has none.
func fillOf(f v5.ReadField) any {
	switch v := f.Variant.(type) {
	case v5.FloatReadField:
		return deref(v.Fill)
	case v5.IntegerReadField:
		return deref(v.Fill)
	case v5.UnsignedReadField:
		return deref(v.Fill)
	case v5.StringReadField:
		return deref(v.Fill)
	case v5.BooleanReadField:
		return deref(v.Fill)
	case v5.TimestampReadField:
		return deref(v.Fill)
	}
	return nil
}

func deref[T any](v *T) any {
	if v == nil {
		return nil
	}
	return *v
}

var _ = Describe("Read field migration", func() {
	Describe("MigrateReadConfig", func() {
		It("Should keep the config and the shared fields of each field", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			cfg := MustSucceed(v5.MigrateReadConfig(ctx, v4.ReadConfig{
				Key:            key,
				Device:         "dev",
				Rate:           5,
				NanFillEnabled: true,
				Endpoints: []v4.ReadEndpoint{{
					Key:    "ep",
					Method: v4.MethodPost,
					Path:   "/values",
					Body:   "{}",
					Index:  "time",
					Fields: []v4.ReadField{{
						Key:      "f",
						Name:     "pressure",
						Disabled: true,
						Channel:  7,
						Pointer:  "/p",
						DataType: telem.Float32T,
					}},
				}},
			}))
			Expect(cfg.Key).To(Equal(key))
			Expect(cfg.Device).To(Equal("dev"))
			Expect(cfg.Rate).To(Equal(telem.Rate(5)))
			Expect(cfg.NanFillEnabled).To(BeTrue())
			ep := cfg.Endpoints[0]
			Expect(ep.Method).To(Equal(v4.MethodPost))
			Expect(ep.Path).To(Equal("/values"))
			Expect(ep.Body).To(Equal("{}"))
			Expect(ep.Index).To(Equal("time"))
			f := ep.Fields[0]
			Expect(f.Base()).To(Equal(v5.BaseReadField{
				Key:      "f",
				Name:     "pressure",
				Disabled: true,
				Channel:  7,
				Pointer:  "/p",
			}))
			float := f.Variant.(v5.FloatReadField)
			Expect(*float.DataType).To(Equal(telem.Float32T))
			Expect(float.Fill).To(BeNil())
		})
	})

	Describe("MigrateReadEndpoint", func() {
		It("Should make an optional index field with a time format a timestamp field",
			func(ctx SpecContext) {
				timing := optionalField(telem.Float64T, nil)
				timing.TimeFormat = new(payload.TimeFormatUnixSec)
				ep := MustSucceed(v5.MigrateReadEndpoint(ctx, v4.ReadEndpoint{
					Index:  "f",
					Fields: []v4.ReadField{timing},
				}))
				timestamp := ep.Fields[0].Variant.(v5.TimestampReadField)
				Expect(timestamp.TimeFormat).To(Equal(payload.TimeFormatUnixSec))
			},
		)

		It("Should make the index field a timestamp field in ISO 8601", func(
			ctx SpecContext,
		) {
			ep := MustSucceed(v5.MigrateReadEndpoint(ctx, v4.ReadEndpoint{
				Index:  "f",
				Fields: []v4.ReadField{untypedField(telem.Float64T)},
			}))
			timestamp := ep.Fields[0].Variant.(v5.TimestampReadField)
			Expect(timestamp.TimeFormat).To(Equal(payload.TimeFormatISO8601))
		})

		It(
			"Should not index a field with an empty key on an endpoint with no index",
			func(ctx SpecContext) {
				field := optionalField(telem.Float64T, nil)
				field.Key = ""
				ep := MustSucceed(v5.MigrateReadEndpoint(ctx, v4.ReadEndpoint{
					Fields: []v4.ReadField{field},
				}))
				float := ep.Fields[0].Variant.(v5.FloatReadField)
				Expect(math.IsNaN(float64(*float.Fill))).To(BeTrue())
			},
		)
	})

	Describe("Field lift", func() {
		DescribeTable("Should pick the kind of the stored data type",
			func(ctx SpecContext, dt telem.DataType, kind payload.Kind) {
				f := lift(untypedField(dt))
				Expect(f.Type()).To(Equal(kind))
			},
			Entry("float64", telem.Float64T, payload.KindFloat),
			Entry("int16", telem.Int16T, payload.KindInteger),
			Entry("uint8", telem.Uint8T, payload.KindUnsigned),
			Entry("boolean", telem.BooleanT, payload.KindBoolean),
			Entry("string", telem.StringT, payload.KindString),
			Entry("timestamp", telem.TimestampT, payload.KindTimestamp),
			Entry("json", telem.JSONT, payload.KindJSON),
			Entry("uuid", telem.UUIDT, payload.KindUUID),
			Entry("bytes", telem.BytesT, payload.KindBytes),
		)

		It("Should read an unknown data type as float64", func() {
			f := lift(untypedField("complex128"))
			Expect(*f.Variant.(v5.FloatReadField).DataType).To(Equal(telem.Float64T))
		})

		It("Should keep the data type of a numeric field", func(ctx SpecContext) {
			f := lift(untypedField(telem.Int16T))
			Expect(*f.Variant.(v5.IntegerReadField).DataType).To(Equal(telem.Int16T))
		})

		It("Should keep the time format of a timestamp field", func() {
			old := untypedField(telem.TimestampT)
			old.TimeFormat = new(payload.TimeFormatUnixMs)
			f := lift(old)
			Expect(f.Variant.(v5.TimestampReadField).TimeFormat).
				To(Equal(payload.TimeFormatUnixMs))
		})

		It("Should read a field with a time format as a timestamp field", func() {
			old := untypedField(telem.Float64T)
			old.TimeFormat = new(payload.TimeFormatUnixSec)
			f := lift(old)
			Expect(f.Variant.(v5.TimestampReadField).TimeFormat).
				To(Equal(payload.TimeFormatUnixSec))
		})

		It("Should read an optional field with a time format as a timestamp field",
			func() {
				old := optionalField(telem.Float64T, nil)
				old.TimeFormat = new(payload.TimeFormatUnixSec)
				f := lift(old)
				Expect(f.Variant.(v5.TimestampReadField).TimeFormat).
					To(Equal(payload.TimeFormatUnixSec))
			},
		)

		It("Should keep the kind of a field with a left-over time format", func() {
			old := untypedField(
				telem.Int16T, payload.EnumEntry{Label: "ON", Value: 1},
			)
			old.TimeFormat = new(payload.TimeFormatUnixSec)
			integer := lift(old).Variant.(v5.IntegerReadField)
			Expect(integer.EnumValues).To(Equal(
				[]payload.IntegerEntry{{Label: "ON", Value: 1}},
			))
		})

		It("Should keep the kind of a float64 field with a time format and entries",
			func() {
				old := untypedField(
					telem.Float64T, payload.EnumEntry{Label: "ON", Value: 1},
				)
				old.TimeFormat = new(payload.TimeFormatUnixSec)
				float := lift(old).Variant.(v5.FloatReadField)
				Expect(float.EnumValues).To(Equal(
					[]payload.FloatEntry{{Label: "ON", Value: 1}},
				))
			},
		)

		It("Should give a timestamp field with no time format ISO 8601", func(
			ctx SpecContext,
		) {
			f := lift(untypedField(telem.TimestampT))
			Expect(f.Variant.(v5.TimestampReadField).TimeFormat).
				To(Equal(payload.TimeFormatISO8601))
		})

		Describe("Entries", func() {
			It("Should keep every entry of a float field", func(ctx SpecContext) {
				f := lift(untypedField(
					telem.Float64T,
					payload.EnumEntry{Label: "HALF", Value: 0.5},
					payload.EnumEntry{Label: "ON", Value: 1},
				))
				Expect(f.Variant.(v5.FloatReadField).EnumValues).To(Equal(
					[]payload.FloatEntry{
						{Label: "HALF", Value: 0.5},
						{Label: "ON", Value: 1},
					},
				))
			})

			It("Should drop the entries an integer field cannot hold", func(
				ctx SpecContext,
			) {
				f := lift(untypedField(
					telem.Int16T,
					payload.EnumEntry{Label: "LOW", Value: -3},
					payload.EnumEntry{Label: "HALF", Value: 0.5},
					payload.EnumEntry{Label: "HUGE", Value: 1e19},
				))
				Expect(f.Variant.(v5.IntegerReadField).EnumValues).To(Equal(
					[]payload.IntegerEntry{{Label: "LOW", Value: -3}},
				))
			})

			It("Should drop the negative entries of an unsigned field", func(
				ctx SpecContext,
			) {
				f := lift(untypedField(
					telem.Uint8T,
					payload.EnumEntry{Label: "LOW", Value: -1},
					payload.EnumEntry{Label: "HIGH", Value: 2},
				))
				Expect(f.Variant.(v5.UnsignedReadField).EnumValues).To(Equal(
					[]payload.UnsignedEntry{{Label: "HIGH", Value: 2}},
				))
			})

			It("Should make a boolean entry true when its value is not 0", func(
				ctx SpecContext,
			) {
				f := lift(untypedField(
					telem.BooleanT,
					payload.EnumEntry{Label: "OFF", Value: 0},
					payload.EnumEntry{Label: "ON", Value: 2},
				))
				Expect(f.Variant.(v5.BooleanReadField).EnumValues).To(Equal(
					[]payload.BooleanEntry{
						{Label: "OFF", Value: false},
						{Label: "ON", Value: true},
					},
				))
			})

			It("Should drop the entries of a string field", func(ctx SpecContext) {
				f := lift(untypedField(
					telem.StringT,
					payload.EnumEntry{Label: "ON", Value: 1},
				))
				Expect(f.Variant.(v5.StringReadField).EnumValues).To(BeEmpty())
			})

			It(
				"Should keep the first entry of a repeated label",
				func(ctx SpecContext) {
					f := lift(untypedField(
						telem.Uint8T,
						payload.EnumEntry{Label: "ON", Value: 1},
						payload.EnumEntry{Label: "HIGH", Value: 1},
						payload.EnumEntry{Label: "ON", Value: 2},
					))
					Expect(f.Variant.(v5.UnsignedReadField).EnumValues).To(Equal(
						[]payload.UnsignedEntry{
							{Label: "ON", Value: 1},
							{Label: "HIGH", Value: 1},
						},
					))
				},
			)
		})

		Describe("Fills", func() {
			onOff := []payload.EnumEntry{
				{Label: "OFF", Value: 0},
				{Label: "ON", Value: 1},
			}

			It("Should fill an optional float field with no fill with NaN", func(
				ctx SpecContext,
			) {
				f := lift(optionalField(telem.Float64T, nil))
				Expect(math.IsNaN(float64(*f.Variant.(v5.FloatReadField).Fill))).
					To(BeTrue())
			})

			DescribeTable("Should convert the fill to the kind",
				func(ctx SpecContext, field v4.ReadField, expected any) {
					f := lift(field)
					Expect(fillOf(f)).To(Equal(expected))
				},
				Entry("a float",
					optionalField(telem.Float32T, 1.5), payload.Float(1.5)),
				Entry("an integer",
					optionalField(telem.Int32T, float64(-7)), payload.Integer(-7)),
				Entry("an unsigned integer",
					optionalField(telem.Uint64T, float64(9)), payload.Unsigned(9)),
				Entry("an enum label",
					optionalField(telem.Uint8T, "ON", onOff...), payload.Unsigned(1)),
				Entry("a boolean", optionalField(telem.BooleanT, true), true),
				Entry("a boolean label",
					optionalField(telem.BooleanT, "ON", onOff...), true),
				Entry("a string", optionalField(telem.StringT, "idle"), "idle"),
				Entry("a number on a string field as its JSON text",
					optionalField(telem.StringT, float64(5)), "5"),
				Entry("numeric text on a float field",
					optionalField(telem.Float64T, "1.5"), payload.Float(1.5)),
				Entry("numeric text on an integer field",
					optionalField(telem.Int64T, "-5"), payload.Integer(-5)),
				Entry("non-finite text on a float field",
					optionalField(telem.Float64T, "-Infinity"),
					payload.Float(math.Inf(-1))),
				Entry("a timestamp in ISO 8601",
					optionalField(telem.TimestampT, "1970-01-01T00:00:01Z"),
					telem.SecondTS),
				Entry("a timestamp in a unix format",
					func() v4.ReadField {
						f := optionalField(telem.TimestampT, float64(2000))
						f.TimeFormat = new(payload.TimeFormatUnixMs)
						return f
					}(),
					2*telem.SecondTS),
			)

			DescribeTable("Should drop a fill the Driver rejected",
				func(ctx SpecContext, field v4.ReadField) {
					f := lift(field)
					Expect(fillOf(f)).To(BeNil())
				},
				Entry("on a field that is not optional", func() v4.ReadField {
					f := optionalField(telem.Float64T, 1.5)
					f.Optional = false
					return f
				}()),
				Entry("absent on an integer field", optionalField(telem.Int32T, nil)),
				Entry("absent on a string field", optionalField(telem.StringT, nil)),
				Entry("with a fraction on an integer field",
					optionalField(telem.Int32T, 1.5)),
				Entry("past the range of the data type",
					optionalField(telem.Int8T, float64(200))),
				Entry("a label no entry has",
					optionalField(telem.Uint8T, "MAYBE", onOff...)),
				Entry("text that is not a number on a float field",
					optionalField(telem.Float64T, "warm")),
				Entry("a bad timestamp",
					optionalField(telem.TimestampT, "yesterday")),
				Entry("on a JSON field", optionalField(telem.JSONT, "{}")),
			)
		})
	})

	Describe("Stored configs", func() {
		var db *gorp.DB
		BeforeEach(func() { db = DeferClose(gorp.Wrap(memkv.New())) })

		migrateStored := func(
			ctx context.Context,
			channels map[channel.Key]v5.Channel,
			migrations ...gorp.Migration,
		) v5.ReadConfig {
			GinkgoHelper()
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:        db,
				Namespace: "http_read_config",
				Migrations: append(
					migrations,
					v5.NewReadMigration(storedChannels(channels)),
				),
			})).To(Succeed())
			var cfgs []v5.ReadConfig
			Expect(gorp.NewRetrieve[uuid.UUID, v5.ReadConfig]().
				Entries(&cfgs).Exec(ctx, db)).To(Succeed())
			Expect(cfgs).To(HaveLen(1))
			return cfgs[0]
		}

		storeV4 := func(ctx context.Context, cfg v4.ReadConfig) {
			GinkgoHelper()
			MustSucceed(gorp.OpenTable(
				ctx, gorp.TableConfig[uuid.UUID, v4.ReadConfig]{DB: db},
			))
			Expect(gorp.NewCreate[uuid.UUID, v4.ReadConfig]().
				Entry(&cfg).Exec(ctx, db)).To(Succeed())
		}

		It("Should rewrite a stored v4 config in place", func(ctx SpecContext) {
			storeV4(ctx, v4.ReadConfig{
				Key: uuid.New(),
				Endpoints: []v4.ReadEndpoint{{
					Key:    "ep",
					Fields: []v4.ReadField{untypedField(telem.Float64T)},
				}},
			})
			migrateStored(
				ctx, map[channel.Key]v5.Channel{7: {DataType: telem.Float64T}},
			)
			iter := MustSucceed(db.OpenIterator(xkv.IterPrefix([]byte("gorp."))))
			records := 0
			for iter.First(); iter.Valid(); iter.Next() {
				if !bytes.HasPrefix(iter.Key(), []byte("gorp.migration.")) {
					records++
				}
			}
			Expect(iter.Close()).To(Succeed())
			Expect(records).To(Equal(1))
		})

		It("Should lift a v3 config to typed fields", func(ctx SpecContext) {
			stored := v3.ReadConfig{
				Key:            uuid.New(),
				Device:         "dev",
				Rate:           5,
				NanFillEnabled: true,
				Endpoints: []v3.ReadEndpoint{{
					Key:   "ep",
					Path:  "/values",
					Index: "f",
					Fields: []v3.ReadField{{
						Key:      "f",
						Channel:  7,
						Pointer:  "/p",
						DataType: telem.Float64T,
					}},
				}},
			}
			MustSucceed(gorp.OpenTable(
				ctx, gorp.TableConfig[uuid.UUID, v3.ReadConfig]{DB: db},
			))
			Expect(gorp.NewCreate[uuid.UUID, v3.ReadConfig]().
				Entry(&stored).Exec(ctx, db)).To(Succeed())
			got := migrateStored(
				ctx,
				map[channel.Key]v5.Channel{7: {DataType: telem.Float32T}},
				v4.Migration,
			)
			Expect(got.Device).To(Equal(stored.Device))
			Expect(got.Rate).To(Equal(stored.Rate))
			Expect(got.NanFillEnabled).To(BeTrue())
			Expect(got.Endpoints).To(HaveLen(1))
			Expect(got.Endpoints[0].Path).To(Equal("/values"))
			Expect(got.Endpoints[0].Index).To(Equal("f"))
			Expect(got.Endpoints[0].Fields).To(HaveLen(1))
			float := got.Endpoints[0].Fields[0].Variant.(v5.FloatReadField)
			Expect(float.Pointer).To(Equal("/p"))
			Expect(float.DataType).To(BeNil())
			Expect(float.Fill).To(BeNil())
		})

		It("Should lift the fills of a config that predates typed fields", func(
			ctx SpecContext,
		) {
			nan := optionalField(telem.Float64T, nil)
			nan.Key, nan.Channel = "g", 8
			storeV4(ctx, v4.ReadConfig{
				Key:    uuid.New(),
				Device: "dev",
				Endpoints: []v4.ReadEndpoint{{
					Key: "ep",
					Fields: []v4.ReadField{
						optionalField(telem.Int32T, float64(-1)),
						nan,
					},
				}},
			})
			got := migrateStored(ctx, map[channel.Key]v5.Channel{
				7: {DataType: telem.Int32T},
				8: {DataType: telem.Float64T},
			})
			fields := got.Endpoints[0].Fields
			Expect(fillOf(fields[0])).To(Equal(payload.Integer(-1)))
			Expect(math.IsNaN(float64(fillOf(fields[1]).(payload.Float)))).
				To(BeTrue())
		})

		It(
			"Should give a field with an unknown data type the kind of its channel",
			func(ctx SpecContext) {
				storeV4(ctx, v4.ReadConfig{
					Key: uuid.New(),
					Endpoints: []v4.ReadEndpoint{{
						Key:    "ep",
						Fields: []v4.ReadField{untypedField("")},
					}},
				})
				got := migrateStored(
					ctx, map[channel.Key]v5.Channel{7: {DataType: telem.StringT}},
				)
				Expect(got.Endpoints[0].Fields[0].Type()).To(Equal(payload.KindString))
			},
		)

		It("Should keep the time format of a v4 Console timing field", func(
			ctx SpecContext,
		) {
			timing := untypedField(telem.Float64T)
			timing.TimeFormat = new(payload.TimeFormatUnixSec)
			storeV4(ctx, v4.ReadConfig{
				Key: uuid.New(),
				Endpoints: []v4.ReadEndpoint{{
					Key:    "ep",
					Index:  "f",
					Fields: []v4.ReadField{timing},
				}},
			})
			got := migrateStored(ctx, map[channel.Key]v5.Channel{
				7: {DataType: telem.TimestampT, IsIndex: true},
			})
			timestamp := got.Endpoints[0].Fields[0].Variant.(v5.TimestampReadField)
			Expect(timestamp.TimeFormat).To(Equal(payload.TimeFormatUnixSec))
		})

		It("Should keep the time format of a field on a timestamp channel", func(
			ctx SpecContext,
		) {
			field := untypedField(telem.Float64T)
			field.TimeFormat = new(payload.TimeFormatUnixMs)
			storeV4(ctx, v4.ReadConfig{
				Key: uuid.New(),
				Endpoints: []v4.ReadEndpoint{{
					Key:    "ep",
					Fields: []v4.ReadField{field},
				}},
			})
			got := migrateStored(
				ctx, map[channel.Key]v5.Channel{7: {DataType: telem.TimestampT}},
			)
			timestamp := got.Endpoints[0].Fields[0].Variant.(v5.TimestampReadField)
			Expect(timestamp.TimeFormat).To(Equal(payload.TimeFormatUnixMs))
		})

		It("Should read a fill by the data type of the field's channel", func(
			ctx SpecContext,
		) {
			storeV4(ctx, v4.ReadConfig{
				Key: uuid.New(),
				Endpoints: []v4.ReadEndpoint{{
					Key:    "ep",
					Fields: []v4.ReadField{optionalField(telem.Float64T, "off")},
				}},
			})
			got := migrateStored(
				ctx, map[channel.Key]v5.Channel{7: {DataType: telem.StringT}},
			)
			Expect(fillOf(got.Endpoints[0].Fields[0])).To(Equal("off"))
		})

		It("Should read entries by the data type of the field's channel", func(
			ctx SpecContext,
		) {
			storeV4(ctx, v4.ReadConfig{
				Key: uuid.New(),
				Endpoints: []v4.ReadEndpoint{{
					Key: "ep",
					Fields: []v4.ReadField{untypedField(
						telem.Float64T,
						payload.EnumEntry{Label: "ON", Value: 1},
						payload.EnumEntry{Label: "OFF", Value: 0},
					)},
				}},
			})
			got := migrateStored(
				ctx, map[channel.Key]v5.Channel{7: {DataType: telem.BooleanT}},
			)
			boolean := got.Endpoints[0].Fields[0].Variant.(v5.BooleanReadField)
			Expect(boolean.EnumValues).To(Equal([]payload.BooleanEntry{
				{Label: "ON", Value: true},
				{Label: "OFF", Value: false},
			}))
		})

		It("Should reconcile each field with its channel", func(ctx SpecContext) {
			storeV4(ctx, v4.ReadConfig{
				Key: uuid.New(),
				Endpoints: []v4.ReadEndpoint{{
					Key: "ep",
					Fields: []v4.ReadField{optionalField(
						telem.Float64T, float64(3),
						payload.EnumEntry{Label: "ON", Value: 1},
						payload.EnumEntry{Label: "HALF", Value: 0.5},
					)},
				}},
			})
			got := migrateStored(
				ctx, map[channel.Key]v5.Channel{7: {DataType: telem.Uint8T}},
			)
			unsigned := got.Endpoints[0].Fields[0].Variant.(v5.UnsignedReadField)
			Expect(unsigned.DataType).To(BeNil())
			Expect(*unsigned.Fill).To(Equal(payload.Unsigned(3)))
			Expect(unsigned.EnumValues).To(Equal(
				[]payload.UnsignedEntry{{Label: "ON", Value: 1}},
			))
		})
	})
})
