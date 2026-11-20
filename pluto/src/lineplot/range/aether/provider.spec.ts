// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, testClientParams } from "@synnaxlabs/client/testutil";
import { box, id, TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { Layout } from "@/lineplot/aether/layout";
import { range } from "@/lineplot/range/aether";
import { ranger } from "@/ranger/aether";
import { renderAether } from "@/testutil/renderAether";
import { render } from "@/vis/render";
import { canvasTest } from "@/vis/render/testutil";

const REGION = box.construct({ x: 0, y: 0 }, { width: 1000, height: 500 });
const PROVIDER_KEY = "annotations";

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

/** Creates one range per [start, end] pair, in seconds past start. */
const createRanges = async (start: TimeStamp, spans: [number, number][]) =>
  await createTestClient().ranges.create(
    spans.map(([from, to]) => ({
      name: `lane-${id.create()}`,
      timeRange: new TimeRange(
        start.add(TimeSpan.seconds(from)),
        start.add(TimeSpan.seconds(to)),
      ),
      color: "#7C3AED",
    })),
  );

describe("Provider", () => {
  // A window an Arc fills with ranges overlaps far more than the strip can draw, and
  // writing the unbounded answer through the cache stalls the worker for seconds,
  // which freezes every plot in the window.
  it("stops drawing annotations once the window passes the cap", async () => {
    const client = createTestClient();
    // Far out enough that no other spec's ranges overlap the window, and stepped by
    // the clock so ranges an earlier run left on the Core stay outside it.
    const start = TimeStamp.now().add(TimeSpan.days(3650));
    const timeRange = new TimeRange(start, start.add(TimeSpan.seconds(1)));
    const create = async (count: number) =>
      await client.ranges.create(
        Array.from({ length: count }, () => ({
          name: `annotation-${id.create()}`,
          timeRange,
          color: "#7C3AED",
        })),
      );
    await create(5);
    const provider = mount();
    const props = renderProps(timeRange);
    const drawn = () => {
      provider.render(props);
      return provider.state.count;
    };
    await expect.poll(drawn).toEqual(5);
    // One past the cap of 100.
    await create(96);
    await expect.poll(drawn).toEqual(0);
  });

  it("draws an open range from its start through every later viewport", async () => {
    const client = createTestClient();
    // Past the cap spec's window, so this range never overlaps it.
    const start = TimeStamp.now().add(TimeSpan.days(7300));
    const range = await client.ranges.create({
      name: `open-${id.create()}`,
      timeRange: new TimeRange(start, TimeStamp.MAX),
      color: "#7C3AED",
    });
    try {
      // A fresh provider per viewport, since a requery keeps the previous answer until
      // the new one lands.
      const drawn = (viewportStart: TimeStamp) => {
        const provider = mount();
        const props = renderProps(
          new TimeRange(viewportStart, viewportStart.add(TimeSpan.seconds(30))),
        );
        return () => {
          provider.render(props);
          return provider.state.count;
        };
      };
      await expect.poll(drawn(start.sub(TimeSpan.seconds(10)))).toEqual(1);
      await expect.poll(drawn(start.add(TimeSpan.hours(1)))).toEqual(1);
    } finally {
      await client.ranges.delete(range.key);
    }
  });

  it("stacks overlapping ranges in lanes above the plot", async () => {
    const start = TimeStamp.now().add(TimeSpan.days(10950));
    const ranges = await createRanges(start, [
      [0, 10],
      [2, 6],
      [4, 8],
    ]);
    try {
      const provider = mount();
      const props = renderProps(new TimeRange(start, start.add(TimeSpan.seconds(10))));
      await expect
        .poll(() => {
          provider.render(props);
          return provider.state.height;
        })
        .toEqual(72);
      // The 2 to 6 second range holds the middle lane, from 46 to 26 px above the plot.
      provider.setState((s) => ({ ...s, cursor: { x: 300, y: -36 } }));
      provider.render(props);
      expect(provider.state.hovered?.key).toEqual(ranges[1].key);
    } finally {
      await createTestClient().ranges.delete(ranges.map((r) => r.key));
    }
  });

  it("keeps the range of an open menu hovered as the cursor leaves it", async () => {
    const start = TimeStamp.now().add(TimeSpan.days(16425));
    const ranges = await createRanges(start, [
      [0, 4],
      [6, 10],
    ]);
    try {
      const provider = mount();
      const shown = new TimeRange(start, start.add(TimeSpan.seconds(10)));
      const props = renderProps(shown);
      await expect
        .poll(() => {
          provider.render(props);
          return provider.state.count;
        })
        .toEqual(2);
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
    } finally {
      await createTestClient().ranges.delete(ranges.map((r) => r.key));
    }
  });

  it("hides the tooltip while the menu is open", async () => {
    const start = TimeStamp.now().add(TimeSpan.days(16790));
    const [rng] = await createRanges(start, [[0, 10]]);
    try {
      const recorder = canvasTest.record();
      const provider = mount(recorder);
      const props = renderProps(new TimeRange(start, start.add(TimeSpan.seconds(10))));
      await expect
        .poll(() => {
          provider.render(props);
          return provider.state.count;
        })
        .toEqual(1);
      const names = (): number => {
        recorder.upper2d.calls.length = 0;
        provider.render(props);
        return recorder.upper2d.calls.filter(
          (c) => c.op === "fillText" && c.args[0] === rng.name,
        ).length;
      };
      provider.setState((s) => ({ ...s, cursor: { x: 500, y: -14 } }));
      // The bar label and the tooltip.
      expect(names()).toEqual(2);
      provider.setState((s) => ({ ...s, menuOpen: true }));
      expect(names()).toEqual(1);
    } finally {
      await createTestClient().ranges.delete(rng.key);
    }
  });

  it("keeps back to back ranges in one lane at any zoom", async () => {
    const start = TimeStamp.now().add(TimeSpan.days(11680));
    const ranges = await createRanges(start, [
      [0, 4],
      [4, 6],
      [6.5, 10],
    ]);
    try {
      const provider = mount();
      const bounds = new TimeRange(start, start.add(TimeSpan.seconds(10)));
      // Zoomed out so far that the half second gap is under a pixel wide.
      const shown = new TimeRange(
        start.sub(TimeSpan.seconds(1000)),
        start.add(TimeSpan.seconds(1000)),
      );
      await expect
        .poll(() => {
          provider.render(renderProps(bounds, shown));
          return provider.state.count;
        })
        .toEqual(3);
      provider.render(renderProps(bounds));
      expect(provider.state.height).toEqual(28);
      provider.render(renderProps(bounds, shown));
      expect(provider.state.height).toEqual(28);
    } finally {
      await createTestClient().ranges.delete(ranges.map((r) => r.key));
    }
  });

  it("draws no bar for a range with no color", async () => {
    const client = createTestClient();
    const start = TimeStamp.now().add(TimeSpan.days(13505));
    const [colored] = await createRanges(start, [[0, 4]]);
    const uncolored = await client.ranges.create({
      name: `uncolored-${id.create()}`,
      timeRange: new TimeRange(
        start.add(TimeSpan.seconds(5)),
        start.add(TimeSpan.seconds(8)),
      ),
    });
    try {
      const provider = mount();
      const props = renderProps(new TimeRange(start, start.add(TimeSpan.seconds(10))));
      await expect
        .poll(() => {
          provider.render(props);
          return provider.state.count;
        })
        .toEqual(1);
      provider.setState((s) => ({ ...s, cursor: { x: 650, y: -14 } }));
      provider.render(props);
      expect(provider.state.hovered).toBeNull();
    } finally {
      await client.ranges.delete([colored.key, uncolored.key]);
    }
  });

  it("keeps a range below the range that holds it", async () => {
    const start = TimeStamp.now().add(TimeSpan.days(11315));
    const ranges = await createRanges(start, [
      [0, 5],
      [2, 30],
      [5, 10],
    ]);
    try {
      const provider = mount();
      const props = renderProps(new TimeRange(start, start.add(TimeSpan.seconds(30))));
      await expect
        .poll(() => {
          provider.render(props);
          return provider.state.height;
        })
        .toEqual(72);
      // The 5 to 10 second range holds the third lane, from 24 to 4 px above the plot.
      provider.setState((s) => ({ ...s, cursor: { x: 250, y: -14 } }));
      provider.render(props);
      expect(provider.state.hovered?.key).toEqual(ranges[2].key);
    } finally {
      await createTestClient().ranges.delete(ranges.map((r) => r.key));
    }
  });

  it("draws ranges past the third lane in thin lanes", async () => {
    const start = TimeStamp.now().add(TimeSpan.days(12775));
    const ranges = await createRanges(start, [
      [0, 10],
      [1, 9],
      [2, 8],
      [3, 7],
      [4, 6],
    ]);
    try {
      const provider = mount();
      const props = renderProps(new TimeRange(start, start.add(TimeSpan.seconds(10))));
      await expect
        .poll(() => {
          provider.render(props);
          return provider.state.height;
        })
        .toEqual(84);
      // The innermost range holds the last thin lane, from 8 to 4 px above the plot.
      provider.setState((s) => ({ ...s, cursor: { x: 500, y: -6 } }));
      provider.render(props);
      expect(provider.state.hovered?.key).toEqual(ranges[4].key);
    } finally {
      await createTestClient().ranges.delete(ranges.map((r) => r.key));
    }
  });

  it("fills only the ranges that hold no other range", async () => {
    const start = TimeStamp.now().add(TimeSpan.days(14600));
    const ranges = await createRanges(start, [
      [0, 10],
      [2, 6],
    ]);
    try {
      const recorder = canvasTest.record();
      const provider = mount(recorder);
      const props = renderProps(new TimeRange(start, start.add(TimeSpan.seconds(10))));
      await expect
        .poll(() => {
          provider.render(props);
          return provider.state.count;
        })
        .toEqual(2);
      recorder.upper2d.calls.length = 0;
      provider.render(props);
      const fills = recorder.upper2d.calls.filter((c) => c.op === "rect");
      expect(fills).toHaveLength(1);
      const [x, y, width, height] = fills[0].args as number[];
      expect(x).toBeCloseTo(200);
      expect(width).toBeCloseTo(400);
      expect([y, height]).toEqual([0, 500]);
    } finally {
      await createTestClient().ranges.delete(ranges.map((r) => r.key));
    }
  });

  // The strip's cap counts ranges in the axis bounds, so a range past them is never
  // drawn, even when a zoomed-out viewport shows it.
  it("draws only the ranges inside the axis bounds", async () => {
    const start = TimeStamp.now().add(TimeSpan.days(16425));
    const ranges = await createRanges(start, [
      [2, 6],
      [12, 14],
    ]);
    try {
      const recorder = canvasTest.record();
      const provider = mount(recorder);
      const bounds = new TimeRange(start, start.add(TimeSpan.seconds(10)));
      const shown = new TimeRange(
        start.sub(TimeSpan.seconds(10)),
        start.add(TimeSpan.seconds(20)),
      );
      const props = renderProps(bounds, shown);
      await expect
        .poll(() => {
          provider.render(props);
          return provider.state.count;
        })
        .toEqual(1);
      recorder.upper2d.calls.length = 0;
      provider.render(props);
      expect(recorder.upper2d.calls.filter((c) => c.op === "rect")).toHaveLength(1);
    } finally {
      await createTestClient().ranges.delete(ranges.map((r) => r.key));
    }
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
    // A minute before now, so few ranges that other specs create near now overlap it.
    const createRunning = async (now: TimeStamp) =>
      await createTestClient().ranges.create({
        name: `running-${id.create()}`,
        timeRange: new TimeRange(
          now.sub(TimeSpan.seconds(90)),
          now.add(TimeSpan.hours(1)),
        ),
        color: "#7C3AED",
      });
    const lastMinute = (now: TimeStamp): TimeRange =>
      new TimeRange(now.sub(TimeSpan.seconds(120)), now.sub(TimeSpan.seconds(60)));
    const ahead = (now: TimeStamp): TimeRange =>
      new TimeRange(now.add(TimeSpan.minutes(1)), now.add(TimeSpan.minutes(20)));

    it("should show the time since its start at the end of its bar", async () => {
      const now = TimeStamp.now();
      const rng = await createRunning(now);
      try {
        const recorder = canvasTest.record();
        const provider = mount(recorder);
        const props = renderProps(lastMinute(now));
        const texts = (): string[] => {
          recorder.upper2d.calls.length = 0;
          provider.render(props);
          return recorder.upper2d.calls
            .filter((c) => c.op === "fillText")
            .map((c) => c.args[0] as string);
        };
        await expect.poll(texts).toContain(rng.name);
        expect(texts().some((t) => /^T\+0:0[1-9]:\d\d$/.test(t))).toBe(true);
        // The bar runs past the view, so it has no head.
        expect(recorder.upper2d.calls.some((c) => c.op === "arcTo")).toBe(false);
        provider.setState((s) => ({ ...s, cursor: { x: 800, y: -14 } }));
        provider.render(props);
        expect(provider.state.hovered?.key).toEqual(rng.key);
        expect(texts().some((t) => t.startsWith("Running for "))).toBe(true);
      } finally {
        await createTestClient().ranges.delete(rng.key);
      }
    });

    it("should end its bar at the newest sample of a live plot", async () => {
      const now = TimeStamp.now();
      const rng = await createRunning(now);
      try {
        const recorder = canvasTest.record();
        const provider = mount(recorder);
        const props = {
          ...renderProps(lastMinute(now)),
          newest: now.sub(TimeSpan.seconds(75)),
        };
        await expect
          .poll(() => {
            provider.render(props);
            return provider.state.count;
          })
          .toBeGreaterThan(0);
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
      } finally {
        await createTestClient().ranges.delete(rng.key);
      }
    });

    it("should redraw as the strip changes, and stop when it hides", async () => {
      const now = TimeStamp.now();
      const rng = await createRunning(now);
      try {
        const provider = mount();
        const live = {
          ...renderProps(lastMinute(now)),
          newest: now.sub(TimeSpan.seconds(75)),
        };
        let props: range.ProviderProps = live;
        provider.setState((s) => ({ ...s, visible: false }));
        await expect
          .poll(() => {
            provider.render(props);
            return provider.state.count;
          })
          .toBeGreaterThan(0);
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
        props = renderProps(lastMinute(now));
        provider.render(props);
        expect(redraws()).toEqual(1);
        provider.setState((s) => ({ ...s, visible: false }));
        provider.render(props);
        expect(redraws()).toEqual(0);
      } finally {
        requested.mockReset();
        vi.useRealTimers();
        await createTestClient().ranges.delete(rng.key);
      }
    });

    // Expects one redraw of the strip at the given time, and none before it.
    const expectRedrawAt = async (
      props: range.ProviderProps,
      at: TimeStamp,
    ): Promise<void> => {
      const provider = mount();
      await expect
        .poll(() => {
          provider.render(props);
          return provider.state.count;
        })
        .toBeGreaterThan(0);
      vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout"] });
      try {
        provider.render(props);
        const wait = new TimeRange(TimeStamp.now(), at).span.milliseconds;
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
      const now = TimeStamp.now();
      // The range starts before the view, so a running range of another spec cannot
      // wake the strip first at the start of the view.
      const rng = await createTestClient().ranges.create({
        name: `planned-${id.create()}`,
        timeRange: new TimeRange(
          now.add(TimeSpan.seconds(30)),
          now.add(TimeSpan.minutes(10)),
        ),
        color: "#7C3AED",
      });
      try {
        await expectRedrawAt(renderProps(ahead(now)), rng.timeRange.start);
      } finally {
        await createTestClient().ranges.delete(rng.key);
      }
    });

    it("should redraw when now reaches a view ahead of a running range", async () => {
      const now = TimeStamp.now();
      const rng = await createRunning(now);
      try {
        const view = ahead(now);
        await expectRedrawAt(renderProps(view), view.start);
      } finally {
        await createTestClient().ranges.delete(rng.key);
      }
    });
  });

  it("should show the length of an ended range at the end of its bar", async () => {
    const start = TimeStamp.now().add(TimeSpan.days(17155));
    const [rng] = await createRanges(start, [[2, 6]]);
    try {
      const recorder = canvasTest.record();
      const provider = mount(recorder);
      const props = renderProps(new TimeRange(start, start.add(TimeSpan.seconds(10))));
      const texts = (): unknown[] => {
        recorder.upper2d.calls.length = 0;
        provider.render(props);
        return recorder.upper2d.calls
          .filter((c) => c.op === "fillText")
          .map((c) => c.args[0]);
      };
      await expect.poll(texts).toContain(rng.name);
      expect(texts()).toContain("4s");
    } finally {
      await createTestClient().ranges.delete(rng.key);
    }
  });

  it("should fade only the bars that the plot edge cuts", async () => {
    const start = TimeStamp.now().add(TimeSpan.days(17520));
    const ranges = await createRanges(start, [
      [0, 5],
      [6, 9],
    ]);
    try {
      const recorder = canvasTest.record();
      const provider = mount(recorder);
      const props = renderProps(
        new TimeRange(start, start.add(TimeSpan.seconds(10))),
        new TimeRange(start.add(TimeSpan.seconds(2)), start.add(TimeSpan.seconds(10))),
      );
      await expect
        .poll(() => {
          provider.render(props);
          return provider.state.count;
        })
        .toEqual(2);
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
    } finally {
      await createTestClient().ranges.delete(ranges.map((r) => r.key));
    }
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
      };
    };

    it("should cut a bar at each seam that its range crosses", async () => {
      const start = TimeStamp.now().add(TimeSpan.days(18250));
      const ranges = await createRanges(start, [[5, 25]]);
      try {
        const recorder = canvasTest.record();
        const provider = mount(recorder);
        const props = segmentedProps(start, [
          [0, 10],
          [20, 30],
        ]);
        await expect
          .poll(() => {
            provider.render(props);
            return provider.state.count;
          })
          .toEqual(1);
        recorder.upper2d.calls.length = 0;
        provider.render(props);
        const { calls } = recorder.upper2d;
        // A fill and a stroke for each of the two bars.
        expect(calls.filter((c) => c.op === "roundRect")).toHaveLength(4);
        const gradients = calls.filter((c) => c.op === "createLinearGradient");
        expect(gradients.length).toBeGreaterThan(0);
      } finally {
        await createTestClient().ranges.delete(ranges.map((r) => r.key));
      }
    });

    it("should draw no bar for a range inside a collapsed gap", async () => {
      const start = TimeStamp.now().add(TimeSpan.days(18615));
      const ranges = await createRanges(start, [
        [2, 8],
        [12, 18],
      ]);
      try {
        const provider = mount();
        const props = segmentedProps(start, [
          [0, 10],
          [20, 30],
        ]);
        await expect
          .poll(() => {
            provider.render(props);
            return provider.state.count;
          })
          .toEqual(1);
      } finally {
        await createTestClient().ranges.delete(ranges.map((r) => r.key));
      }
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
