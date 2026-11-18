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
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func storedField(dt telem.DataType, entries ...payload.EnumEntry) v1.ReadField {
	return v1.ReadField{
		Key:        "f1",
		Name:       "level",
		Channel:    7,
		Pointer:    "/level",
		DataType:   dt,
		EnumValues: entries,
	}
}

func observe() (alamos.Instrumentation, *observer.ObservedLogs) {
	GinkgoHelper()
	core, logs := observer.New(zapcore.WarnLevel)
	logger := MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
		ZapLogger: zap.New(core),
	}))
	return alamos.New("test", alamos.WithLogger(logger)), logs
}

// fields returns the fields of the JSON message msg.
func fields(msg icd.Message) []icd.JSONField {
	GinkgoHelper()
	p, ok := msg.Payload.Variant.(icd.JSONPayload)
	Expect(ok).To(BeTrue())
	return p.Fields
}

// plainEntry returns the plain entry of e.
func plainEntry(e v2.ReadEntry) v2.PlainReadEntry {
	GinkgoHelper()
	plain, ok := e.Variant.(v2.PlainReadEntry)
	Expect(ok).To(BeTrue())
	return plain
}

func inlineEntry(
	at uint32,
	topic, index string,
	fields ...v2.ReadField,
) v2.InlineEntry {
	return v2.InlineEntry{
		Key:    topic,
		At:     at,
		Topic:  topic,
		Fields: fields,
		Index:  index,
	}
}

// inlineConfig returns a config that holds entries inline.
func inlineConfig(entries ...v2.InlineEntry) v2.ReadConfig {
	return v2.ReadConfig{Inline: &v2.InlineRead{Entries: entries}}
}

var _ = Describe("ResolveReadConfig", func() {
	var (
		found    map[channel.Key]v2.Channel
		channels v2.Channels
		changes  []string
	)
	BeforeEach(func() {
		found, changes = nil, nil
		channels = func(
			context.Context,
			gorp.Snapshot,
			[]channel.Key,
		) (map[channel.Key]v2.Channel, error) {
			return found, nil
		}
	})

	// resolve resolves cfg for the task "Plant" and returns the ICD to create.
	resolve := func(ctx SpecContext, cfg *v2.ReadConfig) icd.ICD {
		GinkgoHelper()
		ic, ok := MustSucceed2(v2.ResolveReadConfig(
			ctx,
			nil,
			channels,
			"Plant",
			cfg,
			func(_, change string) { changes = append(changes, change) },
		))
		Expect(ok).To(BeTrue())
		return ic
	}

	// lift resolves one entry that holds old, and returns the field of old in the new
	// message and the read of that message.
	lift := func(
		ctx SpecContext,
		old v2.ReadField,
		index bool,
	) (icd.JSONField, v2.ReadMessage) {
		GinkgoHelper()
		var indexKey string
		if index {
			indexKey = old.Key
		}
		cfg := inlineConfig(inlineEntry(0, "plant/level", indexKey, old))
		ic := resolve(ctx, &cfg)
		return fields(ic.Messages[0])[0], plainEntry(cfg.Entries[0]).Messages[0]
	}

	DescribeTable(
		"Should pick the kind of the stored data type",
		func(ctx SpecContext, dt telem.DataType, kind payload.Kind) {
			f, _ := lift(ctx, storedField(dt), false)
			Expect(f.Type()).To(Equal(kind))
			Expect(changes).To(BeEmpty())
		},
		Entry("float32", telem.Float32T, payload.KindFloat),
		Entry("int16", telem.Int16T, payload.KindInteger),
		Entry("uint8", telem.Uint8T, payload.KindUnsigned),
		Entry("boolean", telem.BooleanT, payload.KindBoolean),
		Entry("string", telem.StringT, payload.KindString),
		Entry("timestamp", telem.TimestampT, payload.KindTimestamp),
		Entry("json", telem.JSONT, payload.KindJSON),
		Entry("uuid", telem.UUIDT, payload.KindUUID),
		Entry("bytes", telem.BytesT, payload.KindBytes),
	)

	It("Should keep the name and the pointer of the field", func(ctx SpecContext) {
		f, read := lift(ctx, storedField(telem.Float64T), false)
		base := f.Base()
		Expect(base.Name).To(Equal("level"))
		Expect(base.Pointer).To(Equal("/level"))
		Expect(base.Scale).To(Equal(1.0))
		Expect(
			read.Bindings,
		).To(Equal([]v2.ReadBinding{{Field: base.Key, DataType: telem.Float64T}}))
	})

	It("Should keep the stored data type as the data type of a new channel",
		func(ctx SpecContext) {
			f, read := lift(ctx, storedField(telem.Int16T), false)
			Expect(read.Bindings).To(Equal([]v2.ReadBinding{
				{Field: f.Base().Key, DataType: telem.Int16T},
			}))
		},
	)

	It("Should bind the channel of a field and take the kind of the channel",
		func(ctx SpecContext) {
			found = map[channel.Key]v2.Channel{
				7: {Name: "level", DataType: telem.Int32T},
			}
			f, read := lift(ctx, storedField(
				telem.Float64T,
				payload.EnumEntry{Label: "ON", Value: 1},
				payload.EnumEntry{Label: "HALF", Value: 0.5},
			), false)
			integer, ok := f.Variant.(icd.IntegerJSONField)
			Expect(ok).To(BeTrue())
			Expect(integer.EnumValues).To(Equal(
				[]payload.IntegerEntry{{Label: "ON", Value: 1}},
			))
			Expect(read.Bindings).To(Equal([]v2.ReadBinding{
				{Field: f.Base().Key, Channel: 7},
			}))
			Expect(changes).To(Equal([]string{
				"data type float64 became int32, the data type of its channel",
				`enum label "HALF" was dropped: 0.5 is not integer`,
			}))
		},
	)

	It("Should lift a field with a time format on a timestamp channel",
		func(ctx SpecContext) {
			found = map[channel.Key]v2.Channel{
				7: {Name: "time", DataType: telem.TimestampT, Index: true},
			}
			old := storedField("")
			old.TimeFormat = new(payload.TimeFormatUnixMs)
			f, _ := lift(ctx, old, false)
			stamp, ok := f.Variant.(icd.TimestampJSONField)
			Expect(ok).To(BeTrue())
			Expect(stamp.TimeFormat).To(Equal(payload.TimeFormatUnixMs))
		},
	)

	It("Should convert whole entries and drop the others on an integer kind",
		func(ctx SpecContext) {
			f, _ := lift(ctx, storedField(
				telem.Int32T,
				payload.EnumEntry{Label: "OFF", Value: 0},
				payload.EnumEntry{Label: "ON", Value: 1},
				payload.EnumEntry{Label: "HALF", Value: 0.5},
				payload.EnumEntry{Label: "", Value: 2},
			), false)
			integer, ok := f.Variant.(icd.IntegerJSONField)
			Expect(ok).To(BeTrue())
			Expect(integer.EnumValues).To(Equal([]payload.IntegerEntry{
				{Label: "OFF", Value: 0},
				{Label: "ON", Value: 1},
			}))
			Expect(changes).To(ConsistOf(
				`enum label "HALF" was dropped: 0.5 is not integer`,
				`enum label "" was dropped: it is empty`,
			))
		},
	)

	It("Should keep the first entry of a repeated label", func(ctx SpecContext) {
		f, _ := lift(ctx, storedField(
			telem.BooleanT,
			payload.EnumEntry{Label: "ON", Value: 1},
			payload.EnumEntry{Label: "ON", Value: 0},
		), false)
		boolean, ok := f.Variant.(icd.BooleanJSONField)
		Expect(ok).To(BeTrue())
		Expect(boolean.EnumValues).To(Equal(
			[]payload.BooleanEntry{{Label: "ON", Value: true}},
		))
		Expect(changes).To(ConsistOf(
			`enum label "ON" was dropped: it repeats an earlier label`,
		))
	})

	It("Should drop a negative entry on an unsigned kind", func(ctx SpecContext) {
		f, _ := lift(ctx, storedField(
			telem.Uint16T,
			payload.EnumEntry{Label: "LOW", Value: -1},
			payload.EnumEntry{Label: "HIGH", Value: 3},
		), false)
		unsigned, ok := f.Variant.(icd.UnsignedJSONField)
		Expect(ok).To(BeTrue())
		Expect(unsigned.EnumValues).To(Equal(
			[]payload.UnsignedEntry{{Label: "HIGH", Value: 3}},
		))
		Expect(
			changes,
		).To(ConsistOf(`enum label "LOW" was dropped: -1 is not unsigned`))
	})

	It("Should keep every finite entry on a float kind", func(ctx SpecContext) {
		f, _ := lift(ctx, storedField(
			telem.Float64T,
			payload.EnumEntry{Label: "HALF", Value: 0.5},
		), false)
		float, ok := f.Variant.(icd.FloatJSONField)
		Expect(ok).To(BeTrue())
		Expect(float.EnumValues).To(Equal(
			[]payload.FloatEntry{{Label: "HALF", Value: 0.5}},
		))
		Expect(changes).To(BeEmpty())
	})

	It(
		"Should map a boolean entry to whether its value is not 0",
		func(ctx SpecContext) {
			f, _ := lift(ctx, storedField(
				telem.BooleanT,
				payload.EnumEntry{Label: "OFF", Value: 0},
				payload.EnumEntry{Label: "ON", Value: 2},
			), false)
			boolean, ok := f.Variant.(icd.BooleanJSONField)
			Expect(ok).To(BeTrue())
			Expect(boolean.EnumValues).To(Equal([]payload.BooleanEntry{
				{Label: "OFF", Value: false},
				{Label: "ON", Value: true},
			}))
		},
	)

	It("Should drop the entries of a string field", func(ctx SpecContext) {
		f, _ := lift(ctx, storedField(
			telem.StringT,
			payload.EnumEntry{Label: "ON", Value: 1},
		), false)
		str, ok := f.Variant.(icd.StringJSONField)
		Expect(ok).To(BeTrue())
		Expect(str.EnumValues).To(BeEmpty())
		Expect(changes).To(ConsistOf(
			`enum label "ON" was dropped: a string field takes no numeric entries`,
		))
	})

	It("Should give a timestamp field with no time format ISO 8601",
		func(ctx SpecContext) {
			f, _ := lift(ctx, storedField(telem.TimestampT), false)
			stamp, ok := f.Variant.(icd.TimestampJSONField)
			Expect(ok).To(BeTrue())
			Expect(stamp.TimeFormat).To(Equal(payload.TimeFormatISO8601))
		},
	)

	It("Should keep the time format of a field stored with another data type",
		func(ctx SpecContext) {
			old := storedField(telem.Float64T)
			old.TimeFormat = new(payload.TimeFormatUnixMs)
			f, read := lift(ctx, old, false)
			stamp, ok := f.Variant.(icd.TimestampJSONField)
			Expect(ok).To(BeTrue())
			Expect(stamp.TimeFormat).To(Equal(payload.TimeFormatUnixMs))
			Expect(read.Bindings[0].DataType).To(BeEmpty())
			Expect(changes).To(ConsistOf(
				"data type float64 became the timestamp kind of an index or time format",
			))
		},
	)

	It("Should give ISO 8601 to a field whose time format is not valid",
		func(ctx SpecContext) {
			old := storedField(telem.Float64T)
			old.TimeFormat = new(payload.TimeFormat("unix"))
			f, _ := lift(ctx, old, false)
			stamp, ok := f.Variant.(icd.TimestampJSONField)
			Expect(ok).To(BeTrue())
			Expect(stamp.TimeFormat).To(Equal(payload.TimeFormatISO8601))
			Expect(stamp.Validate()).To(Succeed())
			Expect(changes).To(ConsistOf(
				"data type float64 became the timestamp kind of an index or time format",
				`time format "unix" is not valid and became iso8601`,
			))
		},
	)

	It("Should keep the kind and entries of a typed field with a time format",
		func(ctx SpecContext) {
			old := storedField(telem.Uint8T, payload.EnumEntry{Label: "ON", Value: 1})
			old.TimeFormat = new(payload.TimeFormatUnixSec)
			f, _ := lift(ctx, old, false)
			unsigned, ok := f.Variant.(icd.UnsignedJSONField)
			Expect(ok).To(BeTrue())
			Expect(unsigned.EnumValues).To(Equal(
				[]payload.UnsignedEntry{{Label: "ON", Value: 1}},
			))
			Expect(changes).To(ConsistOf(
				"time format unix_sec was dropped, because the field is unsigned",
			))
		},
	)

	It("Should lift an index field to the timestamp kind and point the index at it",
		func(ctx SpecContext) {
			f, read := lift(ctx, storedField(""), true)
			stamp, ok := f.Variant.(icd.TimestampJSONField)
			Expect(ok).To(BeTrue())
			Expect(stamp.TimeFormat).To(Equal(payload.TimeFormatISO8601))
			Expect(read.Index).To(HaveValue(Equal(f.Base().Key)))
			Expect(read.Bindings).To(Equal([]v2.ReadBinding{{Field: f.Base().Key}}))
			Expect(changes).To(ConsistOf(
				"data type float64 became the timestamp kind of an index or time format",
			))
		},
	)

	It("Should lift an unknown data type to the float kind with no data type",
		func(ctx SpecContext) {
			f, read := lift(ctx, storedField("complex128"), false)
			Expect(f.Type()).To(Equal(payload.KindFloat))
			Expect(read.Bindings[0].DataType).To(BeEmpty())
			Expect(changes).To(ConsistOf(
				`unknown data type "complex128" became the float kind`,
			))
		},
	)

	It("Should lift an unknown data type with a time format to the timestamp kind",
		func(ctx SpecContext) {
			old := storedField("complex128")
			old.TimeFormat = new(payload.TimeFormatUnixSec)
			f, _ := lift(ctx, old, false)
			stamp, ok := f.Variant.(icd.TimestampJSONField)
			Expect(ok).To(BeTrue())
			Expect(stamp.TimeFormat).To(Equal(payload.TimeFormatUnixSec))
			Expect(changes).To(ConsistOf(
				`unknown data type "complex128" became the float kind`,
				"data type complex128 became the timestamp kind of an index or time format",
			))
		},
	)

	It("Should take float64 for an empty data type", func(ctx SpecContext) {
		f, read := lift(ctx, storedField(""), false)
		Expect(f.Type()).To(Equal(payload.KindFloat))
		Expect(read.Bindings[0].DataType).To(Equal(telem.Float64T))
		Expect(changes).To(BeEmpty())
	})

	It("Should keep a disabled field in the message without a binding",
		func(ctx SpecContext) {
			disabled := storedField(telem.Float64T)
			disabled.Key, disabled.Name, disabled.Disabled = "f2", "spare", true
			cfg := inlineConfig(inlineEntry(
				0, "plant/level", "", storedField(telem.Float64T), disabled,
			))
			msg := resolve(ctx, &cfg).Messages[0]
			Expect(fields(msg)).To(HaveLen(2))
			bindings := plainEntry(cfg.Entries[0]).Messages[0].Bindings
			Expect(bindings).To(HaveLen(1))
			Expect(bindings[0].Field).To(Equal(fields(msg)[0].Base().Key))
		},
	)

	It("Should name a field after its pointer and keep names unique",
		func(ctx SpecContext) {
			named := func(key, name, pointer string) v2.ReadField {
				f := storedField(telem.Float64T)
				f.Key, f.Name, f.Pointer = key, name, pointer
				return f
			}
			cfg := inlineConfig(inlineEntry(
				0,
				"plant/level",
				"",
				named("f1", "", "/out/flow"),
				named("f2", "level", "/a"),
				named("f3", "level", "/b"),
				named("f4", "", ""),
			))
			messageFields := fields(resolve(ctx, &cfg).Messages[0])
			names := make([]string, 0, len(messageFields))
			for _, f := range messageFields {
				names = append(names, f.Base().Name)
			}
			Expect(names).To(Equal([]string{"out/flow", "level", "level (2)", "value"}))
		},
	)

	It("Should drop an index that names no field", func(ctx SpecContext) {
		cfg := inlineConfig(
			inlineEntry(0, "plant/level", "gone", storedField(telem.Float64T)),
		)
		resolve(ctx, &cfg)
		Expect(plainEntry(cfg.Entries[0]).Messages[0].Index).To(BeNil())
		Expect(changes).To(ConsistOf("index field gone does not exist and was dropped"))
	})

	It("Should make one ICD with a message named after the topic of each entry",
		func(ctx SpecContext) {
			sparkplug := v2.ReadEntry{Variant: v2.SparkplugReadEntry{
				BaseReadEntry: v2.BaseReadEntry{Key: "tag"},
				Tag:           "speed",
			}}
			cfg := inlineConfig(
				inlineEntry(0, "plant/a", "", storedField(telem.Float64T)),
				inlineEntry(2, "plant/a", "", storedField(telem.Float64T)),
				inlineEntry(3, "", "", storedField(telem.Float64T)),
			)
			cfg.Entries = []v2.ReadEntry{sparkplug}
			ic := resolve(ctx, &cfg)
			Expect(ic.Name).To(Equal("Plant"))
			Expect(ic.Messages).To(HaveLen(3))
			names := make([]string, 0, len(ic.Messages))
			for _, m := range ic.Messages {
				names = append(names, m.Name)
			}
			Expect(names).To(Equal([]string{"plant/a", "plant/a (2)", "Untitled"}))
			Expect(cfg.Inline).To(BeNil())
			Expect(cfg.Entries).To(HaveLen(4))
			Expect(cfg.Entries[1]).To(Equal(sparkplug))
			for i, j := range []int{0, 2, 3} {
				plain := plainEntry(cfg.Entries[j])
				Expect(plain.ICD).To(Equal(ic.Key))
				Expect(plain.Messages).To(HaveLen(1))
				Expect(plain.Messages[0].Message).To(Equal(ic.Messages[i].Key))
			}
		},
	)

	It("Should keep the base and the subscription of an inline entry",
		func(ctx SpecContext) {
			e := inlineEntry(0, "plant/a", "", storedField(telem.Float64T))
			e.Key, e.Disabled = "e1", true
			e.Qos, e.RetainedIgnored = v2.QoSExactlyOnce, true
			cfg := inlineConfig(e)
			ic := resolve(ctx, &cfg)
			plain := plainEntry(cfg.Entries[0])
			Expect(
				plain.BaseReadEntry,
			).To(Equal(v2.BaseReadEntry{Key: "e1", Disabled: true}))
			Expect(plain.Topic).To(Equal("plant/a"))
			Expect(plain.Qos).To(Equal(v2.QoSExactlyOnce))
			Expect(plain.RetainedIgnored).To(BeTrue())
			Expect(plain.ICD).To(Equal(ic.Key))
		},
	)

	It("Should resolve nothing when the config holds no inline entries",
		func(ctx SpecContext) {
			cfg := v2.ReadConfig{Entries: []v2.ReadEntry{
				{Variant: v2.PlainReadEntry{Topic: "plant/a"}},
				{Variant: v2.SparkplugReadEntry{Tag: "speed"}},
			}}
			before := cfg
			_, ok := MustSucceed2(v2.ResolveReadConfig(
				ctx, nil, channels, "Plant", &cfg, func(_, _ string) {},
			))
			Expect(ok).To(BeFalse())
			Expect(cfg).To(Equal(before))
		},
	)
})

var _ = Describe("MigrateReadConfig", func() {
	It("Should move each plain entry inline and keep a Sparkplug entry",
		func(ctx SpecContext) {
			old := v1.ReadConfig{
				Key:    uuid.New(),
				Device: "broker",
				Entries: []v1.ReadEntry{
					{Variant: v1.SparkplugReadEntry{
						BaseReadEntry: v1.BaseReadEntry{Key: "e1"},
						Tag:           "speed",
						DataType:      telem.Float32T,
					}},
					{Variant: v1.PlainReadEntry{
						BaseReadEntry: v1.BaseReadEntry{Key: "e2"},
						Topic:         "plant/level",
						Qos:           v1.QoSAtLeastOnce,
						Index:         "f1",
						Fields:        []v1.ReadField{storedField(telem.Uint8T)},
					}},
				},
			}
			cfg := MustSucceed(v2.MigrateReadConfig(ctx, old))
			Expect(cfg.Key).To(Equal(old.Key))
			Expect(cfg.Device).To(Equal(old.Device))
			Expect(cfg.Entries).To(HaveLen(1))
			tag, ok := cfg.Entries[0].Variant.(v2.SparkplugReadEntry)
			Expect(ok).To(BeTrue())
			Expect(tag.Tag).To(Equal("speed"))
			Expect(tag.DataType).To(Equal(telem.Float32T))
			Expect(cfg.Inline).To(Equal(&v2.InlineRead{Entries: []v2.InlineEntry{{
				Key:    "e2",
				At:     1,
				Topic:  "plant/level",
				Qos:    v2.QoSAtLeastOnce,
				Fields: []v2.ReadField{storedField(telem.Uint8T)},
				Index:  "f1",
			}}}))
		},
	)

	It("Should hold nothing inline for a config without plain entries",
		func(ctx SpecContext) {
			cfg := MustSucceed(v2.MigrateReadConfig(ctx, v1.ReadConfig{
				Entries: []v1.ReadEntry{{Variant: v1.SparkplugReadEntry{Tag: "speed"}}},
			}))
			Expect(cfg.Inline).To(BeNil())
			Expect(cfg.Entries).To(HaveLen(1))
		},
	)
})

var _ = Describe("NewReadMigration", func() {
	var (
		db       *gorp.DB
		made     []icd.ICD
		create   v2.CreateICD
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
		channels = func(
			context.Context,
			gorp.Snapshot,
			[]channel.Key,
		) (map[channel.Key]v2.Channel, error) {
			return map[channel.Key]v2.Channel{
				7: {Name: "level", DataType: telem.Int32T},
			}, nil
		}
	})

	// store writes old and, unless orphan, a task named name that owns it.
	store := func(ctx SpecContext, name string, old v1.ReadConfig, orphan bool) {
		GinkgoHelper()
		Expect(gorp.WrapWriter[uuid.UUID, v1.ReadConfig](db).Set(ctx, old)).
			To(Succeed())
		if !orphan {
			Expect(gorp.WrapWriter[task.Key, task.Task](db).Set(ctx, task.Task{
				Key:  old.Key,
				Name: name,
				Type: "mqtt_read",
			})).To(Succeed())
		}
	}
	migrate := func(ctx SpecContext, ins alamos.Instrumentation) error {
		return gorp.Migrate(ctx, gorp.MigrateConfig{
			Instrumentation: ins,
			DB:              db,
			Namespace:       "mqtt_read_config",
			Migrations:      []gorp.Migration{v2.NewReadMigration(channels, create)},
		})
	}
	read := func(ctx SpecContext, key uuid.UUID) v2.ReadConfig {
		GinkgoHelper()
		var cfg v2.ReadConfig
		Expect(gorp.NewRetrieve[uuid.UUID, v2.ReadConfig]().
			Where(gorp.MatchKeys[uuid.UUID, v2.ReadConfig](key)).
			Entry(&cfg).
			Exec(ctx, db)).To(Succeed())
		return cfg
	}
	plainConfig := func() v1.ReadConfig {
		return v1.ReadConfig{
			Key: uuid.New(),
			Entries: []v1.ReadEntry{{Variant: v1.PlainReadEntry{
				BaseReadEntry: v1.BaseReadEntry{Key: "e1"},
				Topic:         "plant/level",
				Fields: []v1.ReadField{storedField(
					telem.Float64T,
					payload.EnumEntry{Label: "HALF", Value: 0.5},
				)},
			}}},
		}
	}

	It("Should store the shape that v1 stored under the same table name", func() {
		Expect(v2.ReadConfig{}.CustomTypeName()).
			To(Equal(v1.ReadConfig{}.CustomTypeName()))
	})

	It("Should move the fields of each config into an ICD named after its task",
		func(ctx SpecContext) {
			old := plainConfig()
			store(ctx, "Pump station", old, false)
			ins, logs := observe()
			Expect(migrate(ctx, ins)).To(Succeed())
			Expect(made).To(HaveLen(1))
			Expect(made[0].Name).To(Equal("Pump station"))
			Expect(made[0].Messages).To(HaveLen(1))
			Expect(made[0].Messages[0].Name).To(Equal("plant/level"))
			f := fields(made[0].Messages[0])[0]
			Expect(f.Type()).To(Equal(payload.KindInteger))
			migrated := read(ctx, old.Key)
			Expect(migrated.Inline).To(BeNil())
			plain := plainEntry(migrated.Entries[0])
			Expect(plain.Topic).To(Equal("plant/level"))
			Expect(plain.ICD).To(Equal(made[0].Key))
			Expect(plain.Messages).To(Equal([]v2.ReadMessage{{
				Message:  made[0].Messages[0].Key,
				Bindings: []v2.ReadBinding{{Field: f.Base().Key, Channel: 7}},
			}}))
			logged := logs.FilterField(zap.String("field", "entries.0.fields.0")).All()
			Expect(logged).To(HaveLen(2))
			Expect(logged[0].ContextMap()).To(HaveKeyWithValue(
				"change",
				"data type float64 became int32, the data type of its channel",
			))
			Expect(logged[1].ContextMap()).To(HaveKeyWithValue(
				"change", `enum label "HALF" was dropped: 0.5 is not integer`,
			))
		},
	)

	It("Should make no ICD for the config of a snapshot", func(ctx SpecContext) {
		old := plainConfig()
		Expect(gorp.WrapWriter[uuid.UUID, v1.ReadConfig](db).Set(ctx, old)).
			To(Succeed())
		Expect(gorp.WrapWriter[task.Key, task.Task](db).Set(ctx, task.Task{
			Key:      old.Key,
			Name:     "Pump (snapshot)",
			Type:     "mqtt_read",
			Snapshot: true,
		})).To(Succeed())
		ins, _ := observe()
		Expect(migrate(ctx, ins)).To(Succeed())
		Expect(made).To(BeEmpty())
		Expect(read(ctx, old.Key).Entries).To(BeEmpty())
	})

	It("Should make no ICD for a config without plain entries", func(ctx SpecContext) {
		old := v1.ReadConfig{
			Key: uuid.New(),
			Entries: []v1.ReadEntry{{Variant: v1.SparkplugReadEntry{
				BaseReadEntry: v1.BaseReadEntry{Key: "e1"},
				Tag:           "speed",
			}}},
		}
		store(ctx, "Sparkplug", old, false)
		ins, _ := observe()
		Expect(migrate(ctx, ins)).To(Succeed())
		Expect(made).To(BeEmpty())
		tag, ok := read(ctx, old.Key).Entries[0].Variant.(v2.SparkplugReadEntry)
		Expect(ok).To(BeTrue())
		Expect(tag.Tag).To(Equal("speed"))
	})

	It("Should do nothing without stored configs", func(ctx SpecContext) {
		ins, _ := observe()
		Expect(migrate(ctx, ins)).To(Succeed())
		Expect(made).To(BeEmpty())
	})

	It("Should fail when a config has no task", func(ctx SpecContext) {
		store(ctx, "", plainConfig(), true)
		ins, _ := observe()
		Expect(migrate(ctx, ins)).
			To(MatchError(ContainSubstring("tasks of MQTT read configs")))
	})

	It("Should fail with the config key when an ICD create fails",
		func(ctx SpecContext) {
			old := plainConfig()
			store(ctx, "Pump", old, false)
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
