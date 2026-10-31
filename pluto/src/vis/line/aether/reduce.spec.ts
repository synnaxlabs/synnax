// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type Aggregation,
  DataType,
  Series,
  TimeRange,
  TimeStamp,
} from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { type DrawOperation } from "@/vis/line/aether/draw";
import { groupSize, Reducer } from "@/vis/line/aether/reduce";

const alignment = (domain: number, sample: number): bigint =>
  (BigInt(domain) << 32n) | BigInt(sample);

const f32 = (...values: number[]): number[] => values.map(Math.fround);

const F32_MAX = 3.4028234663852886e38;

const ramp = (length: number, a: bigint = 0n): Series =>
  new Series({
    data: new Float32Array(Array.from({ length }, (_, i) => i)),
    alignment: a,
  });

const op = (x: Series, y: Series): DrawOperation => {
  const xOffset = Number(y.alignment > x.alignment ? y.alignment - x.alignment : 0n);
  const yOffset = Number(x.alignment > y.alignment ? x.alignment - y.alignment : 0n);
  return {
    x,
    y,
    xOffset,
    yOffset,
    count: Math.min(x.length - xOffset, y.length - yOffset),
    span: x.timeRange,
  };
};

const spanned = (o: DrawOperation, start: number, end: number): DrawOperation => ({
  ...o,
  span: new TimeRange(TimeStamp.seconds(start), TimeStamp.seconds(end)),
});

const reduceOne = (o: DrawOperation, aggregation: Aggregation, size: number) =>
  new Reducer().reduce([o], aggregation, size)[0];

describe("groupSize", () => {
  const N = 8000;
  const VIEW = { bounds: { lower: 0, upper: N - 1 }, width: 800 };
  const dense = op(ramp(N), ramp(N));

  it("should hold a pixel of samples per group, rounded up to a power of two", () => {
    expect(groupSize([dense], VIEW, 1 / 2)).toBe(32);
    expect(groupSize([dense], VIEW, 1)).toBe(16);
    expect(groupSize([dense], VIEW, 1 / 4)).toBe(64);
  });

  it("should keep the group size across a small zoom", () => {
    const view = { ...VIEW, bounds: { lower: 0, upper: 7000 } };
    expect(groupSize([dense], view, 1 / 2)).toBe(32);
  });

  it("should measure the x span across every op", () => {
    const later = new Series({
      data: new Float32Array(Array.from({ length: N / 2 }, (_, i) => N / 2 + i)),
      alignment: BigInt(N / 2),
    });
    const halves = [
      op(ramp(N / 2), ramp(N / 2)),
      op(later, ramp(N / 2, BigInt(N / 2))),
    ];
    expect(groupSize(halves, VIEW, 1 / 2)).toBe(32);
  });

  it("should leave out ops of reduced series", () => {
    const reduced = new Series({
      data: new Float32Array(N),
      alignmentMultiple: 2n,
    });
    expect(groupSize([dense, op(reduced, reduced)], VIEW, 1 / 2)).toBe(32);
  });

  it("should return 1 when a group would hold under two samples", () => {
    const view = { ...VIEW, bounds: { lower: 0, upper: 100 } };
    expect(groupSize([dense], view, 1 / 2)).toBe(1);
  });

  it("should return 1 when a span is zero", () => {
    const flat = new Series(new Float32Array([5, 5, 5, 5]));
    expect(groupSize([op(flat, flat)], VIEW, 1 / 2)).toBe(1);
    expect(groupSize([dense], { ...VIEW, width: 0 }, 1 / 2)).toBe(1);
    expect(groupSize([], VIEW, 1 / 2)).toBe(1);
  });
});

// The expected y values below are the output of telem.Series.Reduce in x/go on the same
// input. Each x holds its sample index, so x shows the indices the pair kept.
describe("Reducer", () => {
  describe("min_max", () => {
    it("should keep the y extremes of each group and x at their indices", () => {
      const tr = new TimeRange(TimeStamp.seconds(10), TimeStamp.seconds(20));
      const x = ramp(8, alignment(2, 0));
      const y = new Series({
        data: new Float32Array([3, 1, 4, 1, 5, 9, 2, 6]),
        alignment: alignment(2, 0),
        timeRange: tr,
      });
      const r = reduceOne(op(x, y), "min_max", 4);
      expect(Array.from(r.y)).toEqual([1, 4, 9, 2]);
      expect(Array.from(r.x)).toEqual([1, 2, 5, 6]);
      expect(r.y.alignment).toBe(alignment(2, 0));
      expect(r.x.alignment).toBe(alignment(2, 0));
      expect(r.y.alignmentMultiple).toBe(2n);
      expect(r.x.alignmentMultiple).toBe(2n);
      expect(r.y.timeRange).toBe(tr);
      expect(r).toMatchObject({ xOffset: 0, yOffset: 0, count: 4 });
    });

    it("should repeat the sample of a group with one sample present", () => {
      const y = new Series({ data: new Uint8Array([7, 8, 9]), alignment: 3n });
      const r = reduceOne(op(ramp(3, 3n), y), "min_max", 2);
      expect(Array.from(r.y)).toEqual([7, 7, 8, 9]);
      expect(Array.from(r.x)).toEqual([0, 0, 1, 2]);
      expect(r.y.alignment).toBe(2n);
      expect(r.y.alignmentMultiple).toBe(1n);
    });

    it("should keep a NaN only when it starts a group", () => {
      const y = new Series(new Float32Array([NaN, 1, 2, 3, 4, NaN, 6, 7]));
      const r = reduceOne(op(ramp(8), y), "min_max", 4);
      expect(Array.from(r.y)).toEqual([NaN, NaN, 4, 7]);
      expect(Array.from(r.x)).toEqual([0, 0, 4, 7]);
    });

    it("should take x through the offset of the op", () => {
      const y = new Series({ data: new Float32Array([5, 1, 4, 2]), alignment: 3n });
      const r = reduceOne(op(ramp(10), y), "min_max", 4);
      expect(Array.from(r.y)).toEqual([5, 5, 1, 4]);
      expect(Array.from(r.x)).toEqual([3, 3, 4, 5]);
      expect(r.x.alignment).toBe(0n);
    });
  });

  describe("average", () => {
    it("should keep the mean of x and the mean of y in each group", () => {
      const y = new Series(new Float32Array([1, 2, 3, 4, 5, 6]));
      const r = reduceOne(op(ramp(6), y), "average", 2);
      expect(Array.from(r.y)).toEqual([1.5, 3.5, 5.5]);
      expect(Array.from(r.x)).toEqual([0.5, 2.5, 4.5]);
      expect(r.y.alignmentMultiple).toBe(2n);
    });

    it("should round a float32 mean to float32 once", () => {
      const y = new Series(new Float32Array([0.1, 0.1, 1]));
      const r = reduceOne(op(ramp(3), y), "average", 4);
      expect(Array.from(r.y)).toEqual(f32(0.4));
      expect(Array.from(r.x)).toEqual([1]);
    });

    it.each([
      ["float32 past its range", new Float32Array([-F32_MAX, F32_MAX]), 0],
      [
        "float32 with a mean offset past its range",
        new Float32Array([-F32_MAX, F32_MAX, F32_MAX]),
        Math.fround(F32_MAX / 3),
      ],
      [
        "float64 past its range",
        new Float64Array([-Number.MAX_VALUE, Number.MAX_VALUE]),
        0,
      ],
      ["int8 past its range", new Int8Array([-100, 100]), 0],
      ["int8 with a mean offset past its range", new Int8Array([-128, 127, 127]), 42],
      ["int8 with a negative fraction", new Int8Array([-1, -2]), -2],
      ["int32 past its range", new Int32Array([-(2 ** 31), 2 ** 31 - 1]), -1],
    ])("should average a group of %s as the Core does", (_, data, mean) => {
      const y = new Series(data);
      expect(Array.from(reduceOne(op(ramp(data.length), y), "average", 4).y)).toEqual([
        mean,
      ]);
    });

    it("should round down the average of integer series", () => {
      const x = new Series({
        data: new BigInt64Array([0n, 1n, 2n, 3n]),
        dataType: DataType.TIMESTAMP,
      });
      const y = new Series(new Int16Array([0, 1, 0, 1]));
      const r = reduceOne(op(x, y), "average", 4);
      expect(Array.from(r.x.glData)).toEqual([1]);
      expect(Array.from(r.y)).toEqual([0]);
    });

    it("should scale the mean of a partial group to its count", () => {
      const y = new Series(new Float64Array([0.1, 0.2, 2.3]));
      const r = reduceOne(op(ramp(3), y), "average", 4);
      expect(Array.from(r.y)).toEqual(f32(0.8666666666666665));
    });

    it("should average an integer group whose sum passes 2^53", () => {
      const n = 2 ** 22;
      const data = new Uint32Array(n)
        .fill(2 ** 32 - 3, 0, 3 * 2 ** 20)
        .fill(1, 3 * 2 ** 20);
      const r = reduceOne(op(ramp(n), new Series(data)), "average", n);
      expect(Array.from(r.y)).toEqual(f32(3221225470));
    });

    it("should propagate a NaN in a group", () => {
      const y = new Series(new Float32Array([NaN, 1, 2, 3, 4, NaN, 6, 7]));
      expect(Array.from(reduceOne(op(ramp(8), y), "average", 4).y)).toEqual([NaN, NaN]);
    });
  });

  describe("decimate", () => {
    it("should keep the first sample of each group", () => {
      const y = new Series({
        data: new Uint8Array([1, 2, 3, 4, 5, 6]),
        alignment: 1n,
      });
      const r = reduceOne(op(ramp(6, 1n), y), "decimate", 2);
      expect(Array.from(r.y)).toEqual([1, 2, 4, 6]);
      expect(Array.from(r.x)).toEqual([0, 1, 3, 5]);
      expect(r.y.alignment).toBe(0n);
      expect(r.y.alignmentMultiple).toBe(2n);
    });
  });

  describe("partial groups", () => {
    const x = ramp(7, alignment(5, 1));
    const y = new Series({
      data: new Float32Array([0.1, 0.2, 0.7, 1e7, 1e7 + 1, 3.3, -2.5]),
      alignment: alignment(5, 1),
    });

    it("should reduce the samples present in a partial first and last group", () => {
      const minMax = reduceOne(op(x, y), "min_max", 4);
      expect(Array.from(minMax.y)).toEqual(f32(0.1, 0.7, 1.0000001e7, -2.5));
      expect(Array.from(minMax.x)).toEqual([0, 2, 4, 6]);
      const average = reduceOne(op(x, y), "average", 4);
      expect(Array.from(average.y)).toEqual(f32(0.33333334, 5.0000005e6));
      expect(Array.from(average.x)).toEqual([1, 4.5]);
      const decimate = reduceOne(op(x, y), "decimate", 4);
      expect(Array.from(decimate.y)).toEqual(f32(0.1, 1e7));
      expect(Array.from(decimate.x)).toEqual([0, 3]);
    });

    it("should start at the first group of the domain", () => {
      expect(reduceOne(op(x, y), "min_max", 4).y.alignment).toBe(alignment(5, 0));
      expect(reduceOne(op(x, y), "min_max", 4).x.alignmentMultiple).toBe(2n);
      expect(reduceOne(op(x, y), "average", 4).x.alignmentMultiple).toBe(4n);
      expect(reduceOne(op(x, y), "decimate", 4).y.alignmentMultiple).toBe(4n);
    });
  });

  it("should reduce a uint8 y as uint8", () => {
    const x = ramp(7, alignment(1, 6));
    const y = new Series({
      data: new Uint8Array([5, 2, 2, 9, 9, 1, 7]),
      alignment: alignment(1, 6),
    });
    const minMax = reduceOne(op(x, y), "min_max", 4);
    expect(minMax.y.dataType.equals(DataType.UINT8)).toBe(true);
    expect(minMax.x.dataType.equals(DataType.FLOAT32)).toBe(true);
    expect(Array.from(minMax.y)).toEqual([5, 2, 9, 1, 7, 7]);
    expect(Array.from(minMax.x)).toEqual([0, 1, 3, 5, 6, 6]);
    expect(minMax.y.alignment).toBe(alignment(1, 4));
    const average = reduceOne(op(x, y), "average", 4);
    expect(Array.from(average.y)).toEqual([3, 5, 7]);
    expect(Array.from(average.x)).toEqual([0.5, 3.5, 6]);
    const decimate = reduceOne(op(x, y), "decimate", 4);
    expect(Array.from(decimate.y)).toEqual([5, 2, 7]);
    expect(Array.from(decimate.x)).toEqual([0, 2, 6]);
  });

  it("should keep the time range and sample offset of each series", () => {
    const xRange = new TimeRange(TimeStamp.seconds(1), TimeStamp.seconds(2));
    const x = new Series({
      data: new Float32Array([0, 1, 2, 3]),
      sampleOffset: 100n,
      timeRange: xRange,
    });
    const y = new Series({ data: new Float32Array([4, 5, 6, 7]), sampleOffset: 5 });
    const r = reduceOne(op(x, y), "min_max", 2);
    expect(r.x.sampleOffset).toBe(100n);
    expect(r.y.sampleOffset).toBe(5);
    expect(r.x.timeRange).toBe(xRange);
  });

  it("should reduce a bigint series to float32 off its first sample", () => {
    const first = TimeStamp.seconds(1_750_000_000).valueOf();
    const x = new Series({
      data: new BigInt64Array([first, first + 10n, first + 20n, first + 30n]),
      dataType: DataType.TIMESTAMP,
    });
    const y = new Series(new Float64Array([1, 5, 2, 0]));
    const r = reduceOne(op(x, y), "min_max", 4);
    expect(r.x.dataType).toEqual(DataType.FLOAT32);
    expect(r.y.dataType).toEqual(DataType.FLOAT32);
    expect(Array.from(r.x)).toEqual([first + 10n, first + 30n]);
    expect(Array.from(r.y)).toEqual([5, 0]);
  });

  it("should return ops of reduced series unchanged", () => {
    const reduced = new Series({
      data: new Float32Array([1, 2, 3, 4]),
      alignmentMultiple: 2n,
    });
    const o = op(reduced, reduced);
    expect(reduceOne(o, "min_max", 4)).toBe(o);
  });

  it("should return every op unchanged when a group holds under two samples", () => {
    const o = op(ramp(8), ramp(8));
    expect(reduceOne(o, "min_max", 1)).toBe(o);
  });

  describe("contiguous ops", () => {
    const XS = Array.from({ length: 20 }, (_, i) => i);
    const YS = XS.map((i) => (i * 7) % 11);

    // Returns the ops of XS and YS cut into series at the given sample indices. Each
    // series past the first holds its values off the given sample offsets.
    const cut = (at: number[], xOffset = 0, yOffset = 0): DrawOperation[] => {
      const ends = [...at, XS.length];
      return ends.map((end, i) => {
        const start = i === 0 ? 0 : ends[i - 1];
        const series = (values: number[], offset: number): Series => {
          const sampleOffset = i === 0 ? 0 : offset;
          return new Series({
            data: new Float32Array(
              values.slice(start, end).map((v) => v - sampleOffset),
            ),
            alignment: BigInt(start),
            sampleOffset,
          });
        };
        return op(series(XS, xOffset), series(YS, yOffset));
      });
    };
    const whole = (aggregation: Aggregation): DrawOperation =>
      reduceOne(cut([])[0], aggregation, 8);
    const values = (ops: DrawOperation[], key: "x" | "y"): number[] =>
      ops.flatMap((o) => Array.from(o[key], Number));

    it.each<Aggregation>(["min_max", "average", "decimate"])(
      "should reduce %s groups across ops as one run of samples",
      (aggregation) => {
        const parts = new Reducer().reduce(cut([10]), aggregation, 8);
        const w = whole(aggregation);
        expect(values(parts, "y")).toEqual(Array.from(w.y));
        expect(values(parts, "x")).toEqual(Array.from(w.x));
        expect(parts.map((o) => o.y.alignment)).toEqual([0n, 16n]);
      },
    );

    it("should reduce a group that spans three or more ops", () => {
      const parts = new Reducer().reduce(cut([3, 6, 9, 12, 15, 18]), "min_max", 8);
      expect(values(parts, "y")).toEqual(Array.from(whole("min_max").y));
      expect(parts.map((o) => o.y.alignment)).toEqual([0n, 8n, 16n]);
    });

    it.each<Aggregation>(["min_max", "average"])(
      "should move %s values onto the sample offsets of the op a group starts in",
      (aggregation) => {
        const parts = new Reducer().reduce(cut([6], 100, 50), aggregation, 8);
        expect(values(parts, "y")).toEqual(Array.from(whole(aggregation).y));
        expect(values(parts, "x")).toEqual(Array.from(whole(aggregation).x));
      },
    );

    it("should draw the group an op starts inside once the op before it leaves", () => {
      const [a, b] = cut([10]);
      const reducer = new Reducer();
      reducer.reduce([a, b], "min_max", 8);
      const r = reducer.reduce([b], "min_max", 8)[0];
      const alone = reduceOne(b, "min_max", 8);
      expect(Array.from(r.y)).toEqual(Array.from(alone.y));
      expect(r.y.alignment).toBe(8n);
    });

    it("should reduce a group again when the op that continues it leaves", () => {
      const [a, b] = cut([10]);
      const reducer = new Reducer();
      reducer.reduce([a, b], "min_max", 8);
      const r = reducer.reduce([a], "min_max", 8)[0];
      expect(Array.from(r.y)).toEqual(Array.from(reduceOne(a, "min_max", 8).y));
    });

    it("should leave out an op that draws no group", () => {
      const ops = new Reducer().reduce(cut([3, 6]).slice(0, 2), "min_max", 8);
      expect(ops).toHaveLength(1);
    });

    it("should give the span of an op that draws no group to the op before it", () => {
      const [a, b, c] = cut([3, 6]);
      const [first, last] = new Reducer().reduce(
        [spanned(a, 0, 3), spanned(b, 3, 6), spanned(c, 6, 20)],
        "min_max",
        8,
      );
      expect(first.span.end).toEqual(last.span.start);
    });

    it("should leave the span of an op it passes through unchanged", () => {
      const [a, b, c] = cut([3, 6]);
      const reduced = new Series({
        data: new Float32Array(2),
        alignment: 0n,
        alignmentMultiple: 8n,
      });
      const passed = spanned(op(reduced, reduced), 0, 3);
      new Reducer().reduce(
        [spanned(a, 0, 3), passed, spanned(b, 3, 6), spanned(c, 6, 20)],
        "min_max",
        8,
      );
      expect(passed.span.end).toEqual(TimeStamp.seconds(3));
    });

    it.each<Aggregation>(["min_max", "average", "decimate"])(
      "should keep %s groups apart across ops with time between them",
      (aggregation) => {
        const [a, b] = cut([10]);
        const before = spanned(a, 0, 10);
        const after = spanned(b, 20, 30);
        const parts = new Reducer().reduce([before, after], aggregation, 8);
        const alone = [before, after].flatMap((o) =>
          Array.from(reduceOne(o, aggregation, 8).y),
        );
        expect(values(parts, "y")).toEqual(alone);
      },
    );

    it("should keep a group complete once the ops after it hold its samples", () => {
      const [a] = cut([6]);
      const createGrowing = (): Series =>
        Series.alloc({ capacity: 64, dataType: DataType.FLOAT32, alignment: 6n });
      const x = createGrowing();
      const y = createGrowing();
      const write = (...indices: number[]): void => {
        x.write(new Series(new Float32Array(indices.map((i) => XS[i]))));
        y.write(new Series(new Float32Array(indices.map((i) => YS[i]))));
      };
      const reducer = new Reducer();
      write(6);
      const partial = reducer.reduce([a, op(x, y)], "min_max", 8)[0];
      write(7, 8);
      const complete = reducer.reduce([a, op(x, y)], "min_max", 8)[0];
      expect(complete).not.toBe(partial);
      expect(Array.from(complete.y)).toEqual(
        Array.from(whole("min_max").y).slice(0, 2),
      );
      write(9);
      expect(reducer.reduce([a, op(x, y)], "min_max", 8)[0].y).toBe(complete.y);
    });
  });

  describe("growth", () => {
    const createGrowing = (): Series =>
      Series.alloc({ capacity: 64, dataType: DataType.FLOAT32 });
    const write = (s: Series, ...values: number[]): void => {
      s.write(new Series(new Float32Array(values)));
    };

    it("should reduce only the groups completed since the last call", () => {
      const x = createGrowing();
      const y = createGrowing();
      write(x, 0, 1, 2, 3, 4, 5);
      write(y, 3, 1, 4, 1, 5, 9);
      const reducer = new Reducer();
      const first = reducer.reduce([op(x, y)], "min_max", 4)[0];
      expect(Array.from(first.y)).toEqual([1, 4, 5, 9]);
      // A complete group reduced again would pick up the change.
      (y.data as Float32Array)[0] = -100;
      write(x, 6, 7, 8, 9);
      write(y, 2, 6, 0, 8);
      const second = reducer.reduce([op(x, y)], "min_max", 4)[0];
      expect(Array.from(second.y)).toEqual([1, 4, 9, 2, 0, 8]);
      expect(Array.from(second.x)).toEqual([1, 2, 5, 6, 8, 9]);
      expect(first.y.refCount).toBe(0);
      expect(second.y.refCount).toBe(1);
    });

    it("should return the same op while the count holds", () => {
      const x = createGrowing();
      const y = createGrowing();
      write(x, 0, 1, 2);
      write(y, 0, 1, 2);
      const reducer = new Reducer();
      const first = reducer.reduce([op(x, y)], "min_max", 4)[0];
      expect(reducer.reduce([op(x, y)], "min_max", 4)[0].y).toBe(first.y);
    });
  });

  describe("release", () => {
    const refCounts = (o: DrawOperation): number[] => [o.x.refCount, o.y.refCount];

    it("should release the pairs of ops that leave the list", () => {
      const a = op(ramp(8), ramp(8));
      const b = op(ramp(8, 8n), ramp(8, 8n));
      const reducer = new Reducer();
      const [ra, rb] = reducer.reduce([a, b], "min_max", 4);
      expect(reducer.reduce([b], "min_max", 4)[0].y).toBe(rb.y);
      expect(refCounts(ra)).toEqual([0, 0]);
      expect(refCounts(rb)).toEqual([1, 1]);
    });

    it("should release every pair when the group size changes", () => {
      const o = op(ramp(16), ramp(16));
      const reducer = new Reducer();
      const four = reducer.reduce([o], "min_max", 4)[0];
      const eight = reducer.reduce([o], "min_max", 8)[0];
      expect(refCounts(four)).toEqual([0, 0]);
      expect(refCounts(eight)).toEqual([1, 1]);
      expect(reducer.reduce([o], "min_max", 1)[0]).toBe(o);
      expect(refCounts(eight)).toEqual([0, 0]);
    });

    it("should release every pair when the aggregation changes", () => {
      const o = op(ramp(16), ramp(16));
      const reducer = new Reducer();
      const minMax = reducer.reduce([o], "min_max", 4)[0];
      reducer.reduce([o], "decimate", 4);
      expect(refCounts(minMax)).toEqual([0, 0]);
    });

    it("should release every pair on release", () => {
      const reducer = new Reducer();
      const r = reducer.reduce([op(ramp(8), ramp(8))], "min_max", 4)[0];
      reducer.release();
      expect(refCounts(r)).toEqual([0, 0]);
    });
  });
});
