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
  type CrudeTimeRange,
  DataType,
  MultiSeries,
  scale,
  Series,
  TimeRange,
  TimeSpan,
} from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import {
  bridgeVertices,
  buildDrawOperations,
  type DrawOperation,
  nearestVertex,
} from "@/vis/line/aether/line";

describe("line", () => {
  describe("buildDrawOperations", () => {
    interface SpecEntry {
      timeRange: CrudeTimeRange;
      alignmentBounds: bounds.Bounds<bigint>;
      alignmentMultiple: bigint;
    }

    interface SpecExpected {
      xSeries: number;
      ySeries: number;
      xOffset: number;
      yOffset: number;
      count: number;
    }

    interface Spec {
      name: string;
      x: SpecEntry[];
      y: SpecEntry[];
      expected: SpecExpected[];
    }

    const buildSeriesFromEntries = (entries: SpecEntry[]): MultiSeries =>
      new MultiSeries(
        entries.map(
          ({ alignmentBounds, timeRange, alignmentMultiple }) =>
            new Series({
              data: new Float32Array(Number(bounds.span(alignmentBounds))),
              dataType: DataType.FLOAT32,
              timeRange: new TimeRange(timeRange.start, timeRange.end),
              alignment: alignmentBounds.lower,
              alignmentMultiple,
            }),
        ),
      );

    const CLEARLY_DISTINCT: Spec = {
      name: "clearly distinct",
      x: [
        {
          timeRange: { start: 0, end: 100 },
          alignmentBounds: { lower: 0n, upper: 100n },
          alignmentMultiple: 1n,
        },
        {
          timeRange: { start: 100, end: 200 },
          alignmentBounds: { lower: 100n, upper: 200n },
          alignmentMultiple: 1n,
        },
      ],
      y: [
        {
          timeRange: { start: 0, end: 100 },
          alignmentBounds: { lower: 0n, upper: 100n },
          alignmentMultiple: 1n,
        },
        {
          timeRange: { start: 100, end: 200 },
          alignmentBounds: { lower: 100n, upper: 200n },
          alignmentMultiple: 1n,
        },
      ],
      expected: [
        { xSeries: 0, ySeries: 0, xOffset: 0, yOffset: 0, count: 100 },
        { xSeries: 1, ySeries: 1, xOffset: 0, yOffset: 0, count: 100 },
      ],
    };

    const COMPLETE_OVERLAP_ON_X: Spec = {
      name: "complete overlap on x",
      x: [
        {
          timeRange: { start: 0, end: 100 },
          alignmentBounds: { lower: 0n, upper: 100n },
          alignmentMultiple: 1n,
        },
      ],
      y: [
        {
          timeRange: { start: 0, end: 50 },
          alignmentBounds: { lower: 0n, upper: 50n },
          alignmentMultiple: 1n,
        },
        {
          timeRange: { start: 50, end: 100 },
          alignmentBounds: { lower: 50n, upper: 100n },
          alignmentMultiple: 1n,
        },
      ],
      expected: [
        { ySeries: 0, xSeries: 0, xOffset: 0, yOffset: 0, count: 50 },
        { ySeries: 1, xSeries: 0, xOffset: 50, yOffset: 0, count: 50 },
      ],
    };

    const PARTIAL_OVERLAP_ON_Y_AFTER: Spec = {
      name: "partial overlap on x",
      x: [
        {
          timeRange: { start: 0, end: 100 },
          alignmentBounds: { lower: 0n, upper: 100n },
          alignmentMultiple: 1n,
        },
      ],
      y: [
        {
          timeRange: { start: 50, end: 150 },
          alignmentBounds: { lower: 50n, upper: 150n },
          alignmentMultiple: 1n,
        },
      ],
      expected: [{ xSeries: 0, ySeries: 0, xOffset: 50, yOffset: 0, count: 50 }],
    };

    const Y_COMPLETELY_CONTAINS_X: Spec = {
      name: "y completely contains x",
      x: [
        {
          timeRange: { start: 50, end: 100 },
          alignmentBounds: { lower: 50n, upper: 100n },
          alignmentMultiple: 1n,
        },
      ],
      y: [
        {
          timeRange: { start: 0, end: 200 },
          alignmentBounds: { lower: 0n, upper: 200n },
          alignmentMultiple: 1n,
        },
      ],
      expected: [{ xSeries: 0, ySeries: 0, xOffset: 0, yOffset: 50, count: 50 }],
    };

    const X_COMPLETELY_CONTAINS_Y: Spec = {
      name: "x completely contains y",
      x: [
        {
          timeRange: { start: 0, end: 200 },
          alignmentBounds: { lower: 0n, upper: 200n },
          alignmentMultiple: 1n,
        },
      ],
      y: [
        {
          timeRange: { start: 50, end: 100 },
          alignmentBounds: { lower: 50n, upper: 100n },
          alignmentMultiple: 1n,
        },
      ],
      expected: [{ xSeries: 0, ySeries: 0, xOffset: 50, yOffset: 0, count: 50 }],
    };

    const MULTIPLE_PARTIAL_OVERLAPS: Spec = {
      name: "multiple partial overlaps",
      x: [
        {
          timeRange: { start: 0, end: 100 },
          alignmentBounds: { lower: 0n, upper: 100n },
          alignmentMultiple: 1n,
        },
        {
          timeRange: { start: 100, end: 150 },
          alignmentBounds: { lower: 100n, upper: 150n },
          alignmentMultiple: 1n,
        },
      ],
      y: [
        {
          timeRange: { start: 25, end: 75 },
          alignmentBounds: { lower: 25n, upper: 75n },
          alignmentMultiple: 1n,
        },
        {
          timeRange: { start: 75, end: 125 },
          alignmentBounds: { lower: 75n, upper: 125n },
          alignmentMultiple: 1n,
        },
        {
          timeRange: { start: 125, end: 175 },
          alignmentBounds: { lower: 125n, upper: 175n },
          alignmentMultiple: 1n,
        },
      ],
      expected: [
        { xSeries: 0, ySeries: 0, xOffset: 25, yOffset: 0, count: 50 },
        { xSeries: 0, ySeries: 1, xOffset: 75, yOffset: 0, count: 25 },
        { xSeries: 1, ySeries: 1, xOffset: 0, yOffset: 25, count: 25 },
        { xSeries: 1, ySeries: 2, xOffset: 25, yOffset: 0, count: 25 },
      ],
    };

    const ALIGN_OVERLAP_TIME_RANGE_NO_OVERLAP: Spec = {
      name: "align overlap time range no overlap",
      x: [
        {
          timeRange: { start: 0, end: 100 },
          alignmentBounds: { lower: 0n, upper: 100n },
          alignmentMultiple: 1n,
        },
      ],
      y: [
        {
          timeRange: { start: 100, end: 150 },
          alignmentBounds: { lower: 50n, upper: 150n },
          alignmentMultiple: 1n,
        },
      ],
      expected: [],
    };

    const ALIGN_MULTIPLE_GREATER__THAN_1_PERFECT_ALIGNMENT: Spec = {
      name: "positive align multiple",
      x: [
        {
          timeRange: { start: 0, end: 100 },
          alignmentBounds: { lower: 0n, upper: 60n },
          alignmentMultiple: 3n,
        },
      ],
      y: [
        {
          timeRange: { start: 0, end: 100 },
          alignmentBounds: { lower: 0n, upper: 60n },
          alignmentMultiple: 3n,
        },
      ],
      expected: [{ xSeries: 0, ySeries: 0, xOffset: 0, yOffset: 0, count: 60 }],
    };

    const ALIGNMENT_MULTIPLE_4_MISALIGNMENT: Spec = {
      name: "align multiple 4 misalignment",
      x: [
        {
          timeRange: { start: 0, end: 100 },
          alignmentBounds: { lower: 20n, upper: 80n },
          alignmentMultiple: 4n,
        },
      ],
      y: [
        {
          timeRange: { start: 0, end: 100 },
          alignmentBounds: { lower: 0n, upper: 80n },
          alignmentMultiple: 4n,
        },
      ],
      expected: [{ xSeries: 0, ySeries: 0, xOffset: 0, yOffset: 5, count: 60 }],
    };

    const ALIGN_MULTIPLE_LESS_THAN_1_MISALIGNMENT_BAD_INTERVAL: Spec = {
      name: "align multiple less than 1 misalignment bad interval",
      x: [
        {
          timeRange: { start: 0, end: 100 },
          alignmentBounds: { lower: 13n, upper: 80n },
          alignmentMultiple: 7n,
        },
      ],
      y: [
        {
          timeRange: { start: 0, end: 100 },
          alignmentBounds: { lower: 0n, upper: 80n },
          alignmentMultiple: 7n,
        },
      ],
      expected: [
        {
          xSeries: 0,
          ySeries: 0,
          xOffset: 0,
          yOffset: 1,
          count: 67,
        },
      ],
    };

    const REGRESSION_1: Spec = {
      name: "no alignment no overlap",
      x: [
        {
          timeRange: {
            start: 1756835746434000000n,
            end: 1756835776214375534n,
          },
          alignmentBounds: {
            lower: 81604381164n,
            upper: 81604381314n,
          },
          alignmentMultiple: 3n,
        },
      ],
      y: [
        {
          timeRange: {
            start: 1756835767973000000n,
            end: 1756835797172750864n,
          },
          alignmentBounds: {
            lower: 81604381272n,
            upper: 81604381419n,
          },
          alignmentMultiple: 3n,
        },
      ],
      expected: [{ count: 114, xSeries: 0, ySeries: 0, xOffset: 36, yOffset: 0 }],
    };

    const SPECS: Spec[] = [
      CLEARLY_DISTINCT,
      COMPLETE_OVERLAP_ON_X,
      PARTIAL_OVERLAP_ON_Y_AFTER,
      Y_COMPLETELY_CONTAINS_X,
      X_COMPLETELY_CONTAINS_Y,
      MULTIPLE_PARTIAL_OVERLAPS,
      ALIGN_OVERLAP_TIME_RANGE_NO_OVERLAP,
      ALIGN_MULTIPLE_GREATER__THAN_1_PERFECT_ALIGNMENT,
      ALIGNMENT_MULTIPLE_4_MISALIGNMENT,
      ALIGN_MULTIPLE_LESS_THAN_1_MISALIGNMENT_BAD_INTERVAL,
      REGRESSION_1,
    ];

    SPECS.forEach(({ name, x, y, expected }) => {
      it(`spec ${name}`, () => {
        const xSeries = buildSeriesFromEntries(x);
        const ySeries = buildSeriesFromEntries(y);
        const drawOperations = buildDrawOperations(
          xSeries,
          ySeries,
          1,
          0,
          "decimate",
          TimeSpan.ZERO,
        );
        expect(drawOperations.length).toBe(expected.length);
        drawOperations.forEach((drawOperation: DrawOperation, i: number) => {
          expect(drawOperation.x).toBe(xSeries.series[expected[i].xSeries]);
          expect(drawOperation.y).toBe(ySeries.series[expected[i].ySeries]);
          expect(drawOperation.xOffset).toBe(expected[i].xOffset);
          expect(drawOperation.yOffset).toBe(expected[i].yOffset);
          expect(drawOperation.count).toBe(expected[i].count);
        });
      });
    });
  });

  describe("bridgeVertices", () => {
    const op = (
      x: Series,
      y: Series,
      count: number,
      offset = 0,
      downsample = 1,
    ): DrawOperation => ({ x, y, xOffset: offset, yOffset: offset, count, downsample });
    const f32 = (...data: number[]): Series => new Series(new Float32Array(data));
    const shifted = (sampleOffset: bigint, ...data: number[]): Series =>
      new Series({ data: new Float32Array(data), sampleOffset });
    const vertices = (ops: DrawOperation[]): number[] =>
      Array.from(bridgeVertices(ops, scale.XY.IDENTITY));

    it("should return no vertices for a single op", () => {
      expect(vertices([op(f32(0, 1), f32(2, 3), 2)])).toEqual([]);
    });

    it("should join the last vertex of an op to the first vertex of the next", () => {
      const a = op(f32(0, 1), f32(2, 3), 2);
      const b = op(f32(5, 6), f32(7, 4), 2);
      expect(vertices([a, b])).toEqual([1, 3, 5, 7]);
    });

    it("should apply the x sample offset of each op", () => {
      const a = op(shifted(100n, 0, 1, 2), f32(7, 8, 9), 3);
      const b = op(shifted(110n, 5, 6), f32(3, 4), 2);
      expect(vertices([a, b])).toEqual([102, 9, 115, 3]);
    });

    it("should apply the y sample offset of each op", () => {
      const a = op(f32(0, 1), shifted(50n, 0, 4), 2);
      const b = op(f32(2, 3), shifted(60n, 0, 1), 2);
      expect(vertices([a, b])).toEqual([1, 54, 2, 60]);
    });

    it("should bridge through an op with one sample", () => {
      const a = op(f32(0, 1), f32(2, 3), 2);
      const lone = op(f32(5), f32(7), 1);
      const c = op(f32(8, 9), f32(4, 6), 2);
      expect(vertices([a, lone, c])).toEqual([1, 3, 5, 7, 5, 7, 8, 4]);
    });

    it("should chain bridges through many consecutive ops with one sample each", () => {
      const ys = [2, 9, 4, 7, 1, 8];
      const ops = ys.map((y, x) => op(f32(x), f32(y), 1));
      const expected = ys.slice(1).flatMap((y, i) => [i, ys[i], i + 1, y]);
      expect(vertices(ops)).toEqual(expected);
    });

    describe("decimated ops", () => {
      const ax = f32(0, 1, 2, 3, 4, 5, 6);
      const ay = f32(10, 11, 12, 13, 14, 15, 16);
      const b = op(f32(9), f32(1), 1);

      it("should start at the last vertex the strip draws", () => {
        expect(vertices([op(ax, ay, 7, 0, 3), b])).toEqual([3, 13, 9, 1]);
      });

      it("should start at the first sample of an op too short to draw", () => {
        expect(vertices([op(ax, ay, 5, 0, 3), b])).toEqual([0, 10, 9, 1]);
      });
    });

    it("should respect the op offsets and a uint8 y", () => {
      const a = op(f32(0, 1, 2, 3), new Series(new Uint8Array([0, 1, 1, 0])), 2, 1);
      const b = op(f32(4, 5, 6), new Series(new Uint8Array([1, 0, 0])), 2, 1);
      expect(vertices([a, b])).toEqual([2, 1, 5, 0]);
    });

    it("should map each vertex through the scale", () => {
      const s = new scale.XY(
        scale.Scale.scale<number>(100, 200).scale(1),
        scale.Scale.scale<number>(0, 10).scale(1),
      );
      const a = op(f32(100, 150), f32(0, 5), 2);
      const b = op(f32(175), f32(10), 1);
      const v = bridgeVertices([a, b], s);
      [0.5, 0.5, 0.75, 1].forEach((e, i) => expect(v[i]).toBeCloseTo(e));
    });

    it("should keep a vertex in view precise when the next op is a day away", () => {
      const start = 1_760_000_000_000_000_000n;
      const day = 86_400_000_000_000n;
      const a = op(shifted(start, 0, 1e6, 2e6), f32(1, 2, 3), 3);
      const b = op(shifted(start + day, 0), f32(7), 1);
      const view = 100e6;
      const lower = Number(start) + 2e6 - view / 2;
      const s = new scale.XY(scale.Scale.scale<number>(lower, lower + view).scale(1));
      const [x] = bridgeVertices([a, b], s);
      expect(x).toBeCloseTo(0.5, 5);
    });
  });

  describe("nearestVertex", () => {
    const EMPTY = new Series(new Float32Array());
    const op = (count: number, downsample: number, xOffset = 0): DrawOperation => ({
      x: EMPTY,
      y: EMPTY,
      xOffset,
      yOffset: 0,
      count,
      downsample,
    });

    it("should return the index itself when nothing is decimated", () => {
      expect(nearestVertex(op(30, 1), 17)).toBe(17);
    });

    it("should round to the nearest drawn vertex", () => {
      expect(nearestVertex(op(30, 4), 1)).toBe(0);
      expect(nearestVertex(op(30, 4), 3)).toBe(4);
      expect(nearestVertex(op(30, 4), 14)).toBe(16);
    });

    it("should not pass the last vertex the strip draws", () => {
      expect(nearestVertex(op(30, 4), 29)).toBe(24);
    });

    it("should measure from the op's x offset", () => {
      expect(nearestVertex(op(30, 4, 2), 5)).toBe(6);
      expect(nearestVertex(op(30, 4, 2), 3)).toBe(2);
    });

    it("should return the first sample of an op too short to draw", () => {
      expect(nearestVertex(op(6, 4), 5)).toBe(0);
    });
  });
});
