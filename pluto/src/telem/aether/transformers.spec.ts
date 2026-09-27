// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";

import { TestSource } from "@/telem/aether/test/source";
import {
  RollingAverage,
  ScaleNumber,
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

describe("ScaleNumber", () => {
  it("applies the scale and offset to a number value", () => {
    const t = new ScaleNumber({ scale: { scale: 2, offset: 3 } });
    t.setSources({ in: new TestSource(10) });
    expect(t.value()).toBe(23);
  });

  it("coerces a bigint value to a number before scaling", () => {
    const t = new ScaleNumber({ scale: { scale: 2, offset: 0 } });
    t.setSources({ in: new TestSource(50n) });
    expect(t.value()).toBe(100);
  });

  it("returns NaN when given NaN", () => {
    const t = new ScaleNumber({ scale: { scale: 2, offset: 3 } });
    t.setSources({ in: new TestSource(NaN) });
    expect(Number.isNaN(t.value())).toBe(true);
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
