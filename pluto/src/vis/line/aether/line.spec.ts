// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import {
  bounds,
  box,
  color,
  type CrudeTimeRange,
  DataType,
  id,
  MultiSeries,
  scale,
  Series,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { afterAll, describe, expect, it, vi } from "vitest";

import { telem } from "@/telem/aether";
import { telemTest } from "@/telem/aether/test";
import { renderAether } from "@/testutil/renderAether";
import {
  bridgeVertices,
  buildDrawOperations,
  DEFAULT_OVERLAP_THRESHOLD,
  type DrawOperation,
  Line,
  type LineProps,
  type State,
} from "@/vis/line/aether/line";
import { lineTest } from "@/vis/line/aether/test";
import { canvasTest } from "@/vis/render/test";

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

    const DIFFERENT_ALIGNMENT_MULTIPLES: Spec = {
      name: "different alignment multiples",
      x: [
        {
          timeRange: { start: 0, end: 100 },
          alignmentBounds: { lower: 0n, upper: 64n },
          alignmentMultiple: 8n,
        },
      ],
      y: [
        {
          timeRange: { start: 0, end: 100 },
          alignmentBounds: { lower: 0n, upper: 8n },
          alignmentMultiple: 64n,
        },
      ],
      expected: [],
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
      DIFFERENT_ALIGNMENT_MULTIPLES,
      REGRESSION_1,
    ];

    SPECS.forEach(({ name, x, y, expected }) => {
      it(`spec ${name}`, () => {
        const xSeries = buildSeriesFromEntries(x);
        const ySeries = buildSeriesFromEntries(y);
        const drawOperations = buildDrawOperations(xSeries, ySeries, TimeSpan.ZERO);
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
    const op = (x: Series, y: Series, count: number, offset = 0): DrawOperation => ({
      x,
      y,
      xOffset: offset,
      yOffset: offset,
      count,
    });
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

    it("should respect the op offsets and a uint8 y", () => {
      const a = op(f32(0, 1, 2, 3), new Series(new Uint8Array([0, 1, 1, 0])), 2, 1);
      const b = op(f32(4, 5, 6), new Series(new Uint8Array([1, 0, 0])), 2, 1);
      expect(vertices([a, b])).toEqual([2, 1, 5, 0]);
    });

    it("should leave out a bridge with a non-finite end", () => {
      const a = op(f32(0, 1), f32(2, NaN), 2);
      const b = op(f32(5, 6), f32(7, 4), 2);
      const c = op(f32(8), f32(Infinity), 1);
      const d = op(f32(9, 10), f32(3, 2), 2);
      expect(vertices([a, b, c, d])).toEqual([]);
      expect(vertices([b, d])).toEqual([6, 4, 9, 3]);
    });

    describe("time order", () => {
      const timed = (start: number, end: number, ...data: number[]): Series =>
        new Series({
          data: new Float32Array(data),
          timeRange: new TimeRange(TimeStamp.seconds(start), TimeStamp.seconds(end)),
        });

      it("should join ops in time order whatever their array order", () => {
        const a = op(timed(0, 10, 0, 1), timed(0, 10, 2, 3), 2);
        const b = op(timed(20, 30, 5, 6), timed(20, 30, 7, 4), 2);
        expect(vertices([b, a])).toEqual([1, 3, 5, 7]);
      });

      it("should not join two ops that hold overlapping spans", () => {
        const whole = op(timed(0, 100, 0, 50, 99), timed(0, 100, 1, 2, 3), 3);
        const copy = op(timed(48, 100, 48, 99), timed(48, 100, 5, 6), 2);
        expect(vertices([whole, copy])).toEqual([]);
      });

      it("should join an x series split across two y series", () => {
        const x = timed(0, 20, 0, 1, 2, 3);
        const first = op(x, timed(0, 10, 4, 5), 2);
        const second: DrawOperation = { ...op(x, timed(10, 20, 6, 7), 2), xOffset: 2 };
        expect(vertices([second, first])).toEqual([1, 5, 2, 6]);
      });
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

  describe("Line", () => {
    const series = (
      data: number[],
      alignment: bigint,
      timeRange: CrudeTimeRange,
    ): Series =>
      new Series({
        data: new Float32Array(data),
        dataType: DataType.FLOAT32,
        alignment,
        timeRange: new TimeRange(timeRange.start, timeRange.end),
      });

    const createSource = (data: Series[]) =>
      telemTest.source<[bounds.Bounds, MultiSeries]>([
        bounds.max(data.map((s) => s.bounds)),
        new MultiSeries(data),
      ]);

    const sourceSpec = (
      source: telemTest.TestSource<[bounds.Bounds, MultiSeries]>,
    ): telem.SeriesSourceSpec => ({
      type: telemTest.TestSource.TYPE,
      props: { testId: source.id },
      variant: "source",
      valueType: "series",
    });

    const PROPS: LineProps = {
      region: box.construct(0, 0, 800, 600),
      dataToDecimalScale: scale.XY.IDENTITY,
    };

    const mount = (x: Series[], y: Series[], state: Partial<State> = {}) => {
      const { gl, calls } = canvasTest.createGL();
      const xSource = createSource(x);
      const ySource = createSource(y);
      const recorder = canvasTest.record();
      (recorder as { gl: unknown }).gl = gl;
      recorder.resize(PROPS.region, 1);
      const h = renderAether(lineTest.Host, {
        state: {},
        render: recorder,
        registry: { [Line.TYPE]: Line },
        children: {
          line: {
            type: Line.TYPE,
            state: {
              x: sourceSpec(xSource),
              y: sourceSpec(ySource),
              color: color.construct("#ff0000"),
              ...state,
            },
          },
        },
      });
      const draws = (): [number, number][] =>
        calls
          .filter((c) => c.op === "drawArraysInstanced")
          .map((c) => [Number(c.args[0]), Number(c.args[2])]);
      const strips = (): [number, number][] =>
        calls
          .filter(
            (c) =>
              c.op === "drawArraysInstanced" &&
              c.args[0] === canvasTest.GL_ENUMS.LINE_STRIP,
          )
          .map((c) => [Number(c.args[1]), Number(c.args[2])]);
      const remove = () => h.deleteChild("line");
      return {
        line: h.child<Line>("line"),
        recorder,
        calls,
        draws,
        strips,
        remove,
        xSource,
        ySource,
      };
    };

    const X1 = series([0, 1, 2], 0n, { start: 0, end: 100 });
    const Y1 = series([10, 20, 30], 0n, { start: 0, end: 100 });
    const X2 = series([5, 6], 10n, { start: 200, end: 300 });
    const Y2 = series([40, 50], 10n, { start: 200, end: 300 });

    it("should pass the x range the region shows to its sources", () => {
      const m = mount([X1], [Y1]);
      const values = [vi.spyOn(m.xSource, "value"), vi.spyOn(m.ySource, "value")];
      const x = scale.Scale.scale<number>(100, 300).scale(0, 1);
      const props: LineProps = { ...PROPS, dataToDecimalScale: new scale.XY(x) };
      m.line.render(props);
      m.line.findByXValue(props, 1);
      const view = { view: { bounds: { lower: 100, upper: 300 }, width: 800 } };
      values.forEach((value) => expect(value.mock.calls).toEqual([[view], [view]]));
    });

    it("should read its bounds with the extremes of every sample kept", () => {
      const m = mount([X1], [Y1]);
      const values = [vi.spyOn(m.xSource, "value"), vi.spyOn(m.ySource, "value")];
      expect(m.line.yBounds({ lower: 0, upper: 2 })).toEqual({ lower: 10, upper: 30 });
      values.forEach((value) =>
        expect(value.mock.calls).toEqual([[{ extremesKept: true }]]),
      );
    });

    describe("render", () => {
      it("should draw one strip per domain and no bridge for a single domain", () => {
        const m = mount([X1], [Y1]);
        m.line.render(PROPS);
        expect(m.draws()).toEqual([[canvasTest.GL_ENUMS.LINE_STRIP, 3]]);
      });

      it("should draw a bridge between the strips of two domains", () => {
        const m = mount([X1, X2], [Y1, Y2]);
        m.line.render(PROPS);
        expect(m.draws()).toEqual([
          [canvasTest.GL_ENUMS.LINE_STRIP, 3],
          [canvasTest.GL_ENUMS.LINE_STRIP, 2],
          [canvasTest.GL_ENUMS.LINES, 2],
        ]);
        const upload = m.calls.filter((c) => c.op === "bufferData").at(-1);
        expect(Array.from(upload?.args[1] as Float32Array)).toEqual([2, 30, 5, 40]);
      });

      describe("non-finite samples", () => {
        const RANGE = { start: 0, end: 100 };
        const strips = (x: number[], y: number[], yAlignment = 0n) => {
          const m = mount([series(x, 0n, RANGE)], [series(y, yAlignment, RANGE)]);
          m.line.render(PROPS);
          return m.strips();
        };

        it("should leave a gap at a NaN y sample", () => {
          expect(strips([0, 1, 2, 3, 4], [10, 20, NaN, 40, 50])).toEqual([
            [0, 2],
            [3, 2],
          ]);
        });

        it("should leave a gap at an infinite x or y sample", () => {
          expect(
            strips([0, 1, 2, 3, Infinity, 5, 6], [1, 2, -Infinity, 4, 5, 6, 7]),
          ).toEqual([
            [0, 2],
            [5, 2],
          ]);
        });

        it("should draw nothing for a finite sample between two NaN samples", () => {
          expect(strips([0, 1, 2, 3], [NaN, 1, NaN, 2])).toEqual([]);
        });

        it("should draw nothing when every sample is NaN", () => {
          expect(strips([0, 1, 2], [NaN, NaN, NaN])).toEqual([]);
        });

        it("should find a non-finite x sample through the x offset", () => {
          // x starts one sample before y, so the strip begins at x[1].
          expect(strips([0, 1, 2, 3, Infinity, 5], [10, 11, 12, 13, 14], 1n)).toEqual([
            [0, 3],
          ]);
        });
      });

      it("should upload its series and stroke buffer again after a restore", () => {
        const RANGE = { start: 0, end: 100 };
        const m = mount([series([0, 1, 2], 0n, RANGE)], [series([1, 2, 3], 0n, RANGE)]);
        const uploads = (from: number): string[] =>
          m.calls
            .slice(from)
            .map((c) => c.op)
            .filter((op) => op === "createBuffer" || op === "bufferData");
        m.line.render(PROPS);
        const restored = m.calls.length;
        m.recorder.restore();
        m.line.render(PROPS);
        expect(uploads(restored)).toEqual([
          "createBuffer",
          "bufferData",
          "createBuffer",
          "bufferData",
          "createBuffer",
          "bufferData",
        ]);
      });

      it("should not bridge two domains across an infinite sample", () => {
        const y1 = series([10, 20, Infinity], 0n, { start: 0, end: 100 });
        const m = mount([X1, X2], [y1, Y2]);
        m.line.render(PROPS);
        expect(m.draws()).toEqual([
          [canvasTest.GL_ENUMS.LINE_STRIP, 2],
          [canvasTest.GL_ENUMS.LINE_STRIP, 2],
        ]);
      });
    });

    describe("reduction", () => {
      const N = 8000;
      const RANGE = { start: 0, end: 100 };
      const wave = (i: number): number => Math.sin(i / 10);
      const ramp = (length: number, alignment = 0n): Series =>
        series(
          Array.from({ length }, (_, i) => i),
          alignment,
          RANGE,
        );
      const waves = (length: number, alignment = 0n): Series =>
        series(
          Array.from({ length }, (_, i) => wave(i)),
          alignment,
          RANGE,
        );
      // The region shows x from lower to upper across 800 pixels.
      const showing = (lower: number, upper: number): LineProps => ({
        ...PROPS,
        dataToDecimalScale: new scale.XY(
          scale.Scale.scale<number>(lower, upper).scale(0, 1),
        ),
      });
      const uploads = (m: ReturnType<typeof mount>): number[][] =>
        m.calls
          .filter((c) => c.op === "bufferData" && c.args[1] instanceof ArrayBuffer)
          .map((c) => Array.from(new Float32Array(c.args[1] as ArrayBuffer)));
      const deletes = (m: ReturnType<typeof mount>): number =>
        m.calls.filter((c) => c.op === "deleteBuffer").length;

      it("should draw a dense window at the group size of its pixel density", () => {
        const m = mount([ramp(N)], [waves(N)]);
        m.line.render(showing(0, N - 1));
        // 10 samples per pixel at 2 pixels per group rounds up to groups of 32.
        expect(m.strips()).toEqual([[0, 2 * (N / 32)]]);
        expect(uploads(m).every((u) => u.length < N)).toBe(true);
      });

      it("should draw full resolution when a group holds under two samples", () => {
        const m = mount([ramp(N)], [waves(N)]);
        m.line.render(showing(0, N - 1));
        m.line.render(showing(0, 100));
        expect(m.strips().at(-1)).toEqual([0, N]);
        expect(deletes(m)).toBe(2);
      });

      it("should draw ops of reduced series unchanged", () => {
        const reduced = (data: number[]) =>
          new Series({
            data: new Float32Array(data),
            dataType: DataType.FLOAT32,
            alignment: 0n,
            alignmentMultiple: 2n,
            timeRange: new TimeRange(0, 50),
          });
        const tiles = Array.from({ length: N / 2 }, (_, i) => -2 * i);
        const raw = (s: Series) =>
          new Series({
            data: s.data,
            dataType: DataType.FLOAT32,
            alignment: 1n << 32n,
            timeRange: new TimeRange(50, 100),
          });
        const m = mount(
          [reduced(tiles), raw(ramp(N))],
          [reduced(tiles), raw(waves(N))],
        );
        m.line.render(showing(-N, N - 1));
        expect(m.strips()).toEqual([
          [0, N / 2],
          [0, 2 * (N / 64)],
        ]);
      });

      it("should follow a growing buffer and not reduce complete groups again", () => {
        const createGrowing = () =>
          Series.alloc({
            capacity: N,
            dataType: DataType.FLOAT32,
            timeRange: new TimeRange(0, 100),
          });
        const write = (s: Series, values: number[]) =>
          s.write(new Series(new Float32Array(values)));
        const x = createGrowing();
        const y = createGrowing();
        const indexes = (from: number, to: number) =>
          Array.from({ length: to - from }, (_, i) => from + i);
        write(x, indexes(0, N / 2));
        write(y, indexes(0, N / 2).map(wave));
        const m = mount([x], [y]);
        m.line.render(showing(0, N - 1));
        expect(m.strips()).toEqual([[0, N / 32]]);
        // A complete group reduced again would draw the change.
        (y.data as Float32Array)[0] = 1000;
        write(x, indexes(N / 2, N / 2 + 10));
        const tail = indexes(N / 2, N / 2 + 9).map(wave);
        write(y, [...tail, -5]);
        m.line.render(showing(0, N - 1));
        expect(m.strips().at(-1)).toEqual([0, N / 32 + 2]);
        const drawn = uploads(m).at(-1);
        expect(drawn?.slice(0, 2)).toEqual([0, Math.fround(wave(16))]);
        expect(drawn?.slice(-2)).toEqual([Math.fround(Math.max(...tail)), -5]);
      });

      it("should release its reduced series when deleted", () => {
        const m = mount([ramp(N)], [waves(N)]);
        m.line.render(showing(0, N - 1));
        expect(deletes(m)).toBe(0);
        m.remove();
        expect(deletes(m)).toBe(2);
      });

      it("should reduce by its y source's aggregation and groups per column", () => {
        const decimated = mount([ramp(N)], [waves(N)]);
        decimated.ySource.aggregation = () => "decimate";
        decimated.line.render(showing(0, N - 1));
        expect(decimated.strips()).toEqual([[0, N / 32]]);
        const fine = mount([ramp(N)], [waves(N)], { groupsPerColumn: 1 });
        fine.line.render(showing(0, N - 1));
        expect(fine.strips()).toEqual([[0, 2 * (N / 16)]]);
      });

      it("should average by the data types its sources' channels store", () => {
        const bits = series(
          Array.from({ length: N }, (_, i) => i % 2),
          0n,
          RANGE,
        );
        const m = mount([ramp(N)], [bits]);
        m.xSource.dataType = () => DataType.TIMESTAMP;
        m.ySource.dataType = () => DataType.INT16;
        m.ySource.aggregation = () => "average";
        m.line.render(showing(0, N - 1));
        const [xs, ys] = uploads(m).slice(-2);
        expect(xs.slice(0, 2)).toEqual([15, 47]);
        expect(new Set(ys)).toEqual(new Set([0]));
      });
    });

    describe("findByXValue", () => {
      const X = series(
        Array.from({ length: 30 }, (_, i) => i),
        0n,
        { start: 0, end: 100 },
      );
      const Y = series(
        Array.from({ length: 30 }, (_, i) => i * 10),
        0n,
        { start: 0, end: 100 },
      );

      it("should return the sample under the target", () => {
        const m = mount([X], [Y]);
        expect(m.line.findByXValue(PROPS, 3).value).toEqual({ x: 3, y: 30 });
      });

      it("should return a NaN sample under the target", () => {
        const y = series([0, NaN, 20], 0n, { start: 0, end: 100 });
        const m = mount([X1], [y]);
        const { value } = m.line.findByXValue(PROPS, 1);
        expect(value.x).toBe(1);
        expect(value.y).toBeNaN();
      });

      it("should return a NaN x when no y sample shares the target's alignment", () => {
        const y = series([0, 10, 20], 5n, { start: 0, end: 100 });
        const m = mount([X1], [y]);
        expect(m.line.findByXValue(PROPS, 1).value.x).toBeNaN();
      });

      it("should return a NaN y for a y series of another alignment multiple", () => {
        const y = new Series({
          data: new Float32Array([10, 20]),
          alignment: 0n,
          alignmentMultiple: 2n,
          timeRange: new TimeRange(0, 100),
        });
        const m = mount([X1], [y]);
        expect(m.line.findByXValue(PROPS, 1).value.y).toBeNaN();
      });
    });
  });

  describe("TiledChannelData against a Core", () => {
    const synnax = createTestClient();
    const feed = synnax.openFeed();
    const client: telem.Client = {
      feed,
      channels: synnax.channels,
      now: () => TimeStamp.now(),
    };
    afterAll(async () => await feed.close());

    const COUNT = 10_000;
    const SPIKE = 10;

    // Writes COUNT samples one millisecond apart. Data sample i holds sin(i / 100), or
    // SPIKE at spikeAt.
    const writeChannels = async (spikeAt?: number) => {
      const time = await synnax.channels.create({
        name: id.create(),
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });
      const data = await synnax.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT32,
        index: time.key,
      });
      const start = TimeStamp.seconds(1_000_000);
      await synnax.write(start, {
        [time.key]: Array.from({ length: COUNT }, (_, i) =>
          start.add(TimeSpan.milliseconds(i)),
        ),
        [data.key]: Array.from({ length: COUNT }, (_, i) =>
          i === spikeAt ? SPIKE : Math.sin(i / 100),
        ),
      });
      const home = new TimeRange(start, start.add(TimeSpan.milliseconds(COUNT)));
      return { data, home };
    };

    const create = (props: Partial<telem.TiledChannelDataProps>) =>
      new telem.TiledChannelData(client, props, undefined, undefined, TimeSpan.ZERO);

    const createPair = (dataKey: channel.Key, home: TimeRange) => ({
      x: create({ channel: dataKey, useIndexOfChannel: true, timeRange: home }),
      y: create({ channel: dataKey, timeRange: home }),
    });

    const settle = async (...sources: telem.TiledChannelData[]): Promise<void> =>
      await expect.poll(() => sources.every((s) => !s.loading())).toBe(true);

    const drawn = (x: MultiSeries, y: MultiSeries) =>
      buildDrawOperations(x, y, DEFAULT_OVERLAP_THRESHOLD);

    it("should pair every reduced x sample with a y sample", async () => {
      const { data, home } = await writeChannels();
      const { x, y } = createPair(data.key, home);
      await settle(x, y);
      const [xBounds, xs] = x.value();
      const [, ys] = y.value();
      expect(xBounds).toEqual(home.numericBounds);
      expect(xs.length).toBe(ys.length);
      expect(xs.length).toBeLessThan(COUNT);
      expect(xs.series.map((s) => s.alignment)).toEqual(
        ys.series.map((s) => s.alignment),
      );
      expect(ys.series.every((s) => s.alignmentMultiple > 1n)).toBe(true);
      const ops = drawn(xs, ys);
      expect(ops.reduce((n, op) => n + op.count, 0)).toBe(ys.length);
      expect(y.value()[0].upper).toBeCloseTo(1, 3);
      expect(y.value()[0].lower).toBeCloseTo(-1, 3);
      x.cleanup();
      y.cleanup();
    });

    it("should keep a spike that an average drops inside the line's bounds", async () => {
      const { data, home } = await writeChannels(COUNT / 2);
      const y = create({ channel: data.key, timeRange: home, aggregation: "average" });
      await settle(y);
      expect(y.value()[0].upper).toBeLessThan(SPIKE / 2);
      await expect.poll(() => y.value({ extremesKept: true })[0].upper).toBe(SPIKE);
      y.cleanup();
    });

    it("should draw finer tiles for a zoomed view", async () => {
      const { data, home } = await writeChannels();
      const { x, y } = createPair(data.key, home);
      await settle(x, y);
      const homePoints = y.value()[1].length;
      const start = Number(home.start.valueOf());
      const v: telem.View = {
        bounds: {
          lower: start,
          upper: start + Number(TimeSpan.milliseconds(500).valueOf()),
        },
        width: 1000,
      };
      x.value({ view: v });
      y.value({ view: v });
      await expect
        .poll(
          () => !x.fetching() && !y.fetching() && y.value({ view: v })[1].length > 0,
        )
        .toBe(true);
      const xs = x.value({ view: v })[1];
      const ys = y.value({ view: v })[1];
      expect(ys.series.every((s) => s.alignmentMultiple === 1n)).toBe(true);
      expect(ys.length).toBeGreaterThanOrEqual(500);
      expect(ys.length).toBeLessThan(homePoints + 1024);
      expect(drawn(xs, ys).reduce((n, op) => n + op.count, 0)).toBe(ys.length);
      x.cleanup();
      y.cleanup();
    });
  });
});
