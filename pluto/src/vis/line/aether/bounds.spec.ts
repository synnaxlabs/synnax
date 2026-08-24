// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  bounds,
  DataType,
  MultiSeries,
  Series,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { windowBounds } from "@/vis/line/aether/bounds";

const THRESHOLD = TimeSpan.milliseconds(2);

const stamps = (startSec: number, count: number, alignment: bigint = 0n): Series =>
  new Series({
    data: new BigInt64Array(
      Array.from({ length: count }, (_, i) =>
        TimeStamp.seconds(startSec + i).valueOf(),
      ),
    ),
    dataType: DataType.TIMESTAMP,
    timeRange: TimeStamp.seconds(startSec).range(TimeStamp.seconds(startSec + count)),
    alignment,
  });

const values = (
  startSec: number,
  data: number[],
  alignment: bigint = 0n,
  sampleOffset: number = 0,
): Series =>
  new Series({
    data: new Float32Array(data),
    dataType: DataType.FLOAT32,
    timeRange: TimeStamp.seconds(startSec).range(
      TimeStamp.seconds(startSec + data.length),
    ),
    alignment,
    sampleOffset,
  });

const xWindow = (loSec: number, hiSec: number): bounds.Bounds => ({
  lower: Number(TimeStamp.seconds(loSec).valueOf()),
  upper: Number(TimeStamp.seconds(hiSec).valueOf()),
});

const calc = (x: Series[], y: Series[], win: bounds.Bounds): bounds.Bounds =>
  windowBounds(new MultiSeries(x), new MultiSeries(y), win, THRESHOLD);

describe("windowBounds", () => {
  it("should bound only the samples inside the window", () => {
    const x = stamps(0, 10);
    const y = values(0, [0, 1, 2, 3, 4, 5, 6, 7, 8, 9]);
    expect(calc([x], [y], xWindow(2, 5))).toStrictEqual({ lower: 2, upper: 5 });
  });

  it("should exclude an out-of-window outlier", () => {
    const x = stamps(0, 10);
    const y = values(0, [9999, -250, -240, -230, -220, -210, -205, -202, -201, -200]);
    expect(calc([x], [y], xWindow(1, 9))).toStrictEqual({ lower: -250, upper: -200 });
  });

  it("should return non-finite bounds when no samples fall in the window", () => {
    const x = stamps(0, 10);
    const y = values(0, [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]);
    expect(bounds.isFinite(calc([x], [y], xWindow(20, 30)))).toBe(false);
  });

  it("should pair series with offset alignments", () => {
    const x = stamps(0, 10, 0n);
    // Pairs with x samples at 3s through 9s.
    const y = values(3, [30, 40, 50, 60, 70, 80, 90], 3n);
    expect(calc([x], [y], xWindow(4, 6))).toStrictEqual({ lower: 40, upper: 60 });
  });

  it("should union bounds across consecutive series pairs", () => {
    const x1 = stamps(0, 5, 0n);
    const x2 = stamps(5, 5, 5n);
    const y1 = values(0, [1, 2, 3, 4, 5], 0n);
    const y2 = values(5, [6, 7, 8, 9, 10], 5n);
    expect(calc([x1, x2], [y1, y2], xWindow(3, 7))).toStrictEqual({
      lower: 4,
      upper: 8,
    });
  });

  it("should not pair series whose time ranges do not overlap", () => {
    const x = stamps(0, 5, 0n);
    const y = values(100, [1, 2, 3, 4, 5], 0n);
    expect(bounds.isFinite(calc([x], [y], xWindow(0, 5)))).toBe(false);
  });

  it("should apply the series sample offset", () => {
    const x = stamps(0, 3);
    const y = values(0, [1, 2, 3], 0n, 10);
    expect(calc([x], [y], xWindow(0, 2))).toStrictEqual({ lower: 11, upper: 13 });
  });
});
