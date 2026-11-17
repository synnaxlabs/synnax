// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { labjack } from "@/labjack";

const DEVICE = {
  key: "22222222-2222-4222-8222-222222222222",
  rack: 1,
  name: "T7-0001",
  serial: 470010001,
  connection: "LJM_ctUSB",
  model: "t7",
  ports: { model: "t7" },
} as const;

describe("deviceZ", () => {
  it("should parse a device and apply the defaults", () => {
    const device = labjack.device.deviceZ.parse(DEVICE);
    expect(device.ports).toEqual({ model: "t7", mux80Attached: false });
    expect(device.identifier).toEqual("");
    expect(device.readIndex).toEqual(0);
    expect(device.analogInput).toEqual({});
    expect(device.digitalOutput).toEqual({});
  });

  it("should parse the port maps of a device", () => {
    const device = labjack.device.deviceZ.parse({
      ...DEVICE,
      analogInput: { AIN0: 1, AIN48: 2 },
      digitalOutput: { DIO22: { command: 3, state: 4 } },
    });
    expect(device.analogInput).toEqual({ AIN0: 1, AIN48: 2 });
    expect(device.digitalOutput).toEqual({ DIO22: { command: 3, state: 4 } });
  });

  const INVALID: [string, Record<string, unknown>][] = [
    ["an empty name", { name: "" }],
    ["a zero serial", { serial: 0 }],
    ["an unknown model", { model: "t9" }],
    ["unknown ports", { ports: { model: "t9" } }],
    ["an unknown connection", { connection: "LJM_ctTCP" }],
    ["a port of another model", { model: "t4", digitalInput: { DIO0: 5 } }],
  ];
  it.each(INVALID)("should reject %s", (_, change) => {
    expect(labjack.device.deviceZ.safeParse({ ...DEVICE, ...change }).success).toBe(
      false,
    );
  });
});

describe("updateZ", () => {
  it("should parse a configuration update with the ports of its model", () => {
    const update = labjack.device.updateZ.parse({
      group: "configuration",
      name: "T7-0001",
      ports: { model: "t7" },
    });
    expect(update).toMatchObject({
      group: "configuration",
      ports: { model: "t7", mux80Attached: false },
    });
  });

  it("should parse an identity update", () => {
    const update = labjack.device.updateZ.parse({
      group: "identity",
      rack: 1,
      serial: 470010001,
      connection: "LJM_ctWIFI",
      model: "t7",
    });
    expect(update).toMatchObject({ group: "identity", connection: "LJM_ctWIFI" });
  });

  it("should reject an unknown group", () => {
    expect(labjack.device.updateZ.safeParse({ group: "status" }).success).toBe(false);
  });
});
