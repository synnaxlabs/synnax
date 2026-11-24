// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Unreachable } from "@synnaxlabs/freighter";
import {
  Rate,
  sleep,
  staticWindow,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { assertDefined } from "@synnaxlabs/x/testutil";
import { describe, expect, it } from "vitest";

import type { channel } from "@/channel";
import type { framer } from "@/framer";
import { SyntheticCore } from "@/testutil";

const START = TimeStamp.seconds(1_000_000);
const POLL = { timeout: 5000, interval: 5 };
const RANGE = new TimeRange(START, START.add(TimeSpan.minutes(8)));

const OPTIONS: framer.FeedOptions = {
  batchDebounce: TimeSpan.milliseconds(5),
  removalDelay: TimeSpan.ZERO,
  breaker: {
    baseInterval: TimeSpan.milliseconds(10),
    maxInterval: TimeSpan.milliseconds(50),
  },
};

const createCore = (): SyntheticCore =>
  new SyntheticCore({
    rate: Rate.hz(100),
    start: START,
    now: START.add(TimeSpan.minutes(10)),
    derived: true,
  });

const tiled = (keys: channel.Key[]): framer.PointsRequest => ({
  keys,
  window: staticWindow(RANGE),
  limit: { variant: "points", count: 100, aggregation: "min_max" },
});

const latest = (key: channel.Key): framer.LatestRequest => ({
  keys: [key],
  limit: { variant: "latest", count: 1 },
});

// Wraps the Core's outputs so a spec fails the first lookups, gates them, and counts
// the open subscriptions.
const createOutputs = (core: SyntheticCore, failures = 0) => {
  let failed = 0;
  let open = 0;
  let gate: Promise<void> | null = null;
  const outputs: framer.CalculationOutputs = {
    retrieve: async (keys) => {
      if (gate != null) await gate;
      if (failed < failures) {
        failed++;
        throw new Unreachable({ message: "outputs unavailable" });
      }
      return await core.outputs.retrieve(keys);
    },
    onChange: (key, handler) => {
      open++;
      const stop = core.outputs.onChange(key, handler);
      return () => {
        open--;
        stop();
      };
    },
  };
  return {
    outputs,
    failed: () => failed,
    open: () => open,
    hold: (p: Promise<void>): void => {
      gate = p;
    },
  };
};

describe("framer cache on calculation outputs, cases that hold", () => {
  it("should draw once a failed outputs lookup recovers", async () => {
    const core = createCore();
    assertDefined(core.calculated);
    const { key } = core.calculated;
    const o = createOutputs(core, 2);
    const feed = core.openFeed({ ...OPTIONS, outputs: o.outputs });
    const sub = feed.subscribe(tiled([key]));
    try {
      await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
      expect(o.failed()).toBe(2);
      expect(sub.value(key).series.length).toBeGreaterThan(0);
    } finally {
      sub.close();
      await feed.close();
    }
  });

  it("should read the newest value once a failed outputs lookup recovers", async () => {
    const core = createCore();
    assertDefined(core.calculated);
    const { key } = core.calculated;
    const o = createOutputs(core, 2);
    const feed = core.openFeed({ ...OPTIONS, outputs: o.outputs });
    const sub = feed.subscribe(latest(key));
    try {
      await expect.poll(() => sub.value(key).series.length, POLL).toBe(1);
      expect(o.failed()).toBe(2);
    } finally {
      sub.close();
      await feed.close();
    }
  });

  it("should fail a latest start while its outputs lookup fails", async () => {
    const core = createCore();
    assertDefined(core.calculated);
    const { key } = core.calculated;
    const feed = core.openFeed({
      ...OPTIONS,
      outputs: {
        ...core.outputs,
        retrieve: async () => {
          throw new Error("outputs unavailable");
        },
      },
    });
    const sub = feed.subscribe(latest(key));
    try {
      await expect.poll(() => sub.value(key).status.variant, POLL).toBe("error");
      expect(sub.value(key).status.description).toContain("outputs unavailable");
    } finally {
      sub.close();
      await feed.close();
    }
  });

  it("should move two sources on one channel with its output entry", async () => {
    const core = createCore();
    assertDefined(core.calculated);
    const { key } = core.calculated;
    const feed = core.openFeed(OPTIONS);
    const points = feed.subscribe(tiled([key]));
    const newest = feed.subscribe(latest(key));
    try {
      await expect.poll(() => points.value(key).status.variant, POLL).toBe("success");
      await expect.poll(() => newest.value(key).series.length, POLL).toBe(1);
      core.setOutput(key, false);
      await expect.poll(() => points.value(key).status.variant, POLL).toBe("error");
      await expect.poll(() => newest.value(key).series.length, POLL).toBe(0);
      await sleep.sleep(TimeSpan.milliseconds(50));
      core.setOutput(key, true);
      await expect.poll(() => points.value(key).status.variant, POLL).toBe("success");
      await expect.poll(() => newest.value(key).series.length, POLL).toBe(1);
    } finally {
      points.close();
      newest.close();
      await feed.close();
    }
  });

  it("should unsubscribe when it closes before the lookup answers", async () => {
    const core = createCore();
    assertDefined(core.calculated);
    const { key } = core.calculated;
    const o = createOutputs(core);
    const { promise: released, resolve: release } = Promise.withResolvers<void>();
    o.hold(released);
    const feed = core.openFeed({ ...OPTIONS, outputs: o.outputs });
    const points = feed.subscribe(tiled([key]));
    const newest = feed.subscribe(latest(key));
    try {
      await expect.poll(o.open, POLL).toBe(2);
      points.close();
      newest.close();
      await expect.poll(o.open, POLL).toBe(0);
      release();
      await sleep.sleep(TimeSpan.milliseconds(50));
      core.setOutput(key, false);
      expect(o.open()).toBe(0);
    } finally {
      await feed.close();
    }
  });

  it("should draw many channels of one lookup at once", async () => {
    const core = createCore();
    assertDefined(core.calculated);
    assertDefined(core.calculatedIndex);
    const keys = [core.index.key, core.data[0].key, core.calculated.key];
    const feed = core.openFeed(OPTIONS);
    const sub = feed.subscribe(tiled([...keys, core.calculatedIndex.key]));
    try {
      for (const key of [...keys, core.calculatedIndex.key])
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
    } finally {
      sub.close();
      await feed.close();
    }
  });

  it("should refuse a virtual channel that no calculation writes", async () => {
    const core = createCore();
    assertDefined(core.virtual);
    const { key } = core.virtual;
    const feed = core.openFeed(OPTIONS);
    const sub = feed.subscribe(tiled([key]));
    try {
      await expect.poll(() => sub.value(key).status.variant, POLL).toBe("error");
      expect(sub.value(key).status.description).toContain("has no samples to draw");
    } finally {
      sub.close();
      await feed.close();
    }
  });

  it("should read history once a streamed channel becomes an output", async () => {
    const core = createCore();
    assertDefined(core.calculated);
    const { key } = core.calculated;
    core.setOutput(key, false);
    const feed = core.openFeed(OPTIONS);
    const sub = feed.subscribe(latest(key));
    try {
      await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
      expect(sub.value(key).series.length).toBe(0);
      await sleep.sleep(TimeSpan.milliseconds(50));
      core.setOutput(key, true);
      await expect.poll(() => sub.value(key).series.length, POLL).toBe(1);
    } finally {
      sub.close();
      await feed.close();
    }
  });
});
