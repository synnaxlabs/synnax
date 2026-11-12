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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v4 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v4"
	v5 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v5"
	payloadv0 "github.com/synnaxlabs/x/payload/versions/v0"
	payload "github.com/synnaxlabs/x/payload/versions/v1"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

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
			DB:         db,
			Namespace:  "http_write_config",
			Migrations: []gorp.Migration{v5.WriteMigration},
		})).To(Succeed())
		var got v5.WriteConfig
		Expect(gorp.NewRetrieve[uuid.UUID, v5.WriteConfig]().
			Where(gorp.MatchKeys[uuid.UUID, v5.WriteConfig](stored.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Device).To(Equal(stored.Device))
		Expect(got.Endpoints).To(HaveLen(1))
		ep := got.Endpoints[0]
		Expect(ep.Path).To(Equal("/set"))
		Expect(ep.Channel.Pointer).To(Equal("/v"))
		Expect(ep.Channel.JSONType).To(Equal(payloadv0.JSONTypeString))
		Expect(ep.Channel.Channel).To(BeEquivalentTo(7))
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
