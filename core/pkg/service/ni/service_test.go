// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ni_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ni"
	v2 "github.com/synnaxlabs/synnax/pkg/service/ni/versions/v2"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	tasktestutil "github.com/synnaxlabs/synnax/pkg/service/task/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/patch"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Service", func() {
	var (
		stat  *status.Service
		svc   *ni.Service
		tasks *tasktestutil.Tasks
	)
	BeforeEach(func(ctx SpecContext) {
		MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
		MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: db}))
		stat = MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
		svc = MustOpen(ni.OpenService(ctx, ni.ServiceConfig{
			DB:     db,
			Status: stat,
		}))
		tasks = MustOpen(tasktestutil.OpenTasks(ctx, tasktestutil.TasksConfig{
			DB:     db,
			Status: stat,
			Stores: svc.Stores(),
		}))
	})

	Describe("OpenService", func() {
		DescribeTable(
			"Should reject a config missing a required field",
			func(ctx SpecContext, field string, unset func(*ni.ServiceConfig)) {
				cfg := ni.ServiceConfig{DB: db, Status: stat}
				unset(&cfg)
				Expect(ni.OpenService(ctx, cfg)).Error().
					To(MatchError(ContainSubstring(field + ": must be non-nil")))
			},
			Entry("db", "db", func(c *ni.ServiceConfig) { c.DB = nil }),
			Entry("status", "status", func(c *ni.ServiceConfig) { c.Status = nil }),
		)

		It("Should open the chassis and device services", func() {
			Expect(svc.Chassis).ToNot(BeNil())
			Expect(svc.Devices).ToNot(BeNil())
		})
	})

	Describe("Stores", func() {
		It("Should expose one store per NI task type", func() {
			types := []string{}
			for _, s := range svc.Stores() {
				types = append(types, s.Type())
			}
			Expect(types).To(ConsistOf(
				"ni_analog_read",
				"ni_analog_write",
				"ni_counter_read",
				"ni_digital_read",
				"ni_digital_write",
				"ni_scanner",
			))
		})
	})

	Describe("Write", func() {
		It("Should store a decoded analog read config under the given key", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.AnalogRead, msgpack.EncodedJSON{
				"sample_rate": 25,
				"stream_rate": 5,
				"channels":    []any{},
			}))
			data := MustSucceed(svc.AnalogRead.Read(ctx, db, key))
			Expect(data["key"]).To(Equal(key.String()))
			Expect(data["sample_rate"]).To(BeNumerically("==", 25))
			Expect(data["stream_rate"]).To(BeNumerically("==", 5))
		})

		It("Should apply analog read schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.AnalogRead, msgpack.EncodedJSON{}))
			data := MustSucceed(svc.AnalogRead.Read(ctx, db, key))
			Expect(data["sample_rate"]).To(BeNumerically("==", 10))
			Expect(data["stream_rate"]).To(BeNumerically("==", 5))
		})

		It("Should apply analog write schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := MustSucceed(
				tasks.Create(ctx, svc.AnalogWrite, msgpack.EncodedJSON{}),
			)
			data := MustSucceed(svc.AnalogWrite.Read(ctx, db, key))
			Expect(data["state_rate"]).To(BeNumerically("==", 10))
		})

		It("Should apply counter read schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := MustSucceed(
				tasks.Create(ctx, svc.CounterRead, msgpack.EncodedJSON{}),
			)
			data := MustSucceed(svc.CounterRead.Read(ctx, db, key))
			Expect(data["sample_rate"]).To(BeNumerically("==", 10))
		})

		It("Should apply digital read schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := MustSucceed(
				tasks.Create(ctx, svc.DigitalRead, msgpack.EncodedJSON{}),
			)
			data := MustSucceed(svc.DigitalRead.Read(ctx, db, key))
			Expect(data["sample_rate"]).To(BeNumerically("==", 10))
		})

		It("Should apply digital write schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := MustSucceed(
				tasks.Create(ctx, svc.DigitalWrite, msgpack.EncodedJSON{}),
			)
			data := MustSucceed(svc.DigitalWrite.Read(ctx, db, key))
			Expect(data["state_rate"]).To(BeNumerically("==", 10))
		})

		It("Should apply scanner schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.Scanner, msgpack.EncodedJSON{}))
			data := MustSucceed(svc.Scanner.Read(ctx, db, key))
			Expect(data["rate"]).To(BeNumerically("==", 0.2))
		})

		It("Should keep a deliberate zero on a grouped default", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.AnalogRead, msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type":    "ai_voltage",
					"min_val": -10,
					"max_val": 0,
					"custom_scale": map[string]any{
						"type":       "map",
						"scaled_min": -5,
						"scaled_max": 0,
					},
				}},
			}))
			data := MustSucceed(svc.AnalogRead.Read(ctx, db, key))
			ch := data["channels"].([]any)[0].(map[string]any)
			Expect(ch["min_val"]).To(BeNumerically("==", -10))
			Expect(ch["max_val"]).To(BeNumerically("==", 0))
			scale := ch["custom_scale"].(map[string]any)
			Expect(scale["scaled_min"]).To(BeNumerically("==", -5))
			Expect(scale["scaled_max"]).To(BeNumerically("==", 0))
			// The untouched pre-scaled pair is all zero, so its group still fills.
			Expect(scale["pre_scaled_max"]).To(BeNumerically("==", 1))
		})

		It("Should fill a grouped default when every member is zero", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.AnalogRead, msgpack.EncodedJSON{
				"channels": []any{map[string]any{"type": "ai_voltage"}},
			}))
			data := MustSucceed(svc.AnalogRead.Read(ctx, db, key))
			ch := data["channels"].([]any)[0].(map[string]any)
			Expect(ch["min_val"]).To(BeNumerically("==", 0))
			Expect(ch["max_val"]).To(BeNumerically("==", 1))
		})

		It("Should keep a deliberate zero on a counter channel range", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.CounterRead, msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type":    "ci_velocity_linear",
					"min_val": -10,
					"max_val": 0,
				}},
			}))
			data := MustSucceed(svc.CounterRead.Read(ctx, db, key))
			ch := data["channels"].([]any)[0].(map[string]any)
			Expect(ch["min_val"]).To(BeNumerically("==", -10))
			Expect(ch["max_val"]).To(BeNumerically("==", 0))
		})

		It("Should fill a counter channel range when both bounds are zero", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.CounterRead, msgpack.EncodedJSON{
				"channels": []any{map[string]any{"type": "ci_frequency"}},
			}))
			data := MustSucceed(svc.CounterRead.Read(ctx, db, key))
			ch := data["channels"].([]any)[0].(map[string]any)
			Expect(ch["min_val"]).To(BeNumerically("==", 2))
			Expect(ch["max_val"]).To(BeNumerically("==", 100))
		})

		It(
			"Should return the analog read validation error for an invalid channel",
			func(
				ctx SpecContext,
			) {
				Expect(
					svc.AnalogRead.Prepare(ctx, db, uuid.New(), "", msgpack.EncodedJSON{
						"channels": []any{map[string]any{
							"type":            "ai_voltage",
							"terminal_config": "BOGUS",
						}},
					}),
				).Error().
					To(MatchError(ContainSubstring("invalid terminal_config: BOGUS")))
			},
		)

		It(
			"Should return the analog write validation error for an invalid channel",
			func(
				ctx SpecContext,
			) {
				Expect(
					svc.AnalogWrite.Prepare(
						ctx,
						db,
						uuid.New(),
						"",
						msgpack.EncodedJSON{
							"channels": []any{map[string]any{
								"type": "ao_voltage",
								"custom_scale": map[string]any{
									"type":             "linear",
									"pre_scaled_units": "BOGUS",
								},
							}},
						},
					),
				).Error().
					To(MatchError(ContainSubstring("invalid pre_scaled_units: BOGUS")))
			},
		)

		It(
			"Should reject a voltage-mode sensitivity unit on a charge accelerometer",
			func(ctx SpecContext) {
				Expect(
					svc.AnalogRead.Prepare(ctx, db, uuid.New(), "", msgpack.EncodedJSON{
						"channels": []any{map[string]any{
							"type":              "ai_accel_charge",
							"sensitivity_units": "mVoltsPerG",
						}},
					}),
				).Error().
					To(MatchError(
						ContainSubstring("invalid sensitivity_units: mVoltsPerG"),
					))
			},
		)

		It(
			"Should return the counter read validation error for an invalid channel",
			func(
				ctx SpecContext,
			) {
				Expect(
					svc.CounterRead.Prepare(
						ctx,
						db,
						uuid.New(),
						"",
						msgpack.EncodedJSON{
							"channels": []any{map[string]any{
								"type":  "ci_frequency",
								"units": "BOGUS",
							}},
						},
					),
				).Error().
					To(MatchError(ContainSubstring("invalid units: BOGUS")))
			},
		)

		It("Should reject a digital read line above 31", func(ctx SpecContext) {
			Expect(
				svc.DigitalRead.Prepare(ctx, db, uuid.New(), "", msgpack.EncodedJSON{
					"channels": []any{map[string]any{"port": 0, "line": 32}},
				}),
			).Error().
				To(MatchError(ContainSubstring("must be less than or equal to 31")))
		})

		It("Should reject a digital write line above 31", func(ctx SpecContext) {
			Expect(
				svc.DigitalWrite.Prepare(ctx, db, uuid.New(), "", msgpack.EncodedJSON{
					"channels": []any{map[string]any{"port": 0, "line": 32}},
				}),
			).Error().
				To(MatchError(ContainSubstring("must be less than or equal to 31")))
		})
	})
	Describe("Explicit zeros", func() {
		channel := func(data msgpack.EncodedJSON) map[string]any {
			GinkgoHelper()
			channels := data["channels"].([]any)
			Expect(channels).To(HaveLen(1))
			return channels[0].(map[string]any)
		}

		It("Should keep a linear custom scale slope of 0", func(ctx SpecContext) {
			key := MustSucceed(tasks.Create(ctx, svc.AnalogRead, msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type": "ai_voltage",
					"key":  "ai",
					"custom_scale": map[string]any{
						"type":        "linear",
						"slope":       0,
						"y_intercept": 5,
					},
				}},
			}))
			data := MustSucceed(svc.AnalogRead.Read(ctx, db, key))
			Expect(channel(data)["custom_scale"]).
				To(HaveKeyWithValue("slope", BeNumerically("==", 0)))
		})

		It("Should give a v2 divisor of 0 the default on import", func(
			ctx SpecContext,
		) {
			out := MustSucceed(svc.CounterRead.Normalize(ctx, db, v2.Version,
				msgpack.EncodedJSON{"channels": []any{map[string]any{
					"type":    "ci_frequency",
					"key":     "ci",
					"divisor": 0,
				}}},
			))
			Expect(channel(out)).To(HaveKeyWithValue("divisor", BeNumerically("==", 4)))
		})

		It("Should fill a grouped default that an edit leaves all zero", func(
			ctx SpecContext,
		) {
			key := MustSucceed(tasks.Create(ctx, svc.AnalogRead, msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type":    "ai_voltage",
					"key":     "ai",
					"min_val": 0,
					"max_val": 5,
				}},
			}))
			c := MustSucceed(patch.Begin(patch.Set{
				Path:  []string{"config", "channels", "ai", "max_val"},
				Value: 0.0,
			})).Next()
			rec, changed := MustSucceed2(svc.AnalogRead.Edit(
				ctx, db, key, "", []patch.Cursor{c},
			))
			Expect(changed).To(BeTrue())
			Expect(rec.Write(ctx, db)).To(Succeed())
			data := MustSucceed(svc.AnalogRead.Read(ctx, db, key))
			Expect(
				channel(data),
			).To(HaveKeyWithValue("max_val", BeNumerically("==", 1)))
		})
	})

	DescribeTable("Should reject a scan config whose rate is not positive",
		func(ctx SpecContext, rate float64) {
			Expect(svc.Scanner.Prepare(ctx, db, uuid.New(), "", msgpack.EncodedJSON{
				"rate": rate,
			})).Error().To(MatchError(ContainSubstring("rate: must be greater than 0")))
		},
		Entry("zero", 0.0),
		Entry("negative", -1.0),
	)

	DescribeTable(
		"Should reject a read config with an invalid rate",
		func(ctx SpecContext, sampleRate, streamRate float64, msg string) {
			for _, store := range []config.Store{
				svc.AnalogRead,
				svc.CounterRead,
				svc.DigitalRead,
			} {
				Expect(store.Prepare(ctx, db, uuid.New(), "", msgpack.EncodedJSON{
					"sample_rate": sampleRate,
					"stream_rate": streamRate,
				})).Error().To(MatchError(ContainSubstring(msg)), store.Type())
			}
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
			for _, store := range []config.Store{svc.AnalogWrite, svc.DigitalWrite} {
				Expect(store.Prepare(ctx, db, uuid.New(), "", msgpack.EncodedJSON{
					"state_rate": rate,
				})).Error().To(
					MatchError(ContainSubstring("state_rate: must be greater than 0")),
					store.Type(),
				)
			}
		},
		Entry("zero", 0.0),
		Entry("negative", -1.0),
	)
})
