// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { telem } from "@/telem";

describe("ClockSkewCalculator", () => {
  it("should correctly calculate clock skew from a single measurement", () => {
    let mockTime = telem.TimeStamp.seconds(0);
    const calc = new telem.ClockSkewCalculator(() => mockTime);
    calc.start();
    mockTime = telem.TimeStamp.seconds(10);
    // Remote midpoint is 3s, local midpoint is 5s, so skew is 2s
    calc.end(telem.TimeStamp.seconds(3));
    expect(calc.skew).toEqual(telem.TimeSpan.seconds(2));
    expect(calc.exceeds(telem.TimeSpan.seconds(1))).toBe(true);
    expect(calc.exceeds(telem.TimeSpan.seconds(3))).toBe(false);
  });

  it("should report zero skew when times match perfectly", () => {
    let mockTime = telem.TimeStamp.seconds(0);
    const calc = new telem.ClockSkewCalculator(() => mockTime);
    calc.start();
    mockTime = telem.TimeStamp.seconds(10);
    // Remote midpoint matches local midpoint at 5s
    calc.end(telem.TimeStamp.seconds(5));
    expect(calc.skew).toEqual(telem.TimeSpan.ZERO);
    expect(calc.exceeds(telem.TimeSpan.seconds(1))).toBe(false);
  });

  it("should return the most recent measurement", () => {
    let mockTime = telem.TimeStamp.seconds(0);
    const calc = new telem.ClockSkewCalculator(() => mockTime);
    calc.start();
    mockTime = telem.TimeStamp.seconds(10);
    calc.end(telem.TimeStamp.seconds(3));
    expect(calc.skew).toEqual(telem.TimeSpan.seconds(2));
    mockTime = telem.TimeStamp.seconds(0);
    calc.start();
    mockTime = telem.TimeStamp.seconds(10);
    // Remote midpoint is 7s, local midpoint is 5s, so skew is -2s
    calc.end(telem.TimeStamp.seconds(7));
    expect(calc.skew).toEqual(telem.TimeSpan.seconds(-2));
  });

  it("should return zero skew when no measurements taken", () => {
    const calc = new telem.ClockSkewCalculator();
    expect(calc.skew).toEqual(telem.TimeSpan.ZERO);
    expect(calc.exceeds(telem.TimeSpan.seconds(1))).toBe(false);
  });
});
