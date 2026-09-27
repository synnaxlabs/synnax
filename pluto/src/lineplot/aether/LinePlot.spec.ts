// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, TEST_CLIENT_PARAMS } from "@synnaxlabs/client/testutil";
import { box, DataType, id, TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { afterEach, describe, expect, it } from "vitest";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { aetherTest } from "@/aether/test";
import { LinePlot } from "@/lineplot/aether/LinePlot";
import { XAxis } from "@/lineplot/aether/XAxis";
import { YAxis } from "@/lineplot/aether/YAxis";
import { telem } from "@/telem/aether";
import { buildStack } from "@/testutil/providers";
import { Context as LineContext, Line } from "@/vis/line/aether/line";
import { render } from "@/vis/render";
import { canvasTest } from "@/vis/render/test";

const stubLineStateZ = z.object({
  loading: z.boolean(),
  fetching: z.boolean().default(false),
});

// Line needs a live GL context, so a stub with its TYPE stands in for the walk.
class StubLine extends aether.Leaf<typeof stubLineStateZ> {
  static readonly TYPE = Line.TYPE;
  schema = stubLineStateZ;

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

interface Mount {
  plot: LinePlot;
  recorder: canvasTest.Recorder;
  setLineLoading: (loading: boolean) => void;
  setLineFetching: (fetching: boolean) => void;
  pump: () => void;
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
    const pump = (): void => {
      const entry = recorder.loopCalls
        .map((call) => call.args[0] as LoopEntry)
        .reverse()
        .find((e) => e.key?.startsWith(LinePlot.TYPE) ?? false);
      entry?.render?.();
    };
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
      pump,
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
});

const lineContextProviderStateZ = z.object({});

// Creates the line GL context the way the Canvas component does in production.
class LineContextProvider extends aether.Composite<typeof lineContextProviderStateZ> {
  static readonly TYPE = "LineContextProvider";
  schema = lineContextProviderStateZ;

  afterUpdate(ctx: aether.Context): void {
    LineContext.create(ctx, render.Context.use(ctx));
  }
}

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

  const mount = (props: telem.TiledChannelDataProps) => {
    const stack = buildStack({
      registry: {
        [telem.PROVIDER_TYPE]: telem.Provider,
        [LineContextProvider.TYPE]: LineContextProvider,
        [LinePlot.TYPE]: LinePlot,
        [XAxis.TYPE]: XAxis,
        [YAxis.TYPE]: YAxis,
        [Line.TYPE]: Line,
      },
      synnax: { props: TEST_CLIENT_PARAMS },
      render: true,
    });
    const gl = [...stack.basePath, "gl"];
    stack.driver.update(gl, LineContextProvider.TYPE, {});
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
    stack.driver.update([...base, "x1", "y1", "l1"], Line.TYPE, {
      x: telem.tiledChannelData({ ...props, useIndexOfChannel: true }),
      y: telem.tiledChannelData(props),
      color: "#ff0000",
    });
    teardowns.push(() => stack.driver.delete([aetherTest.ROOT_KEY]));
    return { plot: stack.driver.find<LinePlot>(base), setViewport };
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
    const { plot, setViewport } = mount({ channel: data.key, timeRange: home });
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
    const { plot } = mount({ channel: data.key, timeSpan: TimeSpan.seconds(10) });
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
        await new Promise((resolve) => setTimeout(resolve, 100));
      }
      const target = Number(last.valueOf());
      await expect.poll(() => gaps(plot, [target])[0], { timeout: 10_000 }).toBe(0);
    } finally {
      await writer.close();
    }
  });
});
