// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2_test

import (
	"context"
	"fmt"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	icd "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/mqtt/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/mqtt/versions/v2"
	task "github.com/synnaxlabs/synnax/pkg/service/task/versions/v3"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

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

// plainWrite returns the plain v2 target of t.
func plainWrite(t v2.WriteTarget) v2.PlainWriteTarget {
	GinkgoHelper()
	plain, ok := t.Variant.(v2.PlainWriteTarget)
	Expect(ok).To(BeTrue())
	return plain
}

var _ = Describe("NewWriteMigration", func() {
	var (
		db       *gorp.DB
		made     []icd.ICD
		create   v2.CreateICD
		found    map[channel.Key]v2.Channel
		channels v2.Channels
	)
	BeforeEach(func() {
		db = DeferClose(gorp.Wrap(memkv.New()))
		made = nil
		create = func(_ context.Context, tx gorp.Tx, ic icd.ICD) error {
			GinkgoHelper()
			Expect(tx).ToNot(BeNil())
			made = append(made, ic)
			return nil
		}
		found = map[channel.Key]v2.Channel{
			7: {Name: "valve_cmd", DataType: telem.Float64T},
		}
		channels = func(
			context.Context,
			gorp.Snapshot,
			[]channel.Key,
		) (map[channel.Key]v2.Channel, error) {
			return found, nil
		}
	})

	// store writes old and, unless orphan, a task named name that owns it.
	store := func(ctx SpecContext, name string, old v1.WriteConfig, orphan bool) {
		GinkgoHelper()
		Expect(gorp.WrapWriter[uuid.UUID, v1.WriteConfig](db).Set(ctx, old)).
			To(Succeed())
		if !orphan {
			Expect(gorp.WrapWriter[task.Key, task.Task](db).Set(ctx, task.Task{
				Key:  old.Key,
				Name: name,
				Type: "mqtt_write",
			})).To(Succeed())
		}
	}
	migrate := func(ctx SpecContext, ins alamos.Instrumentation) error {
		return gorp.Migrate(ctx, gorp.MigrateConfig{
			Instrumentation: ins,
			DB:              db,
			Namespace:       "mqtt_write_config",
			Migrations:      []gorp.Migration{v2.NewWriteMigration(channels, create)},
		})
	}
	read := func(ctx SpecContext, key uuid.UUID) v2.WriteConfig {
		GinkgoHelper()
		var cfg v2.WriteConfig
		Expect(gorp.NewRetrieve[uuid.UUID, v2.WriteConfig]().
			Where(gorp.MatchKeys[uuid.UUID, v2.WriteConfig](key)).
			Entry(&cfg).
			Exec(ctx, db)).To(Succeed())
		return cfg
	}
	config := func(targets ...v1.WriteTarget) v1.WriteConfig {
		return v1.WriteConfig{Key: uuid.New(), Targets: targets}
	}
	// lift migrates one config whose only target sends ch and fields, and returns
	// the fields of its message and the message of the target.
	lift := func(
		ctx SpecContext,
		ch v1.ChannelField,
		extra ...v1.WriteField,
	) ([]icd.JSONField, v2.WriteMessage) {
		GinkgoHelper()
		old := config(plainTarget("plant/valve/set", ch, extra...))
		store(ctx, "Valves", old, false)
		ins, _ := observe()
		Expect(migrate(ctx, ins)).To(Succeed())
		Expect(made).To(HaveLen(1))
		return fields(
				made[0].Messages[0],
			), plainWrite(
				read(ctx, old.Key).Targets[0],
			).Message
	}

	It("Should store the shape that v1 stored under the same table name", func() {
		Expect(v2.WriteConfig{}.CustomTypeName()).
			To(Equal(v1.WriteConfig{}.CustomTypeName()))
	})

	It("Should move each plain target into a message of an ICD named after its task",
		func(ctx SpecContext) {
			target := plainTarget("plant/valve/set", valueField())
			plain := target.Variant.(v1.PlainWriteTarget)
			plain.Retained, plain.Disabled = true, true
			old := config(v1.WriteTarget{Variant: plain})
			store(ctx, "Valves", old, false)
			ins, _ := observe()
			Expect(migrate(ctx, ins)).To(Succeed())
			Expect(made).To(HaveLen(1))
			Expect(made[0].Name).To(Equal("Valves"))
			Expect(made[0].Messages).To(HaveLen(1))
			msg := made[0].Messages[0]
			Expect(msg.Name).To(Equal("plant/valve/set"))
			f := fields(msg)
			Expect(f).To(HaveLen(1))
			Expect(f[0].Type()).To(Equal(payload.KindFloat))
			Expect(f[0].Base().Name).To(Equal("value"))
			Expect(f[0].Base().Pointer).To(Equal("/value"))
			Expect(f[0].Base().Scale).To(Equal(1.0))
			migrated := plainWrite(read(ctx, old.Key).Targets[0])
			Expect(migrated.BaseWriteTarget).To(Equal(v2.BaseWriteTarget{
				Key:      "plant/valve/set",
				Disabled: true,
			}))
			Expect(migrated.Topic).To(Equal("plant/valve/set"))
			Expect(migrated.Qos).To(Equal(v2.QoSAtLeastOnce))
			Expect(migrated.Retained).To(BeTrue())
			Expect(migrated.ICD).To(Equal(made[0].Key))
			Expect(migrated.Message).To(Equal(v2.WriteMessage{
				Message:  msg.Key,
				Bindings: []v2.WriteBinding{{Field: f[0].Base().Key, Channel: 7}},
			}))
		},
	)

	It("Should put the plain targets of one config into one ICD with unique names",
		func(ctx SpecContext) {
			old := config(
				plainTarget("plant/cmd", valueField()),
				plainTarget("plant/cmd", valueField()),
				plainTarget("", valueField()),
			)
			store(ctx, "Plant", old, false)
			ins, _ := observe()
			Expect(migrate(ctx, ins)).To(Succeed())
			Expect(made).To(HaveLen(1))
			names := make([]string, len(made[0].Messages))
			for i, m := range made[0].Messages {
				names[i] = m.Name
			}
			Expect(names).To(Equal([]string{"plant/cmd", "plant/cmd (2)", "Untitled"}))
			for i, t := range read(ctx, old.Key).Targets {
				Expect(plainWrite(t).Message.Message).To(Equal(made[0].Messages[i].Key))
			}
		},
	)

	It("Should bind no channel and keep the stored data type when the channel is gone",
		func(ctx SpecContext) {
			ch := valueField()
			ch.Channel, ch.DataType = 12, telem.Int16T
			f, msg := lift(ctx, ch)
			Expect(f[0].Type()).To(Equal(payload.KindInteger))
			Expect(msg.Bindings).To(Equal([]v2.WriteBinding{{
				Field:    f[0].Base().Key,
				DataType: telem.Int16T,
			}}))
		},
	)

	It("Should take float64 for a channel field with no channel or data type",
		func(ctx SpecContext) {
			ch := valueField()
			ch.Channel, ch.DataType = 0, ""
			f, msg := lift(ctx, ch)
			Expect(f[0].Type()).To(Equal(payload.KindFloat))
			Expect(msg.Bindings[0].DataType).To(Equal(telem.Float64T))
		},
	)

	DescribeTable(
		"Should pick the kind that writes the channel as its JSON type",
		func(
			ctx SpecContext,
			dt telem.DataType,
			t payload.JSONType,
			kind payload.Kind,
		) {
			found[7] = v2.Channel{Name: "valve_cmd", DataType: dt}
			ch := valueField()
			ch.JSONType = t
			f, _ := lift(ctx, ch)
			Expect(f[0].Type()).To(Equal(kind))
		},
		Entry(
			"a float number",
			telem.Float32T,
			payload.JSONTypeNumber,
			payload.KindFloat,
		),
		Entry(
			"an integer number",
			telem.Int64T,
			payload.JSONTypeNumber,
			payload.KindInteger,
		),
		Entry(
			"an unsigned number",
			telem.Uint8T,
			payload.JSONTypeNumber,
			payload.KindUnsigned,
		),
		Entry("a boolean", telem.Uint8T, payload.JSONTypeBoolean, payload.KindBoolean),
		Entry("a string", telem.StringT, payload.JSONTypeString, payload.KindString),
		Entry("a UUID", telem.UUIDT, payload.JSONTypeString, payload.KindUUID),
		Entry("bytes", telem.BytesT, payload.JSONTypeString, payload.KindBytes),
		Entry("JSON", telem.JSONT, payload.JSONTypeString, payload.KindJSON),
	)

	It("Should lift a timestamp channel with its time format", func(ctx SpecContext) {
		found[7] = v2.Channel{Name: "valve_cmd", DataType: telem.TimestampT}
		ch := valueField()
		format := payload.TimeFormatUnixMs
		ch.TimeFormat = &format
		f, _ := lift(ctx, ch)
		Expect(f[0].Variant).To(BeAssignableToTypeOf(icd.TimestampJSONField{}))
		Expect(f[0].Variant.(icd.TimestampJSONField).TimeFormat).
			To(Equal(payload.TimeFormatUnixMs))
	})

	It("Should give a timestamp channel with no time format ISO 8601",
		func(ctx SpecContext) {
			found[7] = v2.Channel{Name: "valve_cmd", DataType: telem.TimestampT}
			f, _ := lift(ctx, valueField())
			Expect(f[0].Variant.(icd.TimestampJSONField).TimeFormat).
				To(Equal(payload.TimeFormatISO8601))
		},
	)

	It("Should keep the enum values of a number channel sent as a string",
		func(ctx SpecContext) {
			found[7] = v2.Channel{Name: "valve_cmd", DataType: telem.Int32T}
			ch := valueField()
			ch.JSONType = payload.JSONTypeString
			ch.EnumValues = []payload.EnumEntry{
				{Label: "OPEN", Value: 1},
				{Label: "HALF", Value: 0.5},
			}
			f, _ := lift(ctx, ch)
			Expect(f[0].Variant.(icd.IntegerJSONField).EnumValues).To(Equal(
				[]payload.IntegerEntry{{Label: "OPEN", Value: 1}},
			))
		},
	)

	DescribeTable(
		"Should log each change to what a field sends, with its path",
		func(
			ctx SpecContext,
			dt telem.DataType,
			t payload.JSONType,
			kind payload.Kind,
			change string,
		) {
			found[7] = v2.Channel{Name: "valve_cmd", DataType: dt}
			ch := valueField()
			ch.JSONType = t
			old := config(plainTarget("plant/valve/set", ch))
			store(ctx, "Valves", old, false)
			ins, logs := observe()
			Expect(migrate(ctx, ins)).To(Succeed())
			Expect(fields(made[0].Messages[0])[0].Type()).To(Equal(kind))
			logged := logs.FilterField(zap.String("field", "targets.0.channel")).All()
			Expect(logged).To(HaveLen(1))
			Expect(logged[0].Message).To(Equal("changed an MQTT write field to v2"))
			Expect(logged[0].ContextMap()).To(HaveKeyWithValue("change", change))
		},
		Entry(
			"a number sent as a string",
			telem.Float64T,
			payload.JSONTypeString,
			payload.KindFloat,
			"a float64 channel sent as a JSON string now sends a JSON number",
		),
		Entry(
			"a number sent as a boolean",
			telem.Uint8T,
			payload.JSONTypeBoolean,
			payload.KindBoolean,
			"a uint8 channel sent as a JSON boolean now sends only 0 and 1",
		),
		Entry(
			"JSON sent as a string",
			telem.JSONT,
			payload.JSONTypeString,
			payload.KindJSON,
			"a json channel sent as a JSON string now sends its JSON value",
		),
	)

	It("Should give each static field a constant default", func(ctx SpecContext) {
		f, msg := lift(
			ctx,
			valueField(),
			staticField("/mode", payload.JSONTypeString, "auto"),
			staticField("/level", payload.JSONTypeNumber, 2.5),
			staticField("/armed", payload.JSONTypeBoolean, true),
			staticField("/limits", payload.JSONTypeNumber, map[string]any{"max": 10.0}),
		)
		Expect(f).To(HaveLen(5))
		Expect(*f[1].Variant.(icd.StringJSONField).Default).To(Equal("auto"))
		Expect(*f[2].Variant.(icd.FloatJSONField).Default).To(Equal(payload.Float(2.5)))
		Expect(*f[3].Variant.(icd.BooleanJSONField).Default).To(BeTrue())
		Expect(*f[4].Variant.(icd.JSONJSONField).Default).
			To(Equal(map[string]any{"max": 10.0}))
		Expect(f[1].Base().Name).To(Equal("mode"))
		Expect(f[1].Base().Pointer).To(Equal("/mode"))
		Expect(msg.Bindings).To(HaveLen(1))
	})

	It("Should give each generated field a generator default", func(ctx SpecContext) {
		format := payload.TimeFormatUnixNs
		f, msg := lift(
			ctx,
			valueField(),
			generatedField("/id", payload.GeneratorTypeUUID, nil),
			generatedField("/time", payload.GeneratorTypeTimestamp, &format),
			generatedField("/sent", payload.GeneratorTypeTimestamp, nil),
		)
		Expect(f[1].Variant).To(Equal(icd.UuidJSONField{
			JSONBaseField: f[1].Base(),
			Default:       &icd.UUIDDefault{Variant: icd.RandomUUIDDefault{}},
		}))
		now := &icd.TimestampDefault{Variant: icd.NowTimestampDefault{}}
		Expect(f[2].Variant).To(Equal(icd.TimestampJSONField{
			JSONBaseField: f[2].Base(),
			TimeFormat:    payload.TimeFormatUnixNs,
			Default:       now,
		}))
		Expect(f[3].Variant.(icd.TimestampJSONField).TimeFormat).
			To(Equal(payload.TimeFormatISO8601))
		Expect(msg.Bindings).To(HaveLen(1))
	})

	It("Should name the field of an empty pointer value and keep names unique",
		func(ctx SpecContext) {
			ch := valueField()
			ch.Pointer = ""
			f, _ := lift(ctx, ch, staticField("/value", payload.JSONTypeNumber, 1.0))
			Expect(f[0].Base().Name).To(Equal("value"))
			Expect(f[0].Base().Pointer).To(Equal(""))
			Expect(f[1].Base().Name).To(Equal("value (2)"))
		},
	)

	It("Should keep a Sparkplug target and make no ICD without plain targets",
		func(ctx SpecContext) {
			old := config(v1.WriteTarget{Variant: v1.SparkplugWriteTarget{
				BaseWriteTarget: v1.BaseWriteTarget{Key: "t1"},
				Tag:             "speed",
				Channel:         7,
			}})
			store(ctx, "Sparkplug", old, false)
			ins, _ := observe()
			Expect(migrate(ctx, ins)).To(Succeed())
			Expect(made).To(BeEmpty())
			tag, ok := read(ctx, old.Key).Targets[0].Variant.(v2.SparkplugWriteTarget)
			Expect(ok).To(BeTrue())
			Expect(tag.Tag).To(Equal("speed"))
			Expect(tag.Channel).To(Equal(channel.Key(7)))
		},
	)

	It("Should keep the order of plain and Sparkplug targets", func(ctx SpecContext) {
		old := config(
			v1.WriteTarget{Variant: v1.SparkplugWriteTarget{
				BaseWriteTarget: v1.BaseWriteTarget{Key: "t1"},
				Tag:             "speed",
			}},
			plainTarget("plant/cmd", valueField()),
		)
		store(ctx, "Plant", old, false)
		ins, _ := observe()
		Expect(migrate(ctx, ins)).To(Succeed())
		targets := read(ctx, old.Key).Targets
		Expect(targets).To(HaveLen(2))
		Expect(targets[0].Variant).To(BeAssignableToTypeOf(v2.SparkplugWriteTarget{}))
		Expect(plainWrite(targets[1]).Topic).To(Equal("plant/cmd"))
	})

	It("Should make no ICD for the config of a snapshot", func(ctx SpecContext) {
		old := config(plainTarget("plant/cmd", valueField()))
		Expect(gorp.WrapWriter[uuid.UUID, v1.WriteConfig](db).Set(ctx, old)).
			To(Succeed())
		Expect(gorp.WrapWriter[task.Key, task.Task](db).Set(ctx, task.Task{
			Key:      old.Key,
			Name:     "Valves (snapshot)",
			Type:     "mqtt_write",
			Snapshot: true,
		})).To(Succeed())
		ins, _ := observe()
		Expect(migrate(ctx, ins)).To(Succeed())
		Expect(made).To(BeEmpty())
		Expect(read(ctx, old.Key).Targets).To(BeEmpty())
	})

	It("Should do nothing without stored configs", func(ctx SpecContext) {
		ins, _ := observe()
		Expect(migrate(ctx, ins)).To(Succeed())
		Expect(made).To(BeEmpty())
	})

	It("Should fail when a config has no task", func(ctx SpecContext) {
		store(ctx, "", config(plainTarget("plant/cmd", valueField())), true)
		ins, _ := observe()
		Expect(migrate(ctx, ins)).
			To(MatchError(ContainSubstring("tasks of MQTT write configs")))
	})

	It("Should fail with the config key when an ICD create fails",
		func(ctx SpecContext) {
			old := config(plainTarget("plant/cmd", valueField()))
			store(ctx, "Valves", old, false)
			create = func(context.Context, gorp.Tx, icd.ICD) error {
				return errors.New("create failed")
			}
			ins, _ := observe()
			Expect(migrate(ctx, ins)).To(MatchError(ContainSubstring(
				fmt.Sprintf("config %s: create failed", old.Key),
			)))
		},
	)
})

var _ = Describe("MigrateWriteTarget", func() {
	It("Should keep a Sparkplug target", func(ctx SpecContext) {
		t := MustSucceed(v2.MigrateWriteTarget(ctx, v1.WriteTarget{
			Variant: v1.SparkplugWriteTarget{
				Tag:     "speed",
				TagType: v1.SparkplugDataTypeDouble,
			},
		}))
		Expect(t.Variant).To(Equal(v2.SparkplugWriteTarget{
			Tag:     "speed",
			TagType: v2.SparkplugDataTypeDouble,
		}))
	})

	It("Should refuse a plain target, which needs an ICD", func(ctx SpecContext) {
		_, err := v2.MigrateWriteTarget(ctx, plainTarget("plant/cmd", valueField()))
		Expect(err).To(MatchError(validate.ErrValidation))
		Expect(
			err,
		).To(MatchError(ContainSubstring("plain target plant/cmd needs an ICD")))
	})
})
