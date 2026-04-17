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
    let mockTime = TimeStamp.seconds(0);
    const calc = new ClockSkewCalculator(() => mockTime);
    calc.start();
    mockTime = TimeStamp.seconds(10);
    // Remote midpoint is 3s, local midpoint is 5s, so skew is 2s
    calc.end(TimeStamp.seconds(3));
    expect(calc.skew).toEqual(TimeSpan.seconds(2));
    expect(calc.exceeds(TimeSpan.seconds(1))).toBe(true);
    expect(calc.exceeds(TimeSpan.seconds(3))).toBe(false);
  });

  it("should report zero skew when times match perfectly", () => {
    let mockTime = TimeStamp.seconds(0);
    const calc = new ClockSkewCalculator(() => mockTime);
    calc.start();
    mockTime = TimeStamp.seconds(10);
    // Remote midpoint matches local midpoint at 5s
    calc.end(TimeStamp.seconds(5));
    expect(calc.skew).toEqual(TimeSpan.ZERO);
    expect(calc.exceeds(TimeSpan.seconds(1))).toBe(false);
  });

  it("should return the most recent measurement", () => {
    let mockTime = TimeStamp.seconds(0);
    const calc = new ClockSkewCalculator(() => mockTime);
    calc.start();
    mockTime = TimeStamp.seconds(10);
    calc.end(TimeStamp.seconds(3));
    expect(calc.skew).toEqual(TimeSpan.seconds(2));
    mockTime = TimeStamp.seconds(0);
    calc.start();
    mockTime = TimeStamp.seconds(10);
    // Remote midpoint is 7s, local midpoint is 5s, so skew is -2s
    calc.end(TimeStamp.seconds(7));
    expect(calc.skew).toEqual(TimeSpan.seconds(-2));
  });

  it("should return zero skew when no measurements taken", () => {
    const calc = new ClockSkewCalculator();
    expect(calc.skew).toEqual(TimeSpan.ZERO);
    expect(calc.exceeds(TimeSpan.seconds(1))).toBe(false);
  });
});
