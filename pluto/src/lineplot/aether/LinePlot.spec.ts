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
  type bounds,
  box,
  color,
  DataType,
  dynamicWindow,
  id,
  MultiSeries,
  Series,
  sleep,
  staticWindow,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { afterEach, describe, expect, it } from "vitest";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { aetherTest } from "@/aether/testutil";
import { LinePlot } from "@/lineplot/aether/LinePlot";
import { XAxis } from "@/lineplot/aether/XAxis";
import { YAxis } from "@/lineplot/aether/YAxis";
import { measure } from "@/lineplot/measure/aether";
import { tooltip } from "@/lineplot/tooltip/aether";
import { telem } from "@/telem/aether";
import { telemTest } from "@/telem/aether/testutil";
import { buildStack } from "@/testutil/providers";
import { line } from "@/vis/line/aether";
import { lineTest } from "@/vis/line/aether/testutil";
import type { render } from "@/vis/render";
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

interface LoopEntry {
  key?: string;
  render?: () => render.Cleanup | undefined;
}

// Runs the render that the plot requested last.
const renderPlot = (recorder: canvasTest.Recorder): void => {
  const entry = recorder.loopCalls
    .map((call) => call.args[0] as LoopEntry)
    .reverse()
    .find((e) => e.key?.startsWith(LinePlot.TYPE) ?? false);
  entry?.render?.();
};

interface Mount {
  plot: LinePlot;
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
        [LinePlot.TYPE]: LinePlot,
        [XAxis.TYPE]: XAxis,
        [YAxis.TYPE]: YAxis,
        [StubLine.TYPE]: StubLine,
      },
      render: recorder,
    });
    const base = [...stack.basePath, "plot"];
    stack.driver.update(base, LinePlot.TYPE, {
      container: box.DECIMAL,
      viewport: box.DECIMAL,
      grid: {},
    });
    stack.driver.update([...base, "x1"], XAxis.TYPE, { location: "bottom" });
    stack.driver.update([...base, "x1", "y1"], YAxis.TYPE, { location: "left" });
    if (lineLoading != null)
      stack.driver.update([...base, "x1", "y1", "l1"], StubLine.TYPE, {
        loading: lineLoading,
      });
    teardowns.push(() => stack.driver.delete([aetherTest.ROOT_KEY]));
    return {
      plot: stack.driver.find<LinePlot>(base),
      recorder,
      setLineLoading: (loading) =>
        stack.driver.update([...base, "x1", "y1", "l1"], StubLine.TYPE, { loading }),
      setLineFetching: (fetching) =>
        stack.driver.update([...base, "x1", "y1", "l1"], StubLine.TYPE, {
          loading: false,
          fetching,
        }),
      pump: () => renderPlot(recorder),
      hide: () =>
        stack.driver.update(base, LinePlot.TYPE, {
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
          key: `${LinePlot.TYPE}-plot`,
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

const seriesSpec = (data: number[]): telem.SeriesSourceSpec => {
  const s = new Series({
    data: new Float32Array(data),
    timeRange: new TimeRange(0, 1),
  });
  const source = telemTest.source<[bounds.Bounds, MultiSeries]>([
    s.bounds,
    new MultiSeries([s]),
  ]);
  return {
    type: telemTest.TestSource.TYPE,
    props: { testId: source.id },
    variant: "source",
    valueType: "series",
  };
};

describe("LinePlot with a tooltip", () => {
  const teardowns: (() => void)[] = [];
  afterEach(() => {
    for (const teardown of teardowns) teardown();
    teardowns.length = 0;
  });

  it("should list the drawn point nearest the cursor", () => {
    const recorder = new canvasTest.Recorder();
    (recorder as { gl: unknown }).gl = canvasTest.createGL().gl;
    const stack = buildStack({
      registry: {
        [lineTest.Host.TYPE]: lineTest.Host,
        [LinePlot.TYPE]: LinePlot,
        [XAxis.TYPE]: XAxis,
        [YAxis.TYPE]: YAxis,
        [line.Line.TYPE]: line.Line,
        [tooltip.Tooltip.TYPE]: tooltip.Tooltip,
      },
      render: recorder,
    });
    teardowns.push(() => stack.driver.delete([aetherTest.ROOT_KEY]));
    const gl = [...stack.basePath, "gl"];
    stack.driver.update(gl, lineTest.Host.TYPE, {});
    const base = [...gl, "plot"];
    stack.driver.update(base, LinePlot.TYPE, {
      container: box.construct(0, 0, 1000, 500),
      viewport: box.DECIMAL,
      grid: {},
    });
    stack.driver.update([...base, "x1"], XAxis.TYPE, { location: "bottom" });
    stack.driver.update([...base, "x1", "y1"], YAxis.TYPE, { location: "left" });
    // 4000 samples at zero in [0, 4), then 10 in [100, 1000] that reduce to their
    // extremes: 10 at 300 and -10 at 800. The line draws no sample of 7.
    const dense = Array.from({ length: 4000 }, (_, i) => i / 1000);
    const sparse = Array.from({ length: 10 }, (_, i) => (i + 1) * 100);
    stack.driver.update([...base, "x1", "y1", "l1"], line.Line.TYPE, {
      x: seriesSpec([...dense, ...sparse]),
      y: seriesSpec([...dense.map(() => 0), 7, 7, 10, 7, 7, 7, 7, -10, 7, 7]),
      color: "#ff0000",
    });
    renderPlot(recorder);
    stack.driver.update([...base, "tooltip"], tooltip.Tooltip.TYPE, {
      position: { x: 500, y: 250 },
    });
    renderPlot(recorder);
    const texts = recorder.upper2d.calls
      .filter((c) => c.op === "fillText")
      .map((c) => String(c.args[0]));
    expect(texts).toContain("10");
    expect(texts).not.toContain("7");
  });
});

describe("LinePlot getXBounds", () => {
  const teardowns: (() => void)[] = [];
  afterEach(() => {
    for (const teardown of teardowns) teardown();
    teardowns.length = 0;
  });

  it("should give the values that positions across the home view hold", () => {
    const recorder = new canvasTest.Recorder();
    (recorder as { gl: unknown }).gl = canvasTest.createGL().gl;
    const stack = buildStack({
      registry: {
        [lineTest.Host.TYPE]: lineTest.Host,
        [LinePlot.TYPE]: LinePlot,
        [XAxis.TYPE]: XAxis,
        [YAxis.TYPE]: YAxis,
        [line.Line.TYPE]: line.Line,
      },
      render: recorder,
    });
    teardowns.push(() => stack.driver.delete([aetherTest.ROOT_KEY]));
    const gl = [...stack.basePath, "gl"];
    stack.driver.update(gl, lineTest.Host.TYPE, {});
    const base = [...gl, "plot"];
    stack.driver.update(base, LinePlot.TYPE, {
      container: box.construct(0, 0, 1000, 500),
      viewport: box.construct(0.5, 0, 0.5, 1),
      grid: {},
    });
    stack.driver.update([...base, "x1"], XAxis.TYPE, {
      location: "bottom",
      autoBoundPadding: 0,
    });
    stack.driver.update([...base, "x1", "y1"], YAxis.TYPE, { location: "left" });
    stack.driver.update([...base, "x1", "y1", "l1"], line.Line.TYPE, {
      x: seriesSpec([0, 1000]),
      y: seriesSpec([0, 1]),
      color: "#ff0000",
    });
    renderPlot(recorder);
    const plot = stack.driver.find<LinePlot>(base);
    expect(plot.getXBounds({ lower: 0.25, upper: 0.5 })).toEqual({
      x1: { lower: 250, upper: 500 },
    });
    expect(plot.getXPositions({ lower: 250, upper: 500 })).toEqual({
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

  // mount mounts a plot of two minutes a day apart, whose gap collapses into a break
  // at the middle of the plot.
  const mount = () => {
    const recorder = new canvasTest.Recorder();
    (recorder as { gl: unknown }).gl = canvasTest.createGL().gl;
    const stack = buildStack({
      registry: {
        [lineTest.Host.TYPE]: lineTest.Host,
        [LinePlot.TYPE]: LinePlot,
        [XAxis.TYPE]: XAxis,
        [YAxis.TYPE]: YAxis,
        [line.Line.TYPE]: line.Line,
        [tooltip.Tooltip.TYPE]: tooltip.Tooltip,
        [measure.Measure.TYPE]: measure.Measure,
      },
      render: recorder,
    });
    teardowns.push(() => stack.driver.delete([aetherTest.ROOT_KEY]));
    const gl = [...stack.basePath, "gl"];
    stack.driver.update(gl, lineTest.Host.TYPE, {});
    const base = [...gl, "plot"];
    const setOpenBreaks = (openBreaks: number[]): void =>
      stack.driver.update(base, LinePlot.TYPE, {
        container: box.construct(0, 0, 1000, 500),
        viewport: box.DECIMAL,
        grid: {},
        openBreaks,
      });
    setOpenBreaks([]);
    stack.driver.update([...base, "x1"], XAxis.TYPE, {
      location: "bottom",
      type: "time",
      autoBoundPadding: 0,
    });
    stack.driver.update([...base, "x1", "y1"], YAxis.TYPE, { location: "left" });
    const minute = Number(TimeSpan.MINUTE.valueOf());
    const day = Number(TimeSpan.DAY.valueOf());
    [0, day].forEach((start, i) =>
      stack.driver.update([...base, "x1", "y1", `l${i}`], line.Line.TYPE, {
        x: seriesSpec([start, start + minute]),
        y: seriesSpec([0, 1]),
        color: "#ff0000",
      }),
    );
    const plot = stack.driver.find<LinePlot>(base);
    renderPlot(recorder);
    const texts = (): string[] =>
      recorder.upper2d.calls
        .filter((c) => c.op === "fillText")
        .map((c) => String(c.args[0]));
    const lines = (): number =>
      recorder.upper2d.calls.filter((c) => c.op === "lineTo").length;
    const update = (key: string, type: string, state: Record<string, unknown>) => {
      recorder.upper2d.calls.length = 0;
      stack.driver.update([...base, key], type, state);
      renderPlot(recorder);
    };
    return { plot, recorder, stack, base, setOpenBreaks, texts, lines, update };
  };

  it("should count the open and closed breaks of its first x axis", () => {
    const { plot, recorder, setOpenBreaks } = mount();
    expect(plot.state.breaks).toEqual({ open: 0, closed: 1 });
    const change = plot.getBreakChange({
      action: { variant: "all", open: true },
      viewport: box.DECIMAL,
      held: [],
      selection: box.ZERO,
    });
    setOpenBreaks(change?.openBreaks ?? []);
    renderPlot(recorder);
    expect(plot.state.breaks).toEqual({ open: 1, closed: 0 });
  });

  it("should count the breaks of its first x axis while they are hidden", () => {
    const { plot, recorder, stack, base } = mount();
    stack.driver.update(base, LinePlot.TYPE, {
      container: box.construct(0, 0, 1000, 500),
      viewport: box.DECIMAL,
      grid: {},
      breaksHidden: true,
    });
    renderPlot(recorder);
    expect(plot.state.breaks).toEqual({ open: 0, closed: 1 });
  });

  it("should not count a hidden break outside the manual bounds of x1", () => {
    const { plot, recorder, stack, base } = mount();
    stack.driver.update([...base, "x1"], XAxis.TYPE, {
      location: "bottom",
      type: "time",
      autoBoundPadding: 0,
      bounds: { lower: 0, upper: Number(TimeSpan.MINUTE.valueOf()) * 2 },
      manualBounds: { lower: true, upper: true },
    });
    stack.driver.update(base, LinePlot.TYPE, {
      container: box.construct(0, 0, 1000, 500),
      viewport: box.DECIMAL,
      grid: {},
      breaksHidden: true,
    });
    renderPlot(recorder);
    expect(plot.state.breaks).toEqual({ open: 0, closed: 0 });
  });

  it("should give the seams of its first x axis across the home view", () => {
    const { plot } = mount();
    const seams = plot.getSeams();
    expect(seams).toHaveLength(1);
    expect(seams[0]).toBeCloseTo(0.5, 2);
  });

  it("should list the time that a break hides under the tooltip", () => {
    const { texts, update } = mount();
    update("tooltip", tooltip.Tooltip.TYPE, { position: { x: 500, y: 250 } });
    expect(texts()).toContain("Hidden");
    update("tooltip", tooltip.Tooltip.TYPE, { position: { x: 250, y: 250 } });
    expect(texts()).not.toContain("Hidden");
  });

  it("should not join two measured points across a closed break", () => {
    const { lines, update } = mount();
    const measureAt = (two: number): number => {
      update("measure", measure.Measure.TYPE, {
        one: { x: 100, y: 250 },
        two: { x: two, y: 250 },
        hover: null,
      });
      return lines();
    };
    expect(measureAt(900)).toEqual(measureAt(400) - 1);
  });

  it("should join two measured points of an x axis after the first", () => {
    const recorder = new canvasTest.Recorder();
    (recorder as { gl: unknown }).gl = canvasTest.createGL().gl;
    const stack = buildStack({
      registry: {
        [lineTest.Host.TYPE]: lineTest.Host,
        [LinePlot.TYPE]: LinePlot,
        [XAxis.TYPE]: XAxis,
        [YAxis.TYPE]: YAxis,
        [line.Line.TYPE]: line.Line,
        [measure.Measure.TYPE]: measure.Measure,
      },
      render: recorder,
    });
    teardowns.push(() => stack.driver.delete([aetherTest.ROOT_KEY]));
    const gl = [...stack.basePath, "gl"];
    stack.driver.update(gl, lineTest.Host.TYPE, {});
    const base = [...gl, "plot"];
    stack.driver.update(base, LinePlot.TYPE, {
      container: box.construct(0, 0, 1000, 500),
      viewport: box.DECIMAL,
      grid: {},
    });
    const minute = Number(TimeSpan.MINUTE.valueOf());
    const day = Number(TimeSpan.DAY.valueOf());
    const axis = (axisKey: string, yAxisKey: string) => {
      stack.driver.update([...base, axisKey], XAxis.TYPE, {
        axisKey,
        location: axisKey === "x1" ? "bottom" : "top",
        type: "time",
        autoBoundPadding: 0,
      });
      stack.driver.update([...base, axisKey, yAxisKey], YAxis.TYPE, {
        location: "left",
      });
    };
    // x1 collapses the day between its two lines. The flat line of x2 spans that day,
    // so the measure finds its points at the middle of the plot.
    axis("x1", "y1");
    [0, day].forEach((start, i) =>
      stack.driver.update([...base, "x1", "y1", `l${i}`], line.Line.TYPE, {
        x: seriesSpec([start, start + minute]),
        y: seriesSpec([0, 2]),
        color: "#ff0000",
      }),
    );
    axis("x2", "y2");
    const end = day + minute;
    stack.driver.update([...base, "x2", "y2", "l"], line.Line.TYPE, {
      x: seriesSpec([0, end]),
      y: seriesSpec([1, 1]),
      color: "#0000ff",
    });
    renderPlot(recorder);
    recorder.upper2d.calls.length = 0;
    stack.driver.update([...base, "measure"], measure.Measure.TYPE, {
      one: { x: 1, y: 250 },
      two: { x: 999, y: 250 },
      hover: null,
    });
    renderPlot(recorder);
    const texts = recorder.upper2d.calls
      .filter((c) => c.op === "fillText")
      .map((c) => String(c.args[0]));
    expect(texts).toContain("ΔX");
    expect(texts.filter((t) => t.endsWith(" //"))).toEqual([]);
  });

  it("should lay out every x axis after the first in true time", () => {
    const recorder = new canvasTest.Recorder();
    (recorder as { gl: unknown }).gl = canvasTest.createGL().gl;
    const stack = buildStack({
      registry: {
        [lineTest.Host.TYPE]: lineTest.Host,
        [LinePlot.TYPE]: LinePlot,
        [XAxis.TYPE]: XAxis,
        [YAxis.TYPE]: YAxis,
        [line.Line.TYPE]: line.Line,
      },
      render: recorder,
    });
    teardowns.push(() => stack.driver.delete([aetherTest.ROOT_KEY]));
    const gl = [...stack.basePath, "gl"];
    stack.driver.update(gl, lineTest.Host.TYPE, {});
    const base = [...gl, "plot"];
    stack.driver.update(base, LinePlot.TYPE, {
      container: box.construct(0, 0, 1000, 500),
      viewport: box.DECIMAL,
      grid: {},
    });
    const minute = Number(TimeSpan.MINUTE.valueOf());
    const day = Number(TimeSpan.DAY.valueOf());
    ["x1", "x2"].forEach((axisKey) => {
      stack.driver.update([...base, axisKey], XAxis.TYPE, {
        axisKey,
        location: axisKey === "x1" ? "bottom" : "top",
        type: "time",
        autoBoundPadding: 0,
      });
      stack.driver.update([...base, axisKey, "y1"], YAxis.TYPE, { location: "left" });
      [0, day].forEach((start, i) =>
        stack.driver.update([...base, axisKey, "y1", `l${i}`], line.Line.TYPE, {
          x: seriesSpec([start, start + minute]),
          y: seriesSpec([0, 1]),
          color: "#ff0000",
        }),
      );
    });
    renderPlot(recorder);
    const plot = stack.driver.find<LinePlot>(base);
    const { x1, x2 } = plot.getXBounds({ lower: 0.25, upper: 0.25 });
    expect(x1.lower).toBeLessThan(minute);
    expect(x2.lower).toBeCloseTo((day + minute) / 4, -6);
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
        [LinePlot.TYPE]: LinePlot,
        [XAxis.TYPE]: XAxis,
        [YAxis.TYPE]: YAxis,
        [line.Line.TYPE]: line.Line,
      },
      synnax: { props: testClientParams() },
      render: true,
    });
    const gl = [...stack.basePath, "gl"];
    stack.driver.update(gl, lineTest.Host.TYPE, {});
    const base = [...gl, "plot"];
    const setViewport = (viewport: box.Box): void =>
      stack.driver.update(base, LinePlot.TYPE, {
        container: box.construct(0, 0, 1000, 500),
        viewport,
        grid: {},
      });
    setViewport(box.DECIMAL);
    stack.driver.update([...base, "x1"], XAxis.TYPE, { location: "bottom" });
    stack.driver.update([...base, "x1", "y1"], YAxis.TYPE, { location: "left" });
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
      plot: stack.driver.find<LinePlot>(base),
      xBounds: () => l1.xBounds().bounds,
      setViewport,
      render: () => renderPlot(recorder),
    };
  };

  // The distance from each target to the first drawn sample at or after it.
  const gaps = (plot: LinePlot, targets: number[]): number[] =>
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
    const recorder = new canvasTest.Recorder();
    const { gl, calls } = canvasTest.createGL();
    (recorder as { gl: unknown }).gl = gl;
    const stack = buildStack({
      registry: {
        [lineTest.Host.TYPE]: lineTest.Host,
        [LinePlot.TYPE]: LinePlot,
        [XAxis.TYPE]: XAxis,
        [YAxis.TYPE]: YAxis,
        [line.Line.TYPE]: line.Line,
      },
      render: recorder,
    });
    teardowns.push(() => stack.driver.delete([aetherTest.ROOT_KEY]));
    const host = [...stack.basePath, "gl"];
    stack.driver.update(host, lineTest.Host.TYPE, {});
    const base = [...host, "plot"];
    stack.driver.update(base, LinePlot.TYPE, {
      container: box.construct(0, 0, 1000, 500),
      viewport: box.DECIMAL,
      grid: {},
    });
    stack.driver.update([...base, "x1"], XAxis.TYPE, { location: "bottom" });
    stack.driver.update([...base, "x1", "y1"], YAxis.TYPE, { location: "left" });
    stack.driver.update([...base, "x1", "y2"], YAxis.TYPE, { location: "right" });
    stack.driver.update([...base, "x1", "y1", "raised"], line.Line.TYPE, {
      x: seriesSpec([0, 10]),
      y: seriesSpec([1, 2]),
      color: RAISED,
      raised: true,
    });
    stack.driver.update([...base, "x1", "y2", "plain"], line.Line.TYPE, {
      x: seriesSpec([0, 10]),
      y: seriesSpec([3, 4]),
      color: "#ff0000",
    });
    renderPlot(recorder);
    const colors = calls
      .filter((c) => c.op === "uniform4fv")
      .map((c) => [...(c.args[1] as Iterable<number>)]);
    expect(colors.length).toBeGreaterThan(1);
    expect(colors.at(-1)).toEqual(color.rgba1(color.construct(RAISED)));
  });
});
