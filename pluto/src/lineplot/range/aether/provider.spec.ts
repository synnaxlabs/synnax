// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, type ranger as rangerClient } from "@synnaxlabs/client";
import { createTestClient, testClientParams } from "@synnaxlabs/client/testutil";
import {
  box,
  color as xcolor,
  id,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { describe, expect, it, onTestFinished, vi } from "vitest";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { Layout } from "@/lineplot/aether/layout";
import { range } from "@/lineplot/range/aether";
import { ranger } from "@/ranger/aether";
import { renderAether } from "@/testutil/internal";
import { render } from "@/vis/render";
import { canvasTest } from "@/vis/render/testutil";

const REGION = box.construct({ x: 0, y: 0 }, { width: 1000, height: 500 });
const PROVIDER_KEY = "annotations";

const client = createTestClient();

// The renders the provider requests. The test draws by calling render itself.
const requested = vi.fn<render.Requestor>();

/** Supplies the render requestor the provider reads, which production gets from the
 * plot's canvas. */
class Requestor extends aether.Composite<typeof Requestor.stateZ> {
  static readonly TYPE = "range-provider-test-requestor";
  static readonly stateZ = z.object({});
  override schema = Requestor.stateZ;

  override afterUpdate(ctx: aether.Context): void {
    render.control(ctx, requested);
  }
}

/** @param shown - The time range the plot region shows, wider when zoomed out. */
const renderProps = (
  timeRange: TimeRange,
  shown: TimeRange = timeRange,
): range.ProviderProps => ({
  layout: new Layout([shown.numericBounds], box.DECIMAL),
  region: REGION,
  timeRange,
  raised: [],
});

const mount = (recorder = canvasTest.record()): range.Provider => {
  const h = renderAether(Requestor, {
    state: {},
    synnax: { props: testClientParams() },
    render: recorder,
    registry: { [range.Provider.TYPE]: range.Provider },
    children: {
      [PROVIDER_KEY]: {
        type: range.Provider.TYPE,
        state: { cursor: null, hovered: null, count: 0 },
      },
    },
  });
  return h.child<range.Provider>(PROVIDER_KEY);
};

// The strip draws every range in its window, so each test puts its ranges at a random
// instant in years that ranges near now, and open ranges that start later, never reach.
// The specs of running ranges set the provider's clock to that instant.
const ANCHORS = new TimeRange(new TimeStamp([1975, 1, 1]), new TimeStamp([1995, 1, 1]));
// An open range overlaps every later window, so open ranges start after every anchor.
const OPEN_ANCHORS = new TimeRange(
  new TimeStamp([1995, 1, 1]),
  new TimeStamp([2000, 1, 1]),
);

const createAnchor = (anchors: TimeRange = ANCHORS): TimeStamp =>
  anchors.start.add(TimeSpan.seconds(Math.floor(Math.random() * anchors.span.seconds)));

// The first draw waits for the fetch of a new client, which takes seconds on a loaded
// runner.
const pollDrawn = <T>(draw: () => T) => expect.poll(draw, { timeout: 10_000 });

/** Creates the ranges, and deletes them once the test finishes. */
const create = async (ranges: rangerClient.New[]): Promise<rangerClient.Range[]> => {
  const created = await client.ranges.create(ranges);
  onTestFinished(async () => await client.ranges.delete(created.map((r) => r.key)));
  return created;
};

/** Creates one range per [start, end] pair, in seconds past start. */
const createRanges = async (start: TimeStamp, spans: [number, number][]) =>
  await create(
    spans.map(([from, to]) => ({
      name: `lane-${id.create()}`,
      timeRange: new TimeRange(
        start.add(TimeSpan.seconds(from)),
        start.add(TimeSpan.seconds(to)),
      ),
      color: { type: "fixed" as const, value: "#7C3AED" },
    })),
  );

describe("Provider", () => {
  // A window an Arc fills with ranges overlaps far more than the strip can draw, and
  // writing the unbounded answer through the cache stalls the worker for seconds,
  // which freezes every plot in the window.
  it("stops drawing annotations once the window passes the cap", async () => {
    const start = createAnchor();
    const timeRange = new TimeRange(start, start.add(TimeSpan.seconds(1)));
    const createMany = async (count: number) =>
      await create(
        Array.from({ length: count }, () => ({
          name: `annotation-${id.create()}`,
          timeRange,
          color: { type: "fixed" as const, value: "#7C3AED" },
        })),
      );
    await createMany(5);
    const provider = mount();
    const props = renderProps(timeRange);
    const drawn = () => {
      provider.render(props);
      return provider.state.count;
    };
    await pollDrawn(drawn).toEqual(5);
    // One past the cap of 100.
    await createMany(96);
    await pollDrawn(drawn).toEqual(0);
  });

  it("draws an open range from its start through every later viewport", async () => {
    const start = createAnchor(OPEN_ANCHORS);
    const [rng] = await create([
      {
        name: `open-${id.create()}`,
        timeRange: new TimeRange(start, TimeStamp.MAX),
        color: { type: "fixed", value: "#7C3AED" },
      },
    ]);
    // A fresh provider per viewport, since a requery keeps the previous answer until
    // the new one lands. The open ranges of other runs can reach these viewports, so
    // the spec looks for the name of its own.
    const names = (viewportStart: TimeStamp) => {
      const recorder = canvasTest.record();
      const provider = mount(recorder);
      const props = renderProps(
        new TimeRange(viewportStart, viewportStart.add(TimeSpan.seconds(30))),
      );
      return () => {
        recorder.upper2d.calls.length = 0;
        provider.render(props);
        return recorder.upper2d.calls
          .filter((c) => c.op === "fillText")
          .map((c) => c.args[0]);
      };
    };
    await pollDrawn(names(start.sub(TimeSpan.seconds(10)))).toContain(rng.name);
    await pollDrawn(names(start.add(TimeSpan.hours(1)))).toContain(rng.name);
  });

  it("stacks overlapping ranges in lanes above the plot", async () => {
    const start = createAnchor();
    const ranges = await createRanges(start, [
      [0, 10],
      [2, 6],
      [4, 8],
    ]);
    const provider = mount();
    const props = renderProps(new TimeRange(start, start.add(TimeSpan.seconds(10))));
    await pollDrawn(() => {
      provider.render(props);
      return provider.state.height;
    }).toEqual(72);
    // The 2 to 6 second range holds the middle lane, from 46 to 26 px above the plot.
    provider.setState((s) => ({ ...s, cursor: { x: 300, y: -36 } }));
    provider.render(props);
    expect(provider.state.hovered?.key).toEqual(ranges[1].key);
  });

  it("keeps the range of an open menu hovered as the cursor leaves it", async () => {
    const start = createAnchor();
    const ranges = await createRanges(start, [
      [0, 4],
      [6, 10],
    ]);
    const provider = mount();
    const shown = new TimeRange(start, start.add(TimeSpan.seconds(10)));
    const props = renderProps(shown);
    await pollDrawn(() => {
      provider.render(props);
      return provider.state.count;
    }).toEqual(2);
    provider.setState((s) => ({ ...s, cursor: { x: 200, y: -14 } }));
    provider.render(props);
    expect(provider.state.hovered?.key).toEqual(ranges[0].key);
    provider.setState((s) => ({ ...s, menuOpen: true, cursor: { x: 800, y: -14 } }));
    // The plot slides by two seconds, as a live plot does.
    const two = TimeSpan.seconds(2);
    provider.render(
      renderProps(new TimeRange(shown.start.add(two), shown.end.add(two))),
    );
    expect(provider.state.hovered?.key).toEqual(ranges[0].key);
    expect(provider.state.hovered?.viewport.lower).toBeCloseTo(-0.2);
    provider.setState((s) => ({ ...s, menuOpen: false }));
    provider.render(props);
    expect(provider.state.hovered?.key).toEqual(ranges[1].key);
  });

  it("hides the tooltip while the menu is open", async () => {
    const start = createAnchor();
    const [rng] = await createRanges(start, [[0, 10]]);
    const recorder = canvasTest.record();
    const provider = mount(recorder);
    const props = renderProps(new TimeRange(start, start.add(TimeSpan.seconds(10))));
    await pollDrawn(() => {
      provider.render(props);
      return provider.state.count;
    }).toEqual(1);
    const names = (): number => {
      recorder.upper2d.calls.length = 0;
      const raised: Array<() => void> = [];
      provider.render({ ...props, raised });
      raised.forEach((draw) => draw());
      return recorder.upper2d.calls.filter(
        (c) => c.op === "fillText" && c.args[0] === rng.name,
      ).length;
    };
    provider.setState((s) => ({ ...s, cursor: { x: 500, y: -14 } }));
    // The bar label and the tooltip.
    expect(names()).toEqual(2);
    provider.setState((s) => ({ ...s, menuOpen: true }));
    expect(names()).toEqual(1);
  });

  it("keeps back to back ranges in one lane at any zoom", async () => {
    const start = createAnchor();
    await createRanges(start, [
      [0, 4],
      [4, 6],
      [6.5, 10],
    ]);
    const provider = mount();
    const bounds = new TimeRange(start, start.add(TimeSpan.seconds(10)));
    // Zoomed out so far that the half second gap is under a pixel wide.
    const shown = new TimeRange(
      start.sub(TimeSpan.seconds(1000)),
      start.add(TimeSpan.seconds(1000)),
    );
    await pollDrawn(() => {
      provider.render(renderProps(bounds, shown));
      return provider.state.count;
    }).toEqual(3);
    provider.render(renderProps(bounds));
    expect(provider.state.height).toEqual(28);
    provider.render(renderProps(bounds, shown));
    expect(provider.state.height).toEqual(28);
  });

  it("draws no bar for a range with no color", async () => {
    const start = createAnchor();
    await createRanges(start, [[0, 4]]);
    await create([
      {
        name: `uncolored-${id.create()}`,
        timeRange: new TimeRange(
          start.add(TimeSpan.seconds(5)),
          start.add(TimeSpan.seconds(8)),
        ),
      },
    ]);
    const provider = mount();
    const props = renderProps(new TimeRange(start, start.add(TimeSpan.seconds(10))));
    await pollDrawn(() => {
      provider.render(props);
      return provider.state.count;
    }).toEqual(1);
    provider.setState((s) => ({ ...s, cursor: { x: 650, y: -14 } }));
    provider.render(props);
    expect(provider.state.hovered).toBeNull();
  });

  it("draws a bar for a range with a named color", async () => {
    const start = createAnchor();
    const named = await client.colors.create({
      name: `strip-${id.create()}`,
      light: "#7C3AED",
    });
    onTestFinished(async () => await client.colors.delete(named.key));
    await create([
      {
        name: `named-${id.create()}`,
        timeRange: new TimeRange(start, start.add(TimeSpan.seconds(4))),
        color: { type: "named", key: named.key },
      },
    ]);
    const provider = mount();
    const props = renderProps(new TimeRange(start, start.add(TimeSpan.seconds(10))));
    await pollDrawn(() => {
      provider.render(props);
      return provider.state.count;
    }).toEqual(1);
  });

  it("repaints a bar when its named color changes", async () => {
    const start = createAnchor();
    const named = await client.colors.create({
      name: `strip-${id.create()}`,
      light: "#7C3AED",
    });
    onTestFinished(async () => await client.colors.delete(named.key));
    await create([
      {
        name: `repaint-${id.create()}`,
        timeRange: new TimeRange(start, start.add(TimeSpan.seconds(4))),
        color: { type: "named", key: named.key },
      },
    ]);
    const recorder = canvasTest.record();
    const provider = mount(recorder);
    const props = renderProps(new TimeRange(start, start.add(TimeSpan.seconds(10))));
    const fills = (): unknown[] => {
      recorder.upper2d.calls.length = 0;
      provider.render(props);
      return recorder.upper2d.calls
        .filter(({ op }) => op === "set:fillStyle")
        .map(({ args }) => args[0]);
    };
    let before: unknown[] = [];
    await pollDrawn(() => {
      before = fills();
      return provider.state.count;
    }).toEqual(1);
    await client.colors.dispatch(named.key, [
      color.set({ path: ["light"], value: xcolor.construct("#123456") }),
    ]);
    await expect.poll(fills).not.toEqual(before);
  });

  it("keeps a range below the range that holds it", async () => {
    const start = createAnchor();
    const ranges = await createRanges(start, [
      [0, 5],
      [2, 30],
      [5, 10],
    ]);
    const provider = mount();
    const props = renderProps(new TimeRange(start, start.add(TimeSpan.seconds(30))));
    await pollDrawn(() => {
      provider.render(props);
      return provider.state.height;
    }).toEqual(72);
    // The 5 to 10 second range holds the third lane, from 24 to 4 px above the plot.
    provider.setState((s) => ({ ...s, cursor: { x: 250, y: -14 } }));
    provider.render(props);
    expect(provider.state.hovered?.key).toEqual(ranges[2].key);
  });

  it("draws ranges past the third lane in thin lanes", async () => {
    const start = createAnchor();
    const ranges = await createRanges(start, [
      [0, 10],
      [1, 9],
      [2, 8],
      [3, 7],
      [4, 6],
    ]);
    const provider = mount();
    const props = renderProps(new TimeRange(start, start.add(TimeSpan.seconds(10))));
    await pollDrawn(() => {
      provider.render(props);
      return provider.state.height;
    }).toEqual(84);
    // The innermost range holds the last thin lane, from 8 to 4 px above the plot.
    provider.setState((s) => ({ ...s, cursor: { x: 500, y: -6 } }));
    provider.render(props);
    expect(provider.state.hovered?.key).toEqual(ranges[4].key);
  });

  it("fills only the ranges that hold no other range", async () => {
    const start = createAnchor();
    await createRanges(start, [
      [0, 10],
      [2, 6],
    ]);
    const recorder = canvasTest.record();
    const provider = mount(recorder);
    const props = renderProps(new TimeRange(start, start.add(TimeSpan.seconds(10))));
    await pollDrawn(() => {
      provider.render(props);
      return provider.state.count;
    }).toEqual(2);
    recorder.upper2d.calls.length = 0;
    provider.render(props);
    const fills = recorder.upper2d.calls.filter((c) => c.op === "rect");
    expect(fills).toHaveLength(1);
    const [x, y, width, height] = fills[0].args as number[];
    expect(x).toBeCloseTo(200);
    expect(width).toBeCloseTo(400);
    expect([y, height]).toEqual([0, 500]);
  });

  // The strip's cap counts ranges in the axis bounds, so a range past them is never
  // drawn, even when a zoomed-out viewport shows it.
  it("draws only the ranges inside the axis bounds", async () => {
    const start = createAnchor();
    await createRanges(start, [
      [2, 6],
      [12, 14],
    ]);
    const recorder = canvasTest.record();
    const provider = mount(recorder);
    const bounds = new TimeRange(start, start.add(TimeSpan.seconds(10)));
    const shown = new TimeRange(
      start.sub(TimeSpan.seconds(10)),
      start.add(TimeSpan.seconds(20)),
    );
    const props = renderProps(bounds, shown);
    await pollDrawn(() => {
      provider.render(props);
      return provider.state.count;
    }).toEqual(1);
    recorder.upper2d.calls.length = 0;
    provider.render(props);
    expect(recorder.upper2d.calls.filter((c) => c.op === "rect")).toHaveLength(1);
  });

  it("requeries a live window once per span, not on every sample", () => {
    const retrieve = vi.spyOn(ranger.listDefinition, "retrieve");
    try {
      const provider = mount();
      const span = TimeSpan.seconds(30);
      const start = TimeStamp.now();
      for (let elapsed = 0; elapsed < 60; elapsed += 2) {
        const end = start.add(TimeSpan.seconds(elapsed));
        provider.render(renderProps(new TimeRange(end.sub(span), end)));
      }
      expect(retrieve).toHaveBeenCalledTimes(2);
    } finally {
      retrieve.mockRestore();
    }
  });

  it("keeps the window for a viewport that ends at the window's end", () => {
    const retrieve = vi.spyOn(ranger.listDefinition, "retrieve");
    try {
      const provider = mount();
      const end = TimeStamp.MAX;
      const ending = (span: TimeSpan) =>
        provider.render(renderProps(new TimeRange(end.sub(span), end)));
      ending(TimeSpan.seconds(30));
      ending(TimeSpan.seconds(28));
      expect(retrieve).toHaveBeenCalledTimes(1);
    } finally {
      retrieve.mockRestore();
    }
  });

  it("requeries once the viewport zooms in past 2x", () => {
    const retrieve = vi.spyOn(ranger.listDefinition, "retrieve");
    try {
      const provider = mount();
      const start = TimeStamp.now();
      const zoomed = (span: TimeSpan) =>
        provider.render(renderProps(new TimeRange(start, start.add(span))));
      zoomed(TimeSpan.seconds(60));
      zoomed(TimeSpan.seconds(40));
      expect(retrieve).toHaveBeenCalledTimes(1);
      zoomed(TimeSpan.seconds(20));
      expect(retrieve).toHaveBeenCalledTimes(2);
    } finally {
      retrieve.mockRestore();
    }
  });

  describe("running ranges", () => {
    const createRunning = async (now: TimeStamp): Promise<rangerClient.Range> => {
      const [rng] = await create([
        {
          name: `running-${id.create()}`,
          timeRange: new TimeRange(
            now.sub(TimeSpan.seconds(90)),
            now.add(TimeSpan.hours(1)),
          ),
          color: { type: "fixed", value: "#7C3AED" },
        },
      ]);
      return rng;
    };
    const lastMinute = (now: TimeStamp): range.ProviderProps => ({
      ...renderProps(
        new TimeRange(now.sub(TimeSpan.seconds(120)), now.sub(TimeSpan.seconds(60))),
      ),
      now,
    });
    const ahead = (now: TimeStamp): TimeRange =>
      new TimeRange(now.add(TimeSpan.minutes(1)), now.add(TimeSpan.minutes(20)));

    it("should show the time since its start at the end of its bar", async () => {
      const now = createAnchor();
      const rng = await createRunning(now);
      const recorder = canvasTest.record();
      const provider = mount(recorder);
      const props = lastMinute(now);
      const texts = (): string[] => {
        recorder.upper2d.calls.length = 0;
        const raised: Array<() => void> = [];
        provider.render({ ...props, raised });
        raised.forEach((draw) => draw());
        return recorder.upper2d.calls
          .filter((c) => c.op === "fillText")
          .map((c) => c.args[0] as string);
      };
      await pollDrawn(texts).toContain(rng.name);
      expect(texts()).toContain("T+0:01:30");
      // The bar runs past the view, so it has no head.
      expect(recorder.upper2d.calls.some((c) => c.op === "arcTo")).toBe(false);
      provider.setState((s) => ({ ...s, cursor: { x: 800, y: -14 } }));
      provider.render(props);
      expect(provider.state.hovered?.key).toEqual(rng.key);
      expect(texts().some((t) => t.startsWith("Running for "))).toBe(true);
    });

    it("should end its bar at the newest sample of a live plot", async () => {
      const now = createAnchor();
      const rng = await createRunning(now);
      const recorder = canvasTest.record();
      const provider = mount(recorder);
      const props = { ...lastMinute(now), newest: now.sub(TimeSpan.seconds(75)) };
      await pollDrawn(() => {
        provider.render(props);
        return provider.state.count;
      }).toEqual(1);
      recorder.upper2d.calls.length = 0;
      provider.render(props);
      // The range starts at 500 px, and the newest sample sits at 750 px.
      const bar = recorder.upper2d.calls.find(
        (c) => c.op === "roundRect" && Math.abs((c.args[0] as number) - 500) < 0.01,
      );
      if (bar == null) throw new Error("no bar drawn from 500 px");
      const [x, , width] = bar.args as number[];
      expect(x + width).toBeCloseTo(750);
      // The head bends around both right corners, 1 px inside the bar's end.
      const corners = recorder.upper2d.calls.filter((c) => c.op === "arcTo");
      expect(corners).toHaveLength(2);
      corners.forEach((c) => expect(c.args[0]).toBeCloseTo(749));
      // Zoom to range ends where the bar ends.
      provider.setState((s) => ({ ...s, cursor: { x: 600, y: -14 } }));
      provider.render(props);
      expect(provider.state.hovered?.key).toEqual(rng.key);
      expect(provider.state.hovered?.viewport.upper).toBeCloseTo(0.75);
    });

    it("should redraw as the strip changes, and stop when it hides", async () => {
      const now = createAnchor();
      await createRunning(now);
      try {
        const provider = mount();
        const live = { ...lastMinute(now), newest: now.sub(TimeSpan.seconds(75)) };
        let props: range.ProviderProps = live;
        provider.setState((s) => ({ ...s, visible: false }));
        await pollDrawn(() => {
          provider.render(props);
          return provider.state.count;
        }).toEqual(1);
        vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout"] });
        requested.mockImplementation(() => provider.render(props));
        const redraws = (): number => {
          requested.mockClear();
          vi.advanceTimersByTime(1000);
          return requested.mock.calls.length;
        };
        provider.render(props);
        expect(redraws()).toEqual(0);
        provider.setState((s) => ({ ...s, visible: true }));
        provider.render(props);
        expect(redraws()).toEqual(15);
        // With no live samples the bar runs to now, past the view, so only T+ changes.
        props = lastMinute(now);
        provider.render(props);
        expect(redraws()).toEqual(1);
        provider.setState((s) => ({ ...s, visible: false }));
        provider.render(props);
        expect(redraws()).toEqual(0);
      } finally {
        requested.mockReset();
        vi.useRealTimers();
      }
    });

    // Expects one redraw of the strip at the given time, and none before it.
    const expectRedrawAt = async (
      now: TimeStamp,
      view: TimeRange,
      at: TimeStamp,
    ): Promise<void> => {
      const provider = mount();
      const props = { ...renderProps(view), now };
      await pollDrawn(() => {
        provider.render(props);
        return provider.state.count;
      }).toEqual(1);
      vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout"] });
      try {
        provider.render(props);
        const wait = new TimeRange(now, at).span.milliseconds;
        requested.mockClear();
        vi.advanceTimersByTime(wait - 100);
        expect(requested).not.toHaveBeenCalled();
        vi.advanceTimersByTime(200);
        expect(requested).toHaveBeenCalledTimes(1);
      } finally {
        vi.useRealTimers();
      }
    };

    it("should redraw when a range in view starts", async () => {
      const now = createAnchor();
      // The range starts before the view, so its start, not the view's, wakes the
      // strip.
      const [rng] = await create([
        {
          name: `planned-${id.create()}`,
          timeRange: new TimeRange(
            now.add(TimeSpan.seconds(30)),
            now.add(TimeSpan.minutes(10)),
          ),
          color: { type: "fixed", value: "#7C3AED" },
        },
      ]);
      await expectRedrawAt(now, ahead(now), rng.timeRange.start);
    });

    it("should redraw when now reaches a view ahead of a running range", async () => {
      const now = createAnchor();
      await createRunning(now);
      const view = ahead(now);
      await expectRedrawAt(now, view, view.start);
    });
  });

  it("should show the length of an ended range at the end of its bar", async () => {
    const start = createAnchor();
    const [rng] = await createRanges(start, [[2, 6]]);
    const recorder = canvasTest.record();
    const provider = mount(recorder);
    const props = {
      ...renderProps(new TimeRange(start, start.add(TimeSpan.seconds(10)))),
      now: start.add(TimeSpan.minutes(1)),
    };
    const texts = (): unknown[] => {
      recorder.upper2d.calls.length = 0;
      provider.render(props);
      return recorder.upper2d.calls
        .filter((c) => c.op === "fillText")
        .map((c) => c.args[0]);
    };
    await pollDrawn(texts).toContain(rng.name);
    expect(texts()).toContain("4s");
  });

  it("should fade only the bars that the plot edge cuts", async () => {
    const start = createAnchor();
    await createRanges(start, [
      [0, 5],
      [6, 9],
    ]);
    const recorder = canvasTest.record();
    const provider = mount(recorder);
    const props = renderProps(
      new TimeRange(start, start.add(TimeSpan.seconds(10))),
      new TimeRange(start.add(TimeSpan.seconds(2)), start.add(TimeSpan.seconds(10))),
    );
    await pollDrawn(() => {
      provider.render(props);
      return provider.state.count;
    }).toEqual(2);
    recorder.upper2d.calls.length = 0;
    provider.render(props);
    const gradients = recorder.upper2d.calls.filter(
      (c) => c.op === "createLinearGradient",
    );
    // The 0 to 5 second range runs from the left edge to 375 px.
    expect(gradients.length).toBeGreaterThan(0);
    gradients.forEach((g) => {
      const [x0, y0, x1, y1] = g.args as number[];
      expect([x0, y0, y1]).toEqual([0, 0, 0]);
      expect(x1).toBeCloseTo(375);
    });
  });

  describe("breaks", () => {
    // Props for an x axis with a segment for each [lower, upper] pair, in seconds past
    // start.
    const segmentedProps = (
      start: TimeStamp,
      segments: [number, number][],
    ): range.ProviderProps => {
      const at = (s: number) => Number(start.add(TimeSpan.seconds(s)).valueOf());
      const b = segments.map(([lower, upper]) => ({
        lower: at(lower),
        upper: at(upper),
      }));
      return {
        layout: new Layout(b, box.DECIMAL),
        region: REGION,
        timeRange: new TimeRange(b[0].lower, b[b.length - 1].upper),
        raised: [],
      };
    };

    it("should leave its tooltip to a raised draw over the bands", async () => {
      const start = createAnchor();
      const [rng] = await createRanges(start, [[0, 10]]);
      const recorder = canvasTest.record();
      const provider = mount(recorder);
      const props = segmentedProps(start, [
        [0, 10],
        [3600, 3610],
      ]);
      await pollDrawn(() => {
        provider.render(props);
        return provider.state.count;
      }).toEqual(1);
      const names = (): number =>
        recorder.upper2d.calls.filter(
          (c) => c.op === "fillText" && c.args[0] === rng.name,
        ).length;
      provider.setState((s) => ({ ...s, cursor: { x: 250, y: -14 } }));
      recorder.upper2d.calls.length = 0;
      const raised: Array<() => void> = [];
      provider.render({ ...props, raised });
      // The bar label.
      expect(names()).toEqual(1);
      raised.forEach((draw) => draw());
      expect(names()).toEqual(2);
    });

    it("should cut a bar at each seam that its range crosses", async () => {
      const start = createAnchor();
      await createRanges(start, [[5, 25]]);
      const recorder = canvasTest.record();
      const provider = mount(recorder);
      const props = segmentedProps(start, [
        [0, 10],
        [20, 30],
      ]);
      await pollDrawn(() => {
        provider.render(props);
        return provider.state.count;
      }).toEqual(1);
      recorder.upper2d.calls.length = 0;
      provider.render(props);
      const { calls } = recorder.upper2d;
      // A fill and a stroke for each of the two bars.
      expect(calls.filter((c) => c.op === "roundRect")).toHaveLength(4);
      const gradients = calls.filter((c) => c.op === "createLinearGradient");
      expect(gradients.length).toBeGreaterThan(0);
    });

    it("should light every piece of a hovered range that a seam cuts", async () => {
      const start = createAnchor();
      const ranges = await createRanges(start, [[5, 25]]);
      const recorder = canvasTest.record();
      const provider = mount(recorder);
      const props = segmentedProps(start, [
        [0, 10],
        [20, 30],
      ]);
      await pollDrawn(() => {
        provider.render(props);
        return provider.state.count;
      }).toEqual(1);
      provider.setState((s) => ({ ...s, cursor: { x: 400, y: -14 } }));
      recorder.upper2d.calls.length = 0;
      provider.render(props);
      expect(provider.state.hovered?.key).toEqual(ranges[0].key);
      const alphas = recorder.upper2d.calls
        .filter((c) => c.op === "set:globalAlpha")
        .map((c) => c.args[0]);
      expect(alphas.length).toBeGreaterThan(1);
      alphas.forEach((alpha) => expect(alpha).toEqual(1));
    });

    it("should not pulse a running bar whose head a seam cuts off", async () => {
      const now = createAnchor();
      const start = now.sub(TimeSpan.seconds(120));
      await createRanges(start, [[30, 3720]]);
      try {
        const provider = mount();
        const props = {
          ...segmentedProps(start, [
            [0, 40],
            [60, 100],
          ]),
          now,
        };
        await pollDrawn(() => {
          provider.render(props);
          return provider.state.count;
        }).toEqual(1);
        vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout"] });
        requested.mockImplementation(() => provider.render(props));
        provider.render(props);
        requested.mockClear();
        vi.advanceTimersByTime(1000);
        // Only the time since its start changes, once a second.
        expect(requested).toHaveBeenCalledTimes(1);
      } finally {
        requested.mockReset();
        vi.useRealTimers();
      }
    });

    it("should draw no bar for a range inside a collapsed gap", async () => {
      const start = createAnchor();
      await createRanges(start, [
        [2, 8],
        [12, 18],
      ]);
      const provider = mount();
      const props = segmentedProps(start, [
        [0, 10],
        [20, 30],
      ]);
      await pollDrawn(() => {
        provider.render(props);
        return provider.state.count;
      }).toEqual(1);
    });

    it("should ask for the ranges of each segment and not of the gaps", () => {
      const retrieve = vi.spyOn(ranger.listDefinition, "retrieve");
      try {
        const provider = mount();
        const start = TimeStamp.now();
        provider.render(
          segmentedProps(start, [
            [0, 10],
            [86400, 86410],
          ]),
        );
        const { where } = retrieve.mock.calls[0][0].query;
        expect(where?.or).toHaveLength(2);
      } finally {
        retrieve.mockRestore();
      }
    });
  });
});
