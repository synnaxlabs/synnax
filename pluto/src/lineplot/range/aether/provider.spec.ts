// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger } from "@synnaxlabs/client";
import { createTestClient, TEST_CLIENT_PARAMS } from "@synnaxlabs/client/testutil";
import { box, id, scale, TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { range } from "@/lineplot/range/aether";
import { renderAether } from "@/testutil/renderAether";
import { render } from "@/vis/render";
import { canvasTest } from "@/vis/render/testutil";

const REGION = box.construct({ x: 0, y: 0 }, { width: 1000, height: 500 });
const PROVIDER_KEY = "annotations";

/** Supplies the render requestor the provider reads, which production gets from the
 * plot's canvas. A requested render has nothing to schedule: the test drives drawing
 * by calling render itself. */
class Requestor extends aether.Composite<typeof Requestor.stateZ> {
  static readonly TYPE = "range-provider-test-requestor";
  static readonly stateZ = z.object({});
  schema = Requestor.stateZ;

  afterUpdate(ctx: aether.Context): void {
    render.control(ctx, () => {});
  }
}

/** @param shown - The time range the plot region shows, wider when zoomed out. */
const renderProps = (
  timeRange: TimeRange,
  shown: TimeRange = timeRange,
): range.ProviderProps => ({
  dataToDecimalScale: scale.Scale.scale(
    Number(shown.start.valueOf()),
    Number(shown.end.valueOf()),
  ),
  region: REGION,
  viewport: REGION,
  timeRange,
});

const mount = (recorder = canvasTest.record()): range.Provider => {
  const h = renderAether(Requestor, {
    state: {},
    synnax: { props: TEST_CLIENT_PARAMS },
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
    const retrieve = vi.spyOn(ranger.Client.prototype, "retrieve");
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

  it("requeries once the viewport zooms in past 2x", () => {
    const retrieve = vi.spyOn(ranger.Client.prototype, "retrieve");
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
});
