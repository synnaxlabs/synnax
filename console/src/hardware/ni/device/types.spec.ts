// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { testPropertiesSchema } from "@/hardware/common/device/testutil";
import { propertiesZ, ZERO_PROPERTIES } from "@/hardware/ni/device/types";

testPropertiesSchema("NI", propertiesZ, ZERO_PROPERTIES, [
  [
    "properties missing counterInput (pre-SY-3060)",
    {
      identifier: "Dev1",
      analogInput: { portCount: 4, index: 0, channels: {} },
      analogOutput: { portCount: 2, stateIndex: 0, channels: {} },
      digitalInputOutput: { portCount: 2, lineCounts: [8, 8] },
      digitalInput: { portCount: 2, lineCounts: [8, 8], index: 0, channels: {} },
      digitalOutput: {
        portCount: 2,
        lineCounts: [8, 8],
        stateIndex: 0,
        channels: {},
      },
    },
  ],
  [
    "partially populated analogOutput (shallow merge from enriched.json)",
    {
      identifier: "Dev1",
      analogInput: { portCount: 4, index: 0, channels: {} },
      analogOutput: { portCount: 2 },
      counterInput: { portCount: 0, index: 0, channels: {} },
      digitalInputOutput: { portCount: 0, lineCounts: [] },
      digitalInput: { portCount: 0, lineCounts: [], index: 0, channels: {} },
      digitalOutput: { portCount: 0, lineCounts: [], stateIndex: 0, channels: {} },
    },
  ],
  [
    "scan-only properties from C++ driver",
    { is_simulated: false, resource_name: "Dev1", is_chassis: false },
  ],
  [
    "real v0.47.1 device properties (snake_case, no counterInput, partial analogOutput)",
    {
      identifier: "n9205",
      analog_input: { port_count: 0, index: 1048585, channels: { "0": 1048586 } },
      analog_output: { port_count: 0 },
      digital_input_output: { port_count: 0, line_counts: [] },
      digital_input: { port_count: 0, line_counts: [], index: 0, channels: {} },
      digital_output: {
        port_count: 0,
        line_counts: [],
        state_index: 0,
        channels: {},
      },
      is_simulated: true,
      resource_name: "6C1DE3E6-1E3A-11F1-80FB-9BC4D3F82D6E",
    },
  ],
  [
    "pre-v0.41.0 analogOutput with only portCount (before SY-1335)",
    {
      identifier: "Dev1",
      analogInput: { portCount: 4, index: 0, channels: {} },
      analogOutput: { portCount: 0 },
      digitalInputOutput: { portCount: 0, lineCounts: [] },
      digitalInput: { portCount: 0, lineCounts: [], index: 0, channels: {} },
      digitalOutput: { portCount: 0, lineCounts: [], stateIndex: 0, channels: {} },
    },
  ],
]);
