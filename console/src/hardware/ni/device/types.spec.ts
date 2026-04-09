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
]);
