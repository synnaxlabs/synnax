// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { ClockSkewCalculator, TimeSpan, TimeStamp } from "@/telem";

describe("ClockSkewCalculator", () => {
  it("should correctly calculate clock skew from a single measurement", () => {
    let mockTime = new TimeStamp(0n);
    const calc = new ClockSkewCalculator(() => mockTime);

    calc.start(); // local_start_t = 0
    mockTime = new TimeStamp(10n); // advance local clock by 10ns

    // remote midpoint was 3ns
    calc.end(new TimeStamp(3n));
    // local midpoint = 0 + (10 - 0) / 2 = 5
    // skew = 5 - 3 = 2ns

    expect(calc.skew().valueOf()).toBe(2n);
    expect(calc.exceeds(TimeSpan.nanoseconds(1))).toBe(true);
    expect(calc.exceeds(TimeSpan.nanoseconds(3))).toBe(false);
  });

  it("should report zero skew when local and remote times match perfectly", () => {
    let mockTime = new TimeStamp(0n);
    const calc = new ClockSkewCalculator(() => mockTime);

    calc.start();
    mockTime = new TimeStamp(1000n);
    // remote midpoint matches local midpoint exactly (500)
    calc.end(new TimeStamp(500n));

    expect(calc.skew().valueOf()).toBe(0n);
    expect(calc.exceeds(TimeSpan.nanoseconds(1))).toBe(false);
  });

  it("should average skew across multiple measurements", () => {
    let mockTime = new TimeStamp(0n);
    const calc = new ClockSkewCalculator(() => mockTime);

    // First measurement: local mid = 5, remote = 3, skew = 2
    calc.start();
    mockTime = new TimeStamp(10n);
    calc.end(new TimeStamp(3n));

    // Second measurement: local mid = 15, remote = 11, skew = 4
    mockTime = new TimeStamp(10n);
    calc.start();
    mockTime = new TimeStamp(20n);
    calc.end(new TimeStamp(11n));

    // Average skew = (2 + 4) / 2 = 3
    expect(calc.skew().valueOf()).toBe(3n);
  });

  it("should return zero skew when no measurements taken", () => {
    const calc = new ClockSkewCalculator();
    expect(calc.skew().valueOf()).toBe(0n);
    expect(calc.exceeds(TimeSpan.nanoseconds(1))).toBe(false);
  });

  it("should handle negative skew (local behind remote)", () => {
    let mockTime = new TimeStamp(0n);
    const calc = new ClockSkewCalculator(() => mockTime);

    calc.start();
    mockTime = new TimeStamp(10n);
    // remote midpoint is 8, local midpoint is 5
    // skew = 5 - 8 = -3
    calc.end(new TimeStamp(8n));

    expect(calc.skew().valueOf()).toBe(-3n);
    expect(calc.exceeds(TimeSpan.nanoseconds(2))).toBe(true);
    expect(calc.exceeds(TimeSpan.nanoseconds(3))).toBe(false);
  });
});
