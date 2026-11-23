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

describe("hardwareZ", () => {
  it.each([-1, Infinity, NaN])("should reject an analog input rate of %s", (rate) => {
    expect(() => ni.hardwareZ.parse({ aiMinRate: rate })).toThrow();
  });

  it("should accept an analog input rate of 0", () => {
    expect(ni.hardwareZ.parse({ aiMinRate: 0 }).aiMinRate).toBe(0);
  });
});

const CHANNEL = { type: "ao_func_gen", key: "gen", port: 0 } as const;

describe("aoFuncGenChannelZ", () => {
  it("should accept a positive frequency", () => {
    expect(ni.aoFuncGenChannelZ.parse({ ...CHANNEL, frequency: 12.5 }).frequency).toBe(
      12.5,
    );
  });

  it.each([0, -5])("should reject a frequency of %d", (frequency) => {
    expect(ni.aoFuncGenChannelZ.safeParse({ ...CHANNEL, frequency }).success).toBe(
      false,
    );
  });

  it("should reject a missing frequency", () => {
    expect(ni.aoFuncGenChannelZ.safeParse(CHANNEL).success).toBe(false);
  });
});

describe("minMaxValZ", () => {
  it.each(["minVal", "maxVal"])("should reject the replaced %s key", (key) => {
    const result = ni.aiVoltageChannelZ.safeParse({
      type: "ai_voltage",
      device: "dev",
      port: 0,
      [key]: 5,
    });
    expect(result.success).toBe(false);
    expect(result.error?.issues).toEqual([
      expect.objectContaining({
        path: [key],
        message: `${key} was replaced by range: set range: { lower, upper }`,
      }),
    ]);
  });

  it("should reject the replaced keys in a counter channel", () => {
    expect(
      ni.ciFrequencyChannelZ.safeParse({ type: "ci_frequency", port: 0, minVal: 1 })
        .success,
    ).toBe(false);
  });

  it("should accept a range", () => {
    const ch = ni.aiVoltageChannelZ.parse({
      type: "ai_voltage",
      device: "dev",
      port: 0,
      range: { lower: -10, upper: 10 },
    });
    expect(ch.range).toEqual({ lower: -10, upper: 10 });
  });
});
