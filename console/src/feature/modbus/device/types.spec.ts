// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Modbus } from "@/feature/modbus";
import { testPropertiesSchema } from "@/platform/device/testutil";

testPropertiesSchema(
  "Modbus",
  Modbus.Device.SCHEMAS.properties,
  Modbus.Device.ZERO_PROPERTIES,
  [
    [
      "properties with only connection config",
      { connection: { host: "192.168.1.10", port: 502 } },
    ],
    [
      "properties with a serial connection",
      { connection: { transport: "rtu", serialPort: "COM3" } },
    ],
  ],
);

describe("Modbus.Device.SCHEMAS", () => {
  it("should fill the unit, timing, and request fields of a stored device", () => {
    const props = Modbus.Device.SCHEMAS.properties.parse({
      connection: { host: "10.0.0.7", port: 502, swapBytes: true, swapWords: false },
    });
    expect(props.connection).toEqual({
      transport: "tcp",
      host: "10.0.0.7",
      port: 502,
      swapBytes: true,
      swapWords: false,
      unitId: 255,
      timeoutMs: 1000,
      retries: 1,
    });
    expect(props.requests).toEqual({
      maxRegisters: 125,
      maxGap: 125,
      writeCodes: "multiple",
      maskWriteSupported: false,
      delayMs: 0,
    });
  });

  it("should fill the line settings and unit of a serial connection", () => {
    const props = Modbus.Device.SCHEMAS.properties.parse({
      connection: { transport: "rtu", serialPort: "/dev/ttyUSB0" },
    });
    expect(props.connection).toEqual({
      transport: "rtu",
      serialPort: "/dev/ttyUSB0",
      baudRate: 19200,
      parity: "even",
      stopBits: 1,
      swapBytes: false,
      swapWords: false,
      unitId: 1,
      timeoutMs: 1000,
      retries: 1,
    });
  });

  it("should reject a transport the Driver does not know", () => {
    expect(
      Modbus.Device.SCHEMAS.properties.safeParse({
        connection: { transport: "ascii", serialPort: "COM3" },
      }).success,
    ).toBe(false);
  });

  it.each([
    [{ connection: { unitId: 12.5 } }, "Unit ID must be a whole number"],
    [{ connection: { timeoutMs: 1000.5 } }, "Timeout must be a whole number"],
    [{ connection: { retries: 1.5 } }, "Retries must be a whole number"],
    [{ requests: { maxRegisters: 60.5 } }, "Max registers must be a whole number"],
    [{ requests: { maxGap: 0.5 } }, "Max gap must be a whole number"],
    [{ requests: { delayMs: 2.5 } }, "Request delay must be a whole number"],
    [
      { connection: { transport: "rtu", baudRate: 9600.5 } },
      "Baud rate must be a whole number",
    ],
    [
      { connection: { transport: "rtu", stopBits: 1.5 } },
      "Stop bits must be a whole number",
    ],
  ])("should reject a fraction in %j", (props, message) => {
    const result = Modbus.Device.SCHEMAS.properties.safeParse(props);
    expect(result.error?.issues.map((i) => i.message)).toEqual([message]);
  });

  it("should reject a write code the Driver does not know", () => {
    expect(
      Modbus.Device.SCHEMAS.properties.safeParse({ requests: { writeCodes: "both" } })
        .success,
    ).toBe(false);
  });
});

describe("Modbus.Device byte order", () => {
  it.each([
    ["ABCD", { swapBytes: false, swapWords: true }],
    ["BADC", { swapBytes: true, swapWords: true }],
    ["CDAB", { swapBytes: false, swapWords: false }],
    ["DCBA", { swapBytes: true, swapWords: false }],
  ] as const)("should map %s to the Driver's swap settings", (order, swaps) => {
    expect(Modbus.Device.swapsOf(order)).toEqual(swaps);
    expect(Modbus.Device.byteOrderOf(swaps)).toBe(order);
  });

  it("should give the low word first for a device with no swaps", () => {
    expect(Modbus.Device.byteOrderOf(Modbus.Device.ZERO_PROPERTIES.connection)).toBe(
      "CDAB",
    );
  });
});
