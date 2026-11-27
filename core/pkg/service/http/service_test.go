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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/http"
	"github.com/synnaxlabs/synnax/pkg/service/http/versions/legacy"
	v4 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v4"
	v5 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v5"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	tasktestutil "github.com/synnaxlabs/synnax/pkg/service/task/testutil"
	v2 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v3"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Service", func() {
	var (
		svc   *http.Service
		tasks *tasktestutil.Tasks
	)
	BeforeEach(func(ctx SpecContext) {
		svc = MustOpen(http.OpenService(ctx, http.ServiceConfig{
			DB:      db,
			Channel: channelSvc,
			ICD:     icdSvc,
		}))
		tasks = MustOpen(tasktestutil.OpenTasks(ctx, tasktestutil.TasksConfig{
			DB:     db,
			Status: statusSvc,
			Stores: svc.Stores(),
		}))
	})

	Describe("OpenService", func() {
		DescribeTable(
			"Should reject a config with a missing service",
			func(ctx SpecContext, unset func(*http.ServiceConfig), msg string) {
				cfg := http.ServiceConfig{DB: db, Channel: channelSvc, ICD: icdSvc}
				unset(&cfg)
				Expect(http.OpenService(ctx, cfg)).Error().To(MatchError(msg))
			},
			Entry(
				"db",
				func(c *http.ServiceConfig) { c.DB = nil },
				"db: must be non-nil",
			),
			Entry(
				"channel",
				func(c *http.ServiceConfig) { c.Channel = nil },
				"channel: must be non-nil",
			),
			Entry(
				"icd",
				func(c *http.ServiceConfig) { c.ICD = nil },
				"icd: must be non-nil",
			),
		)

		It("Should stage the channel and fields of each stored v4 write config",
			func(ctx SpecContext) {
				db := DeferClose(gorp.Wrap(memkv.New()))
				StartRun(db.Run)
				old := v4.WriteConfig{Key: uuid.New(), Endpoints: []v4.WriteEndpoint{{
					Key:     "sp",
					Path:    "/setpoint",
					Channel: v4.ChannelField{Pointer: "/value"},
				}}}
				Expect(gorp.WrapWriter[uuid.UUID, v4.WriteConfig](db).Set(ctx, old)).
					To(Succeed())
				MustOpen(http.OpenService(ctx, http.ServiceConfig{
					DB:      db,
					Channel: channelSvc,
					ICD: MustOpen(icd.OpenService(ctx, icd.ServiceConfig{
						DB:   db,
						ImEx: imex.NewService(),
					})),
				}))
				_, closer := MustSucceed2(db.Get(v5.StagedWriteFieldsKVKey(old.Key)))
				Expect(closer.Close()).To(Succeed())
			},
		)

		It(
			"Should move the fields of a write task row from before 0.57 into an ICD",
			func(ctx SpecContext) {
				db := DeferClose(gorp.Wrap(memkv.New()))
				StartRun(db.Run)
				MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: db}))
				statuses := MustOpen(status.OpenService(ctx, status.ServiceConfig{
					DB: db,
				}))
				svc := MustOpen(http.OpenService(ctx, http.ServiceConfig{
					DB:      db,
					Channel: channelSvc,
					ICD: MustOpen(icd.OpenService(ctx, icd.ServiceConfig{
						DB:   db,
						ImEx: imex.NewService(),
					})),
				}))
				row := v2.Task{
					Key:  uuid.New(),
					Name: "Pump",
					Type: "http_write",
					Config: msgpack.EncodedJSON{"endpoints": []any{map[string]any{
						"key":     "sp",
						"path":    "/setpoint",
						"channel": map[string]any{"pointer": "/value"},
					}}},
				}
				Expect(gorp.NewCreate[v2.Key, v2.Task]().Entry(&row).Exec(ctx, db)).
					To(Succeed())
				Expect(ontology.DefineResources(ctx, db, resource.ID{
					Type: resource.TypeTask,
					Key:  row.Key.String(),
				})).To(Succeed())
				MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v2.Key, v2.Task]{
					DB:   db,
					Name: string(resource.TypeTask),
					Migrations: []gorp.Migration{v3.NewMigration(v3.MigrationConfig{
						Status:  statuses,
						Configs: MustSucceed(config.NewRegistry(svc.Stores()...)),
					})},
				}))
				stored := MustSucceed(svc.Write.Read(ctx, db, row.Key))
				Expect(stored).ToNot(HaveKey("inline"))
				Expect(stored["endpoints"]).To(HaveExactElements(HaveKey("icd")))
			},
		)
	})

	Describe("Stores", func() {
		It("Should expose one store per HTTP task type", func() {
			stores := svc.Stores()
			types := make([]string, len(stores))
			for i, s := range stores {
				types[i] = s.Type()
			}
			Expect(types).To(ConsistOf(
				"http_read",
				"http_write",
				"http_scan",
			))
		})
	})

	Describe("Normalize", func() {
		DescribeTable(
			"Should import both read shapes released Cores stamped at version 2",
			func(ctx SpecContext, data msgpack.EncodedJSON, nanFill bool) {
				dev := uuid.New().String()
				Expect(tasks.CreateDevice(ctx, dev)).To(Succeed())
				data["device"] = dev
				out := MustSucceed(svc.Read.Normalize(ctx, db, 2, data))
				Expect(out).To(HaveKeyWithValue("nan_fill_enabled", nanFill))
				Expect(out).To(HaveKeyWithValue("device", dev))
			},
			Entry("v2", msgpack.EncodedJSON{}, false),
			Entry("v3", msgpack.EncodedJSON{"nan_fill_enabled": true}, true),
		)

		It(
			"Should export read configs above every version released Cores stamped",
			func() {
				Expect(svc.Read.Version()).To(BeNumerically(">", legacy.LastVersion))
			},
		)
	})

	Describe("Write", func() {
		It("Should store a decoded read config under the given key", func(
			ctx SpecContext,
		) {
			Expect(tasks.CreateDevice(ctx, "dev-1")).To(Succeed())
			key := MustSucceed(tasks.Create(ctx, svc.Read, msgpack.EncodedJSON{
				"device": "dev-1",
				"rate":   25,
				"endpoints": []any{map[string]any{
					"key":    "ep-1",
					"method": "GET",
					"path":   "/telemetry",
				}},
			}))
			data := MustSucceed(svc.Read.Read(ctx, db, key))
			Expect(data["key"]).To(Equal(key.String()))
			Expect(data["device"]).To(Equal("dev-1"))
			Expect(data["rate"]).To(BeNumerically("==", 25))
		})

		It("Should apply read config schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.Read, msgpack.EncodedJSON{}))
			data := MustSucceed(svc.Read.Read(ctx, db, key))
			Expect(data["rate"]).To(BeNumerically("==", 1))
		})

		It("Should apply write endpoint schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.Write, msgpack.EncodedJSON{
				"endpoints": []any{map[string]any{"key": "ep-1"}},
			}))
			data := MustSucceed(svc.Write.Read(ctx, db, key))
			Expect(data["endpoints"]).To(HaveExactElements(
				HaveKeyWithValue("method", "POST"),
			))
		})

		It("Should apply scan config schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.Scan, msgpack.EncodedJSON{}))
			data := MustSucceed(svc.Scan.Read(ctx, db, key))
			Expect(data["rate"]).To(BeNumerically("==", 0.2))
		})

		It("Should return the read validation error for an invalid endpoint", func(
			ctx SpecContext,
		) {
			Expect(svc.Read.Prepare(ctx, db, uuid.New(), "", msgpack.EncodedJSON{
				"endpoints": []any{map[string]any{"method": "BOGUS"}},
			})).Error().To(MatchError(ContainSubstring("invalid method: BOGUS")))
		})

		It("Should return the write validation error for an invalid endpoint", func(
			ctx SpecContext,
		) {
			Expect(svc.Write.Prepare(ctx, db, uuid.New(), "", msgpack.EncodedJSON{
				"endpoints": []any{map[string]any{"method": "BOGUS"}},
			})).Error().To(MatchError(ContainSubstring("invalid method: BOGUS")))
		})
	})

	Describe("Import", func() {
		It("Should move the fields of a v4 config into an ICD named after its task",
			func(ctx SpecContext) {
				idx := createChannel(ctx, telem.TimestampT, channel.Channel{})
				value := createChannel(ctx, telem.Float32T, idx)
				data := MustSucceed(svc.Read.Normalize(ctx, db, 4, msgpack.EncodedJSON{
					"endpoints": []any{map[string]any{
						"key":    "ep",
						"method": "POST",
						"path":   "/values",
						"index":  "t",
						"fields": []any{
							map[string]any{
								"key":       "t",
								"channel":   idx.Key(),
								"pointer":   "/t",
								"data_type": "timestamp",
							},
							map[string]any{
								"key":       "p",
								"channel":   value.Key(),
								"pointer":   "/p",
								"data_type": "float32",
							},
						},
					}},
				}))
				key := MustSucceed(tasks.Create(ctx, svc.Read, data))
				stored := MustSucceed(svc.Read.Read(ctx, db, key))
				Expect(stored).ToNot(HaveKey("inline"))
				Expect(
					stored["endpoints"],
				).To(HaveExactElements(Not(HaveKey("fields"))))
				var cfg http.ReadConfig
				Expect(stored.Unmarshal(&cfg)).To(Succeed())
				Expect(cfg.Endpoints).To(HaveLen(1))
				ep := cfg.Endpoints[0]
				var ic icd.ICD
				Expect(icdSvc.NewRetrieve().
					Where(icd.MatchKeys(*ep.ICD)).
					Entry(&ic).
					Exec(ctx, db)).To(Succeed())
				Expect(ic.Name).To(Equal("http_read"))
				Expect(ic.Messages).To(HaveLen(1))
				m := ic.Messages[0]
				Expect(m.Name).To(Equal("POST /values"))
				fields := as[icd.JSONPayload](m.Payload.Variant).Fields
				index := fields[0].Base().Key
				Expect(ep.Messages).To(Equal([]icd.ReadMessage{{
					Message: m.Key,
					Index:   &index,
					Bindings: []icd.ReadBinding{
						{Field: index, Channel: idx.Key()},
						{Field: fields[1].Base().Key, Channel: value.Key()},
					},
				}}))
			},
		)

		It("Should move the channel and fields of a v4 write config into an ICD",
			func(ctx SpecContext) {
				idx := createChannel(ctx, telem.TimestampT, channel.Channel{})
				cmd := createChannel(ctx, telem.Float64T, idx)
				data := MustSucceed(svc.Write.Normalize(ctx, db, 4, msgpack.EncodedJSON{
					"endpoints": []any{map[string]any{
						"key":    "sp",
						"method": "PUT",
						"path":   "/setpoint",
						"channel": map[string]any{
							"pointer":   "/value",
							"channel":   cmd.Key(),
							"data_type": "float64",
						},
						"fields": []any{map[string]any{
							"type":      "static",
							"key":       "u",
							"pointer":   "/unit",
							"json_type": "string",
							"value":     "psi",
						}},
					}},
				}))
				key := MustSucceed(tasks.Create(ctx, svc.Write, data))
				stored := MustSucceed(svc.Write.Read(ctx, db, key))
				Expect(stored).ToNot(HaveKey("inline"))
				var cfg http.WriteConfig
				Expect(stored.Unmarshal(&cfg)).To(Succeed())
				Expect(cfg.Endpoints).To(HaveLen(1))
				ep := cfg.Endpoints[0]
				var ic icd.ICD
				Expect(icdSvc.NewRetrieve().
					Where(icd.MatchKeys(*ep.ICD)).
					Entry(&ic).
					Exec(ctx, db)).To(Succeed())
				Expect(ic.Name).To(Equal("http_write"))
				Expect(ic.Messages).To(HaveLen(1))
				m := ic.Messages[0]
				Expect(m.Name).To(Equal("PUT /setpoint"))
				fields := as[icd.JSONPayload](m.Payload.Variant).Fields
				Expect(fields).To(HaveLen(2))
				Expect(ep.Message).To(Equal(&icd.WriteMessage{
					Message: m.Key,
					Bindings: []icd.WriteBinding{{Variant: icd.ChannelWriteBinding{
						WriteBaseBinding: icd.WriteBaseBinding{
							Field: fields[0].Base().Key,
						},
						Channel: cmd.Key(),
					}}},
				}))
			},
		)

		DescribeTable(
			"Should return the write config it stores, with its fields in an ICD",
			func(ctx SpecContext, version int) {
				idx := createChannel(ctx, telem.TimestampT, channel.Channel{})
				cmd := createChannel(ctx, telem.Float64T, idx)
				channelField := map[string]any{
					"pointer":   "/value",
					"json_type": "number",
					"channel":   cmd.Key(),
					"name":      cmd.Name,
					"data_type": "float64",
				}
				data := MustSucceed(svc.Write.Normalize(
					ctx, db, imex.Version(version), msgpack.EncodedJSON{
						"endpoints": []any{map[string]any{
							"key":     "sp",
							"method":  "PUT",
							"path":    "/setpoint",
							"channel": channelField,
							"fields": []any{map[string]any{
								"type":      "static",
								"key":       "u",
								"pointer":   "/unit",
								"json_type": "string",
								"value":     "psi",
							}},
						}},
					},
				))
				Expect(data).To(HaveKey("inline"))
				key := MustSucceed(tasks.Create(ctx, svc.Write, msgpack.EncodedJSON{}))
				rec := MustSucceed(svc.Write.Prepare(ctx, db, key, "Pump", data))
				returned := MustSucceed(rec.Data())
				Expect(returned).ToNot(HaveKey("inline"))
				Expect(
					db.WithTx(
						ctx,
						func(tx gorp.Tx) error { return rec.Write(ctx, tx) },
					),
				).
					To(Succeed())
				Expect(svc.Write.Read(ctx, db, key)).To(Equal(returned))
				Expect(returned["endpoints"]).To(HaveExactElements(SatisfyAll(
					HaveKey("icd"),
					HaveKey("message"),
				)))
			},
			Entry("from a v4 export", 4),
			Entry("from a Console-era config, as a legacy import reads it", 0),
		)
	})

	Describe("Write references", func() {
		var (
			m   icd.Message
			ic  icd.ICD
			cmd channel.Channel
			key uuid.UUID
		)
		BeforeEach(func(ctx SpecContext) {
			m = jsonMessage(
				"Setpoint",
				jsonField("value", "/value", icd.FloatJSONField{}),
			)
			ic = icd.ICD{Name: "Pump", Messages: []icd.Message{m}}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return icdSvc.NewWriter(tx).Create(ctx, &ic)
			})).To(Succeed())
			idx := createChannel(ctx, telem.TimestampT, channel.Channel{})
			cmd = createChannel(ctx, telem.Float64T, idx)
			key = MustSucceed(tasks.Create(ctx, svc.Write, msgpack.EncodedJSON{
				"endpoints": []any{map[string]any{
					"key":  "sp",
					"path": "/setpoint",
					"icd":  ic.Key,
					"message": map[string]any{
						"message": m.Key,
						"bindings": []any{map[string]any{
							"type":    "channel",
							"field":   fieldKey(m, 0),
							"channel": cmd.Key(),
						}},
					},
				}},
			}))
		})

		It("Should refuse deleting the ICD that an endpoint sends", func(
			ctx SpecContext,
		) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return icdSvc.NewWriter(tx).Delete(ctx, ic.Key)
			})).To(MatchError(gorp.RestrictError{
				Target:    gorp.NewTarget("icd", ic.Key),
				Holder:    gorp.NewTarget("http_write_config", key),
				Reference: "endpoints.icd",
			}))
		})

		It("Should refuse deleting a channel that an endpoint binds", func(
			ctx SpecContext,
		) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Delete(ctx, cmd.Key(), false)
			})).To(MatchError(gorp.RestrictError{
				Target:    gorp.NewTarget("channel", cmd.Key()),
				Holder:    gorp.NewTarget("http_write_config", key),
				Reference: "endpoints.message.bindings.channel",
			}))
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

	DescribeTable("Should reject a read config whose rate is not positive",
		func(ctx SpecContext, rate float64) {
			Expect(svc.Read.Prepare(ctx, db, uuid.New(), "", msgpack.EncodedJSON{
				"rate": rate,
			})).Error().To(MatchError(ContainSubstring("rate: must be greater than 0")))
		},
		Entry("zero", 0.0),
		Entry("negative", -1.0),
	)
})
