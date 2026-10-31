// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, status } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import {
  bounds,
  box,
  color,
  type CrudeTimeRange,
  DataType,
  dynamicWindow,
  id,
  MultiSeries,
  scale,
  Series,
  staticWindow,
  TimeRange,
  TimeSpan,
  TimeStamp,
  type Window,
} from "@synnaxlabs/x";
import { afterAll, afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { telem } from "@/telem/aether";
import { telemTest } from "@/telem/aether/testutil";
import { renderAether } from "@/testutil/renderAether";
import {
  bridgeVertices,
  buildDrawOperations,
  DEFAULT_OVERLAP_THRESHOLD,
} from "@/vis/line/aether/draw";
import { Line, type LineProps, type State } from "@/vis/line/aether/line";
import { lineTest } from "@/vis/line/aether/testutil";
import { canvasTest } from "@/vis/render/testutil";

describe("line", () => {
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
      const line = h.child<Line>("line");
      const update = (next: Partial<State>) =>
        h.setChildState("line", Line.TYPE, { ...line.state, ...next });
      const hide = () => update({ visible: false });
      return {
        line,
        recorder,
        calls,
        draws,
        strips,
        remove,
        update,
        hide,
        xSource,
        ySource,
      };
    };

    const X1 = series([0, 1, 2], 0n, { start: 0, end: 100 });
    const Y1 = series([10, 20, 30], 0n, { start: 0, end: 100 });
    const X2 = series([5, 6], 10n, { start: 100, end: 200 });
    const Y2 = series([40, 50], 10n, { start: 100, end: 200 });

    it("should pass the x range the region shows to its sources", () => {
      const m = mount([X1], [Y1]);
      const values = [vi.spyOn(m.xSource, "value"), vi.spyOn(m.ySource, "value")];
      const x = scale.Scale.scale<number>(100, 300).scale(0, 1);
      const props: LineProps = { ...PROPS, dataToDecimalScale: new scale.XY(x) };
      m.line.render(props);
      m.line.findByXValue(props, 1);
      const read = { bounds: { lower: 100, upper: 300 } };
      values.forEach((value) => expect(value.mock.calls).toEqual([[read], [read]]));
    });

    describe("zoom settle", () => {
      beforeEach(() => {
        vi.useFakeTimers();
      });

      afterEach(() => {
        vi.useRealTimers();
      });

      const showing = (lower: number, upper: number): LineProps => ({
        ...PROPS,
        dataToDecimalScale: new scale.XY(
          scale.Scale.scale<number>(lower, upper).scale(0, 1),
        ),
      });

      const mountSpied = (state: Partial<State> = {}) => {
        const m = mount([X1], [Y1], state);
        const value = vi.spyOn(m.ySource, "value");
        const lastPoints = (): number | undefined =>
          value.mock.calls.at(-1)?.[0]?.points;
        return { ...m, lastPoints };
      };

      const settled = (state: Partial<State> = {}) => {
        const m = mountSpied(state);
        m.line.render(showing(100, 300));
        vi.advanceTimersByTime(150);
        m.line.render(showing(100, 300));
        return m;
      };

      it("should give its sources points once a zoom holds", () => {
        const m = mountSpied();
        m.line.render(showing(100, 300));
        expect(m.lastPoints()).toBeUndefined();
        vi.advanceTimersByTime(149);
        m.line.render(showing(100, 300));
        expect(m.lastPoints()).toBeUndefined();
        vi.advanceTimersByTime(1);
        m.line.render(showing(100, 300));
        expect(m.lastPoints()).toBe(800);
      });

      it("should give its sources points for a pan at once", () => {
        const m = settled();
        m.line.render(showing(200, 400));
        expect(m.lastPoints()).toBe(800);
      });

      it("should wait again for a zoom to hold", () => {
        const m = settled();
        m.line.render(showing(100, 150));
        expect(m.lastPoints()).toBeUndefined();
        vi.advanceTimersByTime(150);
        m.line.render(showing(100, 150));
        expect(m.lastPoints()).toBe(800);
      });

      it("should size points by the y aggregation and groups per column", () => {
        const m = mountSpied({ groupsPerColumn: 1 });
        m.ySource.aggregation = "average";
        m.line.render(showing(100, 300));
        vi.advanceTimersByTime(150);
        m.line.render(showing(100, 300));
        expect(m.lastPoints()).toBe(800);
        const coarse = settled({ groupsPerColumn: 1 / 4 });
        expect(coarse.lastPoints()).toBe(400);
      });

      it("should stop its settle timer when deleted", () => {
        const m = mountSpied();
        m.line.render(showing(100, 300));
        expect(vi.getTimerCount()).toBe(1);
        m.remove();
        expect(vi.getTimerCount()).toBe(0);
      });
    });

    it("should read its bounds with the extremes of every sample kept", () => {
      const m = mount([X1], [Y1]);
      const values = [vi.spyOn(m.xSource, "value"), vi.spyOn(m.ySource, "value")];
      expect(m.line.yBounds({ lower: 0, upper: 2 })).toEqual({
        bounds: { lower: 10, upper: 30 },
        loading: false,
      });
      values.forEach((value) =>
        expect(value.mock.calls).toEqual([[{ extremesKept: true }]]),
      );
    });

    it("should report its bounds as loading while a source they read loads", () => {
      const m = mount([X1], [Y1]);
      m.xSource.status = telem.LOADING_STATUS;
      expect(m.line.xBounds().loading).toBe(true);
      expect(m.line.yBounds({ lower: 0, upper: 2 }).loading).toBe(true);
      expect(m.line.yBounds(bounds.INFINITE).loading).toBe(false);
    });

    describe("loading", () => {
      it("should report loading while a source loads with no data", () => {
        const m = mount([X1], []);
        m.ySource.status = telem.LOADING_STATUS;
        m.line.render(PROPS);
        expect([m.line.loading, m.line.fetching]).toEqual([true, false]);
      });

      it("should report fetching while a source loads and data draws", () => {
        const m = mount([X1], [Y1]);
        m.xSource.status = telem.LOADING_STATUS;
        m.line.render(PROPS);
        expect([m.line.loading, m.line.fetching]).toEqual([false, true]);
      });

      it("should report neither while it has no data and hides loading", () => {
        const m = mount([X1], [], { loadingHidden: true });
        m.ySource.status = telem.LOADING_STATUS;
        m.line.render(PROPS);
        expect([m.line.loading, m.line.fetching]).toEqual([false, false]);
      });

      it("should report fetching for a load after the first, with no data", () => {
        const m = mount([X1], []);
        m.line.render(PROPS);
        m.ySource.status = telem.LOADING_STATUS;
        m.line.render(PROPS);
        expect([m.line.loading, m.line.fetching]).toEqual([false, true]);
      });

      it("should report loading for the first load of a new source", () => {
        const m = mount([X1], []);
        m.line.render(PROPS);
        const y = createSource([]);
        y.status = telem.LOADING_STATUS;
        m.update({ y: sourceSpec(y) });
        m.line.render(PROPS);
        expect([m.line.loading, m.line.fetching]).toEqual([true, false]);
      });

      it("should report neither once its sources leave loading", () => {
        const m = mount([X1], []);
        m.ySource.status = status.create({ variant: "error", message: "failed" });
        m.line.render(PROPS);
        expect([m.line.loading, m.line.fetching]).toEqual([false, false]);
      });

      it("should report neither while hidden", () => {
        const m = mount([X1], []);
        m.ySource.status = telem.LOADING_STATUS;
        m.line.render(PROPS);
        m.hide();
        m.line.render(PROPS);
        expect([m.line.loading, m.line.fetching]).toEqual([false, false]);
      });

      it("should report the state of its last render without a read", () => {
        const m = mount([X1], []);
        m.ySource.status = telem.LOADING_STATUS;
        m.line.render(PROPS);
        const values = [vi.spyOn(m.xSource, "value"), vi.spyOn(m.ySource, "value")];
        m.ySource.status = telem.SUCCESS_STATUS;
        expect(m.line.loading).toBe(true);
        values.forEach((value) => expect(value).not.toHaveBeenCalled());
      });
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

      it("should draw no bridge between two domains with time between them", () => {
        const late = { start: 200, end: 300 };
        const m = mount(
          [X1, series([5, 6], 10n, late)],
          [Y1, series([40, 50], 10n, late)],
        );
        m.line.render(PROPS);
        expect(m.draws()).toEqual([
          [canvasTest.GL_ENUMS.LINE_STRIP, 3],
          [canvasTest.GL_ENUMS.LINE_STRIP, 2],
        ]);
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
      // Series upload as static draws, and the stroke and bridge buffers as dynamic.
      const uploads = (m: ReturnType<typeof mount>): number[][] =>
        m.calls
          .filter(
            (c) =>
              c.op === "bufferData" && c.args[2] === canvasTest.GL_ENUMS.STATIC_DRAW,
          )
          .map((c) => Array.from(c.args[1] as Float32Array));
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
        decimated.ySource.aggregation = "decimate";
        decimated.line.render(showing(0, N - 1));
        expect(decimated.strips()).toEqual([[0, N / 32]]);
        const fine = mount([ramp(N)], [waves(N)], { groupsPerColumn: 1 });
        fine.line.render(showing(0, N - 1));
        expect(fine.strips()).toEqual([[0, 2 * (N / 16)]]);
      });

      it("should round down the averages of integer series", () => {
        const timeRange = new TimeRange(RANGE.start, RANGE.end);
        const x = new Series({
          data: new BigInt64Array(Array.from({ length: N }, (_, i) => BigInt(i))),
          dataType: DataType.TIMESTAMP,
          timeRange,
        });
        const bits = new Series({
          data: new Int16Array(Array.from({ length: N }, (_, i) => i % 2)),
          timeRange,
        });
        const m = mount([x], [bits]);
        m.ySource.aggregation = "average";
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

      it("should return the exact value of a float64 sample", () => {
        const y = new Series({
          data: new Float64Array([0.1, 0.2, 0.3]),
          timeRange: new TimeRange(0, 100),
        });
        const m = mount([X1], [y]);
        expect(m.line.findByXValue(PROPS, 1).value).toEqual({ x: 1, y: 0.2 });
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

  describe("ChannelData against a Core", () => {
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
      const { time, data } = await createChannels();
      const start = TimeStamp.seconds(1_000_000);
      await synnax.write(start, {
        [time]: Array.from({ length: COUNT }, (_, i) =>
          start.add(TimeSpan.milliseconds(i)),
        ),
        [data]: Array.from({ length: COUNT }, (_, i) =>
          i === spikeAt ? SPIKE : Math.sin(i / 100),
        ),
      });
      const home = new TimeRange(start, start.add(TimeSpan.milliseconds(COUNT)));
      return { data, home };
    };

    const create = (props: Partial<telem.ChannelDataProps>) =>
      new telem.ChannelData(client, props);

    const createWindowed = (dataKey: channel.Key, window: Window) => ({
      x: create({ channel: dataKey, useIndexOfChannel: true, window }),
      y: create({ channel: dataKey, window }),
    });

    const createPair = (dataKey: channel.Key, home: TimeRange) =>
      createWindowed(dataKey, staticWindow(home));

    const settle = async (...sources: telem.ChannelData[]): Promise<void> =>
      await expect
        .poll(() => sources.every((s) => s.value().status.variant !== "loading"))
        .toBe(true);

    const drawn = (x: MultiSeries, y: MultiSeries) =>
      buildDrawOperations(x, y, DEFAULT_OVERLAP_THRESHOLD);

    // Returns the separate lines the sources draw: their ops less the bridges that join
    // them.
    const linesOf = (x: telem.ChannelData, y: telem.ChannelData): number => {
      const ops = drawn(x.value().data[1], y.value().data[1]);
      return ops.length - bridgeVertices(ops, scale.XY.IDENTITY).length / 4;
    };

    // Writes count samples spaced by step from start in one writer session.
    const writeRun = async (
      keys: { time: channel.Key; data: channel.Key },
      start: TimeStamp,
      count: number,
      step: TimeSpan,
    ): Promise<void> => {
      const writer = await synnax.openWriter({
        start,
        channels: [keys.time, keys.data],
      });
      const BATCH = 100;
      for (let i = 0; i < count; i += BATCH) {
        const n = Math.min(BATCH, count - i);
        await writer.write({
          [keys.time]: Array.from({ length: n }, (_, k) => start.add(step.mult(i + k))),
          [keys.data]: new Float32Array(n).fill(i),
        });
      }
      await writer.close();
    };

    const createChannels = async () => {
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
      return { time: time.key, data: data.key };
    };

    it("should pair every reduced x sample with a y sample", async () => {
      const { data, home } = await writeChannels();
      const { x, y } = createPair(data, home);
      await settle(x, y);
      const [xBounds, xs] = x.value().data;
      const [, ys] = y.value().data;
      expect(xBounds).toEqual(home.numericBounds);
      expect(xs.length).toBe(ys.length);
      expect(xs.length).toBeLessThan(COUNT);
      expect(xs.series.map((s) => s.alignment)).toEqual(
        ys.series.map((s) => s.alignment),
      );
      expect(ys.series.every((s) => s.alignmentMultiple > 1n)).toBe(true);
      const ops = drawn(xs, ys);
      expect(ops.reduce((n, op) => n + op.count, 0)).toBe(ys.length);
      expect(y.value().data[0].upper).toBeCloseTo(1, 3);
      expect(y.value().data[0].lower).toBeCloseTo(-1, 3);
      x.cleanup();
      y.cleanup();
    });

    it("should keep a spike that an average drops inside the line's bounds", async () => {
      const { data, home } = await writeChannels(COUNT / 2);
      const y = create({
        channel: data,
        window: staticWindow(home),
        aggregation: "average",
      });
      await settle(y);
      expect(y.value().data[0].upper).toBeLessThan(SPIKE / 2);
      await expect
        .poll(() => y.value({ extremesKept: true }).data[0].upper)
        .toBe(SPIKE);
      y.cleanup();
    });

    it("should draw finer tiles for a zoomed view", async () => {
      const { data, home } = await writeChannels();
      const { x, y } = createPair(data, home);
      await settle(x, y);
      const homePoints = y.value().data[1].length;
      const start = Number(home.start.valueOf());
      const v: telem.ValueProps = {
        bounds: {
          lower: start,
          upper: start + Number(TimeSpan.milliseconds(500).valueOf()),
        },
        points: 1000,
      };
      x.value(v);
      y.value(v);
      await expect
        .poll(() => {
          const [xr, yr] = [x.value(v), y.value(v)];
          return (
            xr.status.variant !== "loading" &&
            yr.status.variant !== "loading" &&
            yr.data[1].length > 0
          );
        })
        .toBe(true);
      const xs = x.value(v).data[1];
      const ys = y.value(v).data[1];
      expect(ys.series.every((s) => s.alignmentMultiple === 1n)).toBe(true);
      expect(ys.length).toBeGreaterThanOrEqual(500);
      expect(ys.length).toBeLessThan(homePoints + 1024);
      expect(drawn(xs, ys).reduce((n, op) => n + op.count, 0)).toBe(ys.length);
      x.cleanup();
      y.cleanup();
    });

    it("should break a line where a writer restarted, live and read back", async () => {
      const keys = await createChannels();
      const count = 200;
      const step = TimeSpan.milliseconds(1);
      const start = TimeStamp.now().sub(TimeSpan.seconds(10));
      const live = createWindowed(keys.data, dynamicWindow(TimeSpan.seconds(60)));
      await settle(live.x, live.y);
      await writeRun(keys, start, count, step);
      await writeRun(keys, start.add(TimeSpan.seconds(1)), count, step);
      await expect.poll(() => live.y.value().data[1].length).toBe(2 * count);
      expect(linesOf(live.x, live.y)).toBe(2);
      live.x.cleanup();
      live.y.cleanup();
      const end = start.add(TimeSpan.seconds(2));
      const read = createPair(keys.data, new TimeRange(start, end));
      await settle(read.x, read.y);
      await expect.poll(() => read.y.value().data[1].length).toBe(2 * count);
      expect(linesOf(read.x, read.y)).toBe(2);
      read.x.cleanup();
      read.y.cleanup();
    });

    it("should draw one line while a writer's samples cross raw blocks", async () => {
      const keys = await createChannels();
      const count = 7_000;
      const live = createWindowed(keys.data, dynamicWindow(TimeSpan.seconds(100)));
      await settle(live.x, live.y);
      await writeRun(keys, TimeStamp.now(), count, TimeSpan.milliseconds(10));
      await expect.poll(() => live.y.value().data[1].length).toBe(count);
      expect(live.y.value().data[1].series.length).toBeGreaterThan(1);
      expect(linesOf(live.x, live.y)).toBe(1);
      live.x.cleanup();
      live.y.cleanup();
    });
  });
});
