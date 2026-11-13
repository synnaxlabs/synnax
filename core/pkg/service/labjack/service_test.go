// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package labjack_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channeltestutil "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/labjack"
	v2 "github.com/synnaxlabs/synnax/pkg/service/labjack/versions/v2"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	tasktestutil "github.com/synnaxlabs/synnax/pkg/service/task/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/patch"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Service", func() {
	var (
		svc        *labjack.Service
		statusSvc  *status.Service
		channelSvc *channel.Service
		tasks      *tasktestutil.Tasks
	)
	BeforeEach(func(ctx SpecContext) {
		statusSvc = MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB: db,
		}))
		channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
			Channel: node.Channel,
			DB:      db,
			HostKey: node.Identity.HostKey,
			Status:  statusSvc,
		}))
		svc = MustOpen(labjack.OpenService(ctx, labjack.ServiceConfig{
			DB:     db,
			Status: statusSvc,
		}))
		tasks = MustOpen(tasktestutil.OpenTasks(ctx, tasktestutil.TasksConfig{
			DB:     db,
			Status: statusSvc,
			Stores: svc.Stores(),
		}))
	})

	Describe("OpenService", func() {
		DescribeTable(
			"Should reject a config missing a required field",
			func(ctx SpecContext, field string, unset func(*labjack.ServiceConfig)) {
				cfg := labjack.ServiceConfig{DB: db, Status: statusSvc}
				unset(&cfg)
				Expect(labjack.OpenService(ctx, cfg)).Error().
					To(MatchError(ContainSubstring(field + ": must be non-nil")))
			},
			Entry("db", "db", func(c *labjack.ServiceConfig) { c.DB = nil }),
			Entry(
				"status",
				"status",
				func(c *labjack.ServiceConfig) { c.Status = nil },
			),
		)

		It("Should open the device service", func() {
			Expect(svc.Devices).ToNot(BeNil())
		})
	})

	Describe("Stores", func() {
		It("Should expose one store per LabJack task type", func() {
			types := []string{}
			for _, s := range svc.Stores() {
				types = append(types, s.Type())
			}
			Expect(types).To(ConsistOf(
				"labjack_read",
				"labjack_write",
				"labjack_scan",
			))
		})
	})

	Describe("Write", func() {
		It("Should store a decoded read config under the given key", func(
			ctx SpecContext,
		) {
			Expect(tasks.CreateDevice(ctx, "dev-1")).To(Succeed())
			ch := MustSucceed(channeltestutil.CreateVirtual(ctx, db, channelSvc, 1))[0]
			key := MustSucceed(tasks.Create(ctx, svc.Read, msgpack.EncodedJSON{
				"device":      "dev-1",
				"sample_rate": 25,
				"channels": []any{map[string]any{
					"type":    "analog",
					"key":     "chan-1",
					"port":    "AIN0",
					"channel": ch,
					"scale":   map[string]any{"type": "none"},
				}},
			}))
			data := MustSucceed(svc.Read.Read(ctx, db, key))
			Expect(data["key"]).To(Equal(key.String()))
			Expect(data["device"]).To(Equal("dev-1"))
			Expect(data["sample_rate"]).To(BeNumerically("==", 25))
		})

		It("Should apply read config schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.Read, msgpack.EncodedJSON{}))
			data := MustSucceed(svc.Read.Read(ctx, db, key))
			Expect(data["sample_rate"]).To(BeNumerically("==", 10))
			Expect(data["stream_rate"]).To(BeNumerically("==", 5))
		})

		It("Should apply write config schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.Write, msgpack.EncodedJSON{}))
			data := MustSucceed(svc.Write.Read(ctx, db, key))
			Expect(data["state_rate"]).To(BeNumerically("==", 10))
		})

		It("Should apply scan config schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.Scan, msgpack.EncodedJSON{}))
			data := MustSucceed(svc.Scan.Read(ctx, db, key))
			Expect(data["rate"]).To(BeNumerically("==", 0.2))
			Expect(data["tcp_scan_multiplier"]).To(BeNumerically("==", 10))
		})

		It("Should return the read validation error for an invalid channel", func(
			ctx SpecContext,
		) {
			Expect(svc.Read.Prepare(ctx, db, uuid.New(), "", msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type":              "thermocouple",
					"thermocouple_type": "BOGUS",
				}},
			})).Error().To(MatchError(ContainSubstring("invalid thermocouple_type: BOGUS")))
		})

		It("Should reject a negative channel past AIN127", func(ctx SpecContext) {
			Expect(svc.Read.Prepare(ctx, db, uuid.New(), "", msgpack.EncodedJSON{
				"channels": []any{map[string]any{"type": "analog", "neg_chan": 199}},
			})).Error().To(MatchError(ContainSubstring("neg_chan")))
		})

		DescribeTable(
			"Should lift a v2 negative channel",
			func(ctx SpecContext, ch map[string]any, expected types.GomegaMatcher) {
				ch["type"], ch["key"], ch["scale"] = "analog", "ai", map[string]any{
					"type": "none",
				}
				out := MustSucceed(svc.Read.Normalize(ctx, db, v2.Version,
					msgpack.EncodedJSON{"channels": []any{ch}},
				))
				Expect(out["channels"]).To(ConsistOf(expected))
			},
			Entry("ground", map[string]any{"neg_chan": 199}, Not(HaveKey("neg_chan"))),
			Entry("absent", map[string]any{}, Not(HaveKey("neg_chan"))),
			Entry(
				"pair",
				map[string]any{"neg_chan": 3},
				HaveKeyWithValue("neg_chan", BeNumerically("==", 3)),
			),
		)

		It("Should map a v2 backlog count of 0 to absent", func(ctx SpecContext) {
			out := MustSucceed(svc.Read.Normalize(ctx, db, v2.Version,
				msgpack.EncodedJSON{"device_scan_backlog_warn_on_count": 0},
			))
			Expect(out).ToNot(HaveKey("device_scan_backlog_warn_on_count"))
		})

		It("Should give an absent v2 TCP scan multiplier the default", func(
			ctx SpecContext,
		) {
			out := MustSucceed(svc.Scan.Normalize(ctx, db, v2.Version,
				msgpack.EncodedJSON{},
			))
			Expect(out["tcp_scan_multiplier"]).To(BeNumerically("==", 10))
		})

		It("Should reject a scan backlog count of 0", func(ctx SpecContext) {
			Expect(svc.Read.Prepare(ctx, db, uuid.New(), "", msgpack.EncodedJSON{
				"device_scan_backlog_warn_on_count": 0,
			})).Error().To(MatchError(ContainSubstring(
				"device_scan_backlog_warn_on_count",
			)))
		})
	})
	Describe("Explicit zeros", func() {
		analog := func(scale map[string]any) msgpack.EncodedJSON {
			return msgpack.EncodedJSON{"channels": []any{map[string]any{
				"type":  "analog",
				"key":   "ai",
				"port":  "AIN0",
				"scale": scale,
			}}}
		}
		linear := func(slope float64) map[string]any {
			return map[string]any{"type": "linear", "slope": slope, "offset": 5}
		}
		scaleOf := func(data msgpack.EncodedJSON) map[string]any {
			GinkgoHelper()
			channels := data["channels"].([]any)
			Expect(channels).To(HaveLen(1))
			return channels[0].(map[string]any)["scale"].(map[string]any)
		}
		edit := func(ctx SpecContext, key uuid.UUID, a patch.Action) {
			GinkgoHelper()
			c := MustSucceed(patch.Begin(a)).Next()
			rec, _ := MustSucceed2(svc.Read.Edit(ctx, db, key, "", []patch.Cursor{c}))
			Expect(rec.Write(ctx, db)).To(Succeed())
		}
		slopePath := []string{"config", "channels", "ai", "scale", "slope"}

		It("Should keep a linear scale slope of 0 on create", func(ctx SpecContext) {
			key := MustSucceed(tasks.Create(ctx, svc.Read, analog(linear(0))))
			data := MustSucceed(svc.Read.Read(ctx, db, key))
			Expect(scaleOf(data)).To(HaveKeyWithValue("slope", BeNumerically("==", 0)))
		})

		It("Should keep a linear scale slope of 0 on edit", func(ctx SpecContext) {
			key := MustSucceed(tasks.Create(ctx, svc.Read, analog(linear(2))))
			edit(ctx, key, patch.Set{Path: slopePath, Value: 0.0})
			data := MustSucceed(svc.Read.Read(ctx, db, key))
			Expect(scaleOf(data)).To(HaveKeyWithValue("slope", BeNumerically("==", 0)))
		})

		It("Should give a removed linear scale slope its default", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.Read, analog(linear(2))))
			edit(ctx, key, patch.Remove{Path: slopePath[:4], Key: "slope"})
			data := MustSucceed(svc.Read.Read(ctx, db, key))
			Expect(scaleOf(data)).To(HaveKeyWithValue("slope", BeNumerically("==", 1)))
		})

		It("Should keep a linear scale slope of 0 on import", func(ctx SpecContext) {
			out := MustSucceed(svc.Read.Normalize(ctx, db, svc.Read.Version(),
				analog(linear(0)),
			))
			Expect(scaleOf(out)).To(HaveKeyWithValue("slope", BeNumerically("==", 0)))
		})

		It("Should keep a thermocouple CJC slope of 0", func(ctx SpecContext) {
			key := MustSucceed(tasks.Create(ctx, svc.Read, msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type":      "thermocouple",
					"key":       "tc",
					"port":      "AIN0",
					"cjc_slope": 0,
					"scale":     map[string]any{"type": "none"},
				}},
			}))
			data := MustSucceed(svc.Read.Read(ctx, db, key))
			Expect(data["channels"]).To(ConsistOf(
				HaveKeyWithValue("cjc_slope", BeNumerically("==", 0)),
			))
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

	DescribeTable(
		"Should reject a read config with an invalid rate",
		func(ctx SpecContext, sampleRate, streamRate float64, msg string) {
			Expect(svc.Read.Prepare(ctx, db, uuid.New(), "", msgpack.EncodedJSON{
				"sample_rate": sampleRate,
				"stream_rate": streamRate,
			})).Error().To(MatchError(ContainSubstring(msg)))
		},
		Entry("zero sample rate", 0.0, 0.0, "sample_rate: must be greater than 0"),
		Entry(
			"negative sample rate",
			-1.0,
			-2.0,
			"sample_rate: must be greater than 0",
		),
		Entry("zero stream rate", 10.0, 0.0, "stream_rate: must be greater than 0"),
		Entry(
			"negative stream rate",
			10.0,
			-1.0,
			"stream_rate: must be greater than 0",
		),
	)

	DescribeTable("Should reject a write config whose state rate is not positive",
		func(ctx SpecContext, rate float64) {
			Expect(svc.Write.Prepare(ctx, db, uuid.New(), "", msgpack.EncodedJSON{
				"state_rate": rate,
			})).Error().To(MatchError(ContainSubstring("state_rate: must be greater than 0")))
		},
		Entry("zero", 0.0),
		Entry("negative", -1.0),
	)
})
