// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, testClientParams } from "@synnaxlabs/client/testutil";
import {
  box,
  color,
  DataType,
  dynamicWindow,
  id,
  sleep,
  staticWindow,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { afterEach, describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { aetherTest } from "@/aether/testutil";
import { lineplot } from "@/lineplot/aether";
import { measure } from "@/lineplot/measure/aether";
import { tooltip } from "@/lineplot/tooltip/aether";
import { telem } from "@/telem/aether";
import { telemTest } from "@/telem/aether/testutil";
import { buildStack } from "@/testutil/internal";
import { line } from "@/vis/line/aether";
import { lineTest } from "@/vis/line/aether/testutil";
import { canvasTest } from "@/vis/render/testutil";

const stubLineStateZ = z.object({
  loading: z.boolean(),
  fetching: z.boolean().default(false),
});

// Line needs a live GL context, so a stub with its TYPE stands in for the walk.
class StubLine extends aether.Leaf<typeof stubLineStateZ> {
  static readonly TYPE = line.Line.TYPE;
  override schema = stubLineStateZ;

  get loading(): boolean {
    return this.state.loading;
  }

  get fetching(): boolean {
    return this.state.fetching;
  }
}

interface Mount {
  plot: lineplot.LinePlot;
  recorder: canvasTest.Recorder;
  setLineLoading: (loading: boolean) => void;
  setLineFetching: (fetching: boolean) => void;
  pump: () => void;
  hide: () => void;
  remove: () => void;
}

describe("LinePlot", () => {
  const teardowns: (() => void)[] = [];
  afterEach(() => {
    for (const teardown of teardowns) teardown();
    teardowns.length = 0;
  });

  const mount = (lineLoading: boolean | null): Mount => {
    const recorder = new canvasTest.Recorder();
    const stack = buildStack({
      registry: {
        [lineplot.LinePlot.TYPE]: lineplot.LinePlot,
        [lineplot.XAxis.TYPE]: lineplot.XAxis,
        [lineplot.YAxis.TYPE]: lineplot.YAxis,
        [StubLine.TYPE]: StubLine,
      },
      render: recorder,
    });
    const base = [...stack.basePath, "plot"];
    stack.driver.update(base, lineplot.LinePlot.TYPE, {
      container: box.DECIMAL,
      viewport: box.DECIMAL,
      grid: {},
    });
    stack.driver.update([...base, "x1"], lineplot.XAxis.TYPE, { location: "bottom" });
    stack.driver.update([...base, "x1", "y1"], lineplot.YAxis.TYPE, {
      location: "left",
    });
    if (lineLoading != null)
      stack.driver.update([...base, "x1", "y1", "l1"], StubLine.TYPE, {
        loading: lineLoading,
      });
    teardowns.push(() => stack.driver.delete([aetherTest.ROOT_KEY]));
    return {
      plot: stack.driver.find<lineplot.LinePlot>(base),
      recorder,
      setLineLoading: (loading) =>
        stack.driver.update([...base, "x1", "y1", "l1"], StubLine.TYPE, { loading }),
      setLineFetching: (fetching) =>
        stack.driver.update([...base, "x1", "y1", "l1"], StubLine.TYPE, {
          loading: false,
          fetching,
        }),
      pump: () => recorder.renderLast(lineplot.LinePlot.TYPE),
      hide: () =>
        stack.driver.update(base, lineplot.LinePlot.TYPE, {
          container: box.DECIMAL,
          viewport: box.DECIMAL,
          grid: {},
          visible: false,
        }),
      remove: () => stack.driver.delete(base),
    };
  };

  describe("loading", () => {
    it("should keep drawing and sync state while a line loads", () => {
      const m = mount(true);
      m.pump();
      expect(m.plot.state.loading).toBe(true);
      expect(m.recorder.scissorCalls.length).toBeGreaterThan(0);
    });

    it("should clear the state once every line settles", () => {
      const m = mount(true);
      m.pump();
      m.setLineLoading(false);
      m.pump();
      expect(m.plot.state.loading).toBe(false);
    });

    it("should not report loading for a plot with no lines", () => {
      const m = mount(null);
      m.pump();
      expect(m.plot.state.loading).toBe(false);
    });
  });

  describe("fetching", () => {
    it("should keep drawing and sync state while a line fetches", () => {
      const m = mount(false);
      m.setLineFetching(true);
      m.pump();
      expect(m.plot.state.fetching).toBe(true);
      expect(m.plot.state.loading).toBe(false);
      expect(m.recorder.scissorCalls.length).toBeGreaterThan(0);
    });

    it("should clear the state once the line settles", () => {
      const m = mount(false);
      m.setLineFetching(true);
      m.pump();
      m.setLineFetching(false);
      m.pump();
      expect(m.plot.state.fetching).toBe(false);
    });
  });

  describe("after the WebGL context is restored", () => {
    it("should request a render of every canvas until deleted", () => {
      const m = mount(false);
      m.recorder.clear();
      m.recorder.restore();
      expect(m.recorder.loopCalls.map((c) => c.args[0])).toEqual([
        expect.objectContaining({
          key: `${lineplot.LinePlot.TYPE}-plot`,
          priority: "high",
          canvases: ["upper2d", "lower2d", "gl"],
        }),
      ]);
      m.remove();
      m.recorder.clear();
      m.recorder.restore();
      expect(m.recorder.loopCalls).toEqual([]);
    });

    it("should not request a render while hidden", () => {
      const m = mount(false);
      m.hide();
      m.recorder.clear();
      m.recorder.restore();
      expect(m.recorder.loopCalls).toEqual([]);
    });
  });
});

const PLOT_STATE = {
  container: box.construct(0, 0, 1000, 500),
  viewport: box.DECIMAL,
  grid: {},
};

const MINUTE = Number(TimeSpan.MINUTE.valueOf());
const DAY = Number(TimeSpan.DAY.valueOf());

type State = Record<string, unknown>;

interface GLPlot {
  plot: lineplot.LinePlot;
  glCalls: canvasTest.RecordingGL["calls"];
  /** Sets the state of the plot over PLOT_STATE. */
  setPlot: (state?: State) => void;
  /** Sets the state of the element at path, from the plot down. */
  set: (path: string[], type: string, state: State) => void;
  /** Sets the x axis key, with a left y axis y1 inside it. */
  setAxes: (key: string, state?: State) => void;
  /** Sets a line at path, from the plot down, with x and y data. */
  setLine: (path: string[], x: number[], y: number[], state?: State) => void;
  xAxis: (key: string) => lineplot.XAxis;
  /** Renders the plot, and drops the 2D calls of earlier renders. */
  render: () => void;
  /** @returns the text that the last render drew. */
  texts: () => string[];
  /** @returns the count of 2D line segments that the last render drew. */
  lines: () => number;
}

// mountGLPlot mounts a plot whose lines draw through a recording GL context.
const mountGLPlot = (teardowns: (() => void)[]): GLPlot => {
  const recorder = new canvasTest.Recorder();
  const { gl, calls } = canvasTest.createGL();
  (recorder as { gl: unknown }).gl = gl;
  const stack = buildStack({
    registry: {
      [lineTest.Host.TYPE]: lineTest.Host,
      [lineplot.LinePlot.TYPE]: lineplot.LinePlot,
      [lineplot.XAxis.TYPE]: lineplot.XAxis,
      [lineplot.YAxis.TYPE]: lineplot.YAxis,
      [line.Line.TYPE]: line.Line,
      [tooltip.Tooltip.TYPE]: tooltip.Tooltip,
      [measure.Measure.TYPE]: measure.Measure,
    },
    render: recorder,
  });
  teardowns.push(() => stack.driver.delete([aetherTest.ROOT_KEY]));
  const host = [...stack.basePath, "gl"];
  stack.driver.update(host, lineTest.Host.TYPE, {});
  const base = [...host, "plot"];
  const set = (path: string[], type: string, state: State): void =>
    stack.driver.update([...base, ...path], type, state);
  set([], lineplot.LinePlot.TYPE, PLOT_STATE);
  return {
    plot: stack.driver.find<lineplot.LinePlot>(base),
    glCalls: calls,
    setPlot: (state = {}) =>
      set([], lineplot.LinePlot.TYPE, { ...PLOT_STATE, ...state }),
    set,
    setAxes: (key, state = {}) => {
      const location = key === "x1" ? "bottom" : "top";
      set([key], lineplot.XAxis.TYPE, { axisKey: key, location, ...state });
      set([key, "y1"], lineplot.YAxis.TYPE, { location: "left" });
    },
    setLine: (path, x, y, state = {}) =>
      set(path, line.Line.TYPE, {
        x: telemTest.seriesSpec(x),
        y: telemTest.seriesSpec(y),
        color: "#ff0000",
        ...state,
      }),
    xAxis: (key) => stack.driver.find<lineplot.XAxis>([...base, key]),
    render: () => {
      recorder.upper2d.calls.length = 0;
      recorder.renderLast(lineplot.LinePlot.TYPE);
    },
    texts: () =>
      recorder.upper2d.calls
        .filter((c) => c.op === "fillText")
        .map((c) => String(c.args[0])),
    lines: () => recorder.upper2d.calls.filter((c) => c.op === "lineTo").length,
  };
};

describe("LinePlot with a tooltip", () => {
  const teardowns: (() => void)[] = [];
  afterEach(() => {
    for (const teardown of teardowns) teardown();
    teardowns.length = 0;
  });

  it("should list the drawn point nearest the cursor", () => {
    const p = mountGLPlot(teardowns);
    p.setAxes("x1");
    // 4000 samples at zero in [0, 4), then 10 in [100, 1000] that reduce to their
    // extremes: 10 at 300 and -10 at 800. The line draws no sample of 7.
    const dense = Array.from({ length: 4000 }, (_, i) => i / 1000);
    const sparse = Array.from({ length: 10 }, (_, i) => (i + 1) * 100);
    p.setLine(
      ["x1", "y1", "l1"],
      [...dense, ...sparse],
      [...dense.map(() => 0), 7, 7, 10, 7, 7, 7, 7, -10, 7, 7],
    );
    p.render();
    p.set(["tooltip"], tooltip.Tooltip.TYPE, { position: { x: 500, y: 250 } });
    p.render();
    expect(p.texts()).toContain("10");
    expect(p.texts()).not.toContain("7");
  });
});

describe("LinePlot getXBounds and getXPositions", () => {
  const teardowns: (() => void)[] = [];
  afterEach(() => {
    for (const teardown of teardowns) teardown();
    teardowns.length = 0;
  });

  it("should map positions across the home view to values and back", () => {
    const p = mountGLPlot(teardowns);
    p.setPlot({ viewport: box.construct(0.5, 0, 0.5, 1) });
    p.setAxes("x1", { autoBoundPadding: 0 });
    p.setLine(["x1", "y1", "l1"], [0, 1000], [0, 1]);
    p.render();
    expect(p.plot.getXBounds({ lower: 0.25, upper: 0.5 })).toEqual({
      x1: { lower: 250, upper: 500 },
    });
    expect(p.plot.getXPositions({ lower: 250, upper: 500 })).toEqual({
      x1: { lower: 0.25, upper: 0.5 },
    });
  });
});

describe("LinePlot breaks", () => {
  const teardowns: (() => void)[] = [];
  afterEach(() => {
    for (const teardown of teardowns) teardown();
    teardowns.length = 0;
  });

  // mount mounts each x axis with two minutes a day apart. The gap of x1 collapses into
  // a break at the middle of the plot.
  const mount = (axisKeys: string[] = ["x1"]): GLPlot => {
    const p = mountGLPlot(teardowns);
    axisKeys.forEach((key) => {
      p.setAxes(key, { type: "time", autoBoundPadding: 0 });
      [0, DAY].forEach((start, i) =>
        p.setLine([key, "y1", `l${i}`], [start, start + MINUTE], [0, 1]),
      );
    });
    p.render();
    return p;
  };

  it("should count the open and closed breaks of x1", () => {
    const p = mount();
    expect(p.plot.state.breaks).toEqual({ open: 0, closed: 1 });
    const change = p.plot.getBreakChange({
      action: { variant: "all", open: true },
      viewport: box.DECIMAL,
      held: [],
      selection: box.ZERO,
    });
    p.setPlot({ openBreaks: change?.openBreaks ?? [] });
    p.render();
    expect(p.plot.state.breaks).toEqual({ open: 1, closed: 0 });
  });

  it("should count the breaks of x1 while they are hidden", () => {
    const p = mount();
    p.setPlot({ breaksHidden: true });
    p.render();
    expect(p.plot.state.breaks).toEqual({ open: 0, closed: 1 });
  });

  it("should count no hidden breaks while the x source of x1 fails", () => {
    const p = mount();
    p.setPlot({ breaksHidden: true });
    p.render();
    expect(p.plot.state.breaks).toEqual({ open: 0, closed: 1 });
    const failed = vi.spyOn(line.Line.prototype, "xBounds").mockImplementation(() => {
      throw new Error("x source failed");
    });
    teardowns.push(() => failed.mockRestore());
    p.set([], lineplot.LinePlot.TYPE, { ...p.plot.state });
    p.render();
    expect(p.plot.state.breaks).toEqual({ open: 0, closed: 0 });
  });

  it("should not count a hidden break outside the manual bounds of x1", () => {
    const p = mount();
    p.setAxes("x1", {
      type: "time",
      autoBoundPadding: 0,
      bounds: { lower: 0, upper: MINUTE * 2 },
      manualBounds: { lower: true, upper: true },
    });
    p.setPlot({ breaksHidden: true });
    p.render();
    expect(p.plot.state.breaks).toEqual({ open: 0, closed: 0 });
  });

  it("should give the seams of x1 across the home view", () => {
    const seams = mount().plot.getSeams();
    expect(seams).toHaveLength(1);
    expect(seams[0]).toBeCloseTo(0.5, 2);
  });

  it("should list the time that a break hides under the tooltip", () => {
    const p = mount();
    const textsAt = (x: number): string[] => {
      p.set(["tooltip"], tooltip.Tooltip.TYPE, { position: { x, y: 250 } });
      p.render();
      return p.texts();
    };
    expect(textsAt(500)).toContain("Hidden");
    expect(textsAt(250)).not.toContain("Hidden");
  });

  it("should not join two measured points across a closed break", () => {
    const p = mount();
    const measureAt = (two: number): number => {
      p.set(["measure"], measure.Measure.TYPE, {
        one: { x: 100, y: 250 },
        two: { x: two, y: 250 },
        hover: null,
      });
      p.render();
      return p.lines();
    };
    expect(measureAt(900)).toEqual(measureAt(400) - 1);
  });

  it("should join two measured points of an x axis other than x1", () => {
    const p = mountGLPlot(teardowns);
    // x1 collapses the day between its two lines. The flat line of x2 spans that day,
    // so the measure finds its points at the middle of the plot.
    p.setAxes("x1", { type: "time", autoBoundPadding: 0 });
    [0, DAY].forEach((start, i) =>
      p.setLine(["x1", "y1", `l${i}`], [start, start + MINUTE], [0, 2]),
    );
    p.setAxes("x2", { type: "time", autoBoundPadding: 0 });
    p.setLine(["x2", "y1", "l"], [0, DAY + MINUTE], [1, 1], { color: "#0000ff" });
    p.render();
    p.set(["measure"], measure.Measure.TYPE, {
      one: { x: 1, y: 250 },
      two: { x: 999, y: 250 },
      hover: null,
    });
    p.render();
    expect(p.texts()).toContain("ΔX");
    expect(p.texts().filter((t) => t.endsWith(" //"))).toEqual([]);
  });

  it("should count no breaks while the x source of x1 fails", () => {
    const p = mount();
    const failed = vi.spyOn(line.Line.prototype, "xBounds").mockImplementation(() => {
      throw new Error("x source failed");
    });
    teardowns.push(() => failed.mockRestore());
    p.setPlot();
    p.render();
    expect(p.plot.state.breaks).toEqual({ open: 0, closed: 0 });
  });

  it.each([[["x1", "x2"]], [["x2", "x1"]]])(
    "should lay out every x axis other than x1 in true time, in the order %j",
    (keys) => {
      const { x1, x2 } = mount(keys).plot.getXBounds({ lower: 0.25, upper: 0.25 });
      expect(x1.lower).toBeLessThan(MINUTE);
      expect(x2.lower).toBeCloseTo((DAY + MINUTE) / 4, -6);
    },
  );
});

describe("LinePlot trace spans", () => {
  const teardowns: (() => void)[] = [];
  afterEach(() => {
    for (const teardown of teardowns) teardown();
    teardowns.length = 0;
  });

  // mount mounts two lines of one group on x1, with samples from 0 to 10 and 20 to 30.
  const mount = (type: "time" | "relative"): lineplot.XAxis => {
    const p = mountGLPlot(teardowns);
    p.setAxes("x1", { type, autoBoundPadding: 0 });
    [0, 20].forEach((start, i) =>
      p.setLine(["x1", "y1", `l${i}`], [start, start + 10], [i, i + 1], { group: "g" }),
    );
    p.render();
    return p.xAxis("x1");
  };

  const PROPS = {
    grid: {},
    plot: box.construct(0, 0, 1000, 500),
    container: box.construct(0, 0, 1000, 500),
    viewport: box.DECIMAL,
    hold: false,
    breaksHidden: true,
    openBreaks: [],
  };

  it("should give each line of a group its own half of a time axis", () => {
    const layout = mount("time").layout(PROPS);
    expect(layout.span("l0")).toEqual({ lower: -Infinity, upper: 15 });
    expect(layout.span("l1")).toEqual({ lower: 15, upper: Infinity });
  });

  it("should find a value only in the line that owns the time", () => {
    const found = mount("time")
      .findByXValue(PROPS, 25)
      .filter((r) => Number.isFinite(r.value.y));
    expect(found.map((r) => r.key)).toEqual(["l1"]);
  });

  it("should give each line its window with a margin on a relative axis", () => {
    const layout = mount("relative").layout(PROPS);
    const second = Number(TimeSpan.SECOND.valueOf());
    expect(layout.span("l0")).toEqual({ lower: -second, upper: 10 + second });
    expect(layout.span("l1")).toEqual({ lower: 20 - second, upper: 30 + second });
  });
});

describe("LinePlot against a Core", () => {
  const synnax = createTestClient();
  const teardowns: (() => void)[] = [];
  afterEach(() => {
    for (const teardown of teardowns) teardown();
    teardowns.length = 0;
  });

  const RATE = 1000;
  const DURATION = TimeSpan.minutes(10);

  // Writes a 1 kHz sine over DURATION and returns the range it covers.
  const writeData = async () => {
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
    const start = TimeStamp.seconds(2_000_000);
    const count = Number(DURATION.seconds) * RATE;
    const step = TimeSpan.milliseconds(1).valueOf();
    const times = new BigInt64Array(count);
    const values = new Float32Array(count);
    for (let i = 0; i < count; i++) {
      times[i] = start.valueOf() + BigInt(i) * step;
      values[i] = Math.sin(i / 100);
    }
    await synnax.write(start, { [time.key]: times, [data.key]: values });
    return { data, home: new TimeRange(start, start.add(DURATION)) };
  };

  const mount = (props: telem.ChannelDataProps) => {
    const stack = buildStack({
      registry: {
        [telem.PROVIDER_TYPE]: telem.Provider,
        [lineTest.Host.TYPE]: lineTest.Host,
        [lineplot.LinePlot.TYPE]: lineplot.LinePlot,
        [lineplot.XAxis.TYPE]: lineplot.XAxis,
        [lineplot.YAxis.TYPE]: lineplot.YAxis,
        [line.Line.TYPE]: line.Line,
      },
      synnax: { props: testClientParams() },
      render: true,
    });
    const gl = [...stack.basePath, "gl"];
    stack.driver.update(gl, lineTest.Host.TYPE, {});
    const base = [...gl, "plot"];
    const setViewport = (viewport: box.Box): void =>
      stack.driver.update(base, lineplot.LinePlot.TYPE, {
        container: box.construct(0, 0, 1000, 500),
        viewport,
        grid: {},
      });
    setViewport(box.DECIMAL);
    stack.driver.update([...base, "x1"], lineplot.XAxis.TYPE, { location: "bottom" });
    stack.driver.update([...base, "x1", "y1"], lineplot.YAxis.TYPE, {
      location: "left",
    });
    stack.driver.update([...base, "x1", "y1", "l1"], line.Line.TYPE, {
      x: telem.channelData({ ...props, useIndexOfChannel: true }),
      y: telem.channelData(props),
      color: "#ff0000",
    });
    teardowns.push(() => stack.driver.delete([aetherTest.ROOT_KEY]));
    const { recorder } = stack;
    if (recorder == null) throw new Error("the stack has no render recorder");
    const l1 = stack.driver.find<line.Line>([...base, "x1", "y1", "l1"]);
    return {
      plot: stack.driver.find<lineplot.LinePlot>(base),
      xBounds: () => l1.xBounds().bounds,
      setViewport,
      render: () => recorder.renderLast(lineplot.LinePlot.TYPE),
    };
  };

  // The distance from each target to the first drawn sample at or after it.
  const gaps = (plot: lineplot.LinePlot, targets: number[]): number[] =>
    targets.map((t) => {
      const [found] = plot.findByXValue(t);
      const x = found?.value.x ?? NaN;
      return Number.isNaN(x) ? Infinity : x - t;
    });

  it("should draw full detail tiles after a zoom", async () => {
    const { data, home } = await writeData();
    const { plot, setViewport } = mount({
      channel: data.key,
      window: staticWindow(home),
    });
    const homeSpan = Number(home.span.valueOf());
    const lower = Number(home.start.valueOf()) + homeSpan / 2;
    const targets = [0.1, 0.3, 0.5, 0.7, 0.9].map(
      (f) => lower + f * Number(TimeSpan.seconds(2).valueOf()),
    );
    // Home tiles hold one point every half second or so.
    await expect
      .poll(() => Math.max(...gaps(plot, targets)), { timeout: 10_000 })
      .toBeLessThan(Number(TimeSpan.seconds(2).valueOf()));
    expect(Math.max(...gaps(plot, targets))).toBeGreaterThan(
      Number(TimeSpan.milliseconds(50).valueOf()),
    );
    const zoom = Number(TimeSpan.seconds(2).valueOf()) / homeSpan;
    setViewport(box.construct(0.5, 0, zoom, 1));
    await expect
      .poll(() => Math.max(...gaps(plot, targets)), { timeout: 10_000 })
      .toBeLessThan(Number(TimeSpan.milliseconds(20).valueOf()));
  });

  it("should draw samples as a live writer commits them", async () => {
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
    const { plot, render, xBounds } = mount({
      channel: data.key,
      window: dynamicWindow(TimeSpan.seconds(10)),
    });
    // The render loop draws a plot at mount, which subscribes its lines to their
    // channels. A write before the subscription streams reaches the plot only through
    // a history read, which reduces the samples to the point budget.
    await expect
      .poll(
        () => {
          render();
          return plot.state.loading;
        },
        { timeout: 10_000 },
      )
      .toBe(false);
    const writer = await synnax.openWriter({
      start: TimeStamp.now(),
      channels: [time.key, data.key],
    });
    const step = TimeSpan.milliseconds(1).valueOf();
    let last = TimeStamp.now();
    try {
      for (let chunk = 0; chunk < 10; chunk++) {
        const start = TimeStamp.now();
        const times = new BigInt64Array(100);
        const values = new Float32Array(100);
        for (let i = 0; i < 100; i++) {
          times[i] = start.valueOf() + BigInt(i) * step;
          values[i] = Math.sin(i / 10);
        }
        await writer.write({ [time.key]: times, [data.key]: values });
        last = new TimeStamp(times[99]);
        await sleep.sleep(TimeSpan.milliseconds(100));
      }
      // The plot reduces samples to its point budget, so its line can end a sample or
      // so before the newest one.
      const tolerance = Number(TimeSpan.milliseconds(10).valueOf());
      const edge = Number(last.valueOf()) - tolerance;
      await expect
        .poll(() => gaps(plot, [edge])[0], { timeout: 10_000 })
        .toBeLessThan(tolerance);
      // The live window ends at the exclusive end of the newest sample.
      expect(xBounds().upper).toBe(Number(last.valueOf() + 1n));
    } finally {
      await writer.close();
    }
  });
});

describe("LinePlot with lines on two y axes", () => {
  const teardowns: (() => void)[] = [];
  afterEach(() => {
    for (const teardown of teardowns) teardown();
    teardowns.length = 0;
  });

  it("should draw a raised line over the lines of every y axis", () => {
    const RAISED = "#00ff00";
    const p = mountGLPlot(teardowns);
    p.setAxes("x1");
    p.set(["x1", "y2"], lineplot.YAxis.TYPE, { location: "right" });
    p.setLine(["x1", "y1", "raised"], [0, 10], [1, 2], { color: RAISED, raised: true });
    p.setLine(["x1", "y2", "plain"], [0, 10], [3, 4]);
    p.render();
    const colors = p.glCalls
      .filter((c) => c.op === "uniform4fv")
      .map((c) => [...(c.args[1] as Iterable<number>)]);
    expect(colors.length).toBeGreaterThan(1);
    expect(colors.at(-1)).toEqual(color.rgba1(color.construct(RAISED)));
  });
});
