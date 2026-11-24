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
	tasktestutil "github.com/synnaxlabs/synnax/pkg/service/task/testutil"
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
		It("Should reject a config missing the DB", func(ctx SpecContext) {
			Expect(http.OpenService(ctx, http.ServiceConfig{})).Error().
				To(MatchError(ContainSubstring("db: must be non-nil")))
		})

		It("Should reject a config missing the channel service", func(ctx SpecContext) {
			Expect(http.OpenService(ctx, http.ServiceConfig{DB: db})).Error().
				To(MatchError(ContainSubstring("channel: must be non-nil")))
		})

		It("Should reject a config missing the ICD service", func(ctx SpecContext) {
			Expect(http.OpenService(ctx, http.ServiceConfig{
				DB:      db,
				Channel: channelSvc,
			})).Error().To(MatchError(ContainSubstring("icd: must be non-nil")))
		})

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
				var cfg http.ReadConfig
				Expect(stored.Unmarshal(&cfg)).To(Succeed())
				Expect(cfg.Endpoints).To(HaveLen(1))
				ep := cfg.Endpoints[0]
				Expect(ep.Fields).To(HaveLen(2))
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
