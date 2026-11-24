// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  array,
  dynamicWindow,
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

const tiled = (key: channel.Key): framer.PointsRequest => ({
  keys: [key],
  window: staticWindow(RANGE),
  limit: { variant: "points", count: 100, aggregation: "min_max" },
});

const latest = (key: channel.Key): framer.LatestRequest => ({
  keys: [key],
  limit: { variant: "latest", count: 1 },
});

// Returns the status of the channel once it leaves the loading state.
const settledStatus = async (sub: framer.Subscription, key: channel.Key) => {
  await sleep.sleep(TimeSpan.milliseconds(50));
  await expect.poll(() => sub.value(key).status.variant, POLL).not.toBe("loading");
  const { variant, description } = sub.value(key).status;
  return { variant, description };
};

// Opens a feed whose first channel retrieve of key waits for release.
const createGatedFeed = (core: SyntheticCore, key: channel.Key) => {
  const { promise: released, resolve: release } = Promise.withResolvers<void>();
  let retrieves = 0;
  const feed = core.openFeed({
    ...OPTIONS,
    retrieveChannels: async (keys) => {
      if (array.toArray(keys).includes(key) && ++retrieves === 1) await released;
      return await core.retrieveChannels(keys);
    },
  });
  return { feed, release, retrieves: () => retrieves };
};

describe("framer cache on calculation outputs", () => {
  it("should draw a channel whose output goes off and back on", async () => {
    const core = createCore();
    assertDefined(core.calculated);
    const { key } = core.calculated;
    const feed = core.openFeed(OPTIONS);
    const sub = feed.subscribe(tiled(key));
    try {
      await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
      core.setOutput(key, false);
      core.setOutput(key, true);
      const { variant, description } = await settledStatus(sub, key);
      expect(variant, description).toBe("success");
    } finally {
      sub.close();
      await feed.close();
    }
  });

  it("should draw a channel that becomes an output while it loads", async () => {
    const core = createCore();
    assertDefined(core.calculated);
    const { key } = core.calculated;
    core.setOutput(key, false);
    const { feed, release, retrieves } = createGatedFeed(core, key);
    const sub = feed.subscribe(tiled(key));
    try {
      await expect.poll(retrieves, POLL).toBe(1);
      core.setOutput(key, true);
      release();
      const { variant, description } = await settledStatus(sub, key);
      expect(variant, description).toBe("success");
    } finally {
      sub.close();
      await feed.close();
    }
  });

  it("should read the newest value of a new output while it loads", async () => {
    const core = createCore();
    assertDefined(core.calculated);
    const { key } = core.calculated;
    core.setOutput(key, false);
    const { feed, release, retrieves } = createGatedFeed(core, key);
    const sub = feed.subscribe(latest(key));
    try {
      await expect.poll(retrieves, POLL).toBe(1);
      core.setOutput(key, true);
      release();
      await expect.poll(() => sub.value(key).series.length, POLL).toBe(1);
    } finally {
      sub.close();
      await feed.close();
    }
  });

  it("should read the newest value of an output that goes off and on", async () => {
    const core = createCore();
    assertDefined(core.calculated);
    const { key } = core.calculated;
    const feed = core.openFeed(OPTIONS);
    const sub = feed.subscribe(latest(key));
    try {
      await expect.poll(() => sub.value(key).series.length, POLL).toBe(1);
      core.setOutput(key, false);
      core.setOutput(key, true);
      await sleep.sleep(TimeSpan.milliseconds(50));
      await expect.poll(() => sub.value(key).series.length, POLL).toBe(1);
    } finally {
      sub.close();
      await feed.close();
    }
  });

  it("should not read a calculation's index again for a latest", async () => {
    const core = createCore();
    assertDefined(core.calculatedIndex);
    const { key } = core.calculatedIndex;
    let reads = 0;
    const feed = core.openFeed({
      ...OPTIONS,
      readRemote: async (tr, keys, reduction) => {
        if (keys.includes(key)) reads++;
        return core.read(tr, keys, reduction);
      },
    });
    const sub = feed.subscribe({
      keys: [key],
      window: dynamicWindow(TimeSpan.seconds(30)),
      limit: { variant: "points", count: 2 ** 20, aggregation: "min_max" },
    });
    const id = setInterval(() => core.advance(TimeSpan.milliseconds(50)), 2);
    let newest: framer.Subscription | null = null;
    try {
      await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
      await sleep.sleep(TimeSpan.milliseconds(200));
      const before = reads;
      newest = feed.subscribe(latest(key));
      await expect.poll(() => newest?.value(key).status.variant, POLL).toBe("success");
      await sleep.sleep(TimeSpan.milliseconds(500));
      expect(reads).toBe(before);
    } finally {
      clearInterval(id);
      newest?.close();
      sub.close();
      await feed.close();
    }
  });
});
