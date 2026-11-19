// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mqtt_test

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
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt"
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

var _ = Describe("Service", func() {
	var (
		db       *gorp.DB
		node     testutil.Node
		channels *channel.Service
		icds     *icd.Service
		svc      *mqtt.Service
		tasks    *tasktestutil.Tasks
	)
	BeforeEach(func(ctx SpecContext) {
		// The suite's Core holds a mqtt service and a task service of its own.
		node = testutil.NewNode(ctx)
		db = node.DB
		MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: db}))
		statuses := MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
		channels = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
			Channel: node.Channel,
			TS:      node.Storage.TS,
			DB:      db,
			HostKey: node.Identity.HostKey,
			Status:  statuses,
		}))
		icds = MustOpen(icd.OpenService(ctx, icd.ServiceConfig{
			DB:   db,
			ImEx: imex.NewService(),
		}))
		svc = MustOpen(mqtt.OpenService(ctx, mqtt.ServiceConfig{
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
			Expect(mqtt.OpenService(ctx, mqtt.ServiceConfig{})).Error().
				To(MatchError(ContainSubstring("db: must be non-nil")))
		})

		It("Should reject a config missing the channel service", func(ctx SpecContext) {
			Expect(mqtt.OpenService(ctx, mqtt.ServiceConfig{DB: db})).Error().
				To(MatchError(ContainSubstring("channel: must be non-nil")))
		})

		It("Should reject a config missing the ICD service", func(ctx SpecContext) {
			Expect(mqtt.OpenService(ctx, mqtt.ServiceConfig{
				DB:      db,
				Channel: channels,
			})).Error().To(MatchError(ContainSubstring("icd: must be non-nil")))
		})
	})

	Describe("Stores", func() {
		It("Should expose one store per MQTT task type", func() {
			stores := svc.Stores()
			types := make([]string, len(stores))
			for i, s := range stores {
				types[i] = s.Type()
			}
			Expect(types).To(ConsistOf(
				mqtt.ReadTaskType,
				mqtt.WriteTaskType,
				mqtt.ScanTaskType,
				mqtt.EdgeTaskType,
			))
		})
	})

	Describe("Normalize", func() {
		DescribeTable(
			"Should import a config at versions 1 and 2 and reject a later version",
			func(ctx SpecContext, store func(*mqtt.Service) config.Store) {
				s := store(svc)
				Expect(s.Version()).To(BeEquivalentTo(2))
				MustSucceed(s.Normalize(ctx, db, 1, msgpack.EncodedJSON{}))
				MustSucceed(s.Normalize(ctx, db, 2, msgpack.EncodedJSON{}))
				newer := " version 3 is newer than this Core supports (latest: 2)"
				Expect(s.Normalize(ctx, db, 3, msgpack.EncodedJSON{})).Error().
					To(MatchError(ContainSubstring(s.Type() + newer)))
			},
			Entry("read", func(s *mqtt.Service) config.Store { return s.Read }),
			Entry("write", func(s *mqtt.Service) config.Store { return s.Write }),
			Entry("scan", func(s *mqtt.Service) config.Store { return s.Scan }),
			Entry("edge", func(s *mqtt.Service) config.Store { return s.Edge }),
		)
	})

	Describe("Write", func() {
		It("Should store a plain read entry with its schema defaults", func(
			ctx SpecContext,
		) {
			Expect(tasks.CreateDevice(ctx, "broker-1")).To(Succeed())
			ic := icd.ICD{Name: "Plant"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return icds.NewWriter(tx).Create(ctx, &ic)
			})).To(Succeed())
			key := MustSucceed(tasks.Create(ctx, svc.Read, msgpack.EncodedJSON{
				"device": "broker-1",
				"entries": []any{map[string]any{
					"key":   "entry-1",
					"type":  "plain",
					"topic": "plant/line1/temperature",
					"icd":   ic.Key,
				}},
			}))
			data := MustSucceed(svc.Read.Read(ctx, db, key))
			Expect(data["key"]).To(Equal(key.String()))
			Expect(data["device"]).To(Equal("broker-1"))
			Expect(data["entries"]).To(HaveExactElements(SatisfyAll(
				HaveKeyWithValue("topic", "plant/line1/temperature"),
				HaveKeyWithValue("qos", "at_most_once"),
				HaveKeyWithValue("retained_ignored", false),
				HaveKeyWithValue("messages", BeEmpty()),
			)))
		})

		It("Should store a Sparkplug read entry", func(ctx SpecContext) {
			key := MustSucceed(tasks.Create(ctx, svc.Read, msgpack.EncodedJSON{
				"entries": []any{map[string]any{
					"key":       "entry-1",
					"type":      "sparkplug",
					"group":     "plant",
					"edge_node": "line1",
					"tag":       "temperature",
				}},
			}))
			data := MustSucceed(svc.Read.Read(ctx, db, key))
			Expect(data["entries"]).To(HaveExactElements(SatisfyAll(
				HaveKeyWithValue("edge_node", "line1"),
				HaveKeyWithValue("data_type", "float64"),
			)))
		})

		It("Should default a plain write target to QoS 1 and not retained", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.Write, msgpack.EncodedJSON{
				"targets": []any{map[string]any{"key": "target-1", "type": "plain"}},
			}))
			data := MustSucceed(svc.Write.Read(ctx, db, key))
			Expect(data["targets"]).To(HaveExactElements(SatisfyAll(
				HaveKeyWithValue("qos", "at_least_once"),
				HaveKeyWithValue("retained", false),
			)))
		})

		It("Should apply scan config schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.Scan, msgpack.EncodedJSON{}))
			data := MustSucceed(svc.Scan.Read(ctx, db, key))
			Expect(data["rate"]).To(BeNumerically("==", 0.2))
		})

		It("Should default an edge node tag to the double type", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.Edge, msgpack.EncodedJSON{
				"group":     "plant",
				"edge_node": "synnax",
				"tags":      []any{map[string]any{"key": "tag-1", "name": "pressure"}},
			}))
			data := MustSucceed(svc.Edge.Read(ctx, db, key))
			Expect(data["tags"]).To(HaveExactElements(
				HaveKeyWithValue("tag_type", "double"),
			))
		})

		It("Should return the validation error for an invalid QoS", func(
			ctx SpecContext,
		) {
			Expect(svc.Read.Prepare(ctx, db, uuid.New(), "", msgpack.EncodedJSON{
				"entries": []any{map[string]any{"type": "plain", "qos": "twice"}},
			})).Error().
				To(MatchError(ContainSubstring("twice")))
		})

		It("Should return the validation error for an unknown entry type", func(
			ctx SpecContext,
		) {
			Expect(svc.Read.Prepare(ctx, db, uuid.New(), "", msgpack.EncodedJSON{
				"entries": []any{map[string]any{"type": "amqp"}},
			})).Error().
				To(MatchError(ContainSubstring("amqp")))
		})
	})

	Describe("Read config", func() {
		// createChannel creates a channel of dt, an index channel when dt is a
		// timestamp, on the index channel idx otherwise.
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
				IsIndex:     dt == telem.TimestampT,
				LocalIndex:  idx.LocalKey,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channels.NewWriter(tx).Create(ctx, &ch)
			})).To(Succeed())
			return ch
		}

		// format is an ICD with a Status message of a timestamp, a float, a string, and
		// a
		// bytes field at the root, a Stamped message of a timestamp with a default, and
		// channels on one index to bind them to.
		type format struct {
			ic         icd.ICD
			m, stamped icd.Message
			idx, value channel.Channel
			text       channel.Channel
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
			ic := icd.ICD{Name: "Plant", Messages: []icd.Message{m, stamped}}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return icds.NewWriter(tx).Create(ctx, &ic)
			})).To(Succeed())
			idx := createChannel(ctx, telem.TimestampT, channel.Channel{})
			return format{
				ic:      ic,
				m:       m,
				stamped: stamped,
				idx:     idx,
				value:   createChannel(ctx, telem.Float64T, idx),
				text:    createChannel(ctx, telem.StringT, idx),
			}
		}

		readConfigOf := func(entries ...map[string]any) msgpack.EncodedJSON {
			list := make([]any, len(entries))
			for i, e := range entries {
				list[i] = e
			}
			return msgpack.EncodedJSON{"entries": list}
		}

		prepare := func(ctx context.Context, cfg msgpack.EncodedJSON) error {
			_, err := svc.Read.Prepare(ctx, db, uuid.New(), "Pump", cfg)
			return err
		}

		It("Should accept a binding with no channel and the data type of a new one",
			func(ctx SpecContext) {
				f := createFormat(ctx)
				Expect(prepare(ctx, readConfigOf(icdEntry(
					"plant/status", f.ic, readMessage(f.m, map[string]any{
						"field": fieldKey(f.m, 1), "data_type": "float32",
					}),
				)))).To(Succeed())
			},
		)

		It("Should skip the checks of a disabled entry", func(ctx SpecContext) {
			f := createFormat(ctx)
			entry := icdEntry(
				"plant/+",
				f.ic,
				readMessage(f.m, binding(f.m, 2, f.value)),
			)
			entry["disabled"] = true
			Expect(prepare(ctx, readConfigOf(entry))).To(Succeed())
		})

		DescribeTable("Should reject an entry with the path of the problem",
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
				return []map[string]any{icdEntry("plant/x", f.ic, read)}
			}, "entries.0.messages.0.message", "does not exist in ICD Plant"),
			Entry("a field that the message lacks", func(f format) []map[string]any {
				read := readMessage(f.m, map[string]any{
					"field": uuid.New(), "channel": f.value.Key(),
				})
				return []map[string]any{icdEntry("plant/x", f.ic, read)}
			}, "entries.0.messages.0.bindings.0.field", "does not exist in message Status"),
			Entry("a bytes field at the root", func(f format) []map[string]any {
				read := readMessage(f.m, map[string]any{"field": fieldKey(f.m, 3)})
				return []map[string]any{icdEntry("plant/x", f.ic, read)}
			}, "entries.0.messages.0.bindings.0.field",
				"bytes field raw requires a JSON pointer"),
			Entry("a data type that the field kind does not suit",
				func(f format) []map[string]any {
					read := readMessage(f.m, map[string]any{
						"field": fieldKey(f.m, 2), "data_type": "float64",
					})
					return []map[string]any{icdEntry("plant/x", f.ic, read)}
				},
				"entries.0.messages.0.bindings.0.data_type",
				"field state of kind string cannot read into data type float64",
			),
			Entry("a channel that the field kind does not convert to",
				func(f format) []map[string]any {
					read := readMessage(f.m, binding(f.m, 2, f.value))
					return []map[string]any{icdEntry("plant/x", f.ic, read)}
				},
				"entries.0.messages.0.bindings.0.channel",
				"field state of kind string cannot read into channel",
			),
			Entry("an index field that is not a timestamp",
				func(f format) []map[string]any {
					read := readMessage(f.m, binding(f.m, 1, f.value))
					read["index"] = fieldKey(f.m, 1)
					return []map[string]any{icdEntry("plant/x", f.ic, read)}
				},
				"entries.0.messages.0.index",
				"the index field temp must be a timestamp field",
			),
			Entry("an index field with a default", func(f format) []map[string]any {
				read := readMessage(f.stamped, binding(f.stamped, 0, f.idx))
				read["index"] = fieldKey(f.stamped, 0)
				return []map[string]any{icdEntry("plant/x", f.ic, read)}
			}, "entries.0.messages.0.index", "the index field ts cannot take a default"),
			Entry("an index field with no binding", func(f format) []map[string]any {
				read := readMessage(f.m, binding(f.m, 1, f.value))
				read["index"] = fieldKey(f.m, 0)
				return []map[string]any{icdEntry("plant/x", f.ic, read)}
			}, "entries.0.messages.0.index", "the index field ts has no binding"),
			Entry("an index field on a data channel", func(f format) []map[string]any {
				read := readMessage(f.m, map[string]any{
					"field": fieldKey(f.m, 0), "channel": f.value.Key(),
				})
				read["index"] = fieldKey(f.m, 0)
				return []map[string]any{icdEntry("plant/x", f.ic, read)}
			}, "entries.0.messages.0.bindings.0.channel",
				"field ts of kind timestamp cannot read into channel"),
			Entry("a data field on an index channel", func(f format) []map[string]any {
				read := readMessage(f.m, binding(f.m, 0, f.idx))
				return []map[string]any{icdEntry("plant/x", f.ic, read)}
			}, "entries.0.messages.0.bindings.0.channel",
				"only the index field can write to index channel"),
			Entry("a channel that two entries write", func(f format) []map[string]any {
				return []map[string]any{
					icdEntry(
						"plant/a",
						f.ic,
						readMessage(f.m, binding(f.m, 1, f.value)),
					),
					icdEntry(
						"plant/b",
						f.ic,
						readMessage(f.m, binding(f.m, 1, f.value)),
					),
				}
			}, "entries.1.messages.0.bindings.0.channel", "is also written by "+
				"entries.0.messages.0.bindings.0.channel"),
			Entry("a channel that a tag also writes", func(f format) []map[string]any {
				return []map[string]any{
					icdEntry(
						"plant/a",
						f.ic,
						readMessage(f.m, binding(f.m, 1, f.value)),
					),
					{
						"key":       "tag",
						"type":      "sparkplug",
						"group":     "plant",
						"edge_node": "line1",
						"tag":       "temp",
						"channel":   f.value.Key(),
					},
				}
			}, "entries.1.channel", "is also written by "+
				"entries.0.messages.0.bindings.0.channel"),
		)

		Describe("Import", func() {
			// v1 is a v1 read config with one plain entry that reads an index field
			// and a field into channels, and holds a disabled field.
			v1 := func(idx, value channel.Channel) msgpack.EncodedJSON {
				return msgpack.EncodedJSON{"entries": []any{map[string]any{
					"key":   "entry-1",
					"type":  "plain",
					"topic": "plant/level",
					"qos":   "exactly_once",
					"index": "time",
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
				}}}
			}

			It("Should move the fields of a v1 config into an ICD named after its task",
				func(ctx SpecContext) {
					idx := createChannel(ctx, telem.TimestampT, channel.Channel{})
					value := createChannel(ctx, telem.Float32T, idx)
					data := MustSucceed(svc.Read.Normalize(ctx, db, 1, v1(idx, value)))
					key := MustSucceed(tasks.Create(ctx, svc.Read, data))
					stored := MustSucceed(svc.Read.Read(ctx, db, key))
					Expect(stored).ToNot(HaveKey("inline"))
					var cfg mqtt.ReadConfig
					Expect(stored.Unmarshal(&cfg)).To(Succeed())
					Expect(cfg.Entries).To(HaveLen(1))
					entry, ok := cfg.Entries[0].Variant.(mqtt.PlainReadEntry)
					Expect(ok).To(BeTrue())
					Expect(entry.Key).To(Equal("entry-1"))
					Expect(entry.Topic).To(Equal("plant/level"))
					Expect(entry.Qos).To(Equal(mqtt.QoSExactlyOnce))
					var ic icd.ICD
					Expect(icds.NewRetrieve().
						Where(icd.MatchKeys(entry.ICD)).
						Entry(&ic).
						Exec(ctx, db)).To(Succeed())
					Expect(ic.Name).To(Equal(mqtt.ReadTaskType))
					Expect(ic.Messages).To(HaveLen(1))
					m := ic.Messages[0]
					Expect(m.Name).To(Equal("plant/level"))
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
					Expect(entry.Messages).To(Equal([]mqtt.ReadMessage{{
						Message: m.Key,
						Index:   &index,
						Bindings: []mqtt.ReadBinding{
							{Field: index, Channel: idx.Key()},
							{Field: fieldKey(m, 1), Channel: value.Key()},
						},
					}}))
				},
			)

			It("Should leave a v1 config with no plain entry without an ICD",
				func(ctx SpecContext) {
					data := MustSucceed(
						svc.Read.Normalize(ctx, db, 1, msgpack.EncodedJSON{
							"entries": []any{map[string]any{
								"key":       "tag",
								"type":      "sparkplug",
								"group":     "plant",
								"edge_node": "line1",
								"tag":       "temp",
							}},
						}),
					)
					MustSucceed(tasks.Create(ctx, svc.Read, data))
					var found []icd.ICD
					Expect(icds.NewRetrieve().Entries(&found).Exec(ctx, db)).
						To(Succeed())
					Expect(found).To(BeEmpty())
				},
			)
		})

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
				key = MustSucceed(tasks.Create(ctx, svc.Read, readConfigOf(icdEntry(
					"plant/status",
					f.ic,
					readMessage(f.m, binding(f.m, 1, f.value)),
				))))
			})

			It("Should change when the format of a message the task reads changes",
				func(ctx SpecContext) {
					before := hash(ctx)
					set(ctx, pointerPath(f.m, 1), "/temperature")
					Expect(hash(ctx)).ToNot(Equal(before))
				},
			)

			It("Should return to its value when an edit is undone", func(
				ctx SpecContext,
			) {
				before := hash(ctx)
				set(ctx, pointerPath(f.m, 1), "/temperature")
				set(ctx, pointerPath(f.m, 1), "/temp")
				Expect(hash(ctx)).To(Equal(before))
			})

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
					other := jsonMessage(
						"Other",
						nil,
						jsonField("ts", "/ts", icd.TimestampJSONField{
							TimeFormat: payload.TimeFormatUnixSec,
						}),
					)
					set(ctx, []string{"messages"}, []icd.Message{f.m, other})
					set(ctx, pointerPath(other, 0), "/time")
					Expect(hash(ctx)).To(Equal(before))
				},
			)

			It("Should match the hash of the stored config", func(ctx SpecContext) {
				set(ctx, pointerPath(f.m, 1), "/temperature")
				Expect(hash(ctx)).To(Equal(MustSucceed(svc.Read.Hash(ctx, db, key))))
			})
		})
	})

	DescribeTable("Should reject a scan config whose rate is not positive",
		func(ctx SpecContext, rate float64) {
			Expect(svc.Scan.Prepare(ctx, db, uuid.New(), "", msgpack.EncodedJSON{
				"rate": rate,
			})).Error().To(MatchError(ContainSubstring("rate: must be greater than 0")))
		},
		Entry("zero", 0.0),
		Entry("negative", -1.0),
	)
})
