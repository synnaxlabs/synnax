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
