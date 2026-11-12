// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { EtherCAT } from "@/feature/ethercat";
import { createPDOs } from "@/feature/ethercat/testutil";
import { testPropertiesSchema } from "@/platform/device/testutil";

testPropertiesSchema(
  "EtherCAT",
  EtherCAT.Device.slavePropertiesZ,
  EtherCAT.Device.ZERO_SLAVE_PROPERTIES,
  [
    [
      "scan-only properties (identifier and position only)",
      { identifier: "EL3004", position: 1 },
    ],
    [
      "properties missing PDOs",
      {
        identifier: "EL3004",
        serial: 12345,
        vendorId: 2,
        productCode: 0x0bbc3052,
        revision: 0,
        name: "EL3004",
        network: "eth0",
        position: 1,
      },
    ],
    [
      "properties carrying the PDOs written by a device scan",
      { identifier: "EL3004", position: 1, pdos: createPDOs() },
    ],
    [
      "properties carrying PDO names",
      {
        identifier: "EL1008",
        position: 1,
        pdos: {
          inputs: [
            {
              name: "Input",
              pdoName: "Channel 1",
              index: 0x6000,
              subIndex: 1,
              bitLength: 1,
              dataType: "uint8",
            },
          ],
          outputs: [],
        },
      },
    ],
    [
      "properties carrying startup parameters",
      {
        identifier: "EL3004",
        position: 1,
        startupParameters: [
          {
            index: 0x8000,
            subIndex: 0x15,
            dataType: "uint16",
            value: 3,
            name: "Filter",
          },
        ],
      },
    ],
  ],
);

describe("slavePropertiesZ", () => {
  it("should keep the modes of the slave and the chosen mode", () => {
    const opModes: EtherCAT.Device.OpMode[] = [
      {
        name: "SM-Synchron",
        assignActivate: 0,
        sync0Cycle: 0,
        sync0Factor: 0,
        sync0Shift: 0,
        sync1Factor: 0,
        sync1Shift: 0,
      },
      {
        name: "DC",
        assignActivate: 0x0700,
        sync0Cycle: 0,
        sync0Factor: 1,
        sync0Shift: 0,
        sync1Factor: 0,
        sync1Shift: 0,
      },
    ];
    const opMode: EtherCAT.Device.OpModeChoice = {
      index: 1,
      assignActivate: 0x0700,
      sync0Cycle: 1_000,
      sync1Cycle: 1_000_000_000,
    };
    const parsed = EtherCAT.Device.slavePropertiesZ.parse({
      identifier: "EL3004",
      position: 1,
      opModes,
      opMode,
    });
    expect(parsed.opModes).toEqual(opModes);
    expect(parsed.opMode).toEqual(opMode);
  });

  it("should read the channels stored as a map keyed by address as lists", () => {
    const parsed = EtherCAT.Device.slavePropertiesZ.parse({
      read: { channels: { "24576_1": 5, scaled_24576_1: 6, "24576_2": 7 } },
      write: { channels: { "28672_1": 8, state_28672_1: 9, state_28672_2: 10 } },
    });
    expect(parsed.read.channels).toEqual([
      { index: 0x6000, subIndex: 1, scaled: false, calibrated: false, channel: 5 },
      { index: 0x6000, subIndex: 1, scaled: true, calibrated: false, channel: 6 },
      { index: 0x6000, subIndex: 2, scaled: false, calibrated: false, channel: 7 },
    ]);
    expect(parsed.write.channels).toEqual([
      { index: 0x7000, subIndex: 1, command: 8, state: 9 },
      { index: 0x7000, subIndex: 2, command: 0, state: 10 },
    ]);
  });

  it("should drop the stored channel keys that name no address", () => {
    const parsed = EtherCAT.Device.slavePropertiesZ.parse({
      read: { channels: { "24576_1": 5, autoGone: 6, manual_24576_1: 7 } },
      write: { channels: { autoOutputState: 8 } },
    });
    expect(parsed.read.channels).toEqual([
      { index: 0x6000, subIndex: 1, scaled: false, calibrated: false, channel: 5 },
    ]);
    expect(parsed.write.channels).toEqual([]);
  });
});
