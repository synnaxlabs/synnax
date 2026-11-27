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
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channeltestutil "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/labjack"
	v2 "github.com/synnaxlabs/synnax/pkg/service/labjack/versions/v2"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	tasktestutil "github.com/synnaxlabs/synnax/pkg/service/task/testutil"
	task "github.com/synnaxlabs/synnax/pkg/service/task/versions/v2"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/patch"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Service", func() {
	var (
		svc        *labjack.Service
		channelSvc *channel.Service
		tasks      *tasktestutil.Tasks
		deviceKeys map[string]device.Key
	)
	BeforeEach(func(ctx SpecContext) {
		channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
			Channel: node.Channel,
			TS:      node.Storage.TS,
			DB:      db,
			HostKey: node.Identity.HostKey,
			Status:  statusSvc,
		}))
		models := []device.Device{
			{Name: "T4", Model: "LJM_dtT4"},
			{Name: "T7", Model: "LJM_dtT7"},
			{Name: "T8", Model: "LJM_dtT8"},
		}
		deviceKeys = map[string]device.Key{"": ""}
		for i := range models {
			models[i].Key = uuid.New().String()
			deviceKeys[models[i].Name] = models[i].Key
			models[i].Rack = rackSvc.EmbeddedKey
			models[i].Make, models[i].Location = "LabJack", "usb"
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return devices.NewWriter(tx).CreateMany(ctx, &models)
		})).To(Succeed())
		svc = MustOpen(labjack.OpenService(ctx, labjack.ServiceConfig{
			DB:     db,
			Device: devices,
		}))
		tasks = MustOpen(tasktestutil.OpenTasks(ctx, tasktestutil.TasksConfig{
			DB:     db,
			Status: statusSvc,
			Rack:   rackSvc,
			Device: devices,
			Stores: svc.Stores(),
		}))
	})

	Describe("OpenService", func() {
		DescribeTable("Should reject a config missing a required field",
			func(ctx SpecContext, field string) {
				Expect(labjack.OpenService(ctx, labjack.ServiceConfig{})).Error().
					To(MatchError(ContainSubstring(field + ": must be non-nil")))
			},
			Entry("db", "db"),
			Entry("device", "device"),
		)
	})

	Describe("Stores", func() {
		It("Should expose one store per LabJack task type", func() {
			stores := svc.Stores()
			types := make([]string, len(stores))
			for i, s := range stores {
				types[i] = s.Type()
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
			"Should reject an analog range that is not positive",
			func(ctx SpecContext, rng float64) {
				Expect(svc.Read.Prepare(ctx, db, uuid.New(), "", msgpack.EncodedJSON{
					"channels": []any{map[string]any{"type": "analog", "range": rng}},
				})).Error().To(MatchError(ContainSubstring("range")))
			},
			Entry("zero", 0.0),
			Entry("negative", -1.0),
		)

		readChannel := func(typ, port string, neg int) msgpack.EncodedJSON {
			return msgpack.EncodedJSON{
				"stream_rate": 10,
				"channels": []any{map[string]any{
					"type": typ, "port": port, "neg_chan": neg,
				}},
			}
		}

		DescribeTable(
			"Should accept the differential pair of a negative channel",
			func(ctx SpecContext, typ, port string, neg int) {
				MustSucceed(svc.Read.Prepare(
					ctx, db, uuid.New(), "", readChannel(typ, port, neg),
				))
			},
			Entry("base pair", "analog", "AIN2", 3),
			Entry("Mux80 pair", "analog", "AIN64", 72),
			Entry("thermocouple pair", "thermocouple", "AIN0", 1),
		)

		DescribeTable(
			"Should reject a negative channel that is not the differential pair",
			func(ctx SpecContext, typ, port string, neg int) {
				Expect(svc.Read.Prepare(
					ctx, db, uuid.New(), "", readChannel(typ, port, neg),
				)).Error().To(MatchError(ContainSubstring("neg_chan")))
			},
			Entry("wrong base pair", "analog", "AIN2", 5),
			Entry("odd base positive", "analog", "AIN3", 4),
			Entry("Mux80 negative half", "analog", "AIN56", 64),
			Entry("past AIN12", "analog", "AIN13", 14),
			Entry("thermocouple wrong pair", "thermocouple", "AIN0", 2),
		)

		It("Should lower a stored range to the largest range of the model", func(
			ctx SpecContext,
		) {
			out := MustSucceed(svc.Read.Normalize(ctx, db, v2.Version,
				msgpack.EncodedJSON{
					"device": deviceKeys["T7"],
					"channels": []any{map[string]any{
						"type": "analog", "key": "ai", "port": "AIN0", "range": 20,
						"scale": map[string]any{"type": "none"},
					}},
				},
			))
			Expect(out["channels"]).To(ConsistOf(
				HaveKeyWithValue("range", BeNumerically("==", 10)),
			))
		})

		rangeConfig := func(dev string, rng float64) msgpack.EncodedJSON {
			return msgpack.EncodedJSON{
				"device":      dev,
				"stream_rate": 10,
				"channels": []any{map[string]any{
					"type": "analog", "port": "AIN0", "range": rng,
				}},
			}
		}

		DescribeTable(
			"Should accept an analog range the model of the device covers",
			func(ctx SpecContext, dev string, rng float64) {
				MustSucceed(svc.Read.Prepare(
					ctx, db, uuid.New(), "", rangeConfig(deviceKeys[dev], rng),
				))
			},
			Entry("T7 at its largest range", "T7", 10.0),
			Entry("T8 at its largest range", "T8", 11.0),
			Entry("T4, whose ranges are fixed", "T4", 20.0),
			Entry("no device", "", 20.0),
		)

		DescribeTable(
			"Should reject an analog range above the largest range of the model",
			func(ctx SpecContext, dev string, rng float64) {
				Expect(svc.Read.Prepare(
					ctx, db, uuid.New(), "", rangeConfig(deviceKeys[dev], rng),
				)).Error().To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("channels.0.range")),
				))
			},
			Entry("T7", "T7", 10.5),
			Entry("T8", "T8", 12.0),
		)

		It("Should reject a device that does not exist as a validation error", func(
			ctx SpecContext,
		) {
			Expect(svc.Read.Prepare(
				ctx, db, uuid.New(), "", rangeConfig(uuid.New().String(), 20),
			)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("device")),
			))
		})

		It("Should lower a stored range above the largest range of the model on open",
			func(ctx SpecContext) {
				stored := labjack.ReadConfig{
					Key:    uuid.New(),
					Device: deviceKeys["T7"],
					Channels: []labjack.ReadChannel{{
						Variant: labjack.AnalogReadChannel{
							BaseReadChannel: labjack.BaseReadChannel{
								Key:  "ai",
								Port: "AIN0",
							},
							Range: 20,
							Scale: task.Scale{Variant: task.NoneScale{}},
						},
					}},
				}
				fresh := DeferClose(gorp.Wrap(memkv.New()))
				StartRun(fresh.Run)
				Expect(gorp.NewCreate[uuid.UUID, labjack.ReadConfig]().
					Entry(&stored).Exec(ctx, fresh)).To(Succeed())
				var t7 device.Device
				Expect(devices.NewRetrieve().
					Where(device.MatchKeys(deviceKeys["T7"])).
					Entry(&t7).Exec(ctx, db)).To(Succeed())
				Expect(gorp.NewCreate[device.Key, device.Device]().
					Entry(&t7).Exec(ctx, fresh)).To(Succeed())
				MustOpen(labjack.OpenService(ctx, labjack.ServiceConfig{
					DB:     fresh,
					Device: devices,
				}))
				var got labjack.ReadConfig
				Expect(gorp.NewRetrieve[uuid.UUID, labjack.ReadConfig]().
					Where(gorp.MatchKeys[uuid.UUID, labjack.ReadConfig](stored.Key)).
					Entry(&got).Exec(ctx, fresh)).To(Succeed())
				Expect(got.Channels[0].Variant).To(
					HaveField("Range", BeNumerically("==", 10)),
				)
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
				map[string]any{"port": "AIN2", "neg_chan": 3},
				HaveKeyWithValue("neg_chan", BeNumerically("==", 3)),
			),
		)

		DescribeTable(
			"Should reconcile a stored channel that breaks the channel rules",
			func(ctx SpecContext, ch map[string]any, expected types.GomegaMatcher) {
				ch["type"], ch["key"], ch["scale"] = "analog", "ai", map[string]any{
					"type": "none",
				}
				out := MustSucceed(svc.Read.Normalize(ctx, db, v2.Version,
					msgpack.EncodedJSON{"channels": []any{ch}},
				))
				Expect(out["channels"]).To(ConsistOf(expected))
			},
			Entry(
				"a negative channel that is not the pair",
				map[string]any{"port": "AIN2", "neg_chan": 5},
				HaveKeyWithValue("neg_chan", BeNumerically("==", 3)),
			),
			Entry(
				"a negative channel on a port with no pair",
				map[string]any{"port": "AIN3", "neg_chan": 4},
				Not(HaveKey("neg_chan")),
			),
			Entry(
				"a negative range",
				map[string]any{"range": -1},
				HaveKeyWithValue("range", BeNumerically("==", 10)),
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
			channels, ok := data["channels"].([]any)
			Expect(ok).To(BeTrue())
			Expect(channels).To(HaveLen(1))
			channel, ok := channels[0].(map[string]any)
			Expect(ok).To(BeTrue())
			scale, ok := channel["scale"].(map[string]any)
			Expect(ok).To(BeTrue())
			return scale
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
