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
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	v4 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v4"
	v5 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v5"
	payload "github.com/synnaxlabs/x/payload/versions/v0"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

func writeConfig(fields ...v5.ChannelField) v5.WriteConfig {
	cfg := v5.WriteConfig{Device: "dev"}
	for i, f := range fields {
		cfg.Endpoints = append(cfg.Endpoints, v5.WriteEndpoint{
			Key:     string(rune('a' + i)),
			Path:    "/set",
			Channel: f,
		})
	}
	return cfg
}

func noWarn(string, ...zap.Field) {}

var _ = Describe("Channel field", func() {
	stored := func(c v4.ChannelField) v4.ChannelField {
		c.Pointer, c.Name = "/v", "cmd"
		return c
	}
	base := func(ch channel.Key, dt telem.DataType) v5.BaseChannelField {
		return v5.BaseChannelField{Pointer: "/v", Channel: ch, Name: "cmd", DataType: &dt}
	}
	unixMs := payload.TimeFormatUnixMs
	onOff := []payload.EnumEntry{{Label: "OFF", Value: 0}, {Label: "ON", Value: 1}}

	DescribeTable("MigrateChannelField",
		func(old v4.ChannelField, expected v5.ChannelFieldVariant) {
			Expect(v5.MigrateChannelField(stored(old))).
				To(Equal(v5.ChannelField{Variant: expected}))
		},
		Entry("a number as a float",
			v4.ChannelField{
				JSONType: payload.JSONTypeNumber,
				Channel:  7,
				DataType: telem.Float32T,
			},
			v5.FloatChannelField{
				BaseChannelField: base(7, telem.Float32T),
				EnumValues:       []payload.FloatEntry{},
			},
		),
		Entry("an empty JSON type and data type as a float64 number",
			v4.ChannelField{Channel: 7},
			v5.FloatChannelField{
				BaseChannelField: base(7, telem.Float64T),
				EnumValues:       []payload.FloatEntry{},
			},
		),
		Entry("a boolean as a boolean",
			v4.ChannelField{
				JSONType: payload.JSONTypeBoolean,
				Channel:  7,
				DataType: telem.Uint8T,
			},
			v5.BooleanChannelField{BaseChannelField: base(7, telem.Uint8T)},
		),
		Entry("a string with entries as a string with decimal text entries",
			v4.ChannelField{
				JSONType:   payload.JSONTypeString,
				Channel:    7,
				DataType:   telem.Int32T,
				EnumValues: onOff,
			},
			v5.StringChannelField{
				BaseChannelField: base(7, telem.Int32T),
				EnumValues: []payload.StringEntry{
					{Label: "OFF", Value: "0"},
					{Label: "ON", Value: "1"},
				},
			},
		),
		Entry("a -0 entry as the text 0",
			v4.ChannelField{
				JSONType:   payload.JSONTypeString,
				Channel:    7,
				EnumValues: []payload.EnumEntry{{Label: "Z", Value: -0.0}},
			},
			v5.StringChannelField{
				BaseChannelField: base(7, telem.Float64T),
				EnumValues:       []payload.StringEntry{{Label: "Z", Value: "0"}},
			},
		),
		Entry("a repeated entry value by dropping the later entry",
			v4.ChannelField{
				JSONType: payload.JSONTypeString,
				Channel:  7,
				EnumValues: []payload.EnumEntry{
					{Label: "ON", Value: 1},
					{Label: "HIGH", Value: 1},
				},
			},
			v5.StringChannelField{
				BaseChannelField: base(7, telem.Float64T),
				EnumValues:       []payload.StringEntry{{Label: "ON", Value: "1"}},
			},
		),
		Entry("a timestamp data type as a timestamp in ISO 8601",
			v4.ChannelField{
				JSONType: payload.JSONTypeNumber,
				Channel:  7,
				DataType: telem.TimestampT,
			},
			v5.TimestampChannelField{
				BaseChannelField: base(7, telem.Float64T),
				TimeFormat:       payload.TimeFormatISO8601,
			},
		),
		Entry("a string with a time format as a timestamp in that format",
			v4.ChannelField{
				JSONType:   payload.JSONTypeString,
				Channel:    7,
				TimeFormat: &unixMs,
			},
			v5.TimestampChannelField{
				BaseChannelField: base(7, telem.StringT),
				TimeFormat:       payload.TimeFormatUnixMs,
			},
		),
		Entry("a timestamp with string entries as a string",
			v4.ChannelField{
				JSONType:   payload.JSONTypeString,
				Channel:    7,
				DataType:   telem.TimestampT,
				TimeFormat: &unixMs,
				EnumValues: onOff,
			},
			v5.StringChannelField{
				BaseChannelField: base(7, telem.TimestampT),
				EnumValues: []payload.StringEntry{
					{Label: "OFF", Value: "0"},
					{Label: "ON", Value: "1"},
				},
			},
		),
		Entry("an unbound timestamp data type as a timestamp",
			v4.ChannelField{JSONType: payload.JSONTypeString, DataType: telem.TimestampT},
			v5.TimestampChannelField{
				BaseChannelField: base(0, telem.TimestampT),
				TimeFormat:       payload.TimeFormatISO8601,
			},
		),
	)

	Describe("ReconcileWriteConfig", func() {
		reconcile := func(
			ctx SpecContext,
			f v5.ChannelField,
			found map[channel.Key]v5.Channel,
		) v5.ChannelField {
			cfg := writeConfig(f)
			Expect(v5.ReconcileWriteConfig(
				ctx, nil, storedChannels(found), &cfg, noWarn,
			)).To(Succeed())
			return cfg.Endpoints[0].Channel
		}

		It("Should keep a string field on a numeric channel and clear its data type", func(
			ctx SpecContext,
		) {
			f := v5.MigrateChannelField(stored(v4.ChannelField{
				JSONType:   payload.JSONTypeString,
				Channel:    7,
				EnumValues: onOff,
			}))
			Expect(reconcile(ctx, f, map[channel.Key]v5.Channel{
				7: {DataType: telem.Uint8T},
			}).Variant).To(Equal(v5.StringChannelField{
				BaseChannelField: v5.BaseChannelField{
					Pointer: "/v", Channel: 7, Name: "cmd",
				},
				EnumValues: []payload.StringEntry{
					{Label: "OFF", Value: "0"},
					{Label: "ON", Value: "1"},
				},
			}))
		})

		It("Should make a field on a JSON channel write the raw value", func(
			ctx SpecContext,
		) {
			f := v5.MigrateChannelField(stored(v4.ChannelField{Channel: 7}))
			Expect(reconcile(ctx, f, map[channel.Key]v5.Channel{
				7: {DataType: telem.JSONT},
			}).Variant).To(Equal(v5.JsonChannelField{
				BaseChannelField: v5.BaseChannelField{
					Pointer: "/v", Channel: 7, Name: "cmd",
				},
			}))
		})

		It("Should give a string field with entries on a timestamp channel ISO 8601", func(
			ctx SpecContext,
		) {
			f := v5.MigrateChannelField(stored(v4.ChannelField{
				JSONType:   payload.JSONTypeString,
				Channel:    7,
				DataType:   telem.TimestampT,
				TimeFormat: &unixMs,
				EnumValues: onOff,
			}))
			Expect(reconcile(ctx, f, map[channel.Key]v5.Channel{
				7: {DataType: telem.TimestampT},
			}).Variant).To(Equal(v5.TimestampChannelField{
				BaseChannelField: v5.BaseChannelField{
					Pointer: "/v", Channel: 7, Name: "cmd",
				},
				TimeFormat: payload.TimeFormatISO8601,
			}))
		})

		It("Should fall back from a timestamp to the kind of its JSON type", func(
			ctx SpecContext,
		) {
			f := v5.MigrateChannelField(stored(v4.ChannelField{
				JSONType:   payload.JSONTypeBoolean,
				Channel:    7,
				TimeFormat: &unixMs,
			}))
			Expect(reconcile(ctx, f, map[channel.Key]v5.Channel{
				7: {DataType: telem.Uint8T},
			}).Type()).To(Equal(payload.KindBoolean))
		})

		It("Should keep the data type of a field with no channel", func(
			ctx SpecContext,
		) {
			f := v5.MigrateChannelField(stored(v4.ChannelField{
				JSONType: payload.JSONTypeNumber,
				DataType: telem.Int16T,
			}))
			got := reconcile(ctx, f, nil)
			Expect(got.Type()).To(Equal(payload.KindInteger))
			Expect(got.Base().DataType).To(Equal(new(telem.Int16T)))
		})

		It("Should skip a field whose channel does not exist", func(ctx SpecContext) {
			f := v5.MigrateChannelField(stored(v4.ChannelField{Channel: 7}))
			Expect(reconcile(ctx, f, nil)).To(Equal(f))
		})
	})

	Describe("ResolveWriteConfig", func() {
		resolve := func(ctx SpecContext, f v5.ChannelField, dt telem.DataType) (
			v5.WriteConfig,
			error,
		) {
			cfg := writeConfig(f)
			err := v5.ResolveWriteConfig(
				ctx,
				nil,
				storedChannels(map[channel.Key]v5.Channel{7: {DataType: dt}}),
				&cfg,
			)
			return cfg, err
		}

		It("Should clear the data type of a field whose channel allows its kind", func(
			ctx SpecContext,
		) {
			cfg := MustSucceed(resolve(ctx, v5.ChannelField{
				Variant: v5.BooleanChannelField{BaseChannelField: base(7, telem.Uint8T)},
			}, telem.Float32T))
			Expect(cfg.Endpoints[0].Channel.Base().DataType).To(BeNil())
		})

		It("Should reject a kind the channel does not allow", func(ctx SpecContext) {
			_, err := resolve(ctx, v5.ChannelField{
				Variant: v5.JsonChannelField{BaseChannelField: base(7, telem.JSONT)},
			}, telem.Float32T)
			Expect(err).To(MatchError(validate.ErrValidation))
		})

		It("Should reject a string entry that is not a value of the channel", func(
			ctx SpecContext,
		) {
			_, err := resolve(ctx, v5.ChannelField{
				Variant: v5.StringChannelField{
					BaseChannelField: base(7, telem.Uint8T),
					EnumValues:       []payload.StringEntry{{Label: "X", Value: "-1"}},
				},
			}, telem.Uint8T)
			Expect(err).To(MatchError(validate.ErrValidation))
		})
	})

	Describe("ValidateWriteConfig", func() {
		It("Should reject two entries with one value", func() {
			cfg := writeConfig(v5.ChannelField{Variant: v5.FloatChannelField{
				BaseChannelField: base(7, telem.Float64T),
				EnumValues: []payload.FloatEntry{
					{Label: "A", Value: 0},
					{Label: "B", Value: payload.Float(-0.0)},
				},
			}})
			Expect(v5.ValidateWriteConfig(&cfg)).To(MatchError(validate.ErrValidation))
		})

		It("Should reject a channel field with no kind", func() {
			cfg := writeConfig(v5.ChannelField{})
			Expect(v5.ValidateWriteConfig(&cfg)).To(MatchError(validate.ErrValidation))
		})
	})

	It("Should rewrite a stored v4 write config with a time format in place", func(
		ctx SpecContext,
	) {
		old := v4.WriteConfig{
			Key:    uuid.New(),
			Device: "dev",
			Endpoints: []v4.WriteEndpoint{{
				Key: "ep",
				Channel: v4.ChannelField{
					Pointer:    "/t",
					JSONType:   payload.JSONTypeNumber,
					Channel:    7,
					DataType:   telem.TimestampT,
					TimeFormat: &unixMs,
				},
			}},
		}
		db := DeferClose(gorp.Wrap(memkv.New()))
		MustSucceed(
			gorp.OpenTable(ctx, gorp.TableConfig[uuid.UUID, v4.WriteConfig]{DB: db}),
		)
		Expect(
			gorp.NewCreate[uuid.UUID, v4.WriteConfig]().Entry(&old).Exec(ctx, db),
		).To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:        db,
			Namespace: "http_write_config",
			Migrations: []gorp.Migration{v5.NewWriteMigration(storedChannels(
				map[channel.Key]v5.Channel{7: {DataType: telem.TimestampT}},
			))},
		})).To(Succeed())
		var got v5.WriteConfig
		Expect(gorp.NewRetrieve[uuid.UUID, v5.WriteConfig]().
			Where(gorp.MatchKeys[uuid.UUID, v5.WriteConfig](old.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Endpoints[0].Channel.Variant).To(Equal(v5.TimestampChannelField{
			BaseChannelField: v5.BaseChannelField{Pointer: "/t", Channel: 7},
			TimeFormat:       payload.TimeFormatUnixMs,
		}))
	})
})
