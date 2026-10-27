// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { SyntheticCore } from "@synnaxlabs/client/testutil";
import {
  box,
  dynamicWindow,
  sleep,
  staticWindow,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { type Bench, test } from "vitest";

import { aetherTest } from "@/aether/testutil";
import { LinePlot } from "@/lineplot/aether/LinePlot";
import { XAxis } from "@/lineplot/aether/XAxis";
import { YAxis } from "@/lineplot/aether/YAxis";
import { telem } from "@/telem/aether";
import { telemTest } from "@/telem/aether/testutil";
import { buildStack } from "@/testutil/internal";
import { line } from "@/vis/line/aether";
import { lineTest } from "@/vis/line/aether/testutil";
import { canvasTest } from "@/vis/render/testutil";

// Times one frame of a 1000 by 500 pixel plot of 1 kHz channels, with every tile it
// draws cached. The frame runs through a recording GL context, so the timings cover the
// CPU side of a frame and leave out the GPU.

const START = TimeStamp.seconds(1_750_000_000);
const WEEK = new TimeRange(START, START.add(TimeSpan.days(7)));
const HOUR = new TimeRange(
  START.add(TimeSpan.days(3)),
  START.add(TimeSpan.days(3)).add(TimeSpan.hours(1)),
);
const LIVE_SPAN = TimeSpan.minutes(10);
const FRAME = TimeSpan.milliseconds(16);
const CONTAINER = box.construct(0, 0, 1000, 500);
const LINE_COUNTS = [1, 10, 50];

interface LoopEntry {
  key?: string;
  render?: () => unknown;
}

interface Plot {
  core: SyntheticCore;
  setViewport: (viewport: box.Box) => void;
  frame: () => void;
  close: () => Promise<void>;
}

// viewportOf returns the viewport that shows the range of the week.
const viewportOf = (tr: TimeRange): box.Box => {
  const week = Number(WEEK.span.valueOf());
  const x = Number(tr.start.span(WEEK.start).valueOf()) / week;
  return box.construct(x, 0, Number(tr.span.valueOf()) / week, 1);
};

const yieldTask = async (): Promise<void> =>
  await new Promise((resolve) => setImmediate(resolve));

const mount = async (
  lines: number,
  props: Omit<telem.ChannelDataProps, "channel">,
  viewport: box.Box,
): Promise<Plot> => {
  const now =
    props.window.variant === "dynamic" ? WEEK.start.add(TimeSpan.hours(1)) : WEEK.end;
  const core = new SyntheticCore({ rate: 1000, start: START, now, channels: lines });
  const client = telemTest.createClient(core);
  const recorder = canvasTest.record();
  recorder.resize(CONTAINER, 1);
  const stack = buildStack({
    registry: {
      [lineTest.Host.TYPE]: lineTest.Host,
      [LinePlot.TYPE]: LinePlot,
      [XAxis.TYPE]: XAxis,
      [YAxis.TYPE]: YAxis,
      [line.Line.TYPE]: line.Line,
    },
    render: recorder,
    telem: { factories: [new telem.RemoteFactory(client)] },
  });
  const host = [...stack.basePath, "gl"];
  stack.driver.update(host, lineTest.Host.TYPE, {});
  const base = [...host, "plot"];
  const setViewport = (viewport: box.Box): void =>
    stack.driver.update(base, LinePlot.TYPE, {
      container: CONTAINER,
      viewport,
      grid: {},
    });
  setViewport(viewport);
  const yAxis = [...base, "x1", "y1"];
  stack.driver.update([...base, "x1"], XAxis.TYPE, { location: "bottom" });
  stack.driver.update(yAxis, YAxis.TYPE, { location: "left" });
  const linePaths = core.data.map((ch) => {
    const path = [...yAxis, `l${ch.key}`];
    stack.driver.update(path, line.Line.TYPE, {
      x: telem.channelData({ ...props, channel: ch.key, useIndexOfChannel: true }),
      y: telem.channelData({ ...props, channel: ch.key }),
      color: "#ff0000",
    });
    return path;
  });
  const entry = recorder.loopCalls
    .map((call) => call.args[0] as LoopEntry)
    .findLast((e) => e.key?.startsWith(LinePlot.TYPE) ?? false);
  if (entry?.render == null) throw new Error("plot did not request a render");
  const { render } = entry;
  const frame = (): void => {
    recorder.clear();
    render();
  };
  const plot: Plot = {
    core,
    setViewport,
    frame,
    close: async () => {
      stack.driver.delete([aetherTest.ROOT_KEY]);
      await client.feed.close();
    },
  };
  await settle(
    plot,
    linePaths.map((p) => stack.driver.find<line.Line>(p)),
  );
  const drawn = recorder.gl.calls.filter((c) => c.op === "drawArraysInstanced").length;
  if (drawn < lines) throw new Error(`plot drew ${drawn} strips for ${lines} lines`);
  return plot;
};

// settle draws frames until every line holds everything its view draws.
const settle = async (plot: Plot, lines: line.Line[]): Promise<void> => {
  const deadline = TimeStamp.now().add(TimeSpan.seconds(60));
  while (TimeStamp.now().before(deadline)) {
    plot.frame();
    if (lines.every((l) => !l.loading && !l.fetching)) return;
    await sleep.sleep(TimeSpan.milliseconds(1));
  }
  throw new Error("plot did not settle");
};

const compareLineCounts = async (
  bench: Bench,
  create: (lines: number) => Promise<() => unknown>,
): Promise<void> => {
  const frames = [];
  for (const lines of LINE_COUNTS)
    frames.push(bench(`${lines} lines`, await create(lines)));
  await bench.compare(...frames);
};

const plots: Plot[] = [];
const closeAll = async (): Promise<void> => {
  await Promise.all(plots.splice(0).map(async (p) => await p.close()));
};

const createStatic = (viewport: box.Box) => async (lines: number) => {
  const plot = await mount(lines, { window: staticWindow(WEEK) }, viewport);
  plots.push(plot);
  return plot.frame;
};

test("static week, week view", async ({ bench }) => {
  await compareLineCounts(bench, createStatic(box.DECIMAL));
  await closeAll();
});

test("static week, hour view", async ({ bench }) => {
  await compareLineCounts(bench, createStatic(viewportOf(HOUR)));
  await closeAll();
});

// Each frame of a live plot draws 16 ms of new samples, which the stream lands before
// it. The yield that lands them costs about 15 µs.
test("live 10 minutes", async ({ bench }) => {
  await compareLineCounts(bench, async (lines) => {
    const plot = await mount(lines, { window: dynamicWindow(LIVE_SPAN) }, box.DECIMAL);
    plots.push(plot);
    return async () => {
      plot.core.advance(FRAME);
      await yieldTask();
      plot.frame();
    };
  });
  await closeAll();
});

// A pan moves the hour view by a sixtieth of its span per frame. It sweeps one day
// back and forth, so after the first sweep every tile it crosses comes from the cache.
// The yield that lands those tiles costs about 15 µs.
test("pan an hour view", async ({ bench }) => {
  await compareLineCounts(bench, async (lines) => {
    const start = viewportOf(HOUR);
    const step = box.width(start) / 60;
    const steps = Math.round(
      Number(TimeSpan.days(1).valueOf() / HOUR.span.valueOf()) * 60,
    );
    let i = 0;
    const next = (): box.Box => {
      const phase = i++ % (2 * steps);
      const offset = (phase < steps ? phase : 2 * steps - phase) * step;
      return box.translate(start, { x: offset, y: 0 });
    };
    const plot = await mount(lines, { window: staticWindow(WEEK) }, start);
    plots.push(plot);
    const pan = async (): Promise<void> => {
      plot.setViewport(next());
      await yieldTask();
      plot.frame();
    };
    for (let s = 0; s < 2 * steps; s++) await pan();
    // Reads still in flight after the sweep land in the cache.
    await sleep.sleep(TimeSpan.milliseconds(200));
    return pan;
  });
  await closeAll();
});
