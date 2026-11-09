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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/kafka"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	tasktestutil "github.com/synnaxlabs/synnax/pkg/service/task/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/patch"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Service", func() {
	var (
		db    *gorp.DB
		svc   *kafka.Service
		tasks *tasktestutil.Tasks
	)
	BeforeEach(func(ctx SpecContext) {
		// The suite's DB holds a kafka service and a task service of its own.
		db = DeferClose(gorp.Wrap(memkv.New()))
		svc = MustOpen(kafka.OpenService(ctx, kafka.ServiceConfig{DB: db}))
		tasks = MustOpen(tasktestutil.OpenTasks(ctx, tasktestutil.TasksConfig{
			DB:     db,
			Stores: svc.Stores(),
		}))
	})

	Describe("OpenService", func() {
		It("Should reject a config missing the DB", func(ctx SpecContext) {
			Expect(kafka.OpenService(ctx, kafka.ServiceConfig{})).Error().
				To(MatchError(ContainSubstring("db: must be non-nil")))
		})
	})

	Describe("Stores", func() {
		It("Should expose one store per Kafka task type", func() {
			types := []string{}
			for _, s := range svc.Stores() {
				types = append(types, s.Type())
			}
			Expect(types).To(ConsistOf(
				kafka.ReadTaskType,
				kafka.WriteTaskType,
				kafka.ScanTaskType,
			))
		})
	})

	Describe("Write", func() {
		It("Should store a decoded read config under the given key", func(
			ctx SpecContext,
		) {
			Expect(tasks.CreateDevice(ctx, "dev-1")).To(Succeed())
			key := MustSucceed(tasks.Create(ctx, svc.Read, msgpack.EncodedJSON{
				"device": "dev-1",
				"topic":  "telemetry",
				"records": []any{map[string]any{
					"key":        "r-1",
					"record_key": "press_1",
					"fields": []any{map[string]any{
						"key":     "f-1",
						"pointer": "/value",
					}},
				}},
			}))
			data := MustSucceed(svc.Read.Read(ctx, db, key))
			Expect(data["key"]).To(Equal(key.String()))
			Expect(data["device"]).To(Equal("dev-1"))
			Expect(data["topic"]).To(Equal("telemetry"))
			Expect(data["records"]).To(HaveExactElements(SatisfyAll(
				HaveKeyWithValue("record_key", "press_1"),
				HaveKeyWithValue("fields", HaveExactElements(
					HaveKeyWithValue("pointer", "/value"),
				)),
			)))
		})

		It("Should apply read config schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.Read, msgpack.EncodedJSON{
				"records": []any{map[string]any{
					"key":    "r-1",
					"fields": []any{map[string]any{"key": "f-1"}},
				}},
			}))
			data := MustSucceed(svc.Read.Read(ctx, db, key))
			Expect(data["start_offset"]).To(Equal("latest"))
			Expect(data["records"]).To(HaveExactElements(SatisfyAll(
				HaveKeyWithValue("record_key", ""),
				HaveKeyWithValue("fields", HaveExactElements(
					HaveKeyWithValue("data_type", "float64"),
				)),
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
				Expect(svc.Read.Prepare(ctx, db, uuid.New(), msgpack.EncodedJSON{
					"start_offset": "bogus",
				})).Error().To(MatchError(ContainSubstring("invalid start_offset: bogus")))
			},
		)

		It("Should keep an explicit zero scan rate", func(ctx SpecContext) {
			key := MustSucceed(
				tasks.Create(ctx, svc.Scan, msgpack.EncodedJSON{"rate": 0}),
			)
			data := MustSucceed(svc.Scan.Read(ctx, db, key))
			Expect(data["rate"]).To(BeNumerically("==", 0))
		})

		It("Should reject an explicit empty start offset", func(ctx SpecContext) {
			Expect(svc.Read.Prepare(ctx, db, uuid.New(), msgpack.EncodedJSON{
				"start_offset": "",
			})).Error().To(MatchError(ContainSubstring("invalid start_offset")))
		})

		It("Should keep an explicit empty value pointer", func(ctx SpecContext) {
			key := MustSucceed(tasks.Create(ctx, svc.Write, msgpack.EncodedJSON{
				"record": map[string]any{"value_pointer": ""},
			}))
			data := MustSucceed(svc.Write.Read(ctx, db, key))
			Expect(data["record"]).To(HaveKeyWithValue("value_pointer", ""))
		})

		It("Should return the write validation error for an invalid record key",
			func(ctx SpecContext) {
				Expect(svc.Write.Prepare(ctx, db, uuid.New(), msgpack.EncodedJSON{
					"record_key": "bogus",
				})).Error().To(MatchError(ContainSubstring("invalid record_key: bogus")))
			},
		)
	})

	Describe("Edit", func() {
		begin := func(a patch.Action) patch.Cursor {
			GinkgoHelper()
			return MustSucceed(patch.Begin(a)).Next()
		}
		set := func(value any, path ...string) patch.Cursor {
			GinkgoHelper()
			return begin(
				patch.Set{Path: append([]string{"config"}, path...), Value: value},
			)
		}
		remove := func(path ...string) patch.Cursor {
			GinkgoHelper()
			last := len(path) - 1
			return begin(patch.Remove{
				Path: append([]string{"config"}, path[:last]...),
				Key:  path[last],
			})
		}
		edit := func(
			ctx SpecContext,
			store config.Store,
			data msgpack.EncodedJSON,
			cursors ...patch.Cursor,
		) msgpack.EncodedJSON {
			GinkgoHelper()
			key := MustSucceed(tasks.Create(ctx, store, data))
			rec, changed := MustSucceed2(store.Edit(ctx, db, key, cursors))
			Expect(changed).To(BeTrue())
			return MustSucceed(rec.Data())
		}

		It("Should reset a removed scan rate to its default", func(ctx SpecContext) {
			data := edit(ctx, svc.Scan, msgpack.EncodedJSON{"rate": 5}, remove("rate"))
			Expect(data["rate"]).To(BeNumerically("==", 0.2))
		})

		It("Should reset a start offset set to null to its default", func(
			ctx SpecContext,
		) {
			data := edit(
				ctx,
				svc.Read,
				msgpack.EncodedJSON{"start_offset": "earliest"},
				set(nil, "start_offset"),
			)
			Expect(data["start_offset"]).To(Equal("latest"))
		})

		It("Should reset a removed nested value pointer to its default", func(
			ctx SpecContext,
		) {
			data := edit(
				ctx,
				svc.Write,
				msgpack.EncodedJSON{"record": map[string]any{"value_pointer": "/v"}},
				remove("record", "value_pointer"),
			)
			Expect(data["record"]).To(HaveKeyWithValue("value_pointer", "/value"))
		})

		It("Should keep a scan rate that an action sets to zero", func(
			ctx SpecContext,
		) {
			key := MustSucceed(
				tasks.Create(ctx, svc.Scan, msgpack.EncodedJSON{"rate": 5}),
			)
			rec, changed := MustSucceed2(
				svc.Scan.Edit(ctx, db, key, []patch.Cursor{set(0, "rate")}),
			)
			Expect(changed).To(BeFalse())
			Expect(MustSucceed(rec.Data())["rate"]).To(BeNumerically("==", 0))
		})

		It("Should keep a value that an action sets after a removal", func(
			ctx SpecContext,
		) {
			data := edit(
				ctx,
				svc.Scan,
				msgpack.EncodedJSON{"rate": 5},
				remove("rate"),
				set(7, "rate"),
			)
			Expect(data["rate"]).To(BeNumerically("==", 7))
		})
	})
})
