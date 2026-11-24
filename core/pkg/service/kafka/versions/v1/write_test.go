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
	http "github.com/synnaxlabs/synnax/pkg/service/http/versions/v4"
	icd "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v1"
	v0 "github.com/synnaxlabs/synnax/pkg/service/kafka/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/kafka/versions/v1"
	task "github.com/synnaxlabs/synnax/pkg/service/task/versions/v3"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"go.uber.org/zap"
)

// jsonFields returns the fields of the JSON message msg.
func jsonFields(msg icd.Message) []icd.JSONField {
	GinkgoHelper()
	p, ok := msg.Payload.Variant.(icd.JSONPayload)
	Expect(ok).To(BeTrue())
	return p.Fields
}

// variantOf returns the variant of f, which must be a T.
func variantOf[T icd.JSONFieldVariant](f icd.JSONField) T {
	GinkgoHelper()
	v, ok := f.Variant.(T)
	return MustBeOk(v, ok)
}

// writeChannel returns a v0 channel with key that sends ch as a JSON number.
func writeChannel(key string, ch channel.Key) v0.WriteChannel {
	return v0.WriteChannel{Key: key, Channel: ch, JSONType: payload.JSONTypeNumber}
}

func staticField(pointer string, t payload.JSONType, value any) http.WriteField {
	return http.WriteField{Variant: http.StaticWriteField{
		BaseWriteField: http.BaseWriteField{Key: pointer, Pointer: pointer},
		JSONType:       t,
		Value:          value,
	}}
}

func generatedField(
	pointer string,
	g payload.GeneratorType,
	format *payload.TimeFormat,
) http.WriteField {
	return http.WriteField{Variant: http.GeneratedWriteField{
		BaseWriteField: http.BaseWriteField{Key: pointer, Pointer: pointer},
		Generator:      g,
		TimeFormat:     format,
	}}
}

// writeConfig returns a v0 config that sends the samples of channels to the topic
// plant, with the default record.
func writeConfig(channels ...v0.WriteChannel) v0.WriteConfig {
	return v0.WriteConfig{
		Key:       uuid.New(),
		Device:    "cluster",
		Topic:     "plant",
		RecordKey: v0.RecordKeyChannelName,
		Record: v0.Record{
			ValuePointer: "/value",
			TimeFormat:   payload.TimeFormatUnixNs,
			Fields:       []http.WriteField{},
		},
		Channels: channels,
	}
}

var _ = Describe("MigrateWriteConfig", func() {
	It("Should move the channels, topic, record key, and record of a config inline",
		func(ctx SpecContext) {
			old := writeConfig(writeChannel("c1", 7))
			old.AutoStart, old.RecordKey = true, v0.RecordKeyNone
			old.Record.ValuePointer = "/v"
			cfg := MustSucceed(v1.MigrateWriteConfig(ctx, old))
			Expect(cfg.Inline).To(Equal(&v1.InlineWrite{
				Topic:     "plant",
				RecordKey: v1.RecordKeyNone,
				Record:    old.Record,
				Channels:  old.Channels,
			}))
			Expect(cfg.StartConfig).To(Equal(old.StartConfig))
			Expect(cfg.Device).To(Equal(old.Device))
			Expect(cfg.Topic).To(BeEmpty())
			Expect(cfg.RecordKey).To(Equal(v1.RecordKeyChannelName))
			Expect(cfg.Record).To(Equal(v1.Record{
				ValuePointer: "/value",
				TimeFormat:   payload.TimeFormatUnixNs,
				Fields:       []http.WriteField{},
			}))
			Expect(cfg.Channels).To(BeEmpty())
			Expect(cfg.Targets).To(BeEmpty())
		},
	)

	It("Should stage nothing for a config with no channels", func(ctx SpecContext) {
		cfg := MustSucceed(v1.MigrateWriteConfig(ctx, writeConfig()))
		Expect(cfg.Inline).To(BeNil())
		Expect(cfg.Topic).To(Equal("plant"))
	})
})

var _ = Describe("ResolveWriteConfig", func() {
	var (
		found    map[channel.Key]v1.Channel
		channels v1.Channels
		changes  []string
	)
	BeforeEach(func() {
		found = map[channel.Key]v1.Channel{
			7: {Name: "valve_cmd", DataType: telem.Float64T},
			8: {Name: "pump_cmd", DataType: telem.Float64T},
		}
		changes = nil
		channels = func(
			context.Context,
			gorp.Snapshot,
			[]channel.Key,
		) (map[channel.Key]v1.Channel, error) {
			return found, nil
		}
	})

	// lift migrates old and resolves it for the task "Plant". It returns the config and
	// the ICD to create.
	lift := func(ctx SpecContext, old v0.WriteConfig) (v1.WriteConfig, icd.ICD) {
		GinkgoHelper()
		cfg := MustSucceed(v1.MigrateWriteConfig(ctx, old))
		ic, ok := MustSucceed2(v1.ResolveWriteConfig(
			ctx,
			nil,
			channels,
			"Plant",
			&cfg,
			func(path, change string) { changes = append(changes, path+": "+change) },
		))
		Expect(ok).To(BeTrue())
		return cfg, ic
	}
	// liftFields lifts a config whose only channel sends channel 7 in rec, and returns
	// the fields of its message.
	liftFields := func(ctx SpecContext, rec v0.Record) []icd.JSONField {
		GinkgoHelper()
		old := writeConfig(writeChannel("c1", 7))
		old.Record = rec
		_, ic := lift(ctx, old)
		return jsonFields(ic.Messages[0])
	}
	// liftValue lifts a config whose only channel is wc, and returns its value field.
	liftValue := func(ctx SpecContext, wc v0.WriteChannel) icd.JSONField {
		GinkgoHelper()
		_, ic := lift(ctx, writeConfig(wc))
		return jsonFields(ic.Messages[0])[0]
	}
	record := func() v0.Record { return writeConfig().Record }

	It("Should return false and keep a config with no inline channels",
		func(ctx SpecContext) {
			cfg := v1.WriteConfig{Topic: "plant"}
			ic, ok := MustSucceed2(v1.ResolveWriteConfig(
				ctx, nil, channels, "Plant", &cfg, func(string, string) {},
			))
			Expect(ok).To(BeFalse())
			Expect(ic).To(BeZero())
			Expect(cfg).To(Equal(v1.WriteConfig{Topic: "plant"}))
		},
	)

	It("Should add a target for each channel that sends its own message of the ICD",
		func(ctx SpecContext) {
			pump := writeChannel("c2", 8)
			pump.Disabled = true
			cfg, ic := lift(ctx, writeConfig(writeChannel("c1", 7), pump))
			Expect(cfg.Inline).To(BeNil())
			Expect(ic.Name).To(Equal("Plant"))
			Expect(ic.Messages).To(HaveLen(2))
			Expect(ic.Messages[0].Name).To(Equal("valve_cmd"))
			Expect(ic.Messages[1].Name).To(Equal("pump_cmd"))
			value := jsonFields(ic.Messages[0])
			Expect(value).To(HaveLen(1))
			Expect(value[0].Type()).To(Equal(payload.KindFloat))
			Expect(value[0].Base().Name).To(Equal("value"))
			Expect(value[0].Base().Pointer).To(Equal("/value"))
			Expect(value[0].Base().Scale).To(Equal(1.0))
			bound := func(i int, ch channel.Key) icd.WriteMessage {
				return icd.WriteMessage{
					Message: ic.Messages[i].Key,
					Bindings: []icd.WriteBinding{{Variant: icd.ChannelWriteBinding{
						Field:   jsonFields(ic.Messages[i])[0].Base().Key,
						Channel: ch,
					}}},
				}
			}
			Expect(cfg.Targets).To(Equal([]v1.WriteTarget{
				{
					Key:       "c1",
					Topic:     "plant",
					RecordKey: "valve_cmd",
					ICD:       ic.Key,
					Message:   bound(0, 7),
				},
				{
					Key:       "c2",
					Disabled:  true,
					Topic:     "plant",
					RecordKey: "pump_cmd",
					ICD:       ic.Key,
					Message:   bound(1, 8),
				},
			}))
			Expect(changes).To(BeEmpty())
		},
	)

	It("Should send no record key when the config sent none", func(ctx SpecContext) {
		old := writeConfig(writeChannel("c1", 7))
		old.RecordKey = v0.RecordKeyNone
		cfg, _ := lift(ctx, old)
		Expect(cfg.Targets[0].RecordKey).To(BeEmpty())
	})

	It("Should create the key of a target whose channel has none", func(
		ctx SpecContext,
	) {
		cfg, _ := lift(ctx, writeConfig(writeChannel("", 7)))
		Expect(uuid.Parse(cfg.Targets[0].Key)).Error().ToNot(HaveOccurred())
	})

	It("Should give each message a unique name", func(ctx SpecContext) {
		_, ic := lift(ctx, writeConfig(writeChannel("c1", 7), writeChannel("c2", 7)))
		Expect(ic.Messages[0].Name).To(Equal("valve_cmd"))
		Expect(ic.Messages[1].Name).To(Equal("valve_cmd (2)"))
	})

	It("Should name the value field after its pointer", func(ctx SpecContext) {
		rec := record()
		rec.ValuePointer = "/reading"
		f := liftFields(ctx, rec)
		Expect(f[0].Base().Name).To(Equal("reading"))
		Expect(f[0].Base().Pointer).To(Equal("/reading"))
	})

	It("Should send the name of the channel at the channel pointer",
		func(ctx SpecContext) {
			rec := record()
			rec.ChannelPointer = new("/channel")
			f := liftFields(ctx, rec)
			Expect(f).To(HaveLen(2))
			Expect(f[1].Variant).To(Equal(icd.StringJSONField{
				JSONBaseField: f[1].Base(),
				Default:       new("valve_cmd"),
			}))
			Expect(f[1].Base().Name).To(Equal("channel"))
			Expect(f[1].Base().Pointer).To(Equal("/channel"))
		},
	)

	It("Should send the time of the write at the timestamp pointer and note it once",
		func(ctx SpecContext) {
			old := writeConfig(writeChannel("c1", 7), writeChannel("c2", 8))
			old.Record.TimestampPointer = new("/time")
			old.Record.TimeFormat = payload.TimeFormatUnixMs
			_, ic := lift(ctx, old)
			for _, m := range ic.Messages {
				f := jsonFields(m)
				Expect(f).To(HaveLen(2))
				Expect(f[1].Variant).To(Equal(icd.TimestampJSONField{
					JSONBaseField: f[1].Base(),
					TimeFormat:    payload.TimeFormatUnixMs,
					Default: &icd.TimestampDefault{
						Variant: icd.NowTimestampDefault{},
					},
				}))
			}
			Expect(changes).To(Equal([]string{
				"record.timestamp_pointer: the field sends the time of the write, " +
					"not the time of the sample",
			}))
		},
	)

	It("Should send ISO 8601 for a record time format that is not valid",
		func(ctx SpecContext) {
			rec := record()
			rec.TimestampPointer = new("/time")
			rec.TimeFormat = "bogus"
			f := liftFields(ctx, rec)
			Expect(variantOf[icd.TimestampJSONField](f[1]).TimeFormat).
				To(Equal(payload.TimeFormatISO8601))
			Expect(changes).To(ContainElement(fmt.Sprintf(
				`record.time_format: time format "bogus" is not valid and became %s`,
				payload.TimeFormatISO8601,
			)))
		},
	)

	It("Should lift the static and generated fields of the record",
		func(ctx SpecContext) {
			rec := record()
			rec.Fields = []http.WriteField{
				staticField("/mode", payload.JSONTypeString, "auto"),
				generatedField("/id", payload.GeneratorTypeUUID, nil),
			}
			f := liftFields(ctx, rec)
			Expect(f).To(HaveLen(3))
			Expect(*variantOf[icd.StringJSONField](f[1]).Default).To(Equal("auto"))
			Expect(f[1].Base().Pointer).To(Equal("/mode"))
			Expect(f[2].Variant).To(Equal(icd.UuidJSONField{
				JSONBaseField: f[2].Base(),
				Default:       &icd.UUIDDefault{Variant: icd.RandomUUIDDefault{}},
			}))
		},
	)

	It("Should give each message its own copy of the fields of the record",
		func(ctx SpecContext) {
			old := writeConfig(writeChannel("c1", 7), writeChannel("c2", 8))
			old.Record.Fields = []http.WriteField{
				staticField("/mode", payload.JSONTypeString, "auto"),
			}
			_, ic := lift(ctx, old)
			first, second := jsonFields(ic.Messages[0]), jsonFields(ic.Messages[1])
			Expect(second).To(HaveLen(2))
			Expect(second[1].Base().Name).To(Equal("mode"))
			Expect(second[1].Base().Key).ToNot(Equal(first[1].Base().Key))
		},
	)

	It(
		"Should drop a null field of the record and note it once",
		func(ctx SpecContext) {
			old := writeConfig(writeChannel("c1", 7), writeChannel("c2", 8))
			old.Record.Fields = []http.WriteField{{}}
			_, ic := lift(ctx, old)
			Expect(jsonFields(ic.Messages[0])).To(HaveLen(1))
			Expect(jsonFields(ic.Messages[1])).To(HaveLen(1))
			Expect(
				changes,
			).To(Equal([]string{"record.fields.0: a null field was dropped"}))
		},
	)

	It("Should bind no channel and take float64 for a channel that does not exist",
		func(ctx SpecContext) {
			old := writeConfig(writeChannel("c1", 12))
			old.Record.ChannelPointer = new("/channel")
			cfg, ic := lift(ctx, old)
			msg := ic.Messages[0]
			Expect(msg.Name).To(Equal("Untitled"))
			f := jsonFields(msg)
			Expect(f[0].Type()).To(Equal(payload.KindFloat))
			Expect(*variantOf[icd.StringJSONField](f[1]).Default).To(BeEmpty())
			Expect(cfg.Targets[0].RecordKey).To(BeEmpty())
			Expect(cfg.Targets[0].Message.Bindings).To(Equal([]icd.WriteBinding{{
				Variant: icd.ChannelWriteBinding{
					Field:    f[0].Base().Key,
					DataType: new(telem.Float64T),
				},
			}}))
		},
	)

	It("Should send a number channel as a boolean when it was sent as one",
		func(ctx SpecContext) {
			found[7] = v1.Channel{Name: "valve_cmd", DataType: telem.Uint8T}
			wc := writeChannel("c1", 7)
			wc.JSONType = payload.JSONTypeBoolean
			Expect(liftValue(ctx, wc).Type()).To(Equal(payload.KindBoolean))
		},
	)

	It("Should lift a timestamp channel with its time format", func(ctx SpecContext) {
		found[7] = v1.Channel{Name: "valve_cmd", DataType: telem.TimestampT}
		wc := writeChannel("c1", 7)
		wc.TimeFormat = new(payload.TimeFormatUnixMs)
		Expect(variantOf[icd.TimestampJSONField](liftValue(ctx, wc)).TimeFormat).
			To(Equal(payload.TimeFormatUnixMs))
	})

	It("Should keep the enum labels of a number channel sent as a string",
		func(ctx SpecContext) {
			found[7] = v1.Channel{Name: "valve_cmd", DataType: telem.Int32T}
			wc := writeChannel("c1", 7)
			wc.JSONType = payload.JSONTypeString
			wc.EnumValues = []payload.EnumEntry{
				{Label: "OPEN", Value: 1},
				{Label: "HALF", Value: 0.5},
			}
			Expect(variantOf[icd.IntegerJSONField](liftValue(ctx, wc)).EnumValues).
				To(Equal([]payload.IntegerEntry{{Label: "OPEN", Value: 1}}))
			Expect(changes).To(Equal([]string{
				`channels.0: enum label "HALF" was dropped: 0.5 is not integer`,
			}))
		},
	)
})

var _ = Describe("NewWriteMigration", func() {
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
				7: {Name: "valve_cmd", DataType: telem.Float64T},
			}, nil
		}
	})

	// store writes old and a task named name that owns it.
	store := func(ctx SpecContext, name string, old v0.WriteConfig, snapshot bool) {
		GinkgoHelper()
		Expect(gorp.WrapWriter[uuid.UUID, v0.WriteConfig](db).Set(ctx, old)).
			To(Succeed())
		Expect(gorp.WrapWriter[task.Key, task.Task](db).Set(ctx, task.Task{
			Key:      old.Key,
			Name:     name,
			Type:     "kafka_write",
			Snapshot: snapshot,
		})).To(Succeed())
	}
	migrate := func(ctx SpecContext, ins alamos.Instrumentation) error {
		return gorp.Migrate(ctx, gorp.MigrateConfig{
			Instrumentation: ins,
			DB:              db,
			Namespace:       "kafka_write_config",
			Migrations: []gorp.Migration{
				v1.WriteMigration,
				v1.NewWriteMigration(channels, create),
			},
		})
	}
	read := func(ctx SpecContext, key uuid.UUID) v1.WriteConfig {
		GinkgoHelper()
		var cfg v1.WriteConfig
		Expect(gorp.NewRetrieve[uuid.UUID, v1.WriteConfig]().
			Where(gorp.MatchKeys[uuid.UUID, v1.WriteConfig](key)).
			Entry(&cfg).
			Exec(ctx, db)).To(Succeed())
		return cfg
	}

	It(
		"Should move the channels of a config into targets of an ICD named after its task",
		func(ctx SpecContext) {
			old := writeConfig(writeChannel("c1", 7))
			old.Record.TimestampPointer = new("/time")
			store(ctx, "Valves", old, false)
			ins, logs := observe()
			Expect(migrate(ctx, ins)).To(Succeed())
			Expect(made).To(HaveLen(1))
			Expect(made[0].Name).To(Equal("Valves"))
			cfg := read(ctx, old.Key)
			Expect(cfg.Channels).To(BeEmpty())
			Expect(cfg.Inline).To(BeNil())
			Expect(cfg.Targets).To(HaveLen(1))
			Expect(cfg.Targets[0].ICD).To(Equal(made[0].Key))
			Expect(cfg.Targets[0].Message.Message).To(Equal(made[0].Messages[0].Key))
			logged := logs.FilterField(
				zap.String("field", "record.timestamp_pointer"),
			).All()
			Expect(logged).To(HaveLen(1))
			Expect(logged[0].ContextMap()).
				To(HaveKeyWithValue("config", old.Key.String()))
		},
	)

	It("Should leave a config with no channels as it was", func(ctx SpecContext) {
		old := writeConfig()
		store(ctx, "Empty", old, false)
		ins, _ := observe()
		Expect(migrate(ctx, ins)).To(Succeed())
		Expect(made).To(BeEmpty())
		Expect(read(ctx, old.Key).Topic).To(Equal("plant"))
	})

	It("Should make no ICD for the config of a snapshot", func(ctx SpecContext) {
		old := writeConfig(writeChannel("c1", 7))
		store(ctx, "Valves (snapshot)", old, true)
		ins, _ := observe()
		Expect(migrate(ctx, ins)).To(Succeed())
		Expect(made).To(BeEmpty())
		cfg := read(ctx, old.Key)
		Expect(cfg.Channels).To(BeEmpty())
		Expect(cfg.Targets).To(BeEmpty())
	})

	It("Should do nothing without stored configs", func(ctx SpecContext) {
		ins, _ := observe()
		Expect(migrate(ctx, ins)).To(Succeed())
		Expect(made).To(BeEmpty())
	})

	It("Should fail when a config has no task", func(ctx SpecContext) {
		Expect(gorp.WrapWriter[uuid.UUID, v0.WriteConfig](db).
			Set(ctx, writeConfig(writeChannel("c1", 7)))).To(Succeed())
		ins, _ := observe()
		Expect(migrate(ctx, ins)).
			To(MatchError(ContainSubstring("tasks of Kafka write configs")))
	})

	It("Should fail with the config key when an ICD create fails",
		func(ctx SpecContext) {
			old := writeConfig(writeChannel("c1", 7))
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
