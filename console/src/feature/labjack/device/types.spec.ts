// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { LabJack } from "@/feature/labjack";
import { testPropertiesSchema } from "@/platform/device/testutil";

testPropertiesSchema(
  "LabJack",
  LabJack.Device.propertiesZ,
  LabJack.Device.ZERO_PROPERTIES,
  [
    [
      "scan-only properties from C++ driver",
      { serial_number: "470026743", device_type: "T7" },
    ],
    [
      "partially populated (only read channels set)",
      {
        identifier: "lj1",
        readIndex: 123,
        AI: { channels: { "0": 456 } },
      },
    ],
    [
      "real v0.51.3 device properties (connection/read/write format, no readIndex/thermocoupleIndex)",
      {
        identifier: "mlt",
        connection: { identifier: "ANY", connection_type: "USB" },
        read: { index: 0, channels: {} },
        write: { channels: {} },
      },
    ],
    [
      "read task configured but no write task (missing writeStateIndex and thermocoupleIndex)",
      {
        identifier: "lj1",
        readIndex: 1048584,
        AI: { channels: { AIN0: 1048585 } },
      },
    ],
  ],
);

describe("negativeChannelOf", () => {
  it.each([
    ["AIN0", 1],
    ["AIN12", 13],
    ["AIN48", 56],
    ["AIN55", 63],
    ["AIN64", 72],
    ["AIN119", 127],
  ])("should pair %s with AIN%d", (port, negChan) => {
    expect(LabJack.Device.negativeChannelOf(port)).toBe(negChan);
  });

  it.each(["AIN1", "AIN13", "AIN14", "AIN56", "AIN120", "DIO4"])(
    "should pair %s with no channel",
    (port) => {
      expect(LabJack.Device.negativeChannelOf(port)).toBeNull();
    },
  );
});
