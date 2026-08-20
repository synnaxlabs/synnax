// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package legacy_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/ni/versions/legacy"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

var _ = Describe("AnalogRead", func() {
	DescribeTable("legacy shapes",
		func(in, want msgpack.EncodedJSON) {
			Expect(legacy.AnalogRead.Apply(in)).To(Equal(want))
		},
		Entry("copies the v0 config-level device onto channels",
			msgpack.EncodedJSON{
				"device": "dev-1",
				"channels": []any{
					map[string]any{"port": float64(0)},
					map[string]any{"port": float64(1), "device": "dev-2"},
				},
			},
			msgpack.EncodedJSON{
				"channels": []any{
					map[string]any{"port": float64(0), "device": "dev-1"},
					map[string]any{"port": float64(1), "device": "dev-2"},
				},
			},
		),
		Entry("treats an empty channel device as missing during fan-out",
			msgpack.EncodedJSON{
				"device": "dev-1",
				"channels": []any{
					map[string]any{"port": float64(0), "device": ""},
					map[string]any{"port": float64(1), "device": "dev-2"},
				},
			},
			msgpack.EncodedJSON{
				"channels": []any{
					map[string]any{"port": float64(0), "device": "dev-1"},
					map[string]any{"port": float64(1), "device": "dev-2"},
				},
			},
		),
		Entry("drops an empty config-level device without fanning it out",
			msgpack.EncodedJSON{
				"device": "",
				"channels": []any{
					map[string]any{"port": float64(0), "device": "dev-2"},
				},
			},
			msgpack.EncodedJSON{
				"channels": []any{
					map[string]any{"port": float64(0), "device": "dev-2"},
				},
			},
		),
		Entry("replaces the renamed AI type alias",
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{"type": "ai_frequency_voltage"}},
			},
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{"type": "ai_freq_voltage"}},
			},
		),
		Entry("rewrites the coulomb charge units",
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type":  "ai_charge",
					"units": "C",
				}},
			},
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type":  "ai_charge",
					"units": "Coulombs",
				}},
			},
		),
		Entry("rewrites the micro-coulomb charge units to pico-coulombs",
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type":  "ai_charge",
					"units": "uC",
				}},
			},
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type":  "ai_charge",
					"units": "PicoCoulombs",
				}},
			},
		),
		Entry("leaves a non-charge channel's celsius units alone",
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type":  "ai_rtd",
					"units": "C",
				}},
			},
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type":  "ai_rtd",
					"units": "C",
				}},
			},
		),
		Entry("rewrites kebab-case strain bridge names to the DAQmx spellings",
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type":          "ai_strain_gauge",
					"strain_config": "half-bridge-II",
					"units":         "strain",
				}},
			},
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type":          "ai_strain_gauge",
					"strain_config": "HalfBridgeII",
				}},
			},
		),
		Entry("leaves canonical strain bridge names alone",
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type":          "ai_strain_gauge",
					"strain_config": "QuarterBridgeI",
					"units":         "Strain",
				}},
			},
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type":          "ai_strain_gauge",
					"strain_config": "QuarterBridgeI",
				}},
			},
		),
		Entry("leaves a non-strain channel's strain-like values alone",
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type":  "ai_bridge",
					"units": "strain",
				}},
			},
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type":  "ai_bridge",
					"units": "strain",
				}},
			},
		),
		Entry("collapses the constant cold-junction source",
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type":      "ai_thermocouple",
					"cjcSource": "ConstVal",
					"cjcVal":    float64(25),
					"cjcPort":   float64(3),
				}},
			},
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type": "ai_thermocouple",
					"cjc":  map[string]any{"source": "const_val", "val": float64(25)},
				}},
			},
		),
		Entry("collapses the cold-junction reference channel",
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type":       "ai_thermocouple",
					"cjc_source": "Chan",
					"cjc_val":    float64(25),
					"cjc_port":   float64(3),
				}},
			},
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type": "ai_thermocouple",
					"cjc":  map[string]any{"source": "chan", "port": float64(3)},
				}},
			},
		),
		Entry("gives a thermocouple with no cold-junction source the built-in one",
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{"type": "ai_thermocouple"}},
			},
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type": "ai_thermocouple",
					"cjc":  map[string]any{"source": "built_in"},
				}},
			},
		),
		Entry("leaves a non-thermocouple channel's cold junction alone",
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{"type": "ai_voltage"}},
			},
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{"type": "ai_voltage"}},
			},
		),
		Entry("keeps a canonical config unchanged",
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type":   "ai_thermocouple",
					"device": "dev-1",
					"cjc":    map[string]any{"source": "built_in"},
				}},
			},
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type":   "ai_thermocouple",
					"device": "dev-1",
					"cjc":    map[string]any{"source": "built_in"},
				}},
			},
		),
	)
})

var _ = Describe("CounterRead", func() {
	DescribeTable("legacy shapes",
		func(in, want msgpack.EncodedJSON) {
			Expect(legacy.CounterRead.Apply(in)).To(Equal(want))
		},
		Entry("copies the task-level device onto channels without one",
			msgpack.EncodedJSON{
				"device": "dev-ctr",
				"channels": []any{
					map[string]any{"type": "ci_frequency", "device": ""},
					map[string]any{"type": "ci_edge_count", "device": "dev-2"},
				},
			},
			msgpack.EncodedJSON{
				"channels": []any{
					map[string]any{"type": "ci_frequency", "device": "dev-ctr"},
					map[string]any{"type": "ci_edge_count", "device": "dev-2"},
				},
			},
		),
		Entry("rewrites the Python frequency measurement method spelling",
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type":        "ci_frequency",
					"meas_method": "DynAvg",
				}},
			},
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type":        "ci_frequency",
					"meas_method": "DynamicAvg",
				}},
			},
		),
		Entry("keeps a canonical frequency channel unchanged",
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type":        "ci_frequency",
					"meas_method": "DynamicAvg",
					"units":       "Seconds",
				}},
			},
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type":        "ci_frequency",
					"meas_method": "DynamicAvg",
					"units":       "Seconds",
				}},
			},
		),
		Entry("rewrites the measurement method spelling on a period channel",
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type":        "ci_period",
					"meas_method": "DynAvg",
				}},
			},
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type":        "ci_period",
					"meas_method": "DynamicAvg",
				}},
			},
		),
		Entry("leaves other counter channel types alone",
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type":        "ci_pulse_width",
					"meas_method": "DynAvg",
				}},
			},
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type":        "ci_pulse_width",
					"meas_method": "DynAvg",
				}},
			},
		),
	)
})
