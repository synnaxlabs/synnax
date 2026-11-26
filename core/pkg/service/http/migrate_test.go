// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package http_test

import (
	"context"
	"encoding/json/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/http"
	v5 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v5"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	v2 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v3"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

var _ = Describe("Task migration", func() {
	It("Should move the fields of a Console-era task into an ICD named after the task",
		func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			StartRun(db.Run)
			MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: db}))
			statuses := MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
			icds := MustOpen(icd.OpenService(ctx, icd.ServiceConfig{
				DB:   db,
				ImEx: imex.NewService(),
			}))
			svc := MustOpen(http.OpenService(ctx, http.ServiceConfig{
				DB:      db,
				Channel: channelSvc,
				ICD:     icds,
			}))
			t := v2.Task{
				Key:  uuid.New(),
				Name: "Telemetry",
				Type: "http_read",
				Config: msgpack.EncodedJSON{
					"rate": 2,
					"endpoints": []any{map[string]any{
						"key":    "ep",
						"method": "GET",
						"path":   "/telemetry",
						"index":  "ts",
						"fields": []any{map[string]any{
							"enabled":         true,
							"key":             "ts",
							"pointer":         "/ts",
							"dataType":        "timestamp",
							"timestampFormat": "unix_ms",
						}},
					}},
				},
			}
			Expect(gorp.NewCreate[v2.Key, v2.Task]().Entry(&t).Exec(ctx, db)).
				To(Succeed())
			Expect(ontology.DefineResources(ctx, db, resource.ID{
				Type: resource.TypeTask,
				Key:  t.Key.String(),
			})).To(Succeed())
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v2.Key, v2.Task]{
				DB:   db,
				Name: string(resource.TypeTask),
				Migrations: []gorp.Migration{v3.NewMigration(v3.MigrationConfig{
					Status:  statuses,
					Configs: MustSucceed(config.NewRegistry(svc.Stores()...)),
				})},
			}))
			stored := MustSucceed(svc.Read.Read(ctx, db, t.Key))
			Expect(stored).ToNot(HaveKey("inline"))
			Expect(stored["endpoints"]).To(HaveExactElements(Not(HaveKey("fields"))))
			var cfg http.ReadConfig
			Expect(stored.Unmarshal(&cfg)).To(Succeed())
			Expect(cfg.Endpoints).To(HaveLen(1))
			var ic icd.ICD
			Expect(icds.NewRetrieve().
				Where(icd.MatchKeys(*cfg.Endpoints[0].ICD)).
				Entry(&ic).
				Exec(ctx, db)).To(Succeed())
			Expect(ic.Name).To(Equal("Telemetry"))
			Expect(ic.Messages).To(HaveLen(1))
			Expect(ic.Messages[0].Name).To(Equal("GET /telemetry"))
		},
	)
})

var _ = Describe("ICDMigration", func() {
	var (
		svc *http.Service
		// names holds the task names that the lookup returns.
		names map[uuid.UUID]string
		// asked holds the keys of each lookup call.
		asked  [][]uuid.UUID
		lookup http.TaskNames
		ins    alamos.Instrumentation
		// logs holds the warnings that the migration logs.
		logs *observer.ObservedLogs
	)
	BeforeEach(func(ctx SpecContext) {
		svc = MustOpen(http.OpenService(ctx, http.ServiceConfig{
			DB:      db,
			Channel: channelSvc,
			ICD:     icdSvc,
		}))
		names, asked = map[uuid.UUID]string{}, nil
		var core zapcore.Core
		core, logs = observer.New(zapcore.WarnLevel)
		ins = alamos.New("test", alamos.WithLogger(MustSucceed(alamos.NewLogger(
			alamos.LoggerConfig{ZapLogger: zap.New(core)},
		))))
		lookup = func(
			_ context.Context,
			_ gorp.Tx,
			keys []uuid.UUID,
		) (map[uuid.UUID]string, error) {
			asked = append(asked, keys)
			return names, nil
		}
	})

	// stageEndpoint stores a config that the task named name owns, with one endpoint
	// that reads fields and indexes them with index, staged as v5.ReadMigration stages
	// them.
	stageEndpoint := func(
		ctx SpecContext,
		name string,
		index string,
		fields ...http.ReadField,
	) uuid.UUID {
		GinkgoHelper()
		key := uuid.New()
		Expect(gorp.WrapWriter[uuid.UUID, http.ReadConfig](db).Set(
			ctx,
			http.ReadConfig{Key: key, Rate: 1, Endpoints: []http.ReadEndpoint{{
				Key:    "ep",
				Method: "GET",
				Path:   "/level",
			}}},
		)).To(Succeed())
		b := MustSucceed(json.Marshal(http.InlineRead{
			Endpoints: []http.InlineEndpoint{{Key: "ep", Index: index, Fields: fields}},
		}))
		Expect(db.Set(ctx, v5.StagedFieldsKVKey(key), b)).To(Succeed())
		names[key] = name
		return key
	}
	// stage stores a config that the task named name owns, with one staged field.
	stage := func(ctx SpecContext, name string) uuid.UUID {
		GinkgoHelper()
		return stageEndpoint(ctx, name, "", http.ReadField{
			Key:      "level",
			Pointer:  "/level",
			DataType: telem.Float64T,
		})
	}
	run := func(ctx SpecContext) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.ICDMigration(lookup).Run(ctx, tx, ins)
		})
	}
	// expectStaged asserts whether the fields of the config are staged.
	expectStaged := func(key uuid.UUID, staged bool) {
		GinkgoHelper()
		if !staged {
			Expect(db.Get(v5.StagedFieldsKVKey(key))).Error().
				To(MatchError(query.ErrNotFound))
			return
		}
		_, closer := MustSucceed2(db.Get(v5.StagedFieldsKVKey(key)))
		Expect(closer.Close()).To(Succeed())
	}
	read := func(ctx SpecContext, key uuid.UUID) http.ReadConfig {
		GinkgoHelper()
		var cfg http.ReadConfig
		Expect(gorp.NewRetrieve[uuid.UUID, http.ReadConfig]().
			Where(gorp.MatchKeys[uuid.UUID, http.ReadConfig](key)).
			Entry(&cfg).
			Exec(ctx, db)).To(Succeed())
		return cfg
	}

	It("Should move the staged fields of each config into an ICD named after its task",
		func(ctx SpecContext) {
			key := stage(ctx, "Tank farm")
			Expect(run(ctx)).To(Succeed())
			Expect(asked).To(Equal([][]uuid.UUID{{key}}))
			expectStaged(key, false)
			cfg := read(ctx, key)
			Expect(cfg.Inline).To(BeNil())
			Expect(cfg.Endpoints).To(HaveLen(1))
			ep := cfg.Endpoints[0]
			Expect(ep.Messages).To(HaveLen(1))
			var ic icd.ICD
			Expect(icdSvc.NewRetrieve().
				Where(icd.MatchKeys(*ep.ICD)).
				Entry(&ic).
				Exec(ctx, db)).To(Succeed())
			Expect(ic.Name).To(Equal("Tank farm"))
			Expect(ic.Messages).To(HaveLen(1))
			Expect(ic.Messages[0].Name).To(Equal("GET /level"))
		},
	)

	It("Should store reads that a later save accepts", func(ctx SpecContext) {
		idx := createChannel(ctx, telem.TimestampT, channel.Channel{})
		value := createChannel(ctx, telem.Float64T, idx)
		key := stageEndpoint(
			ctx,
			"Tank farm",
			"ts",
			http.ReadField{
				Key:      "ts",
				Disabled: true,
				Channel:  idx.Key(),
				Pointer:  "/ts",
				DataType: telem.TimestampT,
			},
			http.ReadField{
				Key:      "level",
				Channel:  value.Key(),
				Pointer:  "/level",
				DataType: telem.Float64T,
			},
		)
		Expect(run(ctx)).To(Succeed())
		data := MustSucceed(svc.Read.Read(ctx, db, key))
		MustSucceed(svc.Read.Prepare(ctx, db, key, "Tank farm", data))
	})

	It("Should lift two index fields, note them, and refuse a later save", func(
		ctx SpecContext,
	) {
		sent := createChannel(ctx, telem.TimestampT, channel.Channel{})
		received := createChannel(ctx, telem.TimestampT, channel.Channel{})
		key := stageEndpoint(
			ctx,
			"Tank farm",
			"sent",
			http.ReadField{
				Key:      "sent",
				Name:     "sent",
				Channel:  sent.Key(),
				Pointer:  "/sent",
				DataType: telem.TimestampT,
			},
			http.ReadField{
				Key:      "received",
				Name:     "received",
				Channel:  received.Key(),
				Pointer:  "/received",
				DataType: telem.TimestampT,
			},
		)
		Expect(run(ctx)).To(Succeed())
		ep := read(ctx, key).Endpoints[0]
		Expect(ep.Messages).To(HaveLen(1))
		Expect(ep.Messages[0].Bindings).To(HaveExactElements(
			HaveField("Channel", sent.Key()),
			HaveField("Channel", received.Key()),
		))
		Expect(logs.FilterMessage("changed an HTTP read field to v5").All()).To(
			HaveExactElements(WithTransform(
				func(e observer.LoggedEntry) map[string]any { return e.ContextMap() },
				SatisfyAll(
					HaveKeyWithValue("field", "endpoints.0.fields.1"),
					HaveKeyWithValue("change", "field received and index field sent "+
						"both write index channels. Split the endpoint into two, "+
						"each with one index field"),
				),
			)),
		)
		data := MustSucceed(svc.Read.Read(ctx, db, key))
		Expect(svc.Read.Prepare(ctx, db, key, "Tank farm", data)).Error().To(MatchError(
			"endpoints.0.messages.0.bindings.1.channel: field received cannot write " +
				"to index channel " + received.Name + ", because only the index " +
				"field can. Bind field received to a data channel",
		))
	})

	It("Should do nothing without staged fields", func(ctx SpecContext) {
		Expect(run(ctx)).To(Succeed())
		Expect(asked).To(BeEmpty())
	})

	It("Should delete the staged fields of a config that no longer exists", func(
		ctx SpecContext,
	) {
		key := stage(ctx, "Removed")
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Read.Delete(ctx, tx, key)
		})).To(Succeed())
		Expect(run(ctx)).To(Succeed())
		Expect(asked).To(BeEmpty())
		expectStaged(key, false)
	})

	It("Should keep the staged fields when the task name lookup fails", func(
		ctx SpecContext,
	) {
		key := stage(ctx, "Tank farm")
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.Delete(ctx, v5.StagedFieldsKVKey(key))).To(Succeed())
		})
		lookup = func(
			context.Context,
			gorp.Tx,
			[]uuid.UUID,
		) (map[uuid.UUID]string, error) {
			return nil, errors.New("no task")
		}
		Expect(run(ctx)).To(MatchError("tasks of HTTP read configs: no task"))
		expectStaged(key, true)
		Expect(read(ctx, key).Endpoints[0].ICD).To(BeNil())
	})
})

var _ = Describe("WriteICDMigration", func() {
	var (
		svc *http.Service
		// names holds the task names that the lookup returns.
		names map[uuid.UUID]string
		// asked holds the keys of each lookup call.
		asked  [][]uuid.UUID
		lookup http.TaskNames
		ins    alamos.Instrumentation
		// logs holds the warnings that the migration logs.
		logs *observer.ObservedLogs
	)
	BeforeEach(func(ctx SpecContext) {
		svc = MustOpen(http.OpenService(ctx, http.ServiceConfig{
			DB:      db,
			Channel: channelSvc,
			ICD:     icdSvc,
		}))
		names, asked = map[uuid.UUID]string{}, nil
		var core zapcore.Core
		core, logs = observer.New(zapcore.WarnLevel)
		ins = alamos.New("test", alamos.WithLogger(MustSucceed(alamos.NewLogger(
			alamos.LoggerConfig{ZapLogger: zap.New(core)},
		))))
		lookup = func(
			_ context.Context,
			_ gorp.Tx,
			keys []uuid.UUID,
		) (map[uuid.UUID]string, error) {
			asked = append(asked, keys)
			return names, nil
		}
	})

	// stage stores a config that the task named name owns, with one endpoint whose
	// command channel does not exist, staged as v5.WriteMigration stages it.
	stage := func(ctx SpecContext, name string) uuid.UUID {
		GinkgoHelper()
		key := uuid.New()
		ep := http.WriteEndpoint{Key: "sp", Method: "PUT", Path: "/setpoint"}
		Expect(gorp.WrapWriter[uuid.UUID, http.WriteConfig](db).Set(
			ctx,
			http.WriteConfig{Key: key, Endpoints: []http.WriteEndpoint{ep}},
		)).To(Succeed())
		b := MustSucceed(json.Marshal(http.InlineWrite{
			Endpoints: []http.InlineWriteEndpoint{{
				Key: "sp",
				Channel: http.ChannelField{
					Pointer:  "/value",
					Name:     "setpoint_cmd",
					DataType: telem.Float64T,
				},
			}},
		}))
		Expect(db.Set(ctx, v5.StagedWriteFieldsKVKey(key), b)).To(Succeed())
		names[key] = name
		return key
	}
	run := func(ctx SpecContext) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.WriteICDMigration(lookup).Run(ctx, tx, ins)
		})
	}
	// expectStaged asserts whether the channel and fields of the config are staged.
	expectStaged := func(key uuid.UUID, staged bool) {
		GinkgoHelper()
		if !staged {
			Expect(db.Get(v5.StagedWriteFieldsKVKey(key))).Error().
				To(MatchError(query.ErrNotFound))
			return
		}
		_, closer := MustSucceed2(db.Get(v5.StagedWriteFieldsKVKey(key)))
		Expect(closer.Close()).To(Succeed())
	}
	read := func(ctx SpecContext, key uuid.UUID) http.WriteConfig {
		GinkgoHelper()
		var cfg http.WriteConfig
		Expect(gorp.NewRetrieve[uuid.UUID, http.WriteConfig]().
			Where(gorp.MatchKeys[uuid.UUID, http.WriteConfig](key)).
			Entry(&cfg).
			Exec(ctx, db)).To(Succeed())
		return cfg
	}

	It("Should move the staged channel and fields of each config into an ICD",
		func(ctx SpecContext) {
			key := stage(ctx, "Valves")
			Expect(run(ctx)).To(Succeed())
			Expect(asked).To(Equal([][]uuid.UUID{{key}}))
			expectStaged(key, false)
			cfg := read(ctx, key)
			Expect(cfg.Inline).To(BeNil())
			ep := cfg.Endpoints[0]
			var ic icd.ICD
			Expect(icdSvc.NewRetrieve().
				Where(icd.MatchKeys(*ep.ICD)).
				Entry(&ic).
				Exec(ctx, db)).To(Succeed())
			Expect(ic.Name).To(Equal("Valves"))
			Expect(ic.Messages).To(HaveLen(1))
			Expect(ic.Messages[0].Name).To(Equal("PUT /setpoint"))
			Expect(ep.Message.Message).To(Equal(ic.Messages[0].Key))
			Expect(logs.FilterMessage("changed an HTTP write field to v5").All()).To(
				HaveExactElements(WithTransform(
					func(e observer.LoggedEntry) map[string]any { return e.ContextMap() },
					SatisfyAll(
						HaveKeyWithValue("config", key.String()),
						HaveKeyWithValue("field", "endpoints.0.channel"),
						HaveKeyWithValue("change", `channel name "setpoint_cmd" of `+
							"field value on PUT /setpoint was dropped, so a client "+
							"names the new channel"),
					),
				)),
			)
		},
	)

	It("Should do nothing without staged fields", func(ctx SpecContext) {
		Expect(run(ctx)).To(Succeed())
		Expect(asked).To(BeEmpty())
	})

	It("Should delete the staged fields of a config that no longer exists", func(
		ctx SpecContext,
	) {
		key := stage(ctx, "Removed")
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Write.Delete(ctx, tx, key)
		})).To(Succeed())
		Expect(run(ctx)).To(Succeed())
		Expect(asked).To(BeEmpty())
		expectStaged(key, false)
	})

	It("Should keep the staged fields when the task name lookup fails", func(
		ctx SpecContext,
	) {
		key := stage(ctx, "Valves")
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.Delete(ctx, v5.StagedWriteFieldsKVKey(key))).To(Succeed())
		})
		lookup = func(
			context.Context,
			gorp.Tx,
			[]uuid.UUID,
		) (map[uuid.UUID]string, error) {
			return nil, errors.New("no task")
		}
		Expect(run(ctx)).To(MatchError("tasks of HTTP write configs: no task"))
		expectStaged(key, true)
		Expect(read(ctx, key).Endpoints[0].ICD).To(BeNil())
	})
})
