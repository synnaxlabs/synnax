// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { NI } from "@/feature/ni";

interface ScaleExpectation {
  scale: NI.Task.Scale;
  result: boolean;
}

describe("scales", () => {
  const expectations: ScaleExpectation[] = [
    { scale: NI.Task.ZERO_SCALES.none, result: true },
    { scale: NI.Task.ZERO_SCALES.linear, result: true },
    { scale: NI.Task.ZERO_SCALES.map, result: true },
    { scale: NI.Task.ZERO_SCALES.table, result: true },
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
      expect(NI.Task.SCALE_SCHEMAS[type].safeParse(scale).success).toEqual(result);
    });
  });
});

describe("analog read task", () => {
  it("should be able to parse a valid task", () => {
    expect(
      NI.Task.analogReadConfigZ.safeParse({
        ...NI.Task.ZERO_ANALOG_READ_PAYLOAD.config,
        streamRate: 1000,
        sampleRate: 2000,
        channels: [{ ...NI.Task.ZERO_AI_CHANNEL, key: "0", device: "34" }],
      }).success,
    ).toEqual(true);
  });

  describe("should be able to parse a task on multiple devices", () => {
    it("should properly parse a task with the same ports on different devices", () => {
      expect(
        NI.Task.analogReadConfigZ.safeParse({
          ...NI.Task.ZERO_ANALOG_READ_PAYLOAD.config,
          streamRate: 1000,
          sampleRate: 2000,
          channels: [
            { ...NI.Task.ZERO_AI_CHANNEL, key: "0", device: "34", port: 0 },
            { ...NI.Task.ZERO_AI_CHANNEL, key: "1", device: "35", port: 0 },
          ],
        }).success,
      ).toEqual(true);
    });

    it("should properly parse a task with the same ports on the same device", () => {
      expect(
        NI.Task.analogReadConfigZ.safeParse({
          ...NI.Task.ZERO_ANALOG_READ_PAYLOAD.config,
          streamRate: 1000,
          sampleRate: 2000,
          channels: [
            { ...NI.Task.ZERO_AI_CHANNEL, key: "0", device: "34", port: 0 },
            { ...NI.Task.ZERO_AI_CHANNEL, key: "1", device: "34", port: 0 },
          ],
        }).success,
      ).toEqual(false);
    });
  });

  describe("sample rate limits", () => {
    it("should accept sample rate at 1 MHz (max limit)", () => {
      expect(
        NI.Task.analogReadConfigZ.safeParse({
          ...NI.Task.ZERO_ANALOG_READ_PAYLOAD.config,
          streamRate: 20000,
          sampleRate: 1000000,
          channels: [{ ...NI.Task.ZERO_AI_CHANNEL, key: "0", device: "34" }],
        }).success,
      ).toEqual(true);
    });

    it("should reject sample rate exceeding 1 MHz", () => {
      expect(
        NI.Task.analogReadConfigZ.safeParse({
          ...NI.Task.ZERO_ANALOG_READ_PAYLOAD.config,
          streamRate: 20000,
          sampleRate: 1000001,
          channels: [{ ...NI.Task.ZERO_AI_CHANNEL, key: "0", device: "34" }],
        }).success,
      ).toEqual(false);
    });

    it("should reject negative sample rate", () => {
      expect(
        NI.Task.analogReadConfigZ.safeParse({
          ...NI.Task.ZERO_ANALOG_READ_PAYLOAD.config,
          streamRate: 1000,
          sampleRate: -1,
          channels: [{ ...NI.Task.ZERO_AI_CHANNEL, key: "0", device: "34" }],
        }).success,
      ).toEqual(false);
    });
  });

  describe("analog write task", () => {
    it("should be able to parse a valid task", () => {
      expect(
        NI.Task.analogWriteConfigZ.safeParse({
          ...NI.Task.ZERO_ANALOG_WRITE_PAYLOAD.config,
          device: "Dev1",
          channels: [{ ...NI.Task.ZERO_AO_CHANNEL, key: "0" }],
        }).success,
      ).toEqual(true);
    });
  });

  describe("digital read task", () => {
    it("should be able to parse a valid task", () => {
      expect(
        NI.Task.digitalReadConfigZ.safeParse({
          ...NI.Task.ZERO_DIGITAL_READ_PAYLOAD.config,
          device: "Dev1",
          channels: [{ ...NI.Task.ZERO_DI_CHANNEL, key: "0" }],
        }).success,
      ).toEqual(true);
    });
  });

  describe("digital write task", () => {
    it("should be able to parse a valid task", () => {
      expect(
        NI.Task.digitalWriteConfigZ.safeParse({
          ...NI.Task.ZERO_DIGITAL_WRITE_PAYLOAD.config,
          device: "Dev1",
          channels: [{ ...NI.Task.ZERO_DO_CHANNEL, key: "0" }],
        }).success,
      ).toEqual(true);
    });
  });

  describe("counter read task", () => {
    it("should be able to parse a valid task", () => {
      expect(
        NI.Task.counterReadConfigZ.safeParse({
          ...NI.Task.ZERO_COUNTER_READ_PAYLOAD.config,
          streamRate: 25,
          sampleRate: 1000,
          channels: [{ ...NI.Task.ZERO_CI_CHANNEL, key: "0", device: "Dev1" }],
        }).success,
      ).toEqual(true);
    });

    it("should fail to parse a task with duplicate ports on the same device", () => {
      expect(
        NI.Task.counterReadConfigZ.safeParse({
          ...NI.Task.ZERO_COUNTER_READ_PAYLOAD.config,
          streamRate: 25,
          sampleRate: 1000,
          channels: [
            { ...NI.Task.ZERO_CI_CHANNEL, key: "0", device: "Dev1", port: 0 },
            { ...NI.Task.ZERO_CI_CHANNEL, key: "1", device: "Dev1", port: 0 },
          ],
        }).success,
      ).toEqual(false);
    });

    it("should properly parse a task with the same ports on different devices", () => {
      expect(
        NI.Task.counterReadConfigZ.safeParse({
          ...NI.Task.ZERO_COUNTER_READ_PAYLOAD.config,
          streamRate: 25,
          sampleRate: 1000,
          channels: [
            { ...NI.Task.ZERO_CI_CHANNEL, key: "0", device: "Dev1", port: 0 },
            { ...NI.Task.ZERO_CI_CHANNEL, key: "1", device: "Dev2", port: 0 },
          ],
        }).success,
      ).toEqual(true);
    });

    it("should fail to parse a task with sample rate less than stream rate", () => {
      expect(
        NI.Task.counterReadConfigZ.safeParse({
          ...NI.Task.ZERO_COUNTER_READ_PAYLOAD.config,
          streamRate: 1000,
          sampleRate: 500,
          channels: [{ ...NI.Task.ZERO_CI_CHANNEL, key: "0", device: "Dev1" }],
        }).success,
      ).toEqual(false);
    });

    it("should be able to parse a task with ci_edge_count channels", () => {
      expect(
        NI.Task.counterReadConfigZ.safeParse({
          ...NI.Task.ZERO_COUNTER_READ_PAYLOAD.config,
          streamRate: 25,
          sampleRate: 1000,
          channels: [
            { ...NI.Task.ZERO_CI_CHANNELS.ci_edge_count, key: "0", device: "Dev1" },
          ],
        }).success,
      ).toEqual(true);
    });

    it("should be able to parse a task with mixed ci_frequency and ci_edge_count channels", () => {
      expect(
        NI.Task.counterReadConfigZ.safeParse({
          ...NI.Task.ZERO_COUNTER_READ_PAYLOAD.config,
          streamRate: 25,
          sampleRate: 1000,
          channels: [
            {
              ...NI.Task.ZERO_CI_CHANNELS.ci_frequency,
              key: "0",
              device: "Dev1",
              port: 0,
            },
            {
              ...NI.Task.ZERO_CI_CHANNELS.ci_edge_count,
              key: "1",
              device: "Dev1",
              port: 1,
            },
          ],
        }).success,
      ).toEqual(true);
    });

    it("should be able to parse a task with ci_period channels", () => {
      expect(
        NI.Task.counterReadConfigZ.safeParse({
          ...NI.Task.ZERO_COUNTER_READ_PAYLOAD.config,
          streamRate: 25,
          sampleRate: 1000,
          channels: [
            { ...NI.Task.ZERO_CI_CHANNELS.ci_period, key: "0", device: "Dev1" },
          ],
        }).success,
      ).toEqual(true);
    });

    it("should be able to parse a task with mixed ci_frequency, ci_edge_count, and ci_period channels", () => {
      expect(
        NI.Task.counterReadConfigZ.safeParse({
          ...NI.Task.ZERO_COUNTER_READ_PAYLOAD.config,
          streamRate: 25,
          sampleRate: 1000,
          channels: [
            {
              ...NI.Task.ZERO_CI_CHANNELS.ci_frequency,
              key: "0",
              device: "Dev1",
              port: 0,
            },
            {
              ...NI.Task.ZERO_CI_CHANNELS.ci_edge_count,
              key: "1",
              device: "Dev1",
              port: 1,
            },
            {
              ...NI.Task.ZERO_CI_CHANNELS.ci_period,
              key: "2",
              device: "Dev1",
              port: 2,
            },
          ],
        }).success,
      ).toEqual(true);
    });

    it("should be able to parse a task with ci_pulse_width channels", () => {
      expect(
        NI.Task.counterReadConfigZ.safeParse({
          ...NI.Task.ZERO_COUNTER_READ_PAYLOAD.config,
          streamRate: 25,
          sampleRate: 1000,
          channels: [
            { ...NI.Task.ZERO_CI_CHANNELS.ci_pulse_width, key: "0", device: "Dev1" },
          ],
        }).success,
      ).toEqual(true);
    });

    it("should be able to parse a task with ci_semi_period channels", () => {
      expect(
        NI.Task.counterReadConfigZ.safeParse({
          ...NI.Task.ZERO_COUNTER_READ_PAYLOAD.config,
          streamRate: 25,
          sampleRate: 1000,
          channels: [
            { ...NI.Task.ZERO_CI_CHANNELS.ci_semi_period, key: "0", device: "Dev1" },
          ],
        }).success,
      ).toEqual(true);
    });

    it("should be able to parse a task with ci_two_edge_sep channels", () => {
      expect(
        NI.Task.counterReadConfigZ.safeParse({
          ...NI.Task.ZERO_COUNTER_READ_PAYLOAD.config,
          streamRate: 25,
          sampleRate: 1000,
          channels: [
            { ...NI.Task.ZERO_CI_CHANNELS.ci_two_edge_sep, key: "0", device: "Dev1" },
          ],
        }).success,
      ).toEqual(true);
    });

    it("should be able to parse a task with all CI channel types", () => {
      expect(
        NI.Task.counterReadConfigZ.safeParse({
          ...NI.Task.ZERO_COUNTER_READ_PAYLOAD.config,
          streamRate: 25,
          sampleRate: 1000,
          channels: [
            {
              ...NI.Task.ZERO_CI_CHANNELS.ci_frequency,
              key: "0",
              device: "Dev1",
              port: 0,
            },
            {
              ...NI.Task.ZERO_CI_CHANNELS.ci_edge_count,
              key: "1",
              device: "Dev1",
              port: 1,
            },
            {
              ...NI.Task.ZERO_CI_CHANNELS.ci_period,
              key: "2",
              device: "Dev1",
              port: 2,
            },
            {
              ...NI.Task.ZERO_CI_CHANNELS.ci_pulse_width,
              key: "3",
              device: "Dev1",
              port: 3,
            },
            {
              ...NI.Task.ZERO_CI_CHANNELS.ci_semi_period,
              key: "4",
              device: "Dev1",
              port: 4,
            },
            {
              ...NI.Task.ZERO_CI_CHANNELS.ci_two_edge_sep,
              key: "5",
              device: "Dev1",
              port: 5,
            },
          ],
        }).success,
      ).toEqual(true);
    });

    // NOTE: This test is commented out because the validation for "at least one enabled channel"
    // is handled in the C++ driver code, not in the TypeScript schema. Disabled channels are
    // filtered out during task configuration. This matches the behavior of Analog Read and Digital Read tasks.
    // it("should fail to parse a task with no enabled channels", () => {
    //   expect(
    //     NI.Task.counterReadConfigZ.safeParse({
    //       ...NI.Task.ZERO_COUNTER_READ_PAYLOAD.config,
    //       streamRate: 25,
    //       sampleRate: 1000,
    //       channels: [{ ...NI.Task.ZERO_CI_CHANNEL, key: "0", device: "Dev1", enabled: false }],
    //     }).success,
    //   ).toEqual(false);
    // });
  });
});
