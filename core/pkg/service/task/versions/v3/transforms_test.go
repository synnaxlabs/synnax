// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v3 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v3"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

var _ = Describe("Transform", func() {
	DescribeTable("legacy config shapes",
		func(taskType string, in, want msgpack.EncodedJSON) {
			Expect(v3.Transform(taskType, in)).To(Equal(want))
		},
		Entry("flips data_saving on every type",
			"modbus_read",
			msgpack.EncodedJSON{"data_saving": false},
			msgpack.EncodedJSON{"data_saving_disabled": true},
		),
		Entry("renames camelCase base keys",
			"modbus_read",
			msgpack.EncodedJSON{
				"autoStart":  true,
				"dataSaving": true,
				"sampleRate": float64(50),
				"streamRate": float64(25),
			},
			msgpack.EncodedJSON{
				"auto_start":           true,
				"data_saving_disabled": false,
				"sample_rate":          float64(50),
				"stream_rate":          float64(25),
			},
		),
		Entry("flips channel enabled on every type",
			"modbus_read",
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{"enabled": false}},
			},
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{"disabled": true}},
			},
		),
		Entry("keeps a modern config unchanged",
			"modbus_read",
			msgpack.EncodedJSON{
				"data_saving_disabled": true,
				"channels":             []any{map[string]any{"disabled": false}},
			},
			msgpack.EncodedJSON{
				"data_saving_disabled": true,
				"channels":             []any{map[string]any{"disabled": false}},
			},
		),
		Entry("copies the NI read config device onto channels",
			"ni_analog_read",
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
		Entry("renames NI counter camelCase terminals",
			"ni_counter_read",
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"terminalA":  "PFI0",
					"terminalB":  "PFI1",
					"terminalZ":  "PFI2",
					"activeEdge": "Rising",
				}},
			},
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"terminal_a":  "PFI0",
					"terminal_b":  "PFI1",
					"terminal_z":  "PFI2",
					"active_edge": "Rising",
				}},
			},
		),
		Entry("replaces the renamed NI AI type alias",
			"ni_analog_read",
			msgpack.EncodedJSON{
				"channels": []any{
					map[string]any{"type": "ai_frequency_voltage"},
				},
			},
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{"type": "ai_freq_voltage"}},
			},
		),
		Entry("keeps the NI digital read config device",
			"ni_digital_read",
			msgpack.EncodedJSON{
				"device":   "dev-1",
				"channels": []any{map[string]any{"port": float64(0)}},
			},
			msgpack.EncodedJSON{
				"device":   "dev-1",
				"channels": []any{map[string]any{"port": float64(0)}},
			},
		),
		Entry("renames the OPC write channel key",
			"opc_write",
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{"channel": float64(7)}},
			},
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{"cmd_channel": float64(7)}},
			},
		),
		Entry("keeps the OPC read channel key",
			"opc_read",
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{"channel": float64(7)}},
			},
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{"channel": float64(7)}},
			},
		),
		Entry("renames the LabJack write cmd_key and state_key",
			"labjack_write",
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"cmd_key":   float64(7),
					"state_key": float64(8),
				}},
			},
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"cmd_channel":   float64(7),
					"state_channel": float64(8),
				}},
			},
		),
		Entry("normalizes the EtherCAT sub-index spellings",
			"ethercat_read",
			msgpack.EncodedJSON{
				"channels": []any{
					map[string]any{"subindex": float64(1)},
					map[string]any{
						"address": map[string]any{"subIndex": float64(2)},
					},
				},
			},
			msgpack.EncodedJSON{
				"channels": []any{
					map[string]any{"sub_index": float64(1)},
					map[string]any{
						"address": map[string]any{"sub_index": float64(2)},
					},
				},
			},
		),
		Entry("converts HTTP read records to lists and flips fields",
			"http_read",
			msgpack.EncodedJSON{
				"endpoints": []any{map[string]any{
					"headers":      map[string]any{"Accept": "application/json"},
					"query_params": map[string]any{"limit": "10"},
					"fields": []any{map[string]any{
						"enabled":     false,
						"enum_values": map[string]any{"ON": float64(1)},
					}},
				}},
			},
			msgpack.EncodedJSON{
				"endpoints": []any{map[string]any{
					"headers": []any{map[string]any{
						"name": "Accept", "value": "application/json",
					}},
					"query_params": []any{map[string]any{
						"parameter": "limit", "value": "10",
					}},
					"fields": []any{map[string]any{
						"disabled": true,
						"enum_values": []any{map[string]any{
							"label": "ON", "value": float64(1),
						}},
					}},
				}},
			},
		),
		Entry("flips HTTP write endpoints and converts records",
			"http_write",
			msgpack.EncodedJSON{
				"endpoints": []any{map[string]any{
					"enabled":     true,
					"queryParams": map[string]any{"limit": "10"},
				}},
			},
			msgpack.EncodedJSON{
				"endpoints": []any{map[string]any{
					"disabled": false,
					"query_params": []any{map[string]any{
						"parameter": "limit", "value": "10",
					}},
				}},
			},
		),
		Entry("flips PagerDuty alert enabled",
			"pagerduty_alert",
			msgpack.EncodedJSON{
				"routing_key": "rk",
				"alerts":      []any{map[string]any{"enabled": false}},
			},
			msgpack.EncodedJSON{
				"routing_key": "rk",
				"alerts":      []any{map[string]any{"disabled": true}},
			},
		),
		Entry("renames scan_rate and flips enabled on scan tasks",
			"opc_scan",
			msgpack.EncodedJSON{"scan_rate": float64(0.5), "enabled": false},
			msgpack.EncodedJSON{"rate": float64(0.5), "disabled": true},
		),
		Entry("keeps the LabJack scan tcp_scan_multiplier",
			"labjack_scan",
			msgpack.EncodedJSON{"enabled": true, "tcp_scan_multiplier": float64(5)},
			msgpack.EncodedJSON{"disabled": false, "tcp_scan_multiplier": float64(5)},
		),
		Entry("renames scan_rate on the NI scanner",
			"ni_scanner",
			msgpack.EncodedJSON{"scan_rate": float64(1)},
			msgpack.EncodedJSON{"rate": float64(1)},
		),
	)
	It("Should not mutate the input config", func() {
		in := msgpack.EncodedJSON{"data_saving": true}
		v3.Transform("modbus_read", in)
		Expect(in).To(HaveKeyWithValue("data_saving", true))
	})
})
