// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3_test

import (
	"context"
	"encoding/json/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	icdv0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	icd "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v1"
	v1 "github.com/synnaxlabs/synnax/pkg/service/mqtt/versions/v1"
	v3 "github.com/synnaxlabs/synnax/pkg/service/mqtt/versions/v3"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func observe() (alamos.Instrumentation, *observer.ObservedLogs) {
	GinkgoHelper()
	core, logs := observer.New(zapcore.WarnLevel)
	logger := MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
		ZapLogger: zap.New(core),
	}))
	return alamos.New("test", alamos.WithLogger(logger)), logs
}

// variantOf returns the variant of f, which must be a T.
func variantOf[T icd.JSONFieldVariant](f icd.JSONField) T {
	GinkgoHelper()
	v, ok := f.Variant.(T)
	return MustBeOk(v, ok)
}

// fields returns the fields of the JSON message msg.
func fields(msg icd.Message) []icd.JSONField {
	GinkgoHelper()
	p, ok := msg.Payload.Variant.(icd.JSONPayload)
	Expect(ok).To(BeTrue())
	return p.Fields
}

// plainTarget returns a plain v1 target on topic that sends ch and fields.
func plainTarget(
	topic string,
	ch v1.ChannelField,
	fields ...v1.WriteField,
) v1.WriteTarget {
	return v1.WriteTarget{Variant: v1.PlainWriteTarget{
		BaseWriteTarget: v1.BaseWriteTarget{Key: topic},
		Topic:           topic,
		Qos:             v1.QoSAtLeastOnce,
		Channel:         ch,
		Fields:          fields,
	}}
}

// valueField returns a channel field at /value that sends channel 7 as a number.
func valueField() v1.ChannelField {
	return v1.ChannelField{
		Pointer:  "/value",
		JSONType: payload.JSONTypeNumber,
		Channel:  7,
		Name:     "valve_cmd",
		DataType: telem.Float64T,
	}
}

func staticField(pointer string, t payload.JSONType, value any) v1.WriteField {
	return v1.WriteField{Variant: v1.StaticWriteField{
		BaseWriteField: v1.BaseWriteField{Key: pointer, Pointer: pointer},
		JSONType:       t,
		Value:          value,
	}}
}

func generatedField(
	pointer string,
	g payload.GeneratorType,
	format *payload.TimeFormat,
) v1.WriteField {
	return v1.WriteField{Variant: v1.GeneratedWriteField{
		BaseWriteField: v1.BaseWriteField{Key: pointer, Pointer: pointer},
		Generator:      g,
		TimeFormat:     format,
	}}
}

// plainWrite returns the plain v3 target of t.
func plainWrite(t v3.WriteTarget) v3.PlainWriteTarget {
	GinkgoHelper()
	plain, ok := t.Variant.(v3.PlainWriteTarget)
	Expect(ok).To(BeTrue())
	return plain
}

var _ = Describe("MigrateWriteConfig", func() {
	It("Should stage each plain target with its index and keep the others",
		func(ctx SpecContext) {
			sparkplug := v1.WriteTarget{Variant: v1.SparkplugWriteTarget{
				BaseWriteTarget: v1.BaseWriteTarget{Key: "t1"},
				Tag:             "speed",
				TagType:         v1.SparkplugDataTypeDouble,
			}}
			cfg := MustSucceed(v3.MigrateWriteConfig(ctx, v1.WriteConfig{
				Targets: []v1.WriteTarget{
					plainTarget("plant/cmd", valueField()),
					sparkplug,
				},
			}))
			Expect(cfg.Targets).To(HaveLen(1))
			Expect(cfg.Targets[0].Variant).To(Equal(v3.SparkplugWriteTarget{
				Key:     "t1",
				Tag:     "speed",
				TagType: v3.SparkplugDataTypeDouble,
			}))
			Expect(cfg.Inline).ToNot(BeNil())
			Expect(cfg.Inline.Targets).To(HaveExactElements(v3.InlineTarget{
				Key:     "plant/cmd",
				Topic:   "plant/cmd",
				Qos:     v3.QoSAtLeastOnce,
				Channel: valueField(),
			}))
		},
	)

	It("Should stage nothing without plain targets", func(ctx SpecContext) {
		cfg := MustSucceed(v3.MigrateWriteConfig(ctx, v1.WriteConfig{}))
		Expect(cfg.Inline).To(BeNil())
	})
})

var _ = Describe("MigrateWriteTarget", func() {
	It("Should keep the topic settings of a plain target and give it no message",
		func(ctx SpecContext) {
			old := plainTarget("plant/cmd", valueField())
			Expect(v3.MigrateWriteTarget(ctx, old)).To(Equal(v3.WriteTarget{
				Variant: v3.PlainWriteTarget{
					BaseWriteTarget: v3.BaseWriteTarget{Key: "plant/cmd"},
					Topic:           "plant/cmd",
					Qos:             v3.QoSAtLeastOnce,
				},
			}))
		},
	)

	It("Should keep a Sparkplug target", func(ctx SpecContext) {
		old := v1.WriteTarget{Variant: v1.SparkplugWriteTarget{
			BaseWriteTarget: v1.BaseWriteTarget{Key: "t1"},
			Tag:             "speed",
			TagType:         v1.SparkplugDataTypeDouble,
			Channel:         4,
		}}
		Expect(v3.MigrateWriteTarget(ctx, old)).To(Equal(v3.WriteTarget{
			Variant: v3.SparkplugWriteTarget{
				Key:     "t1",
				Tag:     "speed",
				TagType: v3.SparkplugDataTypeDouble,
				Channel: 4,
			},
		}))
	})
})

var _ = Describe("ResolveWriteConfig", func() {
	var (
		db       *gorp.DB
		channels icdv0.Channels
	)
	BeforeEach(func() {
		db = DeferClose(gorp.Wrap(memkv.New()))
		channels = func(
			context.Context,
			gorp.Snapshot,
			[]channel.Key,
		) (map[channel.Key]icdv0.Channel, error) {
			return map[channel.Key]icdv0.Channel{
				7: {Name: "valve_cmd", DataType: telem.Float64T},
			}, nil
		}
	})
	ignore := func(string, string) {}
	tag := func(key string) v3.WriteTarget {
		return v3.WriteTarget{Variant: v3.SparkplugWriteTarget{Key: key, Tag: key}}
	}
	staged := func(at uint32, topic string) v3.InlineTarget {
		return v3.InlineTarget{Key: topic, At: at, Topic: topic, Channel: valueField()}
	}
	config := func(targets ...v1.WriteTarget) v1.WriteConfig {
		return v1.WriteConfig{Key: uuid.New(), Targets: targets}
	}
	// lift stages the plain targets of old as MigrateWriteConfig does and resolves them
	// into an ICD named Valves, noting each change through ins.
	lift := func(
		ctx SpecContext,
		ins alamos.Instrumentation,
		old v1.WriteConfig,
	) (icd.ICD, v3.WriteConfig) {
		GinkgoHelper()
		cfg := MustSucceed(v3.MigrateWriteConfig(ctx, old))
		ic, ok := MustSucceed2(v3.ResolveWriteConfig(
			ctx,
			db,
			channels,
			"Valves",
			&cfg,
			v3.NoteLift(ins, cfg.Key),
		))
		Expect(ok).To(BeTrue())
		return ic, cfg
	}

	It("Should move each plain target into a message of an ICD with the given name",
		func(ctx SpecContext) {
			target := plainTarget("plant/valve/set", valueField())
			plain, ok := target.Variant.(v1.PlainWriteTarget)
			Expect(ok).To(BeTrue())
			plain.Retained, plain.Disabled = true, true
			ins, _ := observe()
			ic, cfg := lift(ctx, ins, config(v1.WriteTarget{Variant: plain}))
			Expect(ic.Name).To(Equal("Valves"))
			Expect(ic.Messages).To(HaveLen(1))
			msg := ic.Messages[0]
			Expect(msg.Name).To(Equal("plant/valve/set"))
			f := fields(msg)
			Expect(f).To(HaveLen(1))
			Expect(f[0].Type()).To(Equal(payload.KindFloat))
			Expect(f[0].Base().Name).To(Equal("value"))
			Expect(f[0].Base().Pointer).To(Equal("/value"))
			Expect(f[0].Base().Scale).To(Equal(1.0))
			migrated := plainWrite(cfg.Targets[0])
			Expect(migrated.BaseWriteTarget).To(Equal(v3.BaseWriteTarget{
				Key:      "plant/valve/set",
				Disabled: true,
			}))
			Expect(migrated.Topic).To(Equal("plant/valve/set"))
			Expect(migrated.Qos).To(Equal(v3.QoSAtLeastOnce))
			Expect(migrated.Retained).To(BeTrue())
			Expect(migrated.ICD).To(Equal(ic.Key))
			Expect(migrated.Message).To(Equal(icd.WriteMessage{
				Message: msg.Key,
				Bindings: []icd.WriteBinding{{Variant: icd.ChannelWriteBinding{
					Field:   f[0].Base().Key,
					Channel: 7,
				}}},
			}))
		},
	)

	It("Should put the plain targets of one config into one ICD with unique names",
		func(ctx SpecContext) {
			ins, _ := observe()
			ic, cfg := lift(ctx, ins, config(
				plainTarget("plant/cmd", valueField()),
				plainTarget("plant/cmd", valueField()),
				plainTarget("", valueField()),
			))
			names := make([]string, len(ic.Messages))
			for i, m := range ic.Messages {
				names[i] = m.Name
			}
			Expect(names).To(Equal([]string{"plant/cmd", "plant/cmd (2)", "Untitled"}))
			for i, t := range cfg.Targets {
				Expect(plainWrite(t).Message.Message).To(Equal(ic.Messages[i].Key))
			}
		},
	)

	It("Should bind no channel to a target with no channel and log its dropped name",
		func(ctx SpecContext) {
			ch := valueField()
			ch.Channel = 0
			old := config(plainTarget("plant/valve/set", ch))
			ins, logs := observe()
			ic, cfg := lift(ctx, ins, old)
			f := fields(ic.Messages[0])
			Expect(plainWrite(cfg.Targets[0]).Message.Bindings).
				To(Equal([]icd.WriteBinding{{
					Variant: icd.ChannelWriteBinding{
						Field:    f[0].Base().Key,
						DataType: new(telem.Float64T),
					},
				}}))
			const change = `channel name "valve_cmd" of field value on topic ` +
				`plant/valve/set was dropped, so a client names the new channel`
			logged := logs.FilterField(zap.String("field", "targets.0.channel")).All()
			Expect(logged).To(HaveLen(1))
			Expect(logged[0].ContextMap()).To(SatisfyAll(
				HaveKeyWithValue("config", old.Key.String()),
				HaveKeyWithValue("change", change),
			))
		},
	)

	It("Should lift the static and generated fields of a plain target",
		func(ctx SpecContext) {
			format := payload.TimeFormatUnixNs
			ins, _ := observe()
			ic, cfg := lift(ctx, ins, config(plainTarget(
				"plant/valve/set",
				valueField(),
				staticField("/mode", payload.JSONTypeString, "auto"),
				generatedField("/id", payload.GeneratorTypeUUID, nil),
				generatedField("/time", payload.GeneratorTypeTimestamp, &format),
			)))
			f := fields(ic.Messages[0])
			Expect(f).To(HaveLen(4))
			Expect(*variantOf[icd.StringJSONField](f[1]).Default).To(Equal("auto"))
			Expect(f[2].Variant).To(Equal(icd.UuidJSONField{
				JSONBaseField: f[2].Base(),
				Default:       &icd.UUIDDefault{Variant: icd.RandomUUIDDefault{}},
			}))
			Expect(variantOf[icd.TimestampJSONField](f[3]).TimeFormat).
				To(Equal(payload.TimeFormatUnixNs))
			Expect(plainWrite(cfg.Targets[0]).Message.Bindings).To(HaveLen(1))
		},
	)

	It("Should return false and keep a config with no inline targets",
		func(ctx SpecContext) {
			cfg := v3.WriteConfig{Targets: []v3.WriteTarget{tag("speed")}}
			_, ok := MustSucceed2(
				v3.ResolveWriteConfig(ctx, db, channels, "Valves", &cfg, ignore),
			)
			Expect(ok).To(BeFalse())
			Expect(cfg.Targets).To(Equal([]v3.WriteTarget{tag("speed")}))
		},
	)

	It("Should insert each inline target at its index and clear the inline targets",
		func(ctx SpecContext) {
			cfg := v3.WriteConfig{
				Targets: []v3.WriteTarget{tag("a"), tag("b")},
				Inline: &v3.InlineWrite{Targets: []v3.InlineTarget{
					staged(1, "plant/one"),
					staged(3, "plant/three"),
				}},
			}
			ic, ok := MustSucceed2(
				v3.ResolveWriteConfig(ctx, db, channels, "Valves", &cfg, ignore),
			)
			Expect(ok).To(BeTrue())
			Expect(cfg.Inline).To(BeNil())
			Expect(ic.Name).To(Equal("Valves"))
			Expect(ic.Messages).To(HaveLen(2))
			Expect(cfg.Targets).To(HaveLen(4))
			Expect(cfg.Targets[0]).To(Equal(tag("a")))
			Expect(cfg.Targets[2]).To(Equal(tag("b")))
			for i, at := range []int{1, 3} {
				t := plainWrite(cfg.Targets[at])
				Expect(t.ICD).To(Equal(ic.Key))
				Expect(t.Message.Message).To(Equal(ic.Messages[i].Key))
				Expect(ic.Messages[i].Name).To(Equal(t.Topic))
				Expect(t.Message.Bindings).To(HaveLen(1))
				Expect(t.Message.Bindings[0].Variant).
					To(HaveField("Channel", BeEquivalentTo(7)))
			}
		},
	)

	It("Should note each change at the path of its field in the config",
		func(ctx SpecContext) {
			weekday := payload.TimeFormat("weekday")
			target := staged(2, "plant/one")
			target.Channel.Channel = 9
			target.Fields = []v1.WriteField{
				{},
				generatedField("/time", payload.GeneratorTypeTimestamp, &weekday),
			}
			cfg := v3.WriteConfig{
				Targets: []v3.WriteTarget{tag("a"), tag("b")},
				Inline:  &v3.InlineWrite{Targets: []v3.InlineTarget{target}},
			}
			notes := make(map[string]string)
			MustSucceed2(v3.ResolveWriteConfig(
				ctx,
				db,
				channels,
				"Valves",
				&cfg,
				func(path, change string) { notes[path] = change },
			))
			Expect(notes).To(Equal(map[string]string{
				"targets.2.channel": `channel name "valve_cmd" of field value on ` +
					"topic plant/one was dropped, so a client names the new channel",
				"targets.2.fields.0": "a null field was dropped",
				"targets.2.fields.1": `time format "weekday" is not valid and became ` +
					"iso8601",
			}))
		},
	)

	It("Should append an inline target whose index is past the last target",
		func(ctx SpecContext) {
			cfg := v3.WriteConfig{
				Targets: []v3.WriteTarget{tag("a")},
				Inline: &v3.InlineWrite{
					Targets: []v3.InlineTarget{staged(5, "plant/one")},
				},
			}
			MustSucceed2(
				v3.ResolveWriteConfig(ctx, db, channels, "Valves", &cfg, ignore),
			)
			Expect(cfg.Targets).To(HaveLen(2))
			Expect(plainWrite(cfg.Targets[1]).Topic).To(Equal("plant/one"))
		},
	)
})

var _ = Describe("NoteLift", func() {
	It("Should log each change with the config key and the path of its field", func() {
		ins, logs := observe()
		key := uuid.New()
		v3.NoteLift(ins, key)("targets.0.channel", "a change")
		Expect(logs.All()).To(HaveLen(1))
		logged := logs.All()[0]
		Expect(logged.Message).To(Equal("changed an MQTT write field to v3"))
		Expect(logged.ContextMap()).To(Equal(map[string]any{
			"config": key.String(),
			"field":  "targets.0.channel",
			"change": "a change",
		}))
	})
})

var _ = Describe("StagedTargetsKVKey", func() {
	It("Should key staged targets by the prefix and the config key", func() {
		key := uuid.New()
		Expect(string(v3.StagedTargetsKVKey(key))).
			To(Equal("sy_mqtt_write_staged_targets/" + key.String()))
	})
})

var _ = Describe("WriteMigration", func() {
	var db *gorp.DB
	BeforeEach(func() { db = DeferClose(gorp.Wrap(memkv.New())) })

	migrate := func(ctx SpecContext, olds ...v1.WriteConfig) {
		GinkgoHelper()
		Expect(gorp.WrapWriter[uuid.UUID, v1.WriteConfig](db).Set(ctx, olds...)).
			To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "mqtt_write_config",
			Migrations: []gorp.Migration{v3.WriteMigration},
		})).To(Succeed())
	}
	read := func(ctx SpecContext, key uuid.UUID) v3.WriteConfig {
		GinkgoHelper()
		var cfg v3.WriteConfig
		Expect(gorp.NewRetrieve[uuid.UUID, v3.WriteConfig]().
			Where(gorp.MatchKeys[uuid.UUID, v3.WriteConfig](key)).
			Entry(&cfg).
			Exec(ctx, db)).To(Succeed())
		return cfg
	}
	sparkplug := v1.WriteTarget{Variant: v1.SparkplugWriteTarget{
		BaseWriteTarget: v1.BaseWriteTarget{Key: "t1"},
		Tag:             "speed",
	}}

	It("Should store the shape that v1 stored under the same table name", func() {
		Expect(v3.WriteConfig{}.CustomTypeName()).
			To(Equal(v1.WriteConfig{}.CustomTypeName()))
	})

	It("Should stage the plain targets of each config and store it without them",
		func(ctx SpecContext) {
			mode := staticField("/mode", payload.JSONTypeString, "auto")
			old := v1.WriteConfig{Key: uuid.New(), Targets: []v1.WriteTarget{
				plainTarget("plant/cmd", valueField(), mode),
				sparkplug,
			}}
			migrate(ctx, old)
			migrated := read(ctx, old.Key)
			Expect(migrated.Targets).To(HaveLen(1))
			Expect(migrated.Targets[0].Variant).
				To(BeAssignableToTypeOf(v3.SparkplugWriteTarget{}))
			Expect(migrated.Inline).To(BeNil())
			b, closer := MustSucceed2(db.Get(v3.StagedTargetsKVKey(old.Key)))
			var staged v3.InlineWrite
			Expect(json.Unmarshal(b, &staged)).To(Succeed())
			Expect(closer.Close()).To(Succeed())
			ch := valueField()
			ch.EnumValues = []payload.EnumEntry{}
			Expect(staged).To(Equal(v3.InlineWrite{Targets: []v3.InlineTarget{{
				Key:     "plant/cmd",
				Topic:   "plant/cmd",
				Qos:     v3.QoSAtLeastOnce,
				Channel: ch,
				Fields:  []v1.WriteField{mode},
			}}}))
		},
	)

	It("Should stage nothing for a config without plain targets",
		func(ctx SpecContext) {
			old := v1.WriteConfig{Key: uuid.New(), Targets: []v1.WriteTarget{sparkplug}}
			migrate(ctx, old)
			Expect(read(ctx, old.Key).Targets).To(HaveLen(1))
			Expect(db.Get(v3.StagedTargetsKVKey(old.Key))).Error().
				To(MatchError(query.ErrNotFound))
		},
	)
})
