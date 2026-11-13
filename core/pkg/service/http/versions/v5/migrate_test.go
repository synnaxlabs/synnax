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
	xpayload "github.com/synnaxlabs/x/payload"
	payloadv0 "github.com/synnaxlabs/x/payload/versions/v0"
	payload "github.com/synnaxlabs/x/payload/versions/v1"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

func storedChannels(found map[channel.Key]v5.Channel) v5.Channels {
	return func(
		context.Context,
		gorp.Snapshot,
		[]channel.Key,
	) (map[channel.Key]v5.Channel, error) {
		return found, nil
	}
}

func untypedField(dt telem.DataType, entries ...payloadv0.EnumEntry) v4.ReadField {
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
	entries ...payloadv0.EnumEntry,
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

var _ = Describe("Migration", func() {
	It("Should rewrite a stored v4 write config in place", func(ctx SpecContext) {
		stored := v4.WriteConfig{
			Key:    uuid.New(),
			Device: "dev",
			Endpoints: []v4.WriteEndpoint{{
				Key:  "ep",
				Path: "/set",
				Channel: v4.ChannelField{
					Pointer:  "/v",
					JSONType: payloadv0.JSONTypeString,
					Channel:  7,
				},
				Fields: []v4.WriteField{
					{Variant: v4.StaticWriteField{
						BaseWriteField: v4.BaseWriteField{Key: "i", Pointer: "/i"},
						JSONType:       payloadv0.JSONTypeString,
						Value:          int64(-3),
					}},
					{Variant: v4.StaticWriteField{
						BaseWriteField: v4.BaseWriteField{Key: "f", Pointer: "/f"},
						JSONType:       payloadv0.JSONTypeNumber,
						Value:          1.5,
					}},
					{Variant: v4.StaticWriteField{
						BaseWriteField: v4.BaseWriteField{Key: "n", Pointer: "/n"},
					}},
					{Variant: v4.GeneratedWriteField{
						BaseWriteField: v4.BaseWriteField{Key: "t", Pointer: "/t"},
						Generator:      payloadv0.GeneratorTypeTimestamp,
					}},
				},
			}},
		}
		db := DeferClose(gorp.Wrap(memkv.New()))
		MustSucceed(
			gorp.OpenTable(ctx, gorp.TableConfig[uuid.UUID, v4.WriteConfig]{DB: db}),
		)
		Expect(
			gorp.NewCreate[uuid.UUID, v4.WriteConfig]().Entry(&stored).Exec(ctx, db),
		).To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:        db,
			Namespace: "http_write_config",
			Migrations: []gorp.Migration{v5.NewWriteMigration(storedChannels(
				map[channel.Key]v5.Channel{7: {DataType: telem.StringT}},
			))},
		})).To(Succeed())
		var got v5.WriteConfig
		Expect(gorp.NewRetrieve[uuid.UUID, v5.WriteConfig]().
			Where(gorp.MatchKeys[uuid.UUID, v5.WriteConfig](stored.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Device).To(Equal(stored.Device))
		Expect(got.Endpoints).To(HaveLen(1))
		ep := got.Endpoints[0]
		Expect(ep.Path).To(Equal("/set"))
		str, ok := ep.Channel.Variant.(v5.StringChannelField)
		Expect(ok).To(BeTrue())
		Expect(str.BaseChannelField).
			To(Equal(v5.BaseChannelField{Pointer: "/v", Channel: 7}))
		Expect(str.EnumValues).To(BeEmpty())
		Expect(ep.Fields).To(Equal([]payload.WriteField{
			{Variant: payload.IntegerWriteField{
				BaseWriteField: payload.BaseWriteField{Key: "i", Pointer: "/i"},
				Value:          -3,
			}},
			{Variant: payload.FloatWriteField{
				BaseWriteField: payload.BaseWriteField{Key: "f", Pointer: "/f"},
				Value:          1.5,
			}},
			{Variant: payload.JsonWriteField{
				BaseWriteField: payload.BaseWriteField{Key: "n", Pointer: "/n"},
				Value:          "null",
			}},
			{Variant: payload.TimestampWriteField{
				BaseWriteField: payload.BaseWriteField{Key: "t", Pointer: "/t"},
				TimeFormat:     payload.TimeFormatISO8601,
			}},
		}))
		Expect(
			gorp.NewRetrieve[uuid.UUID, v5.WriteConfig]().Count(ctx, db),
		).To(Equal(1))
	})

	Describe("MigrateWriteField", func() {
		base := v4.BaseWriteField{Key: "g", Pointer: "/g"}

		It("Should lift a generator in its time format", func() {
			f, ok := MustSucceed2(v5.MigrateWriteField(v4.WriteField{
				Variant: v4.GeneratedWriteField{
					BaseWriteField: base,
					Generator:      payloadv0.GeneratorTypeTimestamp,
					TimeFormat:     new(payloadv0.TimeFormatUnixMs),
				},
			}))
			Expect(ok).To(BeTrue())
			Expect(f.Variant).To(Equal(payload.TimestampWriteField{
				BaseWriteField: payload.BaseWriteField(base),
				TimeFormat:     payload.TimeFormatUnixMs,
			}))
		})

		It("Should drop a field with an unknown generator", func() {
			_, ok := MustSucceed2(v5.MigrateWriteField(v4.WriteField{
				Variant: v4.GeneratedWriteField{BaseWriteField: base, Generator: "id"},
			}))
			Expect(ok).To(BeFalse())
		})

		It("Should drop a field with no variant", func() {
			_, ok := MustSucceed2(v5.MigrateWriteField(v4.WriteField{}))
			Expect(ok).To(BeFalse())
		})
	})

	It("Should drop each write field with an unknown generator", func(
		ctx SpecContext,
	) {
		ep := MustSucceed(v5.MigrateWriteEndpoint(ctx, v4.WriteEndpoint{
			Key:      "ep",
			Disabled: true,
			Fields: []v4.WriteField{
				{Variant: v4.GeneratedWriteField{Generator: "id"}},
				{Variant: v4.GeneratedWriteField{
					Generator: payloadv0.GeneratorTypeUUID,
				}},
			},
		}))
		Expect(ep.Disabled).To(BeTrue())
		Expect(ep.Fields).To(Equal([]payload.WriteField{
			{Variant: payload.UuidWriteField{}},
		}))
	})
})

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
					Index:  "f",
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
			Expect(ep.Index).To(Equal("f"))
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
				timing.TimeFormat = new(xpayload.TimeFormatUnixSec)
				ep := MustSucceed(v5.MigrateReadEndpoint(ctx, v4.ReadEndpoint{
					Index:  "f",
					Fields: []v4.ReadField{timing},
				}))
				timestamp := ep.Fields[0].Variant.(v5.TimestampReadField)
				Expect(timestamp.TimeFormat).To(Equal(xpayload.TimeFormatUnixSec))
			},
		)
	})

	Describe("MigrateReadField", func() {
		DescribeTable("Should pick the kind of the stored data type",
			func(ctx SpecContext, dt telem.DataType, kind xpayload.Kind) {
				f := v5.MigrateReadField(untypedField(dt))
				Expect(f.Type()).To(Equal(kind))
			},
			Entry("float64", telem.Float64T, xpayload.KindFloat),
			Entry("int16", telem.Int16T, xpayload.KindInteger),
			Entry("uint8", telem.Uint8T, xpayload.KindUnsigned),
			Entry("boolean", telem.BooleanT, xpayload.KindBoolean),
			Entry("string", telem.StringT, xpayload.KindString),
			Entry("timestamp", telem.TimestampT, xpayload.KindTimestamp),
			Entry("json", telem.JSONT, xpayload.KindJSON),
			Entry("uuid", telem.UUIDT, xpayload.KindUUID),
			Entry("bytes", telem.BytesT, xpayload.KindBytes),
		)

		It("Should read an unknown data type as float64", func() {
			f := v5.MigrateReadField(untypedField("complex128"))
			Expect(*f.Variant.(v5.FloatReadField).DataType).To(Equal(telem.Float64T))
		})

		It("Should keep the data type of a numeric field", func(ctx SpecContext) {
			f := v5.MigrateReadField(untypedField(telem.Int16T))
			Expect(*f.Variant.(v5.IntegerReadField).DataType).To(Equal(telem.Int16T))
		})

		It("Should keep the time format of a timestamp field", func() {
			old := untypedField(telem.TimestampT)
			old.TimeFormat = new(xpayload.TimeFormatUnixMs)
			f := v5.MigrateReadField(old)
			Expect(f.Variant.(v5.TimestampReadField).TimeFormat).
				To(Equal(xpayload.TimeFormatUnixMs))
		})

		It("Should read a field with a time format as a timestamp field", func() {
			old := untypedField(telem.Float64T)
			old.TimeFormat = new(xpayload.TimeFormatUnixSec)
			f := v5.MigrateReadField(old)
			Expect(f.Variant.(v5.TimestampReadField).TimeFormat).
				To(Equal(xpayload.TimeFormatUnixSec))
		})

		It("Should keep the stored data type of a field with a time format", func() {
			old := optionalField(
				telem.Float64T, nil, payloadv0.EnumEntry{Label: "ON", Value: 1},
			)
			old.TimeFormat = new(xpayload.TimeFormatUnixSec)
			float := v5.MigrateReadField(old).Variant.(v5.FloatReadField)
			Expect(math.IsNaN(float64(*float.Fill))).To(BeTrue())
			Expect(float.EnumValues).To(Equal(
				[]xpayload.FloatEntry{{Label: "ON", Value: 1}},
			))
		})

		It("Should give a timestamp field with no time format ISO 8601", func(
			ctx SpecContext,
		) {
			f := v5.MigrateReadField(untypedField(telem.TimestampT))
			Expect(f.Variant.(v5.TimestampReadField).TimeFormat).
				To(Equal(xpayload.TimeFormatISO8601))
		})

		Describe("Entries", func() {
			It("Should keep every entry of a float field", func(ctx SpecContext) {
				f := v5.MigrateReadField(untypedField(
					telem.Float64T,
					payloadv0.EnumEntry{Label: "HALF", Value: 0.5},
					payloadv0.EnumEntry{Label: "ON", Value: 1},
				))
				Expect(f.Variant.(v5.FloatReadField).EnumValues).To(Equal(
					[]xpayload.FloatEntry{
						{Label: "HALF", Value: 0.5},
						{Label: "ON", Value: 1},
					},
				))
			})

			It("Should drop the entries an integer field cannot hold", func(
				ctx SpecContext,
			) {
				f := v5.MigrateReadField(untypedField(
					telem.Int16T,
					payloadv0.EnumEntry{Label: "LOW", Value: -3},
					payloadv0.EnumEntry{Label: "HALF", Value: 0.5},
					payloadv0.EnumEntry{Label: "HUGE", Value: 1e19},
				))
				Expect(f.Variant.(v5.IntegerReadField).EnumValues).To(Equal(
					[]xpayload.IntegerEntry{{Label: "LOW", Value: -3}},
				))
			})

			It("Should drop the negative entries of an unsigned field", func(
				ctx SpecContext,
			) {
				f := v5.MigrateReadField(untypedField(
					telem.Uint8T,
					payloadv0.EnumEntry{Label: "LOW", Value: -1},
					payloadv0.EnumEntry{Label: "HIGH", Value: 2},
				))
				Expect(f.Variant.(v5.UnsignedReadField).EnumValues).To(Equal(
					[]xpayload.UnsignedEntry{{Label: "HIGH", Value: 2}},
				))
			})

			It("Should make a boolean entry true when its value is not 0", func(
				ctx SpecContext,
			) {
				f := v5.MigrateReadField(untypedField(
					telem.BooleanT,
					payloadv0.EnumEntry{Label: "OFF", Value: 0},
					payloadv0.EnumEntry{Label: "ON", Value: 2},
				))
				Expect(f.Variant.(v5.BooleanReadField).EnumValues).To(Equal(
					[]xpayload.BooleanEntry{
						{Label: "OFF", Value: false},
						{Label: "ON", Value: true},
					},
				))
			})

			It("Should drop the entries of a string field", func(ctx SpecContext) {
				f := v5.MigrateReadField(untypedField(
					telem.StringT,
					payloadv0.EnumEntry{Label: "ON", Value: 1},
				))
				Expect(f.Variant.(v5.StringReadField).EnumValues).To(BeEmpty())
			})

			It(
				"Should keep the first entry of a repeated label",
				func(ctx SpecContext) {
					f := v5.MigrateReadField(untypedField(
						telem.Uint8T,
						payloadv0.EnumEntry{Label: "ON", Value: 1},
						payloadv0.EnumEntry{Label: "HIGH", Value: 1},
						payloadv0.EnumEntry{Label: "ON", Value: 2},
					))
					Expect(f.Variant.(v5.UnsignedReadField).EnumValues).To(Equal(
						[]xpayload.UnsignedEntry{
							{Label: "ON", Value: 1},
							{Label: "HIGH", Value: 1},
						},
					))
				},
			)
		})

		Describe("Fills", func() {
			onOff := []payloadv0.EnumEntry{
				{Label: "OFF", Value: 0},
				{Label: "ON", Value: 1},
			}

			It("Should fill an optional float field with no fill with NaN", func(
				ctx SpecContext,
			) {
				f := v5.MigrateReadField(optionalField(telem.Float64T, nil))
				Expect(math.IsNaN(float64(*f.Variant.(v5.FloatReadField).Fill))).
					To(BeTrue())
			})

			DescribeTable("Should convert the fill to the kind",
				func(ctx SpecContext, field v4.ReadField, expected any) {
					f := v5.MigrateReadField(field)
					Expect(fillOf(f)).To(Equal(expected))
				},
				Entry("a float",
					optionalField(telem.Float32T, 1.5), xpayload.Float(1.5)),
				Entry("an integer",
					optionalField(telem.Int32T, float64(-7)), xpayload.Integer(-7)),
				Entry("an unsigned integer",
					optionalField(telem.Uint64T, float64(9)), xpayload.Unsigned(9)),
				Entry("an enum label",
					optionalField(telem.Uint8T, "ON", onOff...), xpayload.Unsigned(1)),
				Entry("a boolean", optionalField(telem.BooleanT, true), true),
				Entry("a boolean label",
					optionalField(telem.BooleanT, "ON", onOff...), true),
				Entry("a string", optionalField(telem.StringT, "idle"), "idle"),
				Entry("a timestamp in ISO 8601",
					optionalField(telem.TimestampT, "1970-01-01T00:00:01Z"),
					telem.SecondTS),
				Entry("a timestamp in a unix format",
					func() v4.ReadField {
						f := optionalField(telem.TimestampT, float64(2000))
						f.TimeFormat = new(xpayload.TimeFormatUnixMs)
						return f
					}(),
					2*telem.SecondTS),
			)

			DescribeTable("Should drop a fill the Driver rejected",
				func(ctx SpecContext, field v4.ReadField) {
					f := v5.MigrateReadField(field)
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
				Entry("a number on a string field",
					optionalField(telem.StringT, float64(1))),
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
			Expect(fillOf(fields[0])).To(Equal(xpayload.Integer(-1)))
			Expect(math.IsNaN(float64(fillOf(fields[1]).(xpayload.Float)))).
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
				Expect(got.Endpoints[0].Fields[0].Type()).To(Equal(xpayload.KindString))
			},
		)

		It("Should keep the time format of a v4 Console timing field", func(
			ctx SpecContext,
		) {
			timing := untypedField(telem.Float64T)
			timing.TimeFormat = new(xpayload.TimeFormatUnixSec)
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
			Expect(timestamp.TimeFormat).To(Equal(xpayload.TimeFormatUnixSec))
		})

		It("Should keep the time format of a field on a timestamp channel", func(
			ctx SpecContext,
		) {
			field := untypedField(telem.Float64T)
			field.TimeFormat = new(xpayload.TimeFormatUnixMs)
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
			Expect(timestamp.TimeFormat).To(Equal(xpayload.TimeFormatUnixMs))
		})

		It("Should reconcile each field with its channel", func(ctx SpecContext) {
			storeV4(ctx, v4.ReadConfig{
				Key: uuid.New(),
				Endpoints: []v4.ReadEndpoint{{
					Key: "ep",
					Fields: []v4.ReadField{optionalField(
						telem.Float64T, float64(3),
						payloadv0.EnumEntry{Label: "ON", Value: 1},
						payloadv0.EnumEntry{Label: "HALF", Value: 0.5},
					)},
				}},
			})
			got := migrateStored(
				ctx, map[channel.Key]v5.Channel{7: {DataType: telem.Uint8T}},
			)
			unsigned := got.Endpoints[0].Fields[0].Variant.(v5.UnsignedReadField)
			Expect(unsigned.DataType).To(BeNil())
			Expect(*unsigned.Fill).To(Equal(xpayload.Unsigned(3)))
			Expect(unsigned.EnumValues).To(Equal(
				[]xpayload.UnsignedEntry{{Label: "ON", Value: 1}},
			))
		})
	})
})
