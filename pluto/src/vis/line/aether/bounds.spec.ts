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

import { BoundsCache } from "@/vis/line/aether/bounds";

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

const calc = (
  cache: BoundsCache,
  x: Series[],
  y: Series[],
  win: bounds.Bounds,
): bounds.Bounds => cache.value(new MultiSeries(x), new MultiSeries(y), win, THRESHOLD);

describe("BoundsCache", () => {
  it("should bound only the samples inside the window", () => {
    const cache = new BoundsCache();
    const x = stamps(0, 10);
    const y = values(0, [0, 1, 2, 3, 4, 5, 6, 7, 8, 9]);
    const b = calc(cache, [x], [y], xWindow(2, 5));
    expect(b).toStrictEqual({ lower: 2, upper: 5 });
  });

  it("should exclude an out-of-window outlier", () => {
    const cache = new BoundsCache();
    const x = stamps(0, 10);
    const y = values(0, [9999, -250, -240, -230, -220, -210, -205, -202, -201, -200]);
    const b = calc(cache, [x], [y], xWindow(1, 9));
    expect(b).toStrictEqual({ lower: -250, upper: -200 });
  });

  it("should return non-finite bounds when no samples fall in the window", () => {
    const cache = new BoundsCache();
    const x = stamps(0, 10);
    const y = values(0, [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]);
    const b = calc(cache, [x], [y], xWindow(20, 30));
    expect(bounds.isFinite(b)).toBe(false);
  });

  it("should pair series with offset alignments", () => {
    const cache = new BoundsCache();
    const x = stamps(0, 10, 0n);
    // Pairs with x samples at 3s through 9s.
    const y = values(3, [30, 40, 50, 60, 70, 80, 90], 3n);
    const b = calc(cache, [x], [y], xWindow(4, 6));
    expect(b).toStrictEqual({ lower: 40, upper: 60 });
  });

  it("should union bounds across consecutive series pairs", () => {
    const cache = new BoundsCache();
    const x1 = stamps(0, 5, 0n);
    const x2 = stamps(5, 5, 5n);
    const y1 = values(0, [1, 2, 3, 4, 5], 0n);
    const y2 = values(5, [6, 7, 8, 9, 10], 5n);
    const b = calc(cache, [x1, x2], [y1, y2], xWindow(3, 7));
    expect(b).toStrictEqual({ lower: 4, upper: 8 });
  });

  it("should not pair series whose time ranges do not overlap", () => {
    const cache = new BoundsCache();
    const x = stamps(0, 5, 0n);
    const y = values(100, [1, 2, 3, 4, 5], 0n);
    const b = calc(cache, [x], [y], xWindow(0, 5));
    expect(bounds.isFinite(b)).toBe(false);
  });

  it("should apply the series sample offset", () => {
    const cache = new BoundsCache();
    const x = stamps(0, 3);
    const y = values(0, [1, 2, 3], 0n, 10);
    const b = calc(cache, [x], [y], xWindow(0, 2));
    expect(b).toStrictEqual({ lower: 11, upper: 13 });
  });

  describe("block caching", () => {
    it("should answer queries that mix cached blocks and raw edges", () => {
      const cache = new BoundsCache(4);
      const x = stamps(0, 10);
      const y = values(0, [0, -500, 2, 3, 4, 5, 6, 7, 8, 900]);
      expect(calc(cache, [x], [y], xWindow(0, 9))).toStrictEqual({
        lower: -500,
        upper: 900,
      });
      // Repeats hit the now-warm block summaries.
      expect(calc(cache, [x], [y], xWindow(0, 9))).toStrictEqual({
        lower: -500,
        upper: 900,
      });
      // The partial first block is scanned raw, so the outlier at index 1 is
      // excluded once the window starts past it.
      expect(calc(cache, [x], [y], xWindow(2, 9))).toStrictEqual({
        lower: 2,
        upper: 900,
      });
    });

    it("should include samples appended after blocks were cached", () => {
      const cache = new BoundsCache(4);
      const x = stamps(0, 10);
      const y = Series.alloc({
        capacity: 10,
        dataType: DataType.FLOAT32,
        timeRange: TimeStamp.seconds(0).range(TimeStamp.seconds(10)),
        alignment: 0n,
      });
      y.write(values(0, [1, 2, 3, 4, 5, 6]));
      expect(calc(cache, [x], [y], xWindow(0, 9))).toStrictEqual({
        lower: 1,
        upper: 6,
      });
      y.write(values(6, [7, 8, -900], 6n));
      expect(calc(cache, [x], [y], xWindow(0, 9))).toStrictEqual({
        lower: -900,
        upper: 8,
      });
    });
  });
});
