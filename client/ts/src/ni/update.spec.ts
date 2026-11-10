// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { patch } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { ni } from "@/ni";
import { task } from "@/task";

describe("AIChannelUpdate", () => {
  it("should hold the fields of the named variant", () => {
    const update: ni.AIChannelUpdate = { key: "c3", type: "ai_voltage", maxVal: 10 };
    expect(ni.aiChannelUpdateZ.parse(update)).toEqual(update);
  });

  it("should not compile a field of another variant", () => {
    const update: ni.AIChannelUpdate = {
      key: "c3",
      type: "ai_voltage",
      // @ts-expect-error - a voltage channel has no thermocouple type.
      thermocoupleType: "K",
    };
    expect(ni.aiChannelUpdateZ.parse(update)).toEqual({
      key: "c3",
      type: "ai_voltage",
    });
  });

  it("should hold only base fields without a type", () => {
    const update: ni.AIChannelUpdate = { key: "c3", name: "Pressure" };
    expect(ni.aiChannelUpdateZ.parse(update)).toEqual(update);
    // @ts-expect-error - a variant field needs a type.
    const bad: ni.AIChannelUpdate = { key: "c3", maxVal: 10 };
    expect(ni.aiChannelUpdateZ.safeParse(bad).success).toBe(false);
  });

  it("should need a type for a base field that a variant drops", () => {
    // @ts-expect-error - a built-in temperature channel has no port.
    const bad: ni.AIChannelUpdate = { key: "c3", port: 4 };
    expect(ni.aiChannelUpdateZ.safeParse(bad).success).toBe(false);
  });

  it("should merge a nested union of the same variant", () => {
    const update: ni.AIChannelUpdate = {
      key: "c4",
      type: "ai_voltage",
      customScale: { type: "map", scaledMax: 100 },
    };
    expect(ni.aiChannelUpdateZ.parse(update)).toEqual(update);
  });
});

describe("AnalogReadConfigUpdate", () => {
  const schemas = { u: ni.analogReadConfigUpdateZ, t: ni.analogReadConfigZ };
  const config = ni.analogReadConfigZ.parse({
    device: "d",
    sampleRate: 10,
    streamRate: 5,
    channels: [{ key: "c3", type: "ai_voltage", name: "Pressure", port: 1 }],
  });
  const apply = (channel: ni.AIChannelUpdate): ni.AIChannel =>
    patch.applyUpdate(schemas, config, {
      value: { channels: { update: [channel] } },
      guards: [],
    }).next.channels[0];

  it("should rename a channel without a type", () => {
    expect(apply({ key: "c3", name: "Flow" })).toEqual({
      ...config.channels[0],
      name: "Flow",
    });
  });

  it("should change the variant and keep the fields every variant holds", () => {
    const next = apply({ key: "c3", type: "ai_current" });
    expect(next).toMatchObject({ key: "c3", name: "Pressure", type: "ai_current" });
    expect(next).toMatchObject({ port: 0, extShuntResistorVal: 1 });
  });
});

describe("CONFIG_SCHEMAS", () => {
  it("should merge a partial config into an analog read task", () => {
    const config = ni.CONFIG_SCHEMAS[ni.ANALOG_READ_TASK_TYPE];
    const payloadZ = task.payloadZ({ config: config.t });
    const prev = payloadZ.parse({
      key: "00000000-0000-4000-8000-000000000004",
      name: "Read",
      type: ni.ANALOG_READ_TASK_TYPE,
      config: { device: "d", sampleRate: 10, streamRate: 5, channels: [] },
    });
    const { next } = patch.applyUpdate({ u: task.updateZ(config), t: payloadZ }, prev, {
      value: { config: { sampleRate: 1000 } },
      guards: [],
    });
    expect(next.config).toEqual({ ...prev.config, sampleRate: 1000 });
  });
});
