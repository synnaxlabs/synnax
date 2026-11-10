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
  math,
  MultiSeries,
  Series,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { describe, expect, it, onTestFinished, vi } from "vitest";

import { forEachPair, windowBounds } from "@/vis/line/aether/bounds";

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

const FALLBACK: bounds.Bounds = { lower: -1234, upper: 1234 };

const calc = (x: Series[], y: Series[], win: bounds.Bounds): bounds.Bounds =>
  windowBounds(new MultiSeries(x), new MultiSeries(y), win, THRESHOLD, FALLBACK);

// A seeded mulberry32 generator, so each random case reproduces from its seed.
const random = (seed: number) => (): number => {
  seed = (seed + 0x6d2b79f5) | 0;
  let t = Math.imul(seed ^ (seed >>> 15), 1 | seed);
  t ^= t + Math.imul(t ^ (t >>> 7), 61 | t);
  return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
};

const DOMAIN = 1n << 32n;
const MS = TimeSpan.MILLISECOND.valueOf();

interface Shape {
  alignment: bigint;
  length: number;
  multiple: bigint;
  // Moves the time range, so it can overlap its pair by less than the threshold.
  shift: bigint;
}

const shape = (
  alignment: bigint,
  length: number,
  rest: Partial<Shape> = {},
): Shape => ({
  alignment,
  length,
  multiple: 1n,
  shift: 0n,
  ...rest,
});

// Each alignment step is a millisecond, and each domain starts an hour after the last.
const createSeries = (
  { alignment, length, multiple, shift }: Shape,
  dataType: DataType,
  rand: () => number = () => 0,
): Series => {
  const start =
    (alignment / DOMAIN) * TimeSpan.HOUR.valueOf() + (alignment % DOMAIN) * MS + shift;
  const step = multiple * MS;
  const data = dataType.equals(DataType.TIMESTAMP)
    ? BigInt64Array.from({ length }, (_, i) => start + BigInt(i) * step)
    : Float32Array.from({ length }, () => Math.round(rand() * 200 - 100));
  return new Series({
    data,
    dataType,
    timeRange: new TimeRange(start, start + BigInt(length) * step),
    alignment,
    alignmentMultiple: multiple,
  });
};

const xOf = (s: Shape): Series => createSeries(s, DataType.TIMESTAMP);
const yOf = (s: Shape): Series => createSeries(s, DataType.FLOAT32);

// Cuts one or two domains into series with gaps. A mixed list also holds series that
// overlap others in the list, as a tile that stands in for others does, and may be
// out of order.
const createShapes = (
  rand: () => number,
  multiple: bigint,
  mixed: boolean,
): Shape[] => {
  const int = (n: number): number => Math.floor(rand() * n);
  const shift = (): bigint => (rand() < 0.7 ? 0n : (BigInt(int(9) - 4) * MS) / 2n);
  const shapes: Shape[] = [];
  const domains = 1 + int(2);
  for (let d = 0; d < domains; d++) {
    let index = BigInt(int(5));
    const end = index + BigInt(20 + int(60));
    while (index < end) {
      const length = rand() < 0.05 ? 0 : 1 + int(11);
      const alignment = BigInt(d) * DOMAIN + index;
      if (rand() < 0.8) shapes.push({ alignment, length, multiple, shift: shift() });
      index += BigInt(length) * multiple + (rand() < 0.2 ? BigInt(int(4)) : 0n);
    }
  }
  if (!mixed) return shapes;
  const overlaps = rand() < 0.5 ? 1 + int(3) : 0;
  for (let i = 0; i < overlaps; i++)
    shapes.push({
      alignment: BigInt(int(domains)) * DOMAIN + BigInt(int(80)),
      length: 1 + int(30),
      multiple: rand() < 0.7 ? multiple : 3n - multiple,
      shift: shift(),
    });
  if (rand() < 0.3)
    for (let i = shapes.length - 1; i > 0; i--) {
      const j = int(i + 1);
      [shapes[i], shapes[j]] = [shapes[j], shapes[i]];
    }
  return shapes;
};

const createLists = (seed: number, mixed: boolean) => {
  const rand = random(seed);
  const multiple = rand() < 0.25 ? 2n : 1n;
  const yMultiple = rand() < 0.1 ? 3n - multiple : multiple;
  return {
    rand,
    xs: createShapes(rand, multiple, mixed).map(xOf),
    ys: createShapes(rand, yMultiple, mixed).map((s) =>
      createSeries(s, DataType.FLOAT32, rand),
    ),
  };
};

interface Pair {
  x: number;
  y: number;
  start: bigint;
  end: bigint;
}

const pairsOf = (xs: Series[], ys: Series[]): Pair[] => {
  const pairs: Pair[] = [];
  forEachPair(xs, ys, THRESHOLD, (x, y, start, end) => {
    pairs.push({ x: xs.indexOf(x), y: ys.indexOf(y), start, end });
  });
  return pairs;
};

const byIndex = (a: Pair, b: Pair): number => a.x - b.x || a.y - b.y;

// The all-pairs rule that forEachPair replaces.
const referencePairs = (xs: Series[], ys: Series[]): Pair[] => {
  const pairs: Pair[] = [];
  xs.forEach((x, xi) =>
    ys.forEach((y, yi) => {
      if (x.alignmentMultiple !== y.alignmentMultiple) return;
      const timesOverlap =
        x.timeRange.equals(y.timeRange) ||
        x.timeRange.overlapsWith(y.timeRange, THRESHOLD);
      if (!timesOverlap || !bounds.overlapsWith(x.alignmentBounds, y.alignmentBounds))
        return;
      const xOffset = x.alignment < y.alignment ? y.alignment - x.alignment : 0n;
      const yOffset = y.alignment < x.alignment ? x.alignment - y.alignment : 0n;
      const count = math.min(
        bounds.span(x.alignmentBounds) - xOffset,
        bounds.span(y.alignmentBounds) - yOffset,
      );
      if (count === 0n) return;
      const start = x.alignment + xOffset;
      pairs.push({ x: xi, y: yi, start, end: start + count });
    }),
  );
  return pairs;
};

// The all-pairs windowBounds that forEachPair replaces.
const referenceWindowBounds = (
  xs: Series[],
  ys: Series[],
  win: bounds.Bounds,
): bounds.Bounds => {
  let lower = Infinity;
  let upper = -Infinity;
  for (const { x: xi, y: yi, start, end } of referencePairs(xs, ys)) {
    const x = xs[xi];
    const y = ys[yi];
    const mult = x.alignmentMultiple;
    const xLo = x.binarySearch(win.lower);
    let xHi = x.binarySearch(win.upper);
    if (xHi < x.length && Number(x.at(xHi, true)) === win.upper) xHi++;
    const winStart = x.alignment + BigInt(xLo) * mult;
    const winEnd = x.alignment + BigInt(xHi) * mult;
    const s = winStart > start ? winStart : start;
    const e = winEnd < end ? winEnd : end;
    if (e <= s) continue;
    const b = y.boundsFor(
      Number((s - y.alignment) / mult),
      Number((e - y.alignment) / mult),
    );
    if (b.lower < lower) lower = b.lower;
    if (b.upper > upper) upper = b.upper;
  }
  const result = { lower, upper };
  return bounds.isFinite(result) ? result : FALLBACK;
};

// Picks window edges at sample times and between them, past both ends included.
const createWindow = (rand: () => number, xs: Series[]): bounds.Bounds => {
  const stamps = xs.flatMap((s) => Array.from(s.data as BigInt64Array, Number));
  if (stamps.length === 0) return { lower: 0, upper: 1 };
  const min = Math.min(...stamps) - 5e6;
  const max = Math.max(...stamps) + 5e6;
  const edge = (): number =>
    rand() < 0.5
      ? stamps[Math.floor(rand() * stamps.length)]
      : Math.round(min + rand() * (max - min));
  const a = edge();
  const b = edge();
  return { lower: Math.min(a, b), upper: Math.max(a, b) };
};

const SEEDS = Array.from({ length: 400 }, (_, i) => i + 1);

describe("forEachPair", () => {
  it("should find the pairs of the all-pairs rule in random lists", () => {
    let total = 0;
    for (const seed of SEEDS) {
      const { xs, ys } = createLists(seed, true);
      const want = referencePairs(xs, ys);
      total += want.length;
      expect(pairsOf(xs, ys).toSorted(byIndex), `seed ${seed}`).toEqual(want);
    }
    expect(total).toBeGreaterThan(SEEDS.length * 5);
  });

  it("should pair in x order, then y order, when no list overlaps itself", () => {
    for (const seed of SEEDS) {
      const { xs, ys } = createLists(seed, false);
      expect(pairsOf(xs, ys), `seed ${seed}`).toEqual(referencePairs(xs, ys));
    }
  });

  it("should pair series that overlap others in their own list", () => {
    const xs = [xOf(shape(0n, 100)), xOf(shape(10n, 70))];
    const ys = [yOf(shape(20n, 10)), yOf(shape(90n, 30))];
    expect(pairsOf(xs, ys).toSorted(byIndex)).toEqual([
      { x: 0, y: 0, start: 20n, end: 30n },
      { x: 0, y: 1, start: 90n, end: 100n },
      { x: 1, y: 0, start: 20n, end: 30n },
    ]);
  });

  it("should pair lists out of alignment order", () => {
    const xs = [xOf(shape(10n, 10)), xOf(shape(0n, 10))];
    const ys = [yOf(shape(15n, 10)), yOf(shape(5n, 10))];
    expect(pairsOf(xs, ys).toSorted(byIndex)).toEqual([
      { x: 0, y: 0, start: 15n, end: 20n },
      { x: 0, y: 1, start: 10n, end: 15n },
      { x: 1, y: 1, start: 5n, end: 10n },
    ]);
  });

  it("should give the shared alignments of a partial overlap", () => {
    const xs = [xOf(shape(0n, 10))];
    const ys = [yOf(shape(4n, 16))];
    expect(pairsOf(xs, ys)).toEqual([{ x: 0, y: 0, start: 4n, end: 10n }]);
  });

  it("should share alignments in steps of the alignment multiple", () => {
    const xs = [xOf(shape(0n, 5, { multiple: 2n }))];
    const ys = [yOf(shape(4n, 5, { multiple: 2n }))];
    expect(pairsOf(xs, ys)).toEqual([{ x: 0, y: 0, start: 4n, end: 10n }]);
  });

  it("should not pair series that only touch", () => {
    expect(pairsOf([xOf(shape(0n, 10))], [yOf(shape(10n, 10))])).toEqual([]);
  });

  it("should not pair an empty series", () => {
    expect(pairsOf([xOf(shape(5n, 0))], [yOf(shape(0n, 10))])).toEqual([]);
  });

  it("should not pair series of different alignment multiples", () => {
    const xs = [xOf(shape(0n, 10))];
    const ys = [yOf(shape(0n, 5, { multiple: 2n }))];
    expect(xs[0].timeRange.equals(ys[0].timeRange)).toBe(true);
    expect(pairsOf(xs, ys)).toEqual([]);
  });

  it("should not pair series of different domains", () => {
    expect(pairsOf([xOf(shape(0n, 10))], [yOf(shape(DOMAIN, 10))])).toEqual([]);
  });

  it("should pair equal time ranges shorter than the threshold", () => {
    expect(pairsOf([xOf(shape(0n, 1))], [yOf(shape(0n, 1))])).toEqual([
      { x: 0, y: 0, start: 0n, end: 1n },
    ]);
  });

  it("should not pair time ranges that overlap by less than the threshold", () => {
    const xs = [xOf(shape(0n, 10))];
    const ys = [yOf(shape(0n, 10, { shift: 9n * MS }))];
    expect(pairsOf(xs, ys)).toEqual([]);
  });

  it("should compare each series with a bounded number of others", () => {
    const n = 1000;
    const xs = Array.from({ length: n }, (_, i) => xOf(shape(BigInt(i * 10), 10)));
    const ys = Array.from({ length: n }, (_, i) => yOf(shape(BigInt(i * 10), 10)));
    const reads = vi.spyOn(Series.prototype, "alignmentBounds", "get");
    onTestFinished(() => reads.mockRestore());
    expect(pairsOf(xs, ys)).toHaveLength(n);
    expect(reads.mock.calls.length).toBeLessThan(10 * n);
  });
});

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

  it("should return the fallback when the window is beyond all samples", () => {
    const x = stamps(0, 10);
    const y = values(0, [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]);
    expect(calc([x], [y], xWindow(20, 30))).toStrictEqual(FALLBACK);
  });

  it("should return the fallback when the window predates all samples", () => {
    const x = stamps(20, 10);
    const y = values(20, [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]);
    expect(calc([x], [y], xWindow(0, 10))).toStrictEqual(FALLBACK);
  });

  it("should return the fallback when every value in the window is NaN", () => {
    const x = stamps(0, 3);
    const y = values(0, [NaN, NaN, NaN]);
    expect(calc([x], [y], xWindow(0, 3))).toStrictEqual(FALLBACK);
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

  it("should pair series with equal time ranges shorter than the threshold", () => {
    const timeRange = TimeStamp.seconds(1).spanRange(TimeSpan.NANOSECOND);
    const x = new Series({
      data: new BigInt64Array([TimeStamp.seconds(1).valueOf()]),
      dataType: DataType.TIMESTAMP,
      timeRange,
    });
    const y = new Series({
      data: new Float32Array([42]),
      dataType: DataType.FLOAT32,
      timeRange,
    });
    expect(calc([x], [y], xWindow(0, 2))).toStrictEqual({ lower: 42, upper: 42 });
  });

  it("should not pair series whose time ranges do not overlap", () => {
    const x = stamps(0, 5, 0n);
    const y = values(100, [1, 2, 3, 4, 5], 0n);
    expect(calc([x], [y], xWindow(0, 5))).toStrictEqual(FALLBACK);
  });

  it("should apply the series sample offset", () => {
    const x = stamps(0, 3);
    const y = values(0, [1, 2, 3], 0n, 10);
    expect(calc([x], [y], xWindow(0, 2))).toStrictEqual({ lower: 11, upper: 13 });
  });

  it("should match the all-pairs bounds in random lists and windows", () => {
    for (const seed of SEEDS) {
      const { rand, xs, ys } = createLists(seed, true);
      for (let i = 0; i < 3; i++) {
        const win = createWindow(rand, xs);
        expect(calc(xs, ys, win), `seed ${seed}`).toStrictEqual(
          referenceWindowBounds(xs, ys, win),
        );
      }
    }
  });
});
