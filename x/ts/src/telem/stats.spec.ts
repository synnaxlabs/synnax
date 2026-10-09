// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { stats } from "@/telem";

describe("stats", () => {
  const y = [1, -2, 3, 0];

  it("should compute min, max, and peak to peak", () => {
    expect(stats.min(y)).toBe(-2);
    expect(stats.max(y)).toBe(3);
    expect(stats.peakToPeak(y)).toBe(5);
  });

  it("should compute the mean and RMS", () => {
    expect(stats.mean(y)).toBe(0.5);
    expect(stats.rms([3, 4])).toBeCloseTo(Math.sqrt(12.5), 10);
  });

  it("should return NaN for empty input", () => {
    expect(stats.min([])).toBeNaN();
    expect(stats.mean([])).toBeNaN();
    expect(stats.rms([])).toBeNaN();
  });

  it("should estimate frequency from mean crossings", () => {
    const n = 1000;
    const x = new Float64Array(n);
    const s = new Float64Array(n);
    for (let i = 0; i < n; i++) {
      x[i] = i / 1000;
      s[i] = 2 + Math.sin(2 * Math.PI * 50 * x[i]);
    }
    expect(stats.frequency(x, s)).toBeCloseTo(50, 1);
  });

  it("should return NaN frequency with fewer than two crossings", () => {
    expect(stats.frequency([0, 1, 2], [0, 1, 2])).toBeNaN();
  });

  it("should measure the 10% to 90% rise time", () => {
    const x = [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10];
    const s = [0, 0, 0, 0, 0, 2.5, 5, 7.5, 10, 10, 10];
    expect(stats.riseTime(x, s)).toBeCloseTo(3.2, 10);
  });

  it("should return NaN rise time for a flat signal", () => {
    expect(stats.riseTime([0, 1, 2], [1, 1, 1])).toBeNaN();
  });
});
