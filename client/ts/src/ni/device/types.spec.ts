// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { ni } from "@/ni";

describe("Update", () => {
  it("should parse an identity update by its group", () => {
    const update = ni.device.updateZ.parse({
      group: "identity",
      rack: 1,
      serial: "01A",
      resourceName: "Mod1",
      model: "PXI-6255",
    });
    expect(update.group).toEqual("identity");
    if (update.group !== "identity") return;
    expect(update.serial).toEqual("01A");
  });

  it("should parse the integer keys of the port maps", () => {
    const update = ni.device.updateZ.parse({
      group: "configuration",
      name: "Module",
      analogInput: { index: 1, channels: { "3": 2 } },
      digitalOutput: {
        "0": { stateIndex: 4, lines: { "7": { command: 5, state: 6 } } },
      },
    });
    expect(update.group).toEqual("configuration");
    if (update.group !== "configuration") return;
    expect(update.analogInput.channels[3]).toEqual(2);
    expect(update.digitalOutput[0].lines[7].state).toEqual(6);
  });

  it("should parse a configuration update by its group", () => {
    const update = ni.device.updateZ.parse({ group: "configuration", name: "Module" });
    expect(update.group).toEqual("configuration");
    if (update.group !== "configuration") return;
    expect(update.name).toEqual("Module");
  });

  it("should reject an unknown group", () => {
    expect(
      ni.device.updateZ.safeParse({ group: "other", name: "Module" }).success,
    ).toBe(false);
  });
});

describe("Identity", () => {
  const identity = { rack: 1, serial: "01A", resourceName: "Mod1", model: "PXI-6255" };

  it.each(["resourceName", "model"])("should reject an empty %s", (field) => {
    expect(ni.device.identityZ.safeParse({ ...identity, [field]: "" }).success).toBe(
      false,
    );
  });

  it("should default the serial of simulated hardware to empty", () => {
    const { serial: _, ...simulated } = identity;
    expect(ni.device.identityZ.parse(simulated).serial).toEqual("");
  });
});

describe("Device", () => {
  const device = {
    key: "11111111-1111-4111-8111-111111111111",
    rack: 1,
    name: "PXI-6255",
    resourceName: "Mod1",
    model: "PXI-6255",
  };

  it("should parse a device without a serial", () => {
    expect(ni.device.deviceZ.parse(device).serial).toEqual("");
  });

  it.each(["resourceName", "model"])("should reject an empty %s", (field) => {
    expect(ni.device.deviceZ.safeParse({ ...device, [field]: "" }).success).toBe(false);
  });
});
