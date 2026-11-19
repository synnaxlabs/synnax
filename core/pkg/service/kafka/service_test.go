// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kafka_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/kafka"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	tasktestutil "github.com/synnaxlabs/synnax/pkg/service/task/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// binding returns the config that writes field i of m to ch.
func binding(m icd.Message, i int, ch channel.Channel) map[string]any {
	return map[string]any{"field": fieldKey(m, i), "channel": ch.Key()}
}

// readMessage returns the config that reads m through bindings.
func readMessage(m icd.Message, bindings ...map[string]any) map[string]any {
	list := make([]any, len(bindings))
	for i, b := range bindings {
		list[i] = b
	}
	return map[string]any{"message": m.Key, "bindings": list}
}

// icdRecord returns the config of a record with recordKey that reads messages of ic.
func icdRecord(
	recordKey string,
	ic icd.ICD,
	messages ...map[string]any,
) map[string]any {
	list := make([]any, len(messages))
	for i, m := range messages {
		list[i] = m
	}
	return map[string]any{
		"key":        "r-" + recordKey,
		"record_key": recordKey,
		"icd":        ic.Key,
		"messages":   list,
	}
}

var _ = Describe("Service", func() {
	var (
		db       *gorp.DB
		node     testutil.Node
		channels *channel.Service
		icds     *icd.Service
		svc      *kafka.Service
		tasks    *tasktestutil.Tasks
	)
	BeforeEach(func(ctx SpecContext) {
		// The suite's Core holds a kafka service and a task service of its own.
		node = testutil.NewNode(ctx)
		db = node.DB
		MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: db}))
		statuses := MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
		channels = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
			Channel: node.Channel,
			DB:      db,
			HostKey: node.Identity.HostKey,
			Status:  statuses,
		}))
		icds = MustOpen(icd.OpenService(ctx, icd.ServiceConfig{
			DB:   db,
			ImEx: imex.NewService(),
		}))
		svc = MustOpen(kafka.OpenService(ctx, kafka.ServiceConfig{
			DB:      db,
			Channel: channels,
			ICD:     icds,
		}))
		tasks = MustOpen(tasktestutil.OpenTasks(ctx, tasktestutil.TasksConfig{
			DB:     db,
			Status: statuses,
			Stores: svc.Stores(),
		}))
	})

	Describe("OpenService", func() {
		It("Should reject a config missing the DB", func(ctx SpecContext) {
			Expect(kafka.OpenService(ctx, kafka.ServiceConfig{})).Error().
				To(MatchError(ContainSubstring("db: must be non-nil")))
		})

		It("Should reject a config missing the channel service", func(ctx SpecContext) {
			Expect(kafka.OpenService(ctx, kafka.ServiceConfig{DB: db})).Error().
				To(MatchError(ContainSubstring("channel: must be non-nil")))
		})

		It("Should reject a config missing the ICD service", func(ctx SpecContext) {
			Expect(kafka.OpenService(ctx, kafka.ServiceConfig{
				DB:      db,
				Channel: channels,
			})).Error().To(MatchError(ContainSubstring("icd: must be non-nil")))
		})
	})

	Describe("Stores", func() {
		It("Should expose one store per Kafka task type", func() {
			stores := svc.Stores()
			types := make([]string, len(stores))
			for i, s := range stores {
				types[i] = s.Type()
			}
			Expect(types).To(ConsistOf(
				kafka.ReadTaskType,
				kafka.WriteTaskType,
				kafka.ScanTaskType,
			))
		})
	})

	Describe("Normalize", func() {
		DescribeTable(
			"Should import a config at versions 0 and 1 and reject a later version",
			func(ctx SpecContext, store func(*kafka.Service) config.Store) {
				s := store(svc)
				Expect(s.Version()).To(BeEquivalentTo(1))
				MustSucceed(s.Normalize(ctx, db, 0, msgpack.EncodedJSON{}))
				MustSucceed(s.Normalize(ctx, db, 1, msgpack.EncodedJSON{}))
				newer := " version 2 is newer than this Core supports (latest: 1)"
				Expect(s.Normalize(ctx, db, 2, msgpack.EncodedJSON{})).Error().
					To(MatchError(ContainSubstring(s.Type() + newer)))
			},
			Entry("read", func(s *kafka.Service) config.Store { return s.Read }),
			Entry("write", func(s *kafka.Service) config.Store { return s.Write }),
			Entry("scan", func(s *kafka.Service) config.Store { return s.Scan }),
		)
	})

	Describe("Write", func() {
		It("Should store a read topic and record with their schema defaults", func(
			ctx SpecContext,
		) {
			Expect(tasks.CreateDevice(ctx, "dev-1")).To(Succeed())
			ic := icd.ICD{Name: "Plant"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return icds.NewWriter(tx).Create(ctx, &ic)
			})).To(Succeed())
			key := MustSucceed(tasks.Create(ctx, svc.Read, msgpack.EncodedJSON{
				"device": "dev-1",
				"topics": []any{map[string]any{
					"key":     "t-1",
					"name":    "telemetry",
					"records": []any{map[string]any{"key": "r-1", "icd": ic.Key}},
				}},
			}))
			data := MustSucceed(svc.Read.Read(ctx, db, key))
			Expect(data["key"]).To(Equal(key.String()))
			Expect(data["device"]).To(Equal("dev-1"))
			Expect(data["start_offset"]).To(Equal("latest"))
			Expect(data).ToNot(HaveKey("inline"))
			Expect(data["topics"]).To(HaveExactElements(SatisfyAll(
				HaveKeyWithValue("key", "t-1"),
				HaveKeyWithValue("name", "telemetry"),
				HaveKeyWithValue("records", HaveExactElements(SatisfyAll(
					HaveKeyWithValue("record_key", ""),
					HaveKeyWithValue("disabled", false),
					HaveKeyWithValue("icd", ic.Key.String()),
					HaveKeyWithValue("messages", BeEmpty()),
				))),
			)))
		})

		It("Should apply write config schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.Write, msgpack.EncodedJSON{
				"channels": []any{map[string]any{"key": "ch-1"}},
			}))
			data := MustSucceed(svc.Write.Read(ctx, db, key))
			Expect(data["record_key"]).To(Equal("channel_name"))
			Expect(data["record"]).To(And(
				HaveKeyWithValue("value_pointer", "/value"),
				HaveKeyWithValue("time_format", "unix_ns"),
				Not(HaveKey("channel_pointer")),
				Not(HaveKey("timestamp_pointer")),
			))
			Expect(data["channels"]).To(HaveExactElements(
				HaveKeyWithValue("json_type", "number"),
			))
		})

		It("Should keep the record pointers the caller sets", func(ctx SpecContext) {
			key := MustSucceed(tasks.Create(ctx, svc.Write, msgpack.EncodedJSON{
				"record": map[string]any{
					"channel_pointer":   "/name",
					"timestamp_pointer": "/time",
				},
			}))
			data := MustSucceed(svc.Write.Read(ctx, db, key))
			Expect(data["record"]).To(And(
				HaveKeyWithValue("channel_pointer", "/name"),
				HaveKeyWithValue("timestamp_pointer", "/time"),
			))
		})

		It("Should apply scan config schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.Scan, msgpack.EncodedJSON{}))
			data := MustSucceed(svc.Scan.Read(ctx, db, key))
			Expect(data["rate"]).To(BeNumerically("==", 0.2))
		})

		It("Should return the read validation error for an invalid start offset",
			func(ctx SpecContext) {
				Expect(svc.Read.Prepare(ctx, db, uuid.New(), "", msgpack.EncodedJSON{
					"start_offset": "bogus",
				})).Error().To(MatchError(ContainSubstring("invalid start_offset: bogus")))
			},
		)

		It("Should return the write validation error for an invalid record key",
			func(ctx SpecContext) {
				Expect(svc.Write.Prepare(ctx, db, uuid.New(), "", msgpack.EncodedJSON{
					"record_key": "bogus",
				})).Error().To(MatchError(ContainSubstring("invalid record_key: bogus")))
			},
		)
	})

	Describe("Read config", func() {
		// createChannel creates a channel of dt on the index channel idx, or an index
		// channel when idx is the zero channel.
		createChannel := func(
			ctx context.Context,
			dt telem.DataType,
			idx channel.Channel,
		) channel.Channel {
			GinkgoHelper()
			ch := channel.Channel{
				Name:        UniqueChannelName(),
				DataType:    dt,
				Leaseholder: node.Identity.HostKey,
				IsIndex:     idx.LocalKey == 0,
				LocalIndex:  idx.LocalKey,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channels.NewWriter(tx).Create(ctx, &ch)
			})).To(Succeed())
			return ch
		}

		// format is an ICD with a Status message of a timestamp, a float, a string, and
		// a bytes field at the root, a Stamped message of a timestamp with a default, a
		// binary Frame message, and channels on one index to bind them to.
		type format struct {
			ic                icd.ICD
			m, stamped, frame icd.Message
			idx, value, stamp channel.Channel
		}

		createFormat := func(ctx context.Context) format {
			GinkgoHelper()
			m := jsonMessage("Status", nil,
				jsonField("ts", "/ts", icd.TimestampJSONField{
					TimeFormat: payload.TimeFormatUnixSec,
				}),
				jsonField("temp", "/temp", icd.FloatJSONField{}),
				jsonField("state", "/state", icd.StringJSONField{}),
				jsonField("raw", "", icd.BytesJSONField{}),
			)
			stamp := telem.SecondTS
			stamped := jsonMessage("Stamped", nil, jsonField(
				"ts", "/ts", icd.TimestampJSONField{
					TimeFormat: payload.TimeFormatUnixSec,
					Default:    &stamp,
				},
			))
			frame := icd.Message{
				Key:     uuid.New(),
				Name:    "Frame",
				Payload: icd.Payload{Variant: icd.BinaryPayload{}},
			}
			ic := icd.ICD{Name: "Plant", Messages: []icd.Message{m, stamped, frame}}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return icds.NewWriter(tx).Create(ctx, &ic)
			})).To(Succeed())
			idx := createChannel(ctx, telem.TimestampT, channel.Channel{})
			return format{
				ic:      ic,
				m:       m,
				stamped: stamped,
				frame:   frame,
				idx:     idx,
				value:   createChannel(ctx, telem.Float64T, idx),
				stamp:   createChannel(ctx, telem.TimestampT, idx),
			}
		}

		readConfigOf := func(records ...map[string]any) msgpack.EncodedJSON {
			list := make([]any, len(records))
			for i, r := range records {
				list[i] = r
			}
			return msgpack.EncodedJSON{"topics": []any{map[string]any{
				"key":     "t-1",
				"name":    "plant",
				"records": list,
			}}}
		}

		prepare := func(ctx context.Context, cfg msgpack.EncodedJSON) error {
			_, err := svc.Read.Prepare(ctx, db, uuid.New(), "Pump", cfg)
			return err
		}

		It("Should accept a binding with no channel and the data type of a new one",
			func(ctx SpecContext) {
				f := createFormat(ctx)
				Expect(prepare(ctx, readConfigOf(icdRecord(
					"", f.ic, readMessage(f.m, map[string]any{
						"field": fieldKey(f.m, 1), "data_type": "float32",
					}),
				)))).To(Succeed())
			},
		)

		It("Should skip the checks of a disabled record", func(ctx SpecContext) {
			f := createFormat(ctx)
			rec := icdRecord("", f.ic, readMessage(f.m, binding(f.m, 2, f.value)))
			rec["disabled"] = true
			Expect(prepare(ctx, readConfigOf(rec))).To(Succeed())
		})

		DescribeTable("Should reject a record with the path of the problem",
			func(
				ctx SpecContext,
				build func(f format) []map[string]any,
				path string,
				msg string,
			) {
				err := prepare(ctx, readConfigOf(build(createFormat(ctx))...))
				Expect(err).To(MatchError(validate.ErrValidation))
				Expect(err).To(MatchError(HavePrefix(path + ": ")))
				Expect(err).To(MatchError(ContainSubstring(msg)))
			},
			Entry("a message that the ICD lacks", func(f format) []map[string]any {
				other := jsonMessage(
					"Other",
					nil,
					jsonField("v", "/v", icd.FloatJSONField{}),
				)
				read := readMessage(other, binding(other, 0, f.value))
				return []map[string]any{icdRecord("", f.ic, read)}
			}, "topics.0.records.0.messages.0.message", "does not exist in ICD Plant"),
			Entry("a message that is not JSON", func(f format) []map[string]any {
				read := readMessage(f.frame, map[string]any{"field": uuid.New()})
				return []map[string]any{icdRecord("", f.ic, read)}
			}, "topics.0.records.0.messages.0.message", "message Frame is not a JSON message"),
			Entry("a field that the message lacks", func(f format) []map[string]any {
				read := readMessage(f.m, map[string]any{
					"field": uuid.New(), "channel": f.value.Key(),
				})
				return []map[string]any{icdRecord("", f.ic, read)}
			},
				"topics.0.records.0.messages.0.bindings.0.field",
				"does not exist in message Status",
			),
			Entry("a bytes field at the root", func(f format) []map[string]any {
				read := readMessage(f.m, map[string]any{"field": fieldKey(f.m, 3)})
				return []map[string]any{icdRecord("", f.ic, read)}
			}, "topics.0.records.0.messages.0.bindings.0.field",
				"bytes field raw requires a JSON pointer"),
			Entry("a data type that the field kind does not suit",
				func(f format) []map[string]any {
					read := readMessage(f.m, map[string]any{
						"field": fieldKey(f.m, 2), "data_type": "float64",
					})
					return []map[string]any{icdRecord("", f.ic, read)}
				},
				"topics.0.records.0.messages.0.bindings.0.data_type",
				"field state of kind string cannot read into data type float64",
			),
			Entry("a channel that the field kind does not convert to",
				func(f format) []map[string]any {
					read := readMessage(f.m, binding(f.m, 2, f.value))
					return []map[string]any{icdRecord("", f.ic, read)}
				},
				"topics.0.records.0.messages.0.bindings.0.channel",
				"field state of kind string cannot read into channel",
			),
			Entry("an index field that is not a timestamp",
				func(f format) []map[string]any {
					read := readMessage(f.m, binding(f.m, 1, f.value))
					read["index"] = fieldKey(f.m, 1)
					return []map[string]any{icdRecord("", f.ic, read)}
				},
				"topics.0.records.0.messages.0.index",
				"the index field temp must be a timestamp field",
			),
			Entry("an index field with a default", func(f format) []map[string]any {
				read := readMessage(f.stamped, binding(f.stamped, 0, f.idx))
				read["index"] = fieldKey(f.stamped, 0)
				return []map[string]any{icdRecord("", f.ic, read)}
			},
				"topics.0.records.0.messages.0.index",
				"the index field ts cannot take a default",
			),
			Entry("an index field with no binding", func(f format) []map[string]any {
				read := readMessage(f.m, binding(f.m, 1, f.value))
				read["index"] = fieldKey(f.m, 0)
				return []map[string]any{icdRecord("", f.ic, read)}
			}, "topics.0.records.0.messages.0.index", "the index field ts has no binding"),
			Entry("an index that names no field", func(f format) []map[string]any {
				read := readMessage(f.m, binding(f.m, 1, f.value))
				read["index"] = uuid.New()
				return []map[string]any{icdRecord("", f.ic, read)}
			}, "topics.0.records.0.messages.0.index", "does not exist in message Status"),
			Entry("an index field on a data channel", func(f format) []map[string]any {
				read := readMessage(f.m, binding(f.m, 0, f.stamp))
				read["index"] = fieldKey(f.m, 0)
				return []map[string]any{icdRecord("", f.ic, read)}
			}, "topics.0.records.0.messages.0.bindings.0.channel",
				"the index field ts must write to an index channel"),
			Entry("a data field on an index channel", func(f format) []map[string]any {
				read := readMessage(f.m, binding(f.m, 0, f.idx))
				return []map[string]any{icdRecord("", f.ic, read)}
			}, "topics.0.records.0.messages.0.bindings.0.channel",
				"only the index field can write to index channel"),
			Entry("a channel that two records write", func(f format) []map[string]any {
				return []map[string]any{
					icdRecord("a", f.ic, readMessage(f.m, binding(f.m, 1, f.value))),
					icdRecord("b", f.ic, readMessage(f.m, binding(f.m, 1, f.value))),
				}
			}, "topics.0.records.1.messages.0.bindings.0.channel", "is also written by "+
				"topics.0.records.0.messages.0.bindings.0.channel"),
		)

		Describe("Config hash", func() {
			var (
				f   format
				key task.Key
			)
			hash := func(ctx context.Context) string {
				GinkgoHelper()
				var t task.Task
				Expect(tasks.NewRetrieve().
					Where(task.MatchKeys(key)).
					Entry(&t).
					Exec(ctx, db)).To(Succeed())
				return t.ConfigHash
			}
			set := func(ctx context.Context, path []string, value any) {
				GinkgoHelper()
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return icds.NewWriter(tx).Apply(ctx, f.ic.Key, icd.NewSetAction(
						icd.SetPayload{Path: path, Value: value},
					))
				})).To(Succeed())
			}
			// pointerPath is the path of the pointer of field i of m.
			pointerPath := func(m icd.Message, i int) []string {
				return []string{
					"messages", m.Key.String(),
					"payload", "fields", fieldKey(m, i).String(),
					"pointer",
				}
			}
			BeforeEach(func(ctx SpecContext) {
				f = createFormat(ctx)
				key = MustSucceed(tasks.Create(ctx, svc.Read, readConfigOf(icdRecord(
					"", f.ic, readMessage(f.m, binding(f.m, 1, f.value)),
				))))
			})

			It("Should change when the format of a message the task reads changes",
				func(ctx SpecContext) {
					before := hash(ctx)
					set(ctx, pointerPath(f.m, 1), "/temperature")
					Expect(hash(ctx)).ToNot(Equal(before))
				},
			)

			It("Should not change when the name of the ICD changes", func(
				ctx SpecContext,
			) {
				before := hash(ctx)
				set(ctx, []string{"name"}, "Line")
				Expect(hash(ctx)).To(Equal(before))
			})

			It("Should not change when a message the task does not read changes",
				func(ctx SpecContext) {
					before := hash(ctx)
					set(ctx, pointerPath(f.stamped, 0), "/time")
					Expect(hash(ctx)).To(Equal(before))
				},
			)
		})

		Describe("Import", func() {
			// v0 is a v0 read config with one record that reads an index field and a
			// field into channels, and holds a disabled field.
			v0 := func(idx, value channel.Channel) msgpack.EncodedJSON {
				return msgpack.EncodedJSON{
					"topic":        "plant",
					"start_offset": "earliest",
					"records": []any{map[string]any{
						"key":        "record-1",
						"record_key": "pump",
						"index":      "time",
						"fields": []any{
							map[string]any{
								"key":         "time",
								"name":        "Time",
								"channel":     idx.Key(),
								"pointer":     "/ts",
								"data_type":   "timestamp",
								"time_format": "unix_sec",
							},
							map[string]any{
								"key":       "level",
								"channel":   value.Key(),
								"pointer":   "/level",
								"data_type": "float64",
							},
							map[string]any{
								"key":       "spare",
								"name":      "Spare",
								"pointer":   "/spare",
								"data_type": "int16",
								"disabled":  true,
							},
						},
					}},
				}
			}

			It("Should move the fields of a v0 config into an ICD named after its task",
				func(ctx SpecContext) {
					idx := createChannel(ctx, telem.TimestampT, channel.Channel{})
					value := createChannel(ctx, telem.Float32T, idx)
					data := MustSucceed(svc.Read.Normalize(ctx, db, 0, v0(idx, value)))
					key := MustSucceed(tasks.Create(ctx, svc.Read, data))
					stored := MustSucceed(svc.Read.Read(ctx, db, key))
					Expect(stored).ToNot(HaveKey("inline"))
					var cfg kafka.ReadConfig
					Expect(stored.Unmarshal(&cfg)).To(Succeed())
					Expect(cfg.StartOffset).To(Equal(kafka.StartOffsetEarliest))
					Expect(cfg.Topics).To(HaveLen(1))
					Expect(cfg.Topics[0].Name).To(Equal("plant"))
					Expect(cfg.Topics[0].Records).To(HaveLen(1))
					rec := cfg.Topics[0].Records[0]
					Expect(rec.Key).To(Equal("record-1"))
					Expect(rec.RecordKey).To(Equal("pump"))
					var ic icd.ICD
					Expect(icds.NewRetrieve().
						Where(icd.MatchKeys(rec.ICD)).
						Entry(&ic).
						Exec(ctx, db)).To(Succeed())
					Expect(ic.Name).To(Equal(kafka.ReadTaskType))
					Expect(ic.Messages).To(HaveLen(1))
					m := ic.Messages[0]
					Expect(m.Name).To(Equal("pump"))
					fields := jsonFields(m)
					names := make([]string, 0, len(fields))
					kinds := make([]payload.Kind, 0, len(fields))
					for _, f := range fields {
						names = append(names, f.Base().Name)
						kinds = append(kinds, f.Type())
					}
					Expect(names).To(Equal([]string{"Time", "level", "Spare"}))
					Expect(kinds).To(Equal([]payload.Kind{
						payload.KindTimestamp, payload.KindFloat, payload.KindInteger,
					}))
					index := fieldKey(m, 0)
					Expect(rec.Messages).To(Equal([]kafka.ReadMessage{{
						Message: m.Key,
						Index:   &index,
						Bindings: []kafka.ReadBinding{
							{Field: index, Channel: idx.Key()},
							{Field: fieldKey(m, 1), Channel: value.Key()},
						},
					}}))
				},
			)

			It("Should leave a v0 config with no record without an ICD",
				func(ctx SpecContext) {
					data := MustSucceed(svc.Read.Normalize(
						ctx, db, 0, msgpack.EncodedJSON{"topic": "plant"},
					))
					MustSucceed(tasks.Create(ctx, svc.Read, data))
					var found []icd.ICD
					Expect(icds.NewRetrieve().Entries(&found).Exec(ctx, db)).
						To(Succeed())
					Expect(found).To(BeEmpty())
				},
			)
		})
	})

	DescribeTable("Should reject a scan config whose rate the Driver cannot run",
		func(ctx SpecContext, rate float64, msg string) {
			Expect(svc.Scan.Prepare(ctx, db, uuid.New(), "", msgpack.EncodedJSON{
				"rate": rate,
			})).Error().To(MatchError(ContainSubstring(msg)))
		},
		Entry("zero", 0.0, "rate: must be greater than 0"),
		Entry("negative", -1.0, "rate: must be greater than 0"),
		Entry("too high for a nonzero period", 1e12, "is too high"),
	)
})
