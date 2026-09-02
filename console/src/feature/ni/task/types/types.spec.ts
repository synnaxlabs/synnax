// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";
import { z } from "zod";

import { NI } from "@/feature/ni";

interface ScaleExpectation {
  scale: NI.Task.Scale;
  result: boolean;
}

describe("scales", () => {
  const expectations: ScaleExpectation[] = [
    { scale: NI.Task.createScale("none"), result: true },
    { scale: NI.Task.createScale("linear"), result: true },
    { scale: NI.Task.createScale("map"), result: true },
    { scale: NI.Task.createScale("table"), result: true },
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
      expect(z.validate(NI.Task.SCALE_SCHEMAS[type], scale)).toEqual(result);
    });
  });
});

describe("analog read task", () => {
  it("should be able to parse a valid task", () => {
    expect(
      z.validate(NI.Task.analogReadConfigZ, {
        ...NI.Task.ANALOG_READ_SCHEMAS.config.parse({}),
        streamRate: 1000,
        sampleRate: 2000,
        channels: [{ ...NI.Task.createAIChannel(), key: "0", device: "34" }],
      }),
    ).toEqual(true);
  });

  describe("sample rate limits", () => {
    it("should accept sample rate at 1 MHz (max limit)", () => {
      expect(
        z.validate(NI.Task.analogReadConfigZ, {
          ...NI.Task.ANALOG_READ_SCHEMAS.config.parse({}),
          streamRate: 20000,
          sampleRate: 1000000,
          channels: [{ ...NI.Task.createAIChannel(), key: "0", device: "34" }],
        }),
      ).toEqual(true);
    });

    it("should reject sample rate exceeding 1 MHz", () => {
      expect(
        z.validate(NI.Task.deployAnalogReadConfigZ, {
          ...NI.Task.ANALOG_READ_SCHEMAS.config.parse({}),
          streamRate: 20000,
          sampleRate: 1000001,
          channels: [{ ...NI.Task.createAIChannel(), key: "0", device: "34" }],
        }),
      ).toEqual(false);
    });

    it("should reject negative sample rate", () => {
      expect(
        z.validate(NI.Task.deployAnalogReadConfigZ, {
          ...NI.Task.ANALOG_READ_SCHEMAS.config.parse({}),
          streamRate: 1000,
          sampleRate: -1,
          channels: [{ ...NI.Task.createAIChannel(), key: "0", device: "34" }],
        }),
      ).toEqual(false);
    });
  });

  describe("analog write task", () => {
    it("should be able to parse a valid task", () => {
      expect(
        z.validate(NI.Task.analogWriteConfigZ, {
          ...NI.Task.ANALOG_WRITE_SCHEMAS.config.parse({}),
          device: "Dev1",
          channels: [{ ...NI.Task.createAOChannel(), key: "0" }],
        }),
      ).toEqual(true);
    });
  });

  describe("digital read task", () => {
    it("should be able to parse a valid task", () => {
      expect(
        z.validate(NI.Task.digitalReadConfigZ, {
          ...NI.Task.DIGITAL_READ_SCHEMAS.config.parse({}),
          device: "Dev1",
          channels: [{ ...NI.Task.createDIChannel(), key: "0" }],
        }),
      ).toEqual(true);
    });
  });

  describe("digital write task", () => {
    it("should be able to parse a valid task", () => {
      expect(
        z.validate(NI.Task.digitalWriteConfigZ, {
          ...NI.Task.DIGITAL_WRITE_SCHEMAS.config.parse({}),
          device: "Dev1",
          channels: [{ ...NI.Task.createDOChannel(), key: "0" }],
        }),
      ).toEqual(true);
    });
  });

  describe("counter read task", () => {
    it("should be able to parse a valid task", () => {
      expect(
        z.validate(NI.Task.counterReadConfigZ, {
          ...NI.Task.COUNTER_READ_SCHEMAS.config.parse({}),
          streamRate: 25,
          sampleRate: 1000,
          channels: [{ ...NI.Task.createCIChannel(), key: "0", device: "Dev1" }],
        }),
      ).toEqual(true);
    });

    it("should fail to parse a task with duplicate ports on the same device", () => {
      expect(
        z.validate(NI.Task.deployCounterReadConfigZ, {
          ...NI.Task.COUNTER_READ_SCHEMAS.config.parse({}),
          streamRate: 25,
          sampleRate: 1000,
          channels: [
            { ...NI.Task.createCIChannel(), key: "0", device: "Dev1", port: 0 },
            { ...NI.Task.createCIChannel(), key: "1", device: "Dev1", port: 0 },
          ],
        }),
      ).toEqual(false);
    });

    it("should properly parse a task with the same ports on different devices", () => {
      expect(
        z.validate(NI.Task.counterReadConfigZ, {
          ...NI.Task.COUNTER_READ_SCHEMAS.config.parse({}),
          streamRate: 25,
          sampleRate: 1000,
          channels: [
            { ...NI.Task.createCIChannel(), key: "0", device: "Dev1", port: 0 },
            { ...NI.Task.createCIChannel(), key: "1", device: "Dev2", port: 0 },
          ],
        }),
      ).toEqual(true);
    });

    it("should fail to parse a task with sample rate less than stream rate", () => {
      expect(
        z.validate(NI.Task.deployCounterReadConfigZ, {
          ...NI.Task.COUNTER_READ_SCHEMAS.config.parse({}),
          streamRate: 1000,
          sampleRate: 500,
          channels: [{ ...NI.Task.createCIChannel(), key: "0", device: "Dev1" }],
        }),
      ).toEqual(false);
    });

    it("should be able to parse a task with ci_edge_count channels", () => {
      expect(
        z.validate(NI.Task.counterReadConfigZ, {
          ...NI.Task.COUNTER_READ_SCHEMAS.config.parse({}),
          streamRate: 25,
          sampleRate: 1000,
          channels: [
            { ...NI.Task.createCIChannel("ci_edge_count"), key: "0", device: "Dev1" },
          ],
        }),
      ).toEqual(true);
    });

    it("should be able to parse a task with mixed ci_frequency and ci_edge_count channels", () => {
      expect(
        z.validate(NI.Task.counterReadConfigZ, {
          ...NI.Task.COUNTER_READ_SCHEMAS.config.parse({}),
          streamRate: 25,
          sampleRate: 1000,
          channels: [
            {
              ...NI.Task.createCIChannel("ci_frequency"),
              key: "0",
              device: "Dev1",
              port: 0,
            },
            {
              ...NI.Task.createCIChannel("ci_edge_count"),
              key: "1",
              device: "Dev1",
              port: 1,
            },
          ],
        }),
      ).toEqual(true);
    });

    it("should be able to parse a task with ci_period channels", () => {
      expect(
        z.validate(NI.Task.counterReadConfigZ, {
          ...NI.Task.COUNTER_READ_SCHEMAS.config.parse({}),
          streamRate: 25,
          sampleRate: 1000,
          channels: [
            { ...NI.Task.createCIChannel("ci_period"), key: "0", device: "Dev1" },
          ],
        }),
      ).toEqual(true);
    });

    it("should be able to parse a task with mixed ci_frequency, ci_edge_count, and ci_period channels", () => {
      expect(
        z.validate(NI.Task.counterReadConfigZ, {
          ...NI.Task.COUNTER_READ_SCHEMAS.config.parse({}),
          streamRate: 25,
          sampleRate: 1000,
          channels: [
            {
              ...NI.Task.createCIChannel("ci_frequency"),
              key: "0",
              device: "Dev1",
              port: 0,
            },
            {
              ...NI.Task.createCIChannel("ci_edge_count"),
              key: "1",
              device: "Dev1",
              port: 1,
            },
            {
              ...NI.Task.createCIChannel("ci_period"),
              key: "2",
              device: "Dev1",
              port: 2,
            },
          ],
        }),
      ).toEqual(true);
    });

    it("should be able to parse a task with ci_pulse_width channels", () => {
      expect(
        z.validate(NI.Task.counterReadConfigZ, {
          ...NI.Task.COUNTER_READ_SCHEMAS.config.parse({}),
          streamRate: 25,
          sampleRate: 1000,
          channels: [
            { ...NI.Task.createCIChannel("ci_pulse_width"), key: "0", device: "Dev1" },
          ],
        }),
      ).toEqual(true);
    });

    it("should be able to parse a task with ci_semi_period channels", () => {
      expect(
        z.validate(NI.Task.counterReadConfigZ, {
          ...NI.Task.COUNTER_READ_SCHEMAS.config.parse({}),
          streamRate: 25,
          sampleRate: 1000,
          channels: [
            { ...NI.Task.createCIChannel("ci_semi_period"), key: "0", device: "Dev1" },
          ],
        }),
      ).toEqual(true);
    });

    it("should be able to parse a task with ci_two_edge_sep channels", () => {
      expect(
        z.validate(NI.Task.counterReadConfigZ, {
          ...NI.Task.COUNTER_READ_SCHEMAS.config.parse({}),
          streamRate: 25,
          sampleRate: 1000,
          channels: [
            { ...NI.Task.createCIChannel("ci_two_edge_sep"), key: "0", device: "Dev1" },
          ],
        }),
      ).toEqual(true);
    });

    it("should be able to parse a task with all CI channel types", () => {
      expect(
        z.validate(NI.Task.counterReadConfigZ, {
          ...NI.Task.COUNTER_READ_SCHEMAS.config.parse({}),
          streamRate: 25,
          sampleRate: 1000,
          channels: [
            {
              ...NI.Task.createCIChannel("ci_frequency"),
              key: "0",
              device: "Dev1",
              port: 0,
            },
            {
              ...NI.Task.createCIChannel("ci_edge_count"),
              key: "1",
              device: "Dev1",
              port: 1,
            },
            {
              ...NI.Task.createCIChannel("ci_period"),
              key: "2",
              device: "Dev1",
              port: 2,
            },
            {
              ...NI.Task.createCIChannel("ci_pulse_width"),
              key: "3",
              device: "Dev1",
              port: 3,
            },
            {
              ...NI.Task.createCIChannel("ci_semi_period"),
              key: "4",
              device: "Dev1",
              port: 4,
            },
            {
              ...NI.Task.createCIChannel("ci_two_edge_sep"),
              key: "5",
              device: "Dev1",
              port: 5,
            },
          ],
        }),
      ).toEqual(true);
    });

    // NOTE: This test is commented out because the validation for "at least one enabled channel"
    // is handled in the C++ driver code, not in the TypeScript schema. Disabled channels are
    // filtered out during task configuration. This matches the behavior of Analog Read and Digital Read tasks.
    // it("should fail to parse a task with no enabled channels", () => {
    //   expect(
    //     z.validate(NI.Task.counterReadConfigZ, {
    //       ...NI.Task.COUNTER_READ_SCHEMAS.config.parse({}),
    //       streamRate: 25,
    //       sampleRate: 1000,
    //       channels: [{ ...NI.Task.createCIChannel(), key: "0", device: "Dev1", enabled: false }],
    //     }),
    //   ).toEqual(false);
    // });
  });
});

describe("draft configs", () => {
  // Drafts persist server-side before configuration, so the shape schema must
  // accept every zero config; retrieve parses with it.
  it("should accept the zero analog read config", () => {
    expect(
      z.validate(
        NI.Task.ANALOG_READ_SCHEMAS.config,
        NI.Task.ANALOG_READ_SCHEMAS.config.parse({}),
      ),
    ).toBe(true);
  });
  it("should accept the zero analog write config", () => {
    expect(
      z.validate(
        NI.Task.ANALOG_WRITE_SCHEMAS.config,
        NI.Task.ANALOG_WRITE_SCHEMAS.config.parse({}),
      ),
    ).toBe(true);
  });
  it("should accept the zero counter read config", () => {
    expect(
      z.validate(
        NI.Task.COUNTER_READ_SCHEMAS.config,
        NI.Task.COUNTER_READ_SCHEMAS.config.parse({}),
      ),
    ).toBe(true);
  });
  it("should accept the zero digital read config", () => {
    expect(
      z.validate(
        NI.Task.DIGITAL_READ_SCHEMAS.config,
        NI.Task.DIGITAL_READ_SCHEMAS.config.parse({}),
      ),
    ).toBe(true);
  });
  it("should accept the zero digital write config", () => {
    expect(
      z.validate(
        NI.Task.DIGITAL_WRITE_SCHEMAS.config,
        NI.Task.DIGITAL_WRITE_SCHEMAS.config.parse({}),
      ),
    ).toBe(true);
  });
});
