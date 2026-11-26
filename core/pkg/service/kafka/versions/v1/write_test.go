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
	"fmt"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	http "github.com/synnaxlabs/synnax/pkg/service/http/versions/v4"
	icdv0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	icd "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v1"
	v0 "github.com/synnaxlabs/synnax/pkg/service/kafka/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/kafka/versions/v1"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
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
			Expect(cfg.Targets).To(BeEmpty())
		},
	)

	It("Should stage nothing for a config with no channels", func(ctx SpecContext) {
		cfg := MustSucceed(v1.MigrateWriteConfig(ctx, writeConfig()))
		Expect(cfg.Inline).To(BeNil())
		Expect(cfg.Device).To(Equal("cluster"))
	})
})

var _ = Describe("ResolveWriteConfig", func() {
	var (
		found    map[channel.Key]icdv0.Channel
		channels icdv0.Channels
		changes  []string
	)
	BeforeEach(func() {
		found = map[channel.Key]icdv0.Channel{
			7: {Name: "valve_cmd", DataType: telem.Float64T},
			8: {Name: "pump_cmd", DataType: telem.Float64T},
		}
		changes = nil
		channels = func(
			context.Context,
			gorp.Snapshot,
			[]channel.Key,
		) (map[channel.Key]icdv0.Channel, error) {
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
			cfg := v1.WriteConfig{Device: "cluster"}
			ic, ok := MustSucceed2(v1.ResolveWriteConfig(
				ctx, nil, channels, "Plant", &cfg, func(string, string) {},
			))
			Expect(ok).To(BeFalse())
			Expect(ic).To(BeZero())
			Expect(cfg).To(Equal(v1.WriteConfig{Device: "cluster"}))
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

	It("Should bind the field at the channel pointer to the name of the channel",
		func(ctx SpecContext) {
			old := writeConfig(writeChannel("c1", 7))
			old.Record.ChannelPointer = new("/channel")
			cfg, ic := lift(ctx, old)
			f := jsonFields(ic.Messages[0])
			Expect(f).To(HaveLen(2))
			Expect(f[1].Variant).To(Equal(icd.StringJSONField{
				JSONBaseField: f[1].Base(),
			}))
			Expect(f[1].Base().Name).To(Equal("channel"))
			Expect(f[1].Base().Pointer).To(Equal("/channel"))
			Expect(cfg.Targets[0].Message.Bindings[1]).To(Equal(icd.WriteBinding{
				Variant: icd.ValueWriteBinding{
					Field: f[1].Base().Key,
					Value: icd.WriteValue{
						Variant: icd.StringWriteValue{Value: "valve_cmd"},
					},
				},
			}))
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
			Expect(cfg.Targets[0].RecordKey).To(BeEmpty())
			Expect(cfg.Targets[0].Message.Bindings).To(Equal([]icd.WriteBinding{
				{Variant: icd.ChannelWriteBinding{
					Field:    f[0].Base().Key,
					DataType: new(telem.Float64T),
				}},
				{Variant: icd.ValueWriteBinding{
					Field: f[1].Base().Key,
					Value: icd.WriteValue{Variant: icd.StringWriteValue{}},
				}},
			}))
		},
	)

	It("Should send a number channel as a boolean when it was sent as one",
		func(ctx SpecContext) {
			found[7] = icdv0.Channel{Name: "valve_cmd", DataType: telem.Uint8T}
			wc := writeChannel("c1", 7)
			wc.JSONType = payload.JSONTypeBoolean
			Expect(liftValue(ctx, wc).Type()).To(Equal(payload.KindBoolean))
		},
	)

	It("Should lift a timestamp channel with its time format", func(ctx SpecContext) {
		found[7] = icdv0.Channel{Name: "valve_cmd", DataType: telem.TimestampT}
		wc := writeChannel("c1", 7)
		wc.TimeFormat = new(payload.TimeFormatUnixMs)
		Expect(variantOf[icd.TimestampJSONField](liftValue(ctx, wc)).TimeFormat).
			To(Equal(payload.TimeFormatUnixMs))
	})

	It("Should keep the enum labels of a number channel sent as a string",
		func(ctx SpecContext) {
			found[7] = icdv0.Channel{Name: "valve_cmd", DataType: telem.Int32T}
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

var _ = Describe("StagedChannelsKVKey", func() {
	It("Should key staged channels by the prefix and the config key", func() {
		key := uuid.New()
		Expect(string(v1.StagedChannelsKVKey(key))).
			To(Equal("sy_kafka_write_staged_channels/" + key.String()))
	})
})

var _ = Describe("WriteMigration", func() {
	var db *gorp.DB
	BeforeEach(func() { db = DeferClose(gorp.Wrap(memkv.New())) })

	migrate := func(ctx SpecContext) {
		GinkgoHelper()
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "kafka_write_config",
			Migrations: []gorp.Migration{v1.WriteMigration},
		})).To(Succeed())
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
	staged := func(key uuid.UUID) v1.InlineWrite {
		GinkgoHelper()
		b, closer := MustSucceed2(db.Get(v1.StagedChannelsKVKey(key)))
		var in v1.InlineWrite
		Expect(json.Unmarshal(b, &in)).To(Succeed())
		Expect(closer.Close()).To(Succeed())
		return in
	}

	It("Should store the shape that v0 stored under the same table name", func() {
		Expect(v1.WriteConfig{}.CustomTypeName()).
			To(Equal(v0.WriteConfig{}.CustomTypeName()))
	})

	It("Should stage the channels of each config and store the config without them",
		func(ctx SpecContext) {
			// A JSON round trip reads no enum values as an empty list.
			ch := writeChannel("c1", 7)
			ch.EnumValues = []payload.EnumEntry{}
			old := writeConfig(ch)
			old.Record.TimestampPointer = new("/time")
			old.Record.Fields = []http.WriteField{
				staticField("/site", payload.JSONTypeString, "north"),
				generatedField("/id", payload.GeneratorTypeUUID, nil),
			}
			Expect(gorp.WrapWriter[uuid.UUID, v0.WriteConfig](db).Set(ctx, old)).
				To(Succeed())
			migrate(ctx)
			cfg := read(ctx, old.Key)
			Expect(cfg.Device).To(Equal(old.Device))
			Expect(cfg.Targets).To(BeEmpty())
			Expect(cfg.Inline).To(BeNil())
			Expect(staged(old.Key)).To(Equal(v1.InlineWrite{
				Topic:     "plant",
				RecordKey: v1.RecordKeyChannelName,
				Record:    old.Record,
				Channels:  old.Channels,
			}))
		},
	)

	It("Should stage nothing for a config without channels", func(ctx SpecContext) {
		old := writeConfig()
		Expect(gorp.WrapWriter[uuid.UUID, v0.WriteConfig](db).Set(ctx, old)).
			To(Succeed())
		migrate(ctx)
		Expect(read(ctx, old.Key).Device).To(Equal(old.Device))
		Expect(db.Get(v1.StagedChannelsKVKey(old.Key))).Error().
			To(MatchError(query.ErrNotFound))
	})

	It("Should do nothing without stored configs", func(ctx SpecContext) {
		migrate(ctx)
		var cfgs []v1.WriteConfig
		Expect(gorp.NewRetrieve[uuid.UUID, v1.WriteConfig]().
			Entries(&cfgs).
			Exec(ctx, db)).To(Succeed())
		Expect(cfgs).To(BeEmpty())
	})
})
