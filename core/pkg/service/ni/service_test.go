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
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/ni"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Service", func() {
	var svc *ni.Service
	BeforeEach(func(ctx SpecContext) {
		svc = MustOpen(ni.OpenService(ctx, ni.ServiceConfig{
			DB: db,
		}))
	})

	Describe("OpenService", func() {
		It("Should reject a config missing the DB", func(ctx SpecContext) {
			Expect(ni.OpenService(ctx, ni.ServiceConfig{})).Error().
				To(MatchError(ContainSubstring("db: must be non-nil")))
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
			key := uuid.New()
			Expect(svc.AnalogRead.Write(ctx, nil, key, msgpack.EncodedJSON{
				"sample_rate": 25,
				"stream_rate": 5,
				"channels":    []any{},
			})).To(Succeed())
			data := MustSucceed(svc.AnalogRead.Read(ctx, nil, key))
			Expect(data["key"]).To(Equal(key.String()))
			Expect(data["sample_rate"]).To(BeNumerically("==", 25))
			Expect(data["stream_rate"]).To(BeNumerically("==", 5))
		})

		It("Should apply analog read schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			Expect(svc.AnalogRead.Write(ctx, nil, key, msgpack.EncodedJSON{})).
				To(Succeed())
			data := MustSucceed(svc.AnalogRead.Read(ctx, nil, key))
			Expect(data["sample_rate"]).To(BeNumerically("==", 10))
			Expect(data["stream_rate"]).To(BeNumerically("==", 5))
		})

		It("Should apply analog write schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			Expect(svc.AnalogWrite.Write(ctx, nil, key, msgpack.EncodedJSON{})).
				To(Succeed())
			data := MustSucceed(svc.AnalogWrite.Read(ctx, nil, key))
			Expect(data["state_rate"]).To(BeNumerically("==", 10))
		})

		It("Should apply counter read schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			Expect(svc.CounterRead.Write(ctx, nil, key, msgpack.EncodedJSON{})).
				To(Succeed())
			data := MustSucceed(svc.CounterRead.Read(ctx, nil, key))
			Expect(data["sample_rate"]).To(BeNumerically("==", 10))
		})

		It("Should apply digital read schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			Expect(svc.DigitalRead.Write(ctx, nil, key, msgpack.EncodedJSON{})).
				To(Succeed())
			data := MustSucceed(svc.DigitalRead.Read(ctx, nil, key))
			Expect(data["sample_rate"]).To(BeNumerically("==", 10))
		})

		It("Should apply digital write schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			Expect(svc.DigitalWrite.Write(ctx, nil, key, msgpack.EncodedJSON{})).
				To(Succeed())
			data := MustSucceed(svc.DigitalWrite.Read(ctx, nil, key))
			Expect(data["state_rate"]).To(BeNumerically("==", 10))
		})

		It("Should apply scanner schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			Expect(svc.Scanner.Write(ctx, nil, key, msgpack.EncodedJSON{})).
				To(Succeed())
			data := MustSucceed(svc.Scanner.Read(ctx, nil, key))
			Expect(data["rate"]).To(BeNumerically("==", 0.2))
		})

		It("Should keep a deliberate zero on a grouped default", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			Expect(svc.AnalogRead.Write(ctx, nil, key, msgpack.EncodedJSON{
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
			})).To(Succeed())
			data := MustSucceed(svc.AnalogRead.Read(ctx, nil, key))
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
			key := uuid.New()
			Expect(svc.AnalogRead.Write(ctx, nil, key, msgpack.EncodedJSON{
				"channels": []any{map[string]any{"type": "ai_voltage"}},
			})).To(Succeed())
			data := MustSucceed(svc.AnalogRead.Read(ctx, nil, key))
			ch := data["channels"].([]any)[0].(map[string]any)
			Expect(ch["min_val"]).To(BeNumerically("==", 0))
			Expect(ch["max_val"]).To(BeNumerically("==", 1))
		})

		It(
			"Should return the analog read validation error for an invalid channel",
			func(
				ctx SpecContext,
			) {
				Expect(svc.AnalogRead.Write(ctx, nil, uuid.New(), msgpack.EncodedJSON{
					"channels": []any{map[string]any{
						"type":            "ai_voltage",
						"terminal_config": "BOGUS",
					}},
				})).To(MatchError(ContainSubstring("invalid terminal_config: BOGUS")))
			},
		)

		It(
			"Should return the analog write validation error for an invalid channel",
			func(
				ctx SpecContext,
			) {
				Expect(svc.AnalogWrite.Write(ctx, nil, uuid.New(), msgpack.EncodedJSON{
					"channels": []any{map[string]any{
						"type": "ao_voltage",
						"custom_scale": map[string]any{
							"type":             "linear",
							"pre_scaled_units": "BOGUS",
						},
					}},
				})).To(MatchError(ContainSubstring("invalid pre_scaled_units: BOGUS")))
			},
		)

		It(
			"Should reject a voltage-mode sensitivity unit on a charge accelerometer",
			func(ctx SpecContext) {
				Expect(svc.AnalogRead.Write(ctx, nil, uuid.New(), msgpack.EncodedJSON{
					"channels": []any{map[string]any{
						"type":              "ai_accel_charge",
						"sensitivity_units": "mVoltsPerG",
					}},
				})).To(MatchError(
					ContainSubstring("invalid sensitivity_units: mVoltsPerG"),
				))
			},
		)

		It(
			"Should return the counter read validation error for an invalid channel",
			func(
				ctx SpecContext,
			) {
				Expect(svc.CounterRead.Write(ctx, nil, uuid.New(), msgpack.EncodedJSON{
					"channels": []any{map[string]any{
						"type":  "ci_frequency",
						"units": "BOGUS",
					}},
				})).To(MatchError(ContainSubstring("invalid units: BOGUS")))
			},
		)
	})
})
