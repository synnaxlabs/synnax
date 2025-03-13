// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import {
  analogReadConfigZ,
  analogWriteConfigZ,
  digitalReadConfigZ,
  digitalWriteConfigZ,
  LINEAR_SCALE_TYPE,
  MAP_SCALE_TYPE,
  NO_SCALE_TYPE,
  type Scale,
  SCALE_SCHEMAS,
  TABLE_SCALE_TYPE,
  ZERO_AI_CHANNEL,
  ZERO_ANALOG_READ_PAYLOAD,
  ZERO_ANALOG_WRITE_PAYLOAD,
  ZERO_AO_CHANNEL,
  ZERO_DI_CHANNEL,
  ZERO_DIGITAL_READ_PAYLOAD,
  ZERO_DIGITAL_WRITE_PAYLOAD,
  ZERO_DO_CHANNEL,
  ZERO_SCALES,
} from "@/hardware/ni/task/types";

interface ScaleExpectation {
  scale: Scale;
  result: boolean;
}

describe("scales", () => {
  const expectations: ScaleExpectation[] = [
    { scale: ZERO_SCALES[NO_SCALE_TYPE], result: true },
    { scale: ZERO_SCALES[LINEAR_SCALE_TYPE], result: true },
    { scale: ZERO_SCALES[MAP_SCALE_TYPE], result: true },
    { scale: ZERO_SCALES[TABLE_SCALE_TYPE], result: true },
    {
      scale: {
        type: "linear",
        slope: Infinity,
        yIntercept: 0,
        preScaledUnits: "Volts",
        scaledUnits: "Volts",
      },
      result: false,
    },
    {
      scale: {
        type: "linear",
        slope: Infinity,
        yIntercept: 0,
        preScaledUnits: "Volts",
        scaledUnits: "Volts",
      },
      result: false,
    },
    {
      scale: {
        type: "map",
        preScaledUnits: "Volts",
        scaledUnits: "Volts",
        preScaledMin: 2,
        preScaledMax: 1,
        scaledMin: 0,
        scaledMax: 1,
      },
      result: false,
    },
    {
      scale: {
        type: "map",
        preScaledUnits: "Volts",
        scaledUnits: "Volts",
        preScaledMin: 0,
        preScaledMax: 1,
        scaledMin: 1,
        scaledMax: 0,
      },
      result: false,
    },
    {
      scale: {
        type: "table",
        preScaledUnits: "Volts",
        scaledUnits: "Volts",
        preScaledVals: [0, -1],
        scaledVals: [0, 1],
      },
      result: false,
    },
  ];

  expectations.forEach(({ scale, result }) => {
    const { type } = scale;
    it(`should be able to parse ${type} scale`, () => {
      expect(SCALE_SCHEMAS[type].safeParse(scale).success).toEqual(result);
    });
  });
});

describe("analog read task", () => {
  it("should be able to parse a valid task", () => {
    expect(
      analogReadConfigZ.safeParse({
        ...ZERO_ANALOG_READ_PAYLOAD.config,
        streamRate: 1000,
        sampleRate: 2000,
        channels: [{ ...ZERO_AI_CHANNEL, key: "0", device: "34" }],
      }).success,
    ).toEqual(true);
  });
});

describe("analog write task", () => {
  it("should be able to parse a valid task", () => {
    expect(
      analogWriteConfigZ.safeParse({
        ...ZERO_ANALOG_WRITE_PAYLOAD.config,
        device: "Dev1",
        channels: [{ ...ZERO_AO_CHANNEL, key: "0" }],
      }).success,
    ).toEqual(true);
  });
});

describe("digital read task", () => {
  it("should be able to parse a valid task", () => {
    expect(
      digitalReadConfigZ.safeParse({
        ...ZERO_DIGITAL_READ_PAYLOAD.config,
        device: "Dev1",
        channels: [{ ...ZERO_DI_CHANNEL, key: "0" }],
      }).success,
    ).toEqual(true);
  });
});

describe("digital write task", () => {
  it("should be able to parse a valid task", () => {
    expect(
      digitalWriteConfigZ.safeParse({
        ...ZERO_DIGITAL_WRITE_PAYLOAD.config,
        device: "Dev1",
        channels: [{ ...ZERO_DO_CHANNEL, key: "0" }],
      }).success,
    ).toEqual(true);
  });
});
