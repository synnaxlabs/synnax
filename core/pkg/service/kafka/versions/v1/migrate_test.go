// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

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
	v0 "github.com/synnaxlabs/synnax/pkg/service/kafka/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/kafka/versions/v1"
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

func storedField(
	key string,
	dt telem.DataType,
	entries ...payload.EnumEntry,
) v1.ReadField {
	return v1.ReadField{
		Key:        key,
		Name:       key,
		Channel:    7,
		Pointer:    "/" + key,
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

// inlineConfig returns a config that holds records inline.
func inlineConfig(records ...v1.InlineRecord) v1.ReadConfig {
	return v1.ReadConfig{Inline: &v1.InlineRead{Records: records}}
}

var _ = Describe("ResolveReadConfig", func() {
	var (
		found    map[channel.Key]v1.Channel
		channels v1.Channels
		changes  []string
	)
	BeforeEach(func() {
		found, changes = nil, nil
		channels = func(
			context.Context,
			gorp.Snapshot,
			[]channel.Key,
		) (map[channel.Key]v1.Channel, error) {
			return found, nil
		}
	})

	// resolve resolves cfg for the task "Plant" and returns the ICD to create.
	resolve := func(ctx SpecContext, cfg *v1.ReadConfig) icd.ICD {
		GinkgoHelper()
		ic, ok := MustSucceed2(v1.ResolveReadConfig(
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

	DescribeTable(
		"Should pick the kind of the stored data type",
		func(ctx SpecContext, dt telem.DataType, kind payload.Kind) {
			cfg := inlineConfig(v1.InlineRecord{
				Key:    "r1",
				Fields: []v1.ReadField{storedField("level", dt)},
			})
			ic := resolve(ctx, &cfg)
			Expect(fields(ic.Messages[0])[0].Type()).To(Equal(kind))
			Expect(changes).To(BeEmpty())
		},
		Entry("float32", telem.Float32T, payload.KindFloat),
		Entry("int16", telem.Int16T, payload.KindInteger),
		Entry("uint8", telem.Uint8T, payload.KindUnsigned),
		Entry("boolean", telem.BooleanT, payload.KindBoolean),
		Entry("string", telem.StringT, payload.KindString),
		Entry("timestamp", telem.TimestampT, payload.KindTimestamp),
		Entry("json", telem.JSONT, payload.KindJSON),
	)

	It("Should make one ICD with a message named after the key of each record",
		func(ctx SpecContext) {
			cfg := inlineConfig(
				v1.InlineRecord{
					Key:       "r1",
					RecordKey: "pump",
					Fields:    []v1.ReadField{storedField("level", telem.Float64T)},
				},
				v1.InlineRecord{
					Key:       "r2",
					RecordKey: "pump",
					Fields:    []v1.ReadField{storedField("level", telem.Float64T)},
				},
				v1.InlineRecord{
					Key:    "r3",
					Fields: []v1.ReadField{storedField("level", telem.Float64T)},
				},
			)
			ic := resolve(ctx, &cfg)
			Expect(ic.Name).To(Equal("Plant"))
			Expect(ic.Messages).To(HaveLen(3))
			Expect(ic.Messages[0].Name).To(Equal("pump"))
			Expect(ic.Messages[1].Name).To(Equal("pump (2)"))
			Expect(ic.Messages[2].Name).To(Equal("Any key"))
			Expect(cfg.Inline).To(BeNil())
			Expect(cfg.Records).To(HaveLen(3))
			for i, rec := range cfg.Records {
				Expect(rec.ICD).To(Equal(ic.Key))
				Expect(rec.Messages).To(HaveLen(1))
				Expect(rec.Messages[0].Message).To(Equal(ic.Messages[i].Key))
			}
			Expect(cfg.Records[0].Key).To(Equal("r1"))
			Expect(cfg.Records[0].RecordKey).To(Equal("pump"))
			Expect(cfg.Records[2].RecordKey).To(BeEmpty())
		},
	)

	It("Should bind the channel of a field and take the kind of the channel",
		func(ctx SpecContext) {
			found = map[channel.Key]v1.Channel{
				7: {Name: "level", DataType: telem.Int32T},
			}
			cfg := inlineConfig(v1.InlineRecord{
				Key: "r1",
				Fields: []v1.ReadField{storedField(
					"level",
					telem.Float64T,
					payload.EnumEntry{Label: "ON", Value: 1},
					payload.EnumEntry{Label: "HALF", Value: 0.5},
				)},
			})
			ic := resolve(ctx, &cfg)
			f := fields(ic.Messages[0])[0]
			integer, ok := f.Variant.(icd.IntegerJSONField)
			Expect(ok).To(BeTrue())
			Expect(integer.EnumValues).To(Equal(
				[]payload.IntegerEntry{{Label: "ON", Value: 1}},
			))
			Expect(f.Base().Pointer).To(Equal("/level"))
			Expect(cfg.Records[0].Messages[0].Bindings).To(Equal([]v1.ReadBinding{
				{Field: f.Base().Key, Channel: 7},
			}))
			Expect(changes).To(Equal([]string{
				"data type float64 became int32, the data type of its channel",
				`enum label "HALF" was dropped: 0.5 is not integer`,
			}))
		},
	)

	It("Should keep the stored data type as the data type of a new channel",
		func(ctx SpecContext) {
			cfg := inlineConfig(v1.InlineRecord{
				Key:    "r1",
				Fields: []v1.ReadField{storedField("level", telem.Int16T)},
			})
			ic := resolve(ctx, &cfg)
			Expect(cfg.Records[0].Messages[0].Bindings).To(Equal([]v1.ReadBinding{
				{Field: fields(ic.Messages[0])[0].Base().Key, DataType: telem.Int16T},
			}))
		},
	)

	It("Should lift the index field to a timestamp and point the index at it",
		func(ctx SpecContext) {
			ts := storedField("ts", telem.TimestampT)
			format := payload.TimeFormatUnixMs
			ts.TimeFormat = &format
			cfg := inlineConfig(v1.InlineRecord{
				Key:    "r1",
				Index:  "ts",
				Fields: []v1.ReadField{ts, storedField("level", telem.Float64T)},
			})
			ic := resolve(ctx, &cfg)
			f := fields(ic.Messages[0])[0]
			stamp, ok := f.Variant.(icd.TimestampJSONField)
			Expect(ok).To(BeTrue())
			Expect(stamp.TimeFormat).To(Equal(payload.TimeFormatUnixMs))
			Expect(cfg.Records[0].Messages[0].Index).To(Equal(new(f.Base().Key)))
		},
	)

	It("Should drop an index that names no field", func(ctx SpecContext) {
		cfg := inlineConfig(v1.InlineRecord{
			Key:    "r1",
			Index:  "gone",
			Fields: []v1.ReadField{storedField("level", telem.Float64T)},
		})
		resolve(ctx, &cfg)
		Expect(cfg.Records[0].Messages[0].Index).To(BeNil())
		Expect(
			changes,
		).To(Equal([]string{"index field gone does not exist and was dropped"}))
	})

	It("Should keep a disabled field in the message without a binding",
		func(ctx SpecContext) {
			off := storedField("off", telem.Float64T)
			off.Disabled = true
			cfg := inlineConfig(v1.InlineRecord{
				Key:    "r1",
				Fields: []v1.ReadField{storedField("level", telem.Float64T), off},
			})
			ic := resolve(ctx, &cfg)
			Expect(fields(ic.Messages[0])).To(HaveLen(2))
			Expect(cfg.Records[0].Disabled).To(BeFalse())
			Expect(cfg.Records[0].Messages[0].Bindings).To(HaveLen(1))
		},
	)

	It("Should disable a record whose fields are all disabled", func(ctx SpecContext) {
		off := storedField("off", telem.Float64T)
		off.Disabled = true
		cfg := inlineConfig(v1.InlineRecord{Key: "r1", Fields: []v1.ReadField{off}})
		resolve(ctx, &cfg)
		Expect(cfg.Records[0].Disabled).To(BeTrue())
		Expect(cfg.Records[0].Messages[0].Bindings).To(BeEmpty())
	})

	It("Should name a field after its pointer and keep names unique",
		func(ctx SpecContext) {
			first := storedField("a", telem.Float64T)
			second := storedField("b", telem.Float64T)
			first.Name, second.Name = "", ""
			second.Pointer = "/a"
			cfg := inlineConfig(v1.InlineRecord{
				Key:    "r1",
				Fields: []v1.ReadField{first, second},
			})
			ic := resolve(ctx, &cfg)
			Expect(fields(ic.Messages[0])[0].Base().Name).To(Equal("a"))
			Expect(fields(ic.Messages[0])[1].Base().Name).To(Equal("a (2)"))
		},
	)

	It("Should resolve nothing when the config holds no inline records",
		func(ctx SpecContext) {
			cfg := v1.ReadConfig{Records: []v1.ReadRecord{{Key: "r1"}}}
			_, ok := MustSucceed2(v1.ResolveReadConfig(
				ctx, nil, channels, "Plant", &cfg, func(string, string) {},
			))
			Expect(ok).To(BeFalse())
			Expect(cfg.Records).To(Equal([]v1.ReadRecord{{Key: "r1"}}))
		},
	)
})

var _ = Describe("MigrateReadRecord", func() {
	It("Should keep the key and the record key and name no ICD or messages",
		func(ctx SpecContext) {
			r := MustSucceed(v1.MigrateReadRecord(ctx, v0.ReadRecord{
				Key:       "r1",
				RecordKey: "pump",
				Index:     "ts",
				Fields:    []v0.ReadField{storedField("level", telem.Float64T)},
			}))
			Expect(r).To(Equal(v1.ReadRecord{Key: "r1", RecordKey: "pump"}))
		},
	)
})

var _ = Describe("NoteLift", func() {
	It("Should log each change as a warning with the config and the field",
		func() {
			ins, logs := observe()
			key := uuid.New()
			v1.NoteLift(ins, key)("records.0.fields.1", "a change")
			Expect(logs.All()).To(HaveLen(1))
			entry := logs.All()[0]
			Expect(entry.Level).To(Equal(zapcore.WarnLevel))
			Expect(entry.Message).To(Equal("changed a Kafka read field to v1"))
			Expect(entry.ContextMap()).To(Equal(map[string]any{
				"config": key.String(),
				"field":  "records.0.fields.1",
				"change": "a change",
			}))
		},
	)
})

var _ = Describe("MigrateReadConfig", func() {
	It("Should move each record inline and keep the rest of the config",
		func(ctx SpecContext) {
			old := v0.ReadConfig{
				Key:         uuid.New(),
				Device:      "cluster",
				Topic:       "plant",
				Group:       "group",
				StartOffset: v0.StartOffsetEarliest,
				Records: []v0.ReadRecord{{
					Key:       "r1",
					RecordKey: "pump",
					Index:     "ts",
					Fields:    []v0.ReadField{storedField("level", telem.Uint8T)},
				}},
			}
			cfg := MustSucceed(v1.MigrateReadConfig(ctx, old))
			Expect(cfg.Key).To(Equal(old.Key))
			Expect(cfg.Device).To(Equal(old.Device))
			Expect(cfg.Topic).To(Equal("plant"))
			Expect(cfg.Group).To(Equal("group"))
			Expect(cfg.StartOffset).To(Equal(v1.StartOffsetEarliest))
			Expect(cfg.Records).To(BeEmpty())
			Expect(cfg.Inline).To(Equal(&v1.InlineRead{Records: []v1.InlineRecord{{
				Key:       "r1",
				RecordKey: "pump",
				Index:     "ts",
				Fields:    []v1.ReadField{storedField("level", telem.Uint8T)},
			}}}))
		},
	)

	It(
		"Should hold nothing inline for a config without records",
		func(ctx SpecContext) {
			cfg := MustSucceed(v1.MigrateReadConfig(ctx, v0.ReadConfig{Topic: "plant"}))
			Expect(cfg.Inline).To(BeNil())
		},
	)
})

var _ = Describe("NewReadMigration", func() {
	var (
		db       *gorp.DB
		made     []icd.ICD
		create   v1.CreateICD
		channels v1.Channels
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
		) (map[channel.Key]v1.Channel, error) {
			return map[channel.Key]v1.Channel{
				7: {Name: "level", DataType: telem.Int32T},
			}, nil
		}
	})

	// storeTask writes old and the task t that owns it, with the key of old.
	storeTask := func(ctx SpecContext, old v0.ReadConfig, t task.Task) {
		GinkgoHelper()
		Expect(gorp.WrapWriter[uuid.UUID, v0.ReadConfig](db).Set(ctx, old)).
			To(Succeed())
		t.Key, t.Type = old.Key, "kafka_read"
		Expect(gorp.WrapWriter[task.Key, task.Task](db).Set(ctx, t)).To(Succeed())
	}
	// store writes old and, unless orphan, a task named name that owns it.
	store := func(ctx SpecContext, name string, old v0.ReadConfig, orphan bool) {
		GinkgoHelper()
		if !orphan {
			storeTask(ctx, old, task.Task{Name: name})
			return
		}
		Expect(gorp.WrapWriter[uuid.UUID, v0.ReadConfig](db).Set(ctx, old)).
			To(Succeed())
	}
	migrate := func(ctx SpecContext, ins alamos.Instrumentation) error {
		return gorp.Migrate(ctx, gorp.MigrateConfig{
			Instrumentation: ins,
			DB:              db,
			Namespace:       "kafka_read_config",
			Migrations:      []gorp.Migration{v1.NewReadMigration(channels, create)},
		})
	}
	read := func(ctx SpecContext, key uuid.UUID) v1.ReadConfig {
		GinkgoHelper()
		var cfg v1.ReadConfig
		Expect(gorp.NewRetrieve[uuid.UUID, v1.ReadConfig]().
			Where(gorp.MatchKeys[uuid.UUID, v1.ReadConfig](key)).
			Entry(&cfg).
			Exec(ctx, db)).To(Succeed())
		return cfg
	}
	storedConfig := func() v0.ReadConfig {
		return v0.ReadConfig{
			Key:   uuid.New(),
			Topic: "plant",
			Records: []v0.ReadRecord{{
				Key:       "r1",
				RecordKey: "pump",
				Fields: []v0.ReadField{storedField(
					"level",
					telem.Float64T,
					payload.EnumEntry{Label: "HALF", Value: 0.5},
				)},
			}},
		}
	}

	It("Should store the shape that v0 stored under the same table name", func() {
		Expect(v1.ReadConfig{}.CustomTypeName()).
			To(Equal(v0.ReadConfig{}.CustomTypeName()))
	})

	It("Should move the fields of each config into an ICD named after its task",
		func(ctx SpecContext) {
			old := storedConfig()
			store(ctx, "Pump station", old, false)
			ins, logs := observe()
			Expect(migrate(ctx, ins)).To(Succeed())
			Expect(made).To(HaveLen(1))
			Expect(made[0].Name).To(Equal("Pump station"))
			Expect(made[0].Messages).To(HaveLen(1))
			Expect(made[0].Messages[0].Name).To(Equal("pump"))
			f := fields(made[0].Messages[0])[0]
			Expect(f.Type()).To(Equal(payload.KindInteger))
			migrated := read(ctx, old.Key)
			Expect(migrated.Inline).To(BeNil())
			Expect(migrated.Topic).To(Equal("plant"))
			Expect(migrated.Records).To(Equal([]v1.ReadRecord{{
				Key:       "r1",
				RecordKey: "pump",
				ICD:       made[0].Key,
				Messages: []v1.ReadMessage{{
					Message:  made[0].Messages[0].Key,
					Bindings: []v1.ReadBinding{{Field: f.Base().Key, Channel: 7}},
				}},
			}}))
			logged := logs.FilterField(zap.String("field", "records.0.fields.0")).All()
			Expect(logged).To(HaveLen(2))
			Expect(logged[0].ContextMap()).To(HaveKeyWithValue(
				"change",
				"data type float64 became int32, the data type of its channel",
			))
		},
	)

	It("Should make no ICD for the config of a snapshot and keep no records",
		func(ctx SpecContext) {
			old := storedConfig()
			storeTask(ctx, old, task.Task{Name: "Pump (snapshot)", Snapshot: true})
			ins, _ := observe()
			Expect(migrate(ctx, ins)).To(Succeed())
			Expect(made).To(BeEmpty())
			migrated := read(ctx, old.Key)
			Expect(migrated.Topic).To(Equal("plant"))
			Expect(migrated.Records).To(BeEmpty())
		},
	)

	It("Should make no ICD for a config without records", func(ctx SpecContext) {
		old := v0.ReadConfig{Key: uuid.New(), Topic: "plant"}
		store(ctx, "Empty", old, false)
		ins, _ := observe()
		Expect(migrate(ctx, ins)).To(Succeed())
		Expect(made).To(BeEmpty())
		Expect(read(ctx, old.Key).Topic).To(Equal("plant"))
	})

	It("Should do nothing without stored configs", func(ctx SpecContext) {
		ins, _ := observe()
		Expect(migrate(ctx, ins)).To(Succeed())
		Expect(made).To(BeEmpty())
	})

	It("Should fail when a config has no task", func(ctx SpecContext) {
		store(ctx, "", storedConfig(), true)
		ins, _ := observe()
		Expect(migrate(ctx, ins)).
			To(MatchError(ContainSubstring("tasks of Kafka read configs")))
	})

	It("Should fail with the config key when an ICD create fails",
		func(ctx SpecContext) {
			old := storedConfig()
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
