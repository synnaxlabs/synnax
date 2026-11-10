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
