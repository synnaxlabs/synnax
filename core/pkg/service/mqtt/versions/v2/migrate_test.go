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

func inlineEntry(at uint32, topic string, fields ...v2.ReadField) v2.InlineEntry {
	return v2.InlineEntry{Key: topic, At: at, Topic: topic, Fields: fields}
}

// inlineConfig returns a config that holds entries inline.
func inlineConfig(entries ...v2.InlineEntry) v2.ReadConfig {
	return v2.ReadConfig{Inline: &v2.InlineRead{Entries: entries}}
}

var _ = Describe("ResolveReadConfig", func() {
	var (
		found    map[channel.Key]icd.Channel
		channels icd.Channels
		changes  []string
	)
	BeforeEach(func() {
		found, changes = nil, nil
		channels = func(
			context.Context,
			gorp.Snapshot,
			[]channel.Key,
		) (map[channel.Key]icd.Channel, error) {
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

	It("Should note each change at the path of its entry", func(ctx SpecContext) {
		var paths []string
		cfg := inlineConfig(inlineEntry(2, "plant/a", storedField("complex128")))
		MustSucceed2(v2.ResolveReadConfig(
			ctx,
			nil,
			channels,
			"Plant",
			&cfg,
			func(path, _ string) { paths = append(paths, path) },
		))
		Expect(paths).To(Equal([]string{"entries.2.fields.0"}))
	})

	It("Should take the channels of the fields from channels", func(ctx SpecContext) {
		found = map[channel.Key]icd.Channel{7: {Name: "level", DataType: telem.Int32T}}
		cfg := inlineConfig(inlineEntry(0, "plant/a", storedField(telem.Float64T)))
		ic := resolve(ctx, &cfg)
		Expect(fields(ic.Messages[0])[0].Type()).To(Equal(payload.KindInteger))
		Expect(plainEntry(cfg.Entries[0]).Messages[0].Bindings[0].Channel).
			To(Equal(channel.Key(7)))
	})

	It("Should make one ICD with a message named after the topic of each entry",
		func(ctx SpecContext) {
			sparkplug := v2.ReadEntry{Variant: v2.SparkplugReadEntry{
				BaseReadEntry: v2.BaseReadEntry{Key: "tag"},
				Tag:           "speed",
			}}
			cfg := inlineConfig(
				inlineEntry(0, "plant/a", storedField(telem.Float64T)),
				inlineEntry(2, "plant/a", storedField(telem.Float64T)),
				inlineEntry(3, "", storedField(telem.Float64T)),
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

	It("Should index the read with the field that the inline entry names",
		func(ctx SpecContext) {
			e := inlineEntry(0, "plant/a", storedField(telem.TimestampT))
			e.Index = "f1"
			cfg := inlineConfig(e)
			ic := resolve(ctx, &cfg)
			index := plainEntry(cfg.Entries[0]).Messages[0].Index
			Expect(index).ToNot(BeNil())
			Expect(*index).To(Equal(fields(ic.Messages[0])[0].Base().Key))
			Expect(changes).To(BeEmpty())
		},
	)

	It("Should lift the name, pointer, and time format of a disabled field",
		func(ctx SpecContext) {
			f := storedField(telem.TimestampT)
			f.Name, f.Pointer, f.Disabled = "stamp", "/meta/stamp", true
			f.TimeFormat = new(payload.TimeFormatUnixMs)
			cfg := inlineConfig(inlineEntry(0, "plant/a", f))
			ic := resolve(ctx, &cfg)
			lifted := fields(ic.Messages[0])[0]
			Expect(lifted.Base().Name).To(Equal("stamp"))
			Expect(lifted.Base().Pointer).To(Equal("/meta/stamp"))
			ts, ok := lifted.Variant.(icd.TimestampJSONField)
			Expect(MustBeOk(ts, ok).TimeFormat).To(Equal(payload.TimeFormatUnixMs))
			Expect(plainEntry(cfg.Entries[0]).Messages[0].Bindings).To(BeEmpty())
		},
	)

	It("Should keep the base and the subscription of an inline entry",
		func(ctx SpecContext) {
			e := inlineEntry(0, "plant/a", storedField(telem.Float64T))
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
		channels icd.Channels
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
		) (map[channel.Key]icd.Channel, error) {
			return map[channel.Key]icd.Channel{
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
			Expect(plain.Messages).To(Equal([]icd.ReadMessage{{
				Message:  made[0].Messages[0].Key,
				Bindings: []icd.ReadBinding{{Field: f.Base().Key, Channel: 7}},
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
