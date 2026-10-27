// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import { TestSource } from "@/telem/aether/test/source";
import {
  BandColor,
  type BandColorProps,
  RollingAverage,
  StringifyNumber,
  WithinBounds,
} from "@/telem/aether/transformers";

describe("StringifyNumber", () => {
  it("formats a number value with prefix and suffix", () => {
    const t = new StringifyNumber({ precision: 2, prefix: "$", suffix: " USD" });
    t.setSources({ in: new TestSource(42) });
    expect(t.value()).toBe("$42.00 USD");
  });

  it("preserves full precision for a bigint value above 2^53 in standard notation", () => {
    const t = new StringifyNumber({ precision: 0, notation: "standard" });
    t.setSources({ in: new TestSource(1778020940471336960n) });
    expect(t.value()).toBe("1778020940471336960");
  });

  it("does not crash and returns a finite string for a bigint value", () => {
    const t = new StringifyNumber({ precision: 2, notation: "scientific" });
    t.setSources({ in: new TestSource(1778020940471336960n) });
    expect(t.value()).toBe("1.78ᴇ18");
  });

  it("returns an empty string for NaN", () => {
    const t = new StringifyNumber({ precision: 2 });
    t.setSources({ in: new TestSource(NaN) });
    expect(t.value()).toBe("");
  });
});

describe("RollingAverage", () => {
  it("returns the value unchanged when windowSize is less than 2", () => {
    const t = new RollingAverage({ windowSize: 1 });
    t.setSources({ in: new TestSource(42) });
    expect(t.value()).toBe(42);
  });

  it("coerces a bigint value to a number", () => {
    const t = new RollingAverage({ windowSize: 1 });
    t.setSources({ in: new TestSource(123n) });
    expect(t.value()).toBe(123);
  });

  it("returns the latest value until the window holds a sample", () => {
    const t = new RollingAverage({ windowSize: 3 });
    t.setSources({ in: new TestSource(42) });
    expect(t.value()).toBe(42);
  });

  it("averages the samples the window holds before it fills", () => {
    const t = new RollingAverage({ windowSize: 4 });
    const source = new TestSource(0);
    t.setSources({ in: source });
    t.onChange(() => {});
    source.setValue(2);
    source.setValue(4);
    expect(t.value()).toBe(3);
  });

  it("averages the last windowSize samples once the window fills", () => {
    const t = new RollingAverage({ windowSize: 3 });
    const source = new TestSource(0);
    t.setSources({ in: source });
    t.onChange(() => {});
    [1, 2, 3, 10, 20, 30].forEach((v) => source.setValue(v));
    expect(t.value()).toBe(20);
  });

  it("reports NaN while a NaN sample sits in the window", () => {
    const t = new RollingAverage({ windowSize: 3 });
    const source = new TestSource(0);
    t.setSources({ in: source });
    t.onChange(() => {});
    source.setValue(4);
    source.setValue(NaN);
    expect(Number.isNaN(t.value())).toBe(true);
    source.setValue(6);
    expect(Number.isNaN(t.value())).toBe(true);
  });

  it("recovers once a NaN sample slides out of the window", () => {
    const t = new RollingAverage({ windowSize: 3 });
    const source = new TestSource(0);
    t.setSources({ in: source });
    t.onChange(() => {});
    source.setValue(NaN);
    [2, 4, 6].forEach((v) => source.setValue(v));
    expect(t.value()).toBe(4);
  });

  // Staleness counts arrivals, so a window above 1 must not stretch the countdown.
  it("notifies on every sample, even with a window above 1", () => {
    const t = new RollingAverage({ windowSize: 5 });
    const source = new TestSource(0);
    t.setSources({ in: source });
    const handler = vi.fn();
    t.onChange(handler);
    for (let i = 0; i < 3; i++) source.setValue(i);
    expect(handler).toHaveBeenCalledTimes(3);
  });
});

describe("WithinBounds", () => {
  it("returns true for a value inside the bounds", () => {
    const t = new WithinBounds({ trueBound: { lower: 5, upper: 15 } });
    t.setSources({ in: new TestSource(10) });
    expect(t.value()).toBe(true);
  });

  it("returns false for a value outside the bounds", () => {
    const t = new WithinBounds({ trueBound: { lower: 5, upper: 15 } });
    t.setSources({ in: new TestSource(20) });
    expect(t.value()).toBe(false);
  });

  // Staleness counts arrivals, so a sample that leaves the boolean unchanged must still
  // reach the listener.
  it("notifies on every sample, even when the boolean stays the same", () => {
    const t = new WithinBounds({ trueBound: { lower: 5, upper: 15 } });
    const source = new TestSource(10);
    t.setSources({ in: source });
    const handler = vi.fn();
    t.onChange(handler);
    source.setValue(11);
    source.setValue(12);
    expect(handler).toHaveBeenCalledTimes(2);
  });
});

describe("BandColor", () => {
  const green = color.construct("#00ff00");
  const yellow = color.construct("#ffff00");
  const red = color.construct("#ff0000");
  const blue = color.construct("#0000ff");
  const BANDS: BandColorProps["bands"] = [
    { key: "a", threshold: 100, color: red },
    { key: "b", threshold: 50, color: yellow },
  ];

  const create = (
    props: BandColorProps,
    value: number | string,
  ): [BandColor, TestSource<number | string>] => {
    const t = new BandColor(props);
    const source = new TestSource(value);
    t.setSources({ source });
    return [t, source];
  };

  describe("discrete", () => {
    it("should paint the color of the band owning the value", () => {
      expect(create({ bands: BANDS }, 75)[0].value()).toEqual(yellow);
      expect(create({ bands: BANDS }, 150)[0].value()).toEqual(red);
    });

    it("should give a threshold to the band starting at it", () => {
      expect(create({ bands: BANDS }, 100)[0].value()).toEqual(red);
    });

    it("should paint the background below every threshold", () => {
      expect(create({ bands: BANDS, background: green }, 10)[0].value()).toEqual(green);
    });

    it("should paint no color below every threshold without a background", () => {
      expect(create({ bands: BANDS }, 10)[0].value()).toEqual(color.ZERO);
    });

    it("should paint the background for a value that is not a number", () => {
      const bands = [{ key: "c", threshold: 0, color: red }];
      expect(create({ bands, background: green }, "abc")[0].value()).toEqual(green);
    });

    // The display stringifier renders a missing sample as an empty string.
    it("should paint the background for a missing value", () => {
      const bands = [{ key: "d", threshold: 0, color: red }];
      expect(create({ bands, background: green }, "")[0].value()).toEqual(green);
    });

    it("should read a stringified value", () => {
      expect(create({ bands: BANDS }, "1.2e2")[0].value()).toEqual(red);
    });

    it("should paint the background with no bands", () => {
      expect(create({ bands: [], background: blue }, 10)[0].value()).toEqual(blue);
    });
  });

  describe("smooth", () => {
    it("should interpolate between the bands around the value", () => {
      const [t] = create(
        {
          bands: [
            { key: "e", threshold: 0, color: [0, 0, 0, 1] },
            { key: "f", threshold: 100, color: [200, 100, 0, 1] },
          ],
          smooth: true,
        },
        25,
      );
      expect(t.value()).toEqual([50, 25, 0, 1]);
    });

    it("should hold the highest band's color above its threshold", () => {
      expect(create({ bands: BANDS, smooth: true }, 500)[0].value()).toEqual(red);
    });

    it("should paint the same band as discrete mode at a shared threshold", () => {
      const bands = [
        { key: "g", threshold: 50, color: yellow },
        { key: "h", threshold: 50, color: red },
      ];
      expect(create({ bands, smooth: true }, 50)[0].value()).toEqual(red);
      expect(create({ bands }, 50)[0].value()).toEqual(red);
    });

    it("should paint the background below every threshold", () => {
      expect(
        create({ bands: BANDS, background: green, smooth: true }, 0)[0].value(),
      ).toEqual(green);
    });
  });

  describe("flashing", () => {
    it("should alternate between the band color and the background on each tick", () => {
      const [t] = create(
        {
          bands: [{ key: "i", threshold: 0, color: red, flashing: true }],
          background: green,
        },
        10,
      );
      const phase = new TestSource(0);
      t.setSources({ phase });
      expect(t.value()).toEqual(red);
      phase.setValue(1);
      expect(t.value()).toEqual(green);
      phase.setValue(2);
      expect(t.value()).toEqual(red);
    });

    it("should hold the color of a band that does not flash", () => {
      const [t] = create({ bands: BANDS, background: green }, 75);
      const phase = new TestSource(1);
      t.setSources({ phase });
      expect(t.value()).toEqual(yellow);
    });
  });

  describe("notifications", () => {
    it("should notify only when the color changes", () => {
      const [t, source] = create({ bands: BANDS }, 60);
      const handler = vi.fn();
      t.onChange(handler);
      source.setValue(70);
      source.setValue(80);
      expect(handler).toHaveBeenCalledTimes(1);
      source.setValue(120);
      expect(handler).toHaveBeenCalledTimes(2);
    });

    it("should notify on every tick of a flashing band", () => {
      const [t] = create(
        {
          bands: [{ key: "j", threshold: 0, color: red, flashing: true }],
          background: green,
        },
        10,
      );
      const phase = new TestSource(0);
      t.setSources({ phase });
      const handler = vi.fn();
      t.onChange(handler);
      phase.setValue(1);
      phase.setValue(2);
      phase.setValue(3);
      expect(handler).toHaveBeenCalledTimes(3);
    });
  });
});
