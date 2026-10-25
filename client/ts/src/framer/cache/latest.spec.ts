// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Rate, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import { Joins } from "@/framer/cache/join";
import { Latest, type LatestProps } from "@/framer/cache/latest";
import { Store } from "@/framer/cache/store";
import { MultiplexedStreamer } from "@/framer/cache/streamer";
import { SyntheticCore } from "@/testutil";

const START = TimeStamp.seconds(1_000_000);
const POLL = { timeout: 5000, interval: 5 };

// Wires a latest to the Core as a feed does, counting its reads.
const createLatest = (core: SyntheticCore) => {
  const store = new Store({ now: core.now });
  const streamer = new MultiplexedStreamer({
    store,
    joins: new Joins(store),
    onWait: () => {},
    openStreamer: async (config) => core.openStreamer(config),
    removalDelay: TimeSpan.ZERO,
  });
  let reads = 0;
  const props: LatestProps = {
    streamer,
    retrieveChannels: core.retrieveChannels,
    now: core.now,
    readLatest: async (key, count, tr) => {
      reads++;
      return core.readLatest([key], count, tr).get(key);
    },
  };
  const open = (count: number, notify: (l: Latest) => void = () => {}): Latest => {
    const latest: Latest = new Latest(props, core.data[0].key, undefined, count, () =>
      notify(latest),
    );
    return latest;
  };
  return {
    streamer,
    open,
    reads: () => reads,
    close: async () => await streamer.close(),
  };
};

const createCore = (): SyntheticCore =>
  new SyntheticCore({
    rate: Rate.hz(100),
    start: START,
    now: START.add(TimeSpan.minutes(10)),
  });

describe("Latest", () => {
  it("should keep its samples for an equal window and count", async () => {
    const core = createCore();
    const { open, reads, close } = createLatest(core);
    const latest = open(100);
    try {
      await expect.poll(() => latest.value().status.variant, POLL).toBe("success");
      const before = latest.value().series.series;
      latest.update(undefined, 100);
      expect(latest.value().series.series).toEqual(before);
      expect(latest.value().status.variant).toBe("success");
      expect(reads()).toBe(1);
    } finally {
      latest.close();
      await close();
    }
  });

  it("should skip the read of a load an update replaced after the ack", async () => {
    const core = createCore();
    const { open, reads, close } = createLatest(core);
    let updated = false;
    // The stream acks the channel, then its status changes, before the load goes on.
    const latest = open(100, (l) => {
      if (updated || l.value().status.message !== "Loading channel data") return;
      updated = true;
      l.update(undefined, 50);
    });
    try {
      await expect.poll(() => latest.value().status.variant, POLL).toBe("success");
      expect(updated).toBe(true);
      expect(reads()).toBe(1);
      expect(latest.value().series.length).toBe(50);
    } finally {
      latest.close();
      await close();
    }
  });

  it("should end the load's wait for the stream's ack on close", async () => {
    const core = createCore();
    const { streamer, open, close } = createLatest(core);
    const waitAcked = vi.spyOn(streamer, "waitAcked");
    const latest = open(100);
    try {
      await expect.poll(() => waitAcked.mock.calls.length, POLL).toBe(1);
      const signal = waitAcked.mock.calls[0][1];
      expect(signal?.aborted).toBe(false);
      latest.close();
      expect(signal?.aborted).toBe(true);
    } finally {
      await close();
    }
  });
});
