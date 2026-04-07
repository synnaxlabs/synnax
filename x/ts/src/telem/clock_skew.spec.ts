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
    let mockTime = new TimeStamp(0);
    const calc = new ClockSkewCalculator(() => mockTime);
    calc.start();
    mockTime = new TimeStamp(10);
    calc.end(new TimeStamp(3));
    expect(calc.skew.valueOf()).toBe(2n);
    expect(calc.exceeds(new TimeSpan(1))).toBe(true);
    expect(calc.exceeds(new TimeSpan(3))).toBe(false);
  });

  it("should report zero skew when times match perfectly", () => {
    let mockTime = new TimeStamp(0);
    const calc = new ClockSkewCalculator(() => mockTime);
    calc.start();
    mockTime = new TimeStamp(1000);
    calc.end(new TimeStamp(500));
    expect(calc.skew.valueOf()).toBe(0n);
    expect(calc.exceeds(new TimeSpan(1))).toBe(false);
  });

  it("should average multiple measurements", () => {
    let mockTime = new TimeStamp(0);
    const calc = new ClockSkewCalculator(() => mockTime);
    calc.start();
    mockTime = new TimeStamp(10);
    calc.end(new TimeStamp(3));
    mockTime = new TimeStamp(0);
    calc.start();
    mockTime = new TimeStamp(10);
    calc.end(new TimeStamp(7));
    expect(calc.skew.valueOf()).toBe(0n);
  });

  it("should return zero skew when no measurements taken", () => {
    const calc = new ClockSkewCalculator();
    expect(calc.skew.valueOf()).toBe(0n);
    expect(calc.exceeds(new TimeSpan(1))).toBe(false);
  });
});
