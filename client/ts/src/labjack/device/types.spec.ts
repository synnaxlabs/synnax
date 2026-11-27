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
    expect(device.ports).toEqual({
      model: "t7",
      mux80Attached: false,
      analogInput: {},
      analogOutput: {},
      digitalInput: {},
      digitalOutput: {},
    });
    expect(device.identifier).toEqual("");
    expect(device.readIndex).toBeUndefined();
  });

  it("should parse the port maps of a device", () => {
    const device = labjack.device.deviceZ.parse({
      ...DEVICE,
      ports: {
        model: "t7",
        analogInput: { AIN0: 1, AIN48: 2 },
        digitalOutput: { DIO22: { command: 3, state: 4 } },
      },
    });
    expect(device.ports).toMatchObject({
      analogInput: { AIN0: 1, AIN48: 2 },
      digitalOutput: { DIO22: { command: 3, state: 4 } },
    });
  });

  const INVALID: [string, Record<string, unknown>][] = [
    ["an empty name", { name: "" }],
    ["a zero serial", { serial: 0 }],
    ["an unknown model", { model: "t9" }],
    ["unknown ports", { ports: { model: "t9" } }],
    ["a port of another model", { ports: { model: "t4", digitalInput: { DIO0: 5 } } }],
    ["an unknown connection", { connection: "LJM_ctTCP" }],
  ];
  it.each(INVALID)("should reject %s", (_, change) => {
    expect(labjack.device.deviceZ.safeParse({ ...DEVICE, ...change }).success).toBe(
      false,
    );
  });
});

describe("actionZ", () => {
  it("should parse a set on the ports of a device", () => {
    const action = labjack.device.actionZ.parse({
      type: "set",
      set: { path: ["ports", "mux80_attached"], value: true },
    });
    expect(action).toMatchObject({
      type: "set",
      set: { path: ["ports", "mux80_attached"], value: true },
    });
  });

  it("should reject an unknown action type", () => {
    expect(labjack.device.actionZ.safeParse({ type: "update" }).success).toBe(false);
  });
});

describe("retrieveRequestZ", () => {
  it("should parse a request with a where and a cursor", () => {
    const req = labjack.device.retrieveRequestZ.parse({
      where: { rack: { eq: 1 } },
      after: { key: DEVICE.key },
      limit: 10,
    });
    expect(req.after?.key).toEqual(DEVICE.key);
  });
});

describe("retrieveResponseZ", () => {
  it("should parse the devices of a response", () => {
    const res = labjack.device.retrieveResponseZ.parse({ devices: [DEVICE] });
    expect(res.devices).toHaveLength(1);
  });
});
