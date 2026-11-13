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
	tasktestutil "github.com/synnaxlabs/synnax/pkg/service/task/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
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
	})

	Describe("Stores", func() {
		It("Should expose one store per HTTP task type", func() {
			types := []string{}
			for _, s := range svc.Stores() {
				types = append(types, s.Type())
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

		Describe("Read fields", func() {
			var index, data channel.Channel
			BeforeEach(func(ctx SpecContext) {
				suffix := uuid.New().String()[:8]
				index = channel.Channel{
					Name:     "idx_" + suffix,
					DataType: telem.TimestampT,
					IsIndex:  true,
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return channelSvc.NewWriter(tx).Create(ctx, &index)
				})).To(Succeed())
				data = channel.Channel{
					Name:       "data_" + suffix,
					DataType:   telem.Int16T,
					LocalIndex: index.LocalKey,
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return channelSvc.NewWriter(tx).Create(ctx, &data)
				})).To(Succeed())
			})

			readConfig := func(field map[string]any) msgpack.EncodedJSON {
				return msgpack.EncodedJSON{"endpoints": []any{map[string]any{
					"key":    "ep-1",
					"method": "GET",
					"fields": []any{field},
				}}}
			}

			It("Should drop the data type of a field with a channel", func(
				ctx SpecContext,
			) {
				key := MustSucceed(
					tasks.Create(ctx, svc.Read, readConfig(map[string]any{
						"type":      "integer",
						"key":       "f",
						"channel":   uint32(data.Key()),
						"pointer":   "/v",
						"data_type": "int64",
						"fill":      "-3",
					})),
				)
				out := MustSucceed(svc.Read.Read(ctx, db, key))
				endpoints := out["endpoints"].([]any)
				field := endpoints[0].(map[string]any)["fields"].([]any)[0]
				Expect(field).ToNot(HaveKey("data_type"))
				Expect(field).To(HaveKeyWithValue("fill", "-3"))
			})

			DescribeTable("Should reject a field that does not fit its channel",
				func(ctx SpecContext, field func() map[string]any, msg string) {
					cfg := readConfig(field())
					Expect(svc.Read.Prepare(ctx, db, uuid.New(), "", cfg)).
						Error().To(MatchError(ContainSubstring(msg)))
				},
				Entry("another kind", func() map[string]any {
					return map[string]any{
						"type": "float", "key": "f", "channel": uint32(data.Key()),
					}
				}, "kind float does not match kind integer of data type int16"),
				Entry("a fill past the channel's range", func() map[string]any {
					return map[string]any{
						"type":    "integer",
						"key":     "f",
						"channel": uint32(data.Key()),
						"fill":    "40000",
					}
				}, "40000 does not fit data type int16"),
				Entry("a fill on an index channel", func() map[string]any {
					return map[string]any{
						"type":        "timestamp",
						"key":         "f",
						"channel":     uint32(index.Key()),
						"time_format": "unix_sec",
						"fill":        "0",
					}
				}, "an index channel cannot have a fill"),
			)

			It("Should reject a repeated enum label", func(ctx SpecContext) {
				cfg := readConfig(map[string]any{
					"type": "integer",
					"key":  "f",
					"enum_values": []any{
						map[string]any{"label": "ON", "value": "1"},
						map[string]any{"label": "ON", "value": "2"},
					},
				})
				Expect(svc.Read.Prepare(ctx, db, uuid.New(), "", cfg)).Error().
					To(MatchError(ContainSubstring(`enum label "ON" repeats`)))
			})

			It("Should give an imported field the kind of its channel", func(
				ctx SpecContext,
			) {
				out := MustSucceed(svc.Read.Normalize(
					ctx, db, svc.Read.Version(), readConfig(map[string]any{
						"type":      "float",
						"key":       "f",
						"channel":   uint32(data.Key()),
						"data_type": "float64",
						"fill":      2,
						"enum_values": []any{
							map[string]any{"label": "HALF", "value": 0.5},
						},
					}),
				))
				endpoints := out["endpoints"].([]any)
				field := endpoints[0].(map[string]any)["fields"].([]any)[0]
				Expect(field).To(HaveKeyWithValue("type", "integer"))
				Expect(field).To(HaveKeyWithValue("fill", "2"))
				Expect(field).To(HaveKeyWithValue("enum_values", BeEmpty()))
				Expect(field).ToNot(HaveKey("data_type"))
			})
		})

		It("Should return the write validation error for an invalid endpoint", func(
			ctx SpecContext,
		) {
			Expect(svc.Write.Prepare(ctx, db, uuid.New(), "", msgpack.EncodedJSON{
				"endpoints": []any{map[string]any{"method": "BOGUS"}},
			})).Error().To(MatchError(ContainSubstring("invalid method: BOGUS")))
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
