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
	"encoding/json/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	icd "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	v0 "github.com/synnaxlabs/synnax/pkg/service/kafka/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/kafka/versions/v1"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/query"
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

// inlineConfig returns a config that holds records of the topic plant inline.
func inlineConfig(records ...v1.InlineRecord) v1.ReadConfig {
	return v1.ReadConfig{Inline: &v1.InlineRead{Topic: "plant", Records: records}}
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
			Expect(cfg.Topics).To(HaveLen(1))
			Expect(cfg.Topics[0].Name).To(Equal("plant"))
			Expect(cfg.Topics[0].Key).ToNot(BeEmpty())
			Expect(cfg.Topics[0].Records).To(HaveLen(3))
			for i, rec := range cfg.Topics[0].Records {
				Expect(rec.ICD).To(Equal(ic.Key))
				Expect(rec.Messages).To(HaveLen(1))
				Expect(rec.Messages[0].Message).To(Equal(ic.Messages[i].Key))
			}
			Expect(cfg.Topics[0].Records[0].Key).To(Equal("r1"))
			Expect(cfg.Topics[0].Records[0].RecordKey).To(Equal("pump"))
			Expect(cfg.Topics[0].Records[2].RecordKey).To(BeEmpty())
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
			Expect(
				cfg.Topics[0].Records[0].Messages[0].Bindings,
			).To(Equal([]v1.ReadBinding{
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
			Expect(
				cfg.Topics[0].Records[0].Messages[0].Bindings,
			).To(Equal([]v1.ReadBinding{
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
			Expect(
				cfg.Topics[0].Records[0].Messages[0].Index,
			).To(Equal(new(f.Base().Key)))
		},
	)

	It("Should drop an index that names no field", func(ctx SpecContext) {
		cfg := inlineConfig(v1.InlineRecord{
			Key:    "r1",
			Index:  "gone",
			Fields: []v1.ReadField{storedField("level", telem.Float64T)},
		})
		resolve(ctx, &cfg)
		Expect(cfg.Topics[0].Records[0].Messages[0].Index).To(BeNil())
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
			Expect(cfg.Topics[0].Records[0].Disabled).To(BeFalse())
			Expect(cfg.Topics[0].Records[0].Messages[0].Bindings).To(HaveLen(1))
		},
	)

	It("Should disable a record whose fields are all disabled", func(ctx SpecContext) {
		off := storedField("off", telem.Float64T)
		off.Disabled = true
		cfg := inlineConfig(v1.InlineRecord{Key: "r1", Fields: []v1.ReadField{off}})
		resolve(ctx, &cfg)
		Expect(cfg.Topics[0].Records[0].Disabled).To(BeTrue())
		Expect(cfg.Topics[0].Records[0].Messages[0].Bindings).To(BeEmpty())
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

	It("Should add the records to the topic of the config with the inline topic",
		func(ctx SpecContext) {
			cfg := inlineConfig(v1.InlineRecord{
				Key:    "r1",
				Fields: []v1.ReadField{storedField("level", telem.Float64T)},
			})
			cfg.Topics = []v1.ReadTopic{
				{Key: "t0", Name: "other", Records: []v1.ReadRecord{}},
				{Key: "t1", Name: "plant", Records: []v1.ReadRecord{}},
			}
			resolve(ctx, &cfg)
			Expect(cfg.Topics).To(HaveLen(2))
			Expect(cfg.Topics[0].Records).To(BeEmpty())
			Expect(cfg.Topics[1].Key).To(Equal("t1"))
			Expect(cfg.Topics[1].Records).To(HaveLen(1))
			Expect(cfg.Topics[1].Records[0].Key).To(Equal("r1"))
		},
	)

	It("Should resolve nothing when the config holds no inline records",
		func(ctx SpecContext) {
			topics := []v1.ReadTopic{{
				Key:     "t1",
				Name:    "plant",
				Records: []v1.ReadRecord{{Key: "r1"}},
			}}
			cfg := v1.ReadConfig{Topics: topics}
			_, ok := MustSucceed2(v1.ResolveReadConfig(
				ctx, nil, channels, "Plant", &cfg, func(string, string) {},
			))
			Expect(ok).To(BeFalse())
			Expect(cfg.Topics).To(Equal(topics))
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
			Expect(cfg.Group).To(Equal("group"))
			Expect(cfg.StartOffset).To(Equal(v1.StartOffsetEarliest))
			Expect(cfg.Topics).To(HaveLen(1))
			Expect(cfg.Topics[0].Name).To(Equal("plant"))
			Expect(cfg.Topics[0].Records).To(BeEmpty())
			Expect(cfg.Inline).To(Equal(&v1.InlineRead{
				Topic: "plant",
				Records: []v1.InlineRecord{{
					Key:       "r1",
					RecordKey: "pump",
					Index:     "ts",
					Fields:    []v1.ReadField{storedField("level", telem.Uint8T)},
				}},
			}))
		},
	)

	It(
		"Should keep the topic of a config without records and hold nothing inline",
		func(ctx SpecContext) {
			cfg := MustSucceed(v1.MigrateReadConfig(ctx, v0.ReadConfig{Topic: "plant"}))
			Expect(cfg.Inline).To(BeNil())
			Expect(cfg.Topics).To(HaveLen(1))
			Expect(cfg.Topics[0].Name).To(Equal("plant"))
			Expect(cfg.Topics[0].Records).To(BeEmpty())
		},
	)

	It("Should add no topic for a config without one", func(ctx SpecContext) {
		cfg := MustSucceed(v1.MigrateReadConfig(ctx, v0.ReadConfig{}))
		Expect(cfg.Topics).To(BeEmpty())
	})
})

// storedConfig returns a v0 config with one record of one field, bound to channel 7.
func storedConfig() v0.ReadConfig {
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

var _ = Describe("StagedRecordsKVKey", func() {
	It("Should key staged records by the prefix and the config key", func() {
		key := uuid.New()
		Expect(string(v1.StagedRecordsKVKey(key))).
			To(Equal("sy_kafka_read_staged_records/" + key.String()))
	})
})

var _ = Describe("ReadMigration", func() {
	var db *gorp.DB
	BeforeEach(func() { db = DeferClose(gorp.Wrap(memkv.New())) })

	migrate := func(ctx SpecContext, olds ...v0.ReadConfig) {
		GinkgoHelper()
		Expect(gorp.WrapWriter[uuid.UUID, v0.ReadConfig](db).Set(ctx, olds...)).
			To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "kafka_read_config",
			Migrations: []gorp.Migration{v1.ReadMigration},
		})).To(Succeed())
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

	It("Should store the shape that v0 stored under the same table name", func() {
		Expect(v1.ReadConfig{}.CustomTypeName()).
			To(Equal(v0.ReadConfig{}.CustomTypeName()))
	})

	It("Should stage the records of each config and store the config without them",
		func(ctx SpecContext) {
			old := storedConfig()
			migrate(ctx, old)
			migrated := read(ctx, old.Key)
			Expect(migrated.Topics).To(HaveLen(1))
			Expect(migrated.Topics[0].Name).To(Equal("plant"))
			Expect(migrated.Topics[0].Records).To(BeEmpty())
			Expect(migrated.Inline).To(BeNil())
			b, closer := MustSucceed2(db.Get(v1.StagedRecordsKVKey(old.Key)))
			var staged v1.InlineRead
			Expect(json.Unmarshal(b, &staged)).To(Succeed())
			Expect(closer.Close()).To(Succeed())
			Expect(staged).To(Equal(v1.InlineRead{
				Topic: "plant",
				Records: []v1.InlineRecord{{
					Key:       "r1",
					RecordKey: "pump",
					Fields:    old.Records[0].Fields,
				}},
			}))
		},
	)

	It("Should stage nothing for a config without records", func(ctx SpecContext) {
		old := v0.ReadConfig{Key: uuid.New(), Topic: "plant"}
		migrate(ctx, old)
		topics := read(ctx, old.Key).Topics
		Expect(topics).To(HaveLen(1))
		Expect(topics[0].Name).To(Equal("plant"))
		Expect(db.Get(v1.StagedRecordsKVKey(old.Key))).Error().
			To(MatchError(query.ErrNotFound))
	})
})
