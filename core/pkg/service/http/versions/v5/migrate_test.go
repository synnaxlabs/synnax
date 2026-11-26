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
	"encoding/json/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	v2 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v2"
	v4 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v4"
	v5 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v5"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// field returns a read field as the Console stores it.
func field(key string, ch channel.Key, pointer string) v4.ReadField {
	return v4.ReadField{
		Key:        key,
		Channel:    ch,
		Pointer:    pointer,
		EnumValues: []payload.EnumEntry{},
	}
}

// storedConfig returns a v4 config with two endpoints, one indexed by its field t.
func storedConfig() v4.ReadConfig {
	return v4.ReadConfig{
		Key:    uuid.New(),
		Device: "dev",
		Rate:   5,
		Endpoints: []v4.ReadEndpoint{
			{
				Key:     "ep1",
				Method:  v4.MethodPost,
				Path:    "/values",
				Headers: []v4.Header{{Name: "Accept", Value: "application/json"}},
				Body:    `{"all":true}`,
				Index:   "t",
				Fields: []v4.ReadField{
					field("t", 6, "/time"),
					field("p", 7, "/pressure"),
				},
			},
			{
				Key:    "ep2",
				Path:   "/status",
				Fields: []v4.ReadField{field("s", 8, "/state")},
			},
		},
	}
}

var _ = Describe("MigrateReadConfig", func() {
	It("Should stage the fields of each endpoint and keep its request",
		func(ctx SpecContext) {
			old := storedConfig()
			cfg := MustSucceed(v5.MigrateReadConfig(ctx, old))
			Expect(cfg.Device).To(Equal(old.Device))
			Expect(cfg.Rate).To(Equal(old.Rate))
			Expect(cfg.Endpoints).To(HaveLen(2))
			ep := cfg.Endpoints[0]
			Expect(ep.Method).To(Equal(v4.MethodPost))
			Expect(ep.Path).To(Equal("/values"))
			Expect(ep.Headers).To(Equal(old.Endpoints[0].Headers))
			Expect(ep.Body).To(Equal(`{"all":true}`))
			Expect(ep.ICD).To(BeNil())
			Expect(ep.Messages).To(BeEmpty())
			Expect(cfg.Inline).To(Equal(&v5.InlineRead{Endpoints: []v5.InlineEndpoint{
				{Key: "ep1", Index: "t", Fields: old.Endpoints[0].Fields},
				{Key: "ep2", Fields: old.Endpoints[1].Fields},
			}}))
		},
	)

	It("Should stage nothing for a config with no endpoints", func(ctx SpecContext) {
		cfg := MustSucceed(v5.MigrateReadConfig(ctx, v4.ReadConfig{Key: uuid.New()}))
		Expect(cfg.Inline).To(BeNil())
	})

	DescribeTable("Should keep the NaN fill of v4",
		func(ctx SpecContext, nanFill bool) {
			old := storedConfig()
			old.NanFillEnabled = nanFill
			old.Endpoints[1].Fields = append(
				old.Endpoints[1].Fields,
				v4.ReadField{Key: "o", Optional: true},
			)
			cfg := MustSucceed(v5.MigrateReadConfig(ctx, old))
			Expect(cfg.NanFillEnabled).To(Equal(nanFill))
		},
		Entry("on", true),
		Entry("off, with an optional field that has no fill value", false),
	)
})

var _ = Describe("MigrateReadEndpoint", func() {
	It("Should keep the request and fields of an endpoint, with no ICD",
		func(ctx SpecContext) {
			old := storedConfig().Endpoints[0]
			ep := MustSucceed(v5.MigrateReadEndpoint(ctx, old))
			Expect(ep.Key).To(Equal("ep1"))
			Expect(ep.Method).To(Equal(v4.MethodPost))
			Expect(ep.Path).To(Equal("/values"))
			Expect(ep.Headers).To(Equal(old.Headers))
			Expect(ep.Body).To(Equal(`{"all":true}`))
			Expect(ep.Fields).To(Equal(old.Fields))
			Expect(ep.Index).To(Equal("t"))
			Expect(ep.ICD).To(BeNil())
			Expect(ep.Messages).To(BeEmpty())
		},
	)
})

var _ = Describe("StagedFieldsKVKey", func() {
	It("Should key the staged fields of a config under the prefix", func() {
		key := uuid.New()
		Expect(string(v5.StagedFieldsKVKey(key))).
			To(Equal(v5.StagedFieldsKVPrefix + key.String()))
	})
})

var _ = Describe("ReadMigration", func() {
	var db *gorp.DB
	BeforeEach(func() { db = DeferClose(gorp.Wrap(memkv.New())) })

	migrate := func(ctx SpecContext, olds ...v4.ReadConfig) {
		GinkgoHelper()
		Expect(gorp.WrapWriter[uuid.UUID, v4.ReadConfig](db).Set(ctx, olds...)).
			To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "http_read_config",
			Migrations: []gorp.Migration{v5.ReadMigration},
		})).To(Succeed())
	}
	read := func(ctx SpecContext, key uuid.UUID) v5.ReadConfig {
		GinkgoHelper()
		var cfg v5.ReadConfig
		Expect(gorp.NewRetrieve[uuid.UUID, v5.ReadConfig]().
			Where(gorp.MatchKeys[uuid.UUID, v5.ReadConfig](key)).
			Entry(&cfg).
			Exec(ctx, db)).To(Succeed())
		return cfg
	}

	It("Should store the shape that v4 stored under the same table name", func() {
		Expect(v5.ReadConfig{}.CustomTypeName()).
			To(Equal(v4.ReadConfig{}.CustomTypeName()))
	})

	It("Should stage the fields of each config under its own KV key",
		func(ctx SpecContext) {
			old := storedConfig()
			old.Endpoints[1].Fields[0].Optional = true
			migrate(ctx, old)
			migrated := read(ctx, old.Key)
			Expect(migrated.Inline).To(BeNil())
			Expect(migrated.NanFillEnabled).To(BeFalse())
			Expect(migrated.Endpoints).To(HaveLen(2))
			Expect(migrated.Endpoints[0].Path).To(Equal("/values"))
			b, closer := MustSucceed2(db.Get(v5.StagedFieldsKVKey(old.Key)))
			var staged v5.InlineRead
			Expect(json.Unmarshal(b, &staged)).To(Succeed())
			Expect(closer.Close()).To(Succeed())
			Expect(staged).To(Equal(v5.InlineRead{Endpoints: []v5.InlineEndpoint{
				{Key: "ep1", Index: "t", Fields: old.Endpoints[0].Fields},
				{Key: "ep2", Fields: old.Endpoints[1].Fields},
			}}))
		},
	)

	It("Should stage nothing for a config without endpoints", func(ctx SpecContext) {
		old := v4.ReadConfig{Key: uuid.New(), Device: "dev"}
		migrate(ctx, old)
		Expect(read(ctx, old.Key).Device).To(Equal("dev"))
		Expect(db.Get(v5.StagedFieldsKVKey(old.Key))).Error().
			To(MatchError(query.ErrNotFound))
	})
})

// storedWriteConfig returns a v4 write config with two endpoints, as the Console stores
// it. The first sends a static and a generated field beside its channel, and the second
// is disabled.
func storedWriteConfig() v4.WriteConfig {
	return v4.WriteConfig{
		Key:    uuid.New(),
		Device: "dev",
		Endpoints: []v4.WriteEndpoint{
			{
				Key:     "sp",
				Method:  v4.MethodPut,
				Path:    "/setpoint",
				Headers: []v4.Header{{Name: "Accept", Value: "application/json"}},
				Channel: v4.ChannelField{
					Pointer:    "/value",
					JSONType:   payload.JSONTypeNumber,
					Channel:    42,
					Name:       "setpoint_cmd",
					DataType:   "float64",
					EnumValues: []payload.EnumEntry{},
				},
				Fields: []v4.WriteField{
					{Variant: v4.StaticWriteField{
						BaseWriteField: v4.BaseWriteField{Key: "u", Pointer: "/unit"},
						JSONType:       payload.JSONTypeString,
						Value:          "psi",
					}},
					{Variant: v4.GeneratedWriteField{
						BaseWriteField: v4.BaseWriteField{
							Key:     "id",
							Pointer: "/request_id",
						},
						Generator: payload.GeneratorTypeUUID,
					}},
				},
			},
			{
				Key:      "valve",
				Disabled: true,
				Method:   v4.MethodPost,
				Path:     "/valve",
				Channel: v4.ChannelField{
					Channel:    43,
					DataType:   "uint8",
					EnumValues: []payload.EnumEntry{},
				},
				Fields: []v4.WriteField{},
			},
		},
	}
}

// stagedWrite returns the endpoints of old as MigrateWriteConfig stages them.
func stagedWrite(old v4.WriteConfig) v5.InlineWrite {
	return v5.InlineWrite{Endpoints: []v5.InlineWriteEndpoint{
		{
			Key:     "sp",
			Channel: old.Endpoints[0].Channel,
			Fields:  old.Endpoints[0].Fields,
		},
		{
			Key:     "valve",
			Channel: old.Endpoints[1].Channel,
			Fields:  old.Endpoints[1].Fields,
		},
	}}
}

var _ = Describe("MigrateWriteConfig", func() {
	It("Should stage the channel and fields of each endpoint and keep its request",
		func(ctx SpecContext) {
			old := storedWriteConfig()
			cfg := MustSucceed(v5.MigrateWriteConfig(ctx, old))
			Expect(cfg.Device).To(Equal(old.Device))
			Expect(cfg.Endpoints).To(HaveLen(2))
			ep := cfg.Endpoints[0]
			Expect(ep.Method).To(Equal(v4.MethodPut))
			Expect(ep.Path).To(Equal("/setpoint"))
			Expect(ep.Headers).To(Equal(old.Endpoints[0].Headers))
			Expect(ep.ICD).To(BeNil())
			Expect(ep.Message).To(BeNil())
			Expect(cfg.Endpoints[1].Disabled).To(BeTrue())
			Expect(cfg.Inline).To(Equal(new(stagedWrite(old))))
		},
	)

	It("Should stage nothing for a config with no endpoints", func(ctx SpecContext) {
		cfg := MustSucceed(v5.MigrateWriteConfig(ctx, v4.WriteConfig{Key: uuid.New()}))
		Expect(cfg.Inline).To(BeNil())
	})
})

var _ = Describe("MigrateWriteEndpoint", func() {
	It("Should keep the request of an endpoint, with no ICD",
		func(ctx SpecContext) {
			old := storedWriteConfig().Endpoints[0]
			ep := MustSucceed(v5.MigrateWriteEndpoint(ctx, old))
			Expect(ep.Key).To(Equal("sp"))
			Expect(ep.Method).To(Equal(v4.MethodPut))
			Expect(ep.Path).To(Equal("/setpoint"))
			Expect(ep.Headers).To(Equal(old.Headers))
			Expect(ep.Disabled).To(BeFalse())
			Expect(ep.ICD).To(BeNil())
			Expect(ep.Message).To(BeNil())
		},
	)

	It("Should keep a disabled endpoint disabled", func(ctx SpecContext) {
		ep := MustSucceed(
			v5.MigrateWriteEndpoint(ctx, storedWriteConfig().Endpoints[1]),
		)
		Expect(ep.Disabled).To(BeTrue())
	})
})

var _ = Describe("StagedWriteFieldsKVKey", func() {
	It("Should key the staged channel and fields of a config under the prefix",
		func() {
			key := uuid.New()
			Expect(string(v5.StagedWriteFieldsKVKey(key))).
				To(Equal(v5.StagedWriteFieldsKVPrefix + key.String()))
		},
	)
})

var _ = Describe("WriteMigration", func() {
	var db *gorp.DB
	BeforeEach(func() { db = DeferClose(gorp.Wrap(memkv.New())) })

	run := func(ctx SpecContext) {
		GinkgoHelper()
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "http_write_config",
			Migrations: []gorp.Migration{v5.WriteMigration},
		})).To(Succeed())
	}
	migrate := func(ctx SpecContext, olds ...v4.WriteConfig) {
		GinkgoHelper()
		Expect(gorp.WrapWriter[uuid.UUID, v4.WriteConfig](db).Set(ctx, olds...)).
			To(Succeed())
		run(ctx)
	}
	staged := func(key uuid.UUID) v5.InlineWrite {
		GinkgoHelper()
		b, closer := MustSucceed2(db.Get(v5.StagedWriteFieldsKVKey(key)))
		var staged v5.InlineWrite
		Expect(json.Unmarshal(b, &staged)).To(Succeed())
		Expect(closer.Close()).To(Succeed())
		return staged
	}
	read := func(ctx SpecContext, key uuid.UUID) v5.WriteConfig {
		GinkgoHelper()
		var cfg v5.WriteConfig
		Expect(gorp.NewRetrieve[uuid.UUID, v5.WriteConfig]().
			Where(gorp.MatchKeys[uuid.UUID, v5.WriteConfig](key)).
			Entry(&cfg).
			Exec(ctx, db)).To(Succeed())
		return cfg
	}

	It("Should store the shape that v4 stored under the same table name", func() {
		Expect(v5.WriteConfig{}.CustomTypeName()).
			To(Equal(v4.WriteConfig{}.CustomTypeName()))
	})

	It("Should stage the channel and fields of each config under its own KV key",
		func(ctx SpecContext) {
			old := storedWriteConfig()
			migrate(ctx, old)
			migrated := read(ctx, old.Key)
			Expect(migrated.Inline).To(BeNil())
			Expect(migrated.Endpoints).To(HaveLen(2))
			Expect(migrated.Endpoints[0].Path).To(Equal("/setpoint"))
			Expect(staged(old.Key)).To(Equal(stagedWrite(old)))
		},
	)

	It("Should stage the channel and fields of a config in the shape 0.61 stored",
		func(ctx SpecContext) {
			old := v2.WriteConfig{Key: uuid.New(), Endpoints: []v2.WriteEndpoint{{
				Key:  "mode",
				Path: "/mode",
				Channel: v2.ChannelField{
					Pointer:    "/mode",
					JSONType:   v2.JSONTypeString,
					Channel:    42,
					DataType:   "uint8",
					TimeFormat: new(v2.TimeFormatUnixMs),
					EnumValues: []v2.EnumEntry{{Label: "auto", Value: 1}},
				},
				Fields: []v2.WriteField{{Variant: v2.GeneratedWriteField{
					BaseWriteField: v2.BaseWriteField{Key: "at", Pointer: "/at"},
					Generator:      v2.GeneratorTypeTimestamp,
					TimeFormat:     new(v2.TimeFormatUnixMs),
				}}},
			}}}
			Expect(gorp.WrapWriter[uuid.UUID, v2.WriteConfig](db).Set(ctx, old)).
				To(Succeed())
			run(ctx)
			Expect(read(ctx, old.Key).Endpoints[0].Path).To(Equal("/mode"))
			Expect(staged(old.Key)).To(Equal(v5.InlineWrite{
				Endpoints: []v5.InlineWriteEndpoint{{
					Key: "mode",
					Channel: v4.ChannelField{
						Pointer:    "/mode",
						JSONType:   payload.JSONTypeString,
						Channel:    42,
						DataType:   "uint8",
						TimeFormat: new(payload.TimeFormatUnixMs),
						EnumValues: []payload.EnumEntry{{Label: "auto", Value: 1}},
					},
					Fields: []v4.WriteField{{Variant: v4.GeneratedWriteField{
						BaseWriteField: v4.BaseWriteField{Key: "at", Pointer: "/at"},
						Generator:      payload.GeneratorTypeTimestamp,
						TimeFormat:     new(payload.TimeFormatUnixMs),
					}}},
				}},
			}))
		},
	)

	It("Should stage nothing for a config without endpoints", func(ctx SpecContext) {
		old := v4.WriteConfig{Key: uuid.New(), Device: "dev"}
		migrate(ctx, old)
		Expect(read(ctx, old.Key).Device).To(Equal("dev"))
		Expect(db.Get(v5.StagedWriteFieldsKVKey(old.Key))).Error().
			To(MatchError(query.ErrNotFound))
	})
})
