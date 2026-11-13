// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v3 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v4"
	payload "github.com/synnaxlabs/x/payload/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migration", func() {
	It(
		"Should keep every read config field and leave fields not optional",
		func(ctx SpecContext) {
			stored := v3.ReadConfig{
				Key:            uuid.New(),
				Device:         "dev",
				Rate:           5,
				NanFillEnabled: true,
				Endpoints: []v3.ReadEndpoint{{
					Key:    "ep",
					Path:   "/values",
					Index:  "f",
					Fields: []v3.ReadField{{Key: "f", Channel: 7, Pointer: "/p"}},
				}},
			}
			db := DeferClose(gorp.Wrap(memkv.New()))
			MustSucceed(
				gorp.OpenTable(ctx, gorp.TableConfig[uuid.UUID, v3.ReadConfig]{DB: db}),
			)
			Expect(
				gorp.NewCreate[uuid.UUID, v3.ReadConfig]().Entry(&stored).Exec(ctx, db),
			).
				To(Succeed())
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:         db,
				Namespace:  "http_read_config",
				Migrations: []gorp.Migration{v4.Migration},
			})).To(Succeed())
			var got v4.ReadConfig
			Expect(gorp.NewRetrieve[uuid.UUID, v4.ReadConfig]().
				Where(gorp.MatchKeys[uuid.UUID, v4.ReadConfig](stored.Key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			Expect(got.Device).To(Equal(stored.Device))
			Expect(got.NanFillEnabled).To(BeTrue())
			Expect(got.Endpoints).To(HaveLen(1))
			Expect(got.Endpoints[0].Index).To(Equal("f"))
			Expect(got.Endpoints[0].Fields).To(HaveLen(1))
			Expect(got.Endpoints[0].Fields[0].Channel).To(BeEquivalentTo(7))
			Expect(got.Endpoints[0].Fields[0].Pointer).To(Equal("/p"))
			Expect(got.Endpoints[0].Fields[0].Optional).To(BeFalse())
			Expect(got.Endpoints[0].Fields[0].FillValue).To(BeNil())
		},
	)

	It("Should keep the enums and time formats of a read field", func(ctx SpecContext) {
		unixMs := v3.TimeFormatUnixMs
		f := MustSucceed(v4.MigrateReadField(ctx, v3.ReadField{
			Key:        "f",
			TimeFormat: &unixMs,
			EnumValues: []v3.EnumEntry{{Label: "ON", Value: 1}},
		}))
		Expect(*f.TimeFormat).To(Equal(payload.TimeFormatUnixMs))
		Expect(f.EnumValues).To(Equal([]payload.EnumEntry{{Label: "ON", Value: 1}}))
	})

	It("Should keep every write config field", func(ctx SpecContext) {
		unixMs := v3.TimeFormatUnixMs
		cfg := MustSucceed(v4.MigrateWriteConfig(ctx, v3.WriteConfig{
			Device: "dev",
			Endpoints: []v3.WriteEndpoint{{
				Key:  "ep",
				Path: "/set",
				Channel: v3.ChannelField{
					Pointer:    "/v",
					JSONType:   v3.JSONTypeString,
					Channel:    7,
					TimeFormat: &unixMs,
					EnumValues: []v3.EnumEntry{{Label: "HIGH", Value: 2}},
				},
				Fields: []v3.WriteField{
					{Variant: v3.StaticWriteField{
						BaseWriteField: v3.BaseWriteField{Key: "s", Pointer: "/s"},
						JSONType:       v3.JSONTypeBoolean,
						Value:          true,
					}},
					{Variant: v3.GeneratedWriteField{
						BaseWriteField: v3.BaseWriteField{Key: "g", Pointer: "/g"},
						Generator:      v3.GeneratorTypeTimestamp,
						TimeFormat:     &unixMs,
					}},
				},
			}},
		}))
		Expect(cfg.Device).To(Equal("dev"))
		Expect(cfg.Endpoints).To(HaveLen(1))
		ep := cfg.Endpoints[0]
		Expect(ep.Path).To(Equal("/set"))
		Expect(ep.Channel.Pointer).To(Equal("/v"))
		Expect(ep.Channel.JSONType).To(Equal(payload.JSONTypeString))
		Expect(ep.Channel.Channel).To(BeEquivalentTo(7))
		Expect(*ep.Channel.TimeFormat).To(Equal(payload.TimeFormatUnixMs))
		Expect(ep.Channel.EnumValues).To(Equal(
			[]payload.EnumEntry{{Label: "HIGH", Value: 2}},
		))
		Expect(ep.Fields).To(HaveLen(2))
		static := ep.Fields[0].Variant.(v4.StaticWriteField)
		Expect(static.Key).To(Equal("s"))
		Expect(static.JSONType).To(Equal(payload.JSONTypeBoolean))
		Expect(static.Value).To(BeTrue())
		generated := ep.Fields[1].Variant.(v4.GeneratedWriteField)
		Expect(generated.Pointer).To(Equal("/g"))
		Expect(generated.Generator).To(Equal(payload.GeneratorTypeTimestamp))
		Expect(*generated.TimeFormat).To(Equal(payload.TimeFormatUnixMs))
	})

	Describe("MigrateWriteEndpoint", func() {
		It(
			"Should keep the endpoint, its channel, and its fields",
			func(ctx SpecContext) {
				ep := MustSucceed(v4.MigrateWriteEndpoint(ctx, v3.WriteEndpoint{
					Key:      "ep",
					Disabled: true,
					Path:     "/set",
					Channel:  v3.ChannelField{JSONType: v3.JSONTypeBoolean},
					Fields: []v3.WriteField{{Variant: v3.StaticWriteField{
						BaseWriteField: v3.BaseWriteField{Key: "s"},
					}}},
				}))
				Expect(ep.Key).To(Equal("ep"))
				Expect(ep.Disabled).To(BeTrue())
				Expect(ep.Path).To(Equal("/set"))
				Expect(ep.Channel.JSONType).To(Equal(payload.JSONTypeBoolean))
				Expect(ep.Fields).To(HaveLen(1))
			},
		)
	})

	Describe("MigrateChannelField", func() {
		It("Should keep every field", func(ctx SpecContext) {
			iso := v3.TimeFormatISO8601
			f := MustSucceed(v4.MigrateChannelField(ctx, v3.ChannelField{
				Pointer:    "/v",
				JSONType:   v3.JSONTypeNumber,
				Channel:    3,
				Name:       "cmd",
				DataType:   "float32",
				TimeFormat: &iso,
				EnumValues: []v3.EnumEntry{{Label: "LOW", Value: 1}},
			}))
			Expect(f.Pointer).To(Equal("/v"))
			Expect(f.JSONType).To(Equal(payload.JSONTypeNumber))
			Expect(f.Channel).To(BeEquivalentTo(3))
			Expect(f.Name).To(Equal("cmd"))
			Expect(f.DataType).To(BeEquivalentTo("float32"))
			Expect(*f.TimeFormat).To(Equal(payload.TimeFormatISO8601))
			Expect(
				f.EnumValues,
			).To(Equal([]payload.EnumEntry{{Label: "LOW", Value: 1}}))
		})
	})

	Describe("MigrateWriteField", func() {
		It("Should keep a static field", func(ctx SpecContext) {
			f := MustSucceed(v4.MigrateWriteField(ctx, v3.WriteField{
				Variant: v3.StaticWriteField{
					BaseWriteField: v3.BaseWriteField{Key: "s", Pointer: "/s"},
					JSONType:       v3.JSONTypeString,
					Value:          "on",
				},
			}))
			static := f.Variant.(v4.StaticWriteField)
			Expect(static.Pointer).To(Equal("/s"))
			Expect(static.JSONType).To(Equal(payload.JSONTypeString))
			Expect(static.Value).To(Equal("on"))
		})

		It("Should keep a generated field", func(ctx SpecContext) {
			f := MustSucceed(v4.MigrateWriteField(ctx, v3.WriteField{
				Variant: v3.GeneratedWriteField{Generator: v3.GeneratorTypeUUID},
			}))
			generated := f.Variant.(v4.GeneratedWriteField)
			Expect(generated.Generator).To(Equal(payload.GeneratorTypeUUID))
			Expect(generated.TimeFormat).To(BeNil())
		})

		It("Should keep a field with no variant empty", func(ctx SpecContext) {
			f := MustSucceed(v4.MigrateWriteField(ctx, v3.WriteField{}))
			Expect(f.Variant).To(BeNil())
		})
	})
})
