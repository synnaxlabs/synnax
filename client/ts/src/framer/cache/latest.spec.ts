// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  DataType,
  dynamicWindow,
  Rate,
  Series,
  sleep,
  TimeRange,
  TimeSpan,
  TimeStamp,
  type Window,
} from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import { framer } from "@/framer";
import { domainOf } from "@/framer/cache/block";
import { Joins } from "@/framer/cache/join";
import { Latest, type LatestProps } from "@/framer/cache/latest";
import { Store } from "@/framer/cache/store";
import { MultiplexedStreamer } from "@/framer/cache/streamer";
import { SyntheticCore } from "@/testutil";

const START = TimeStamp.seconds(1_000_000);
const POLL = { timeout: 5000, interval: 5 };

// Wires a latest to the Core as a feed does, counting its reads. onRead runs before
// each read, which waits for it.
const createLatest = (
  core: SyntheticCore,
  onRead: () => void | Promise<void> = () => {},
) => {
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
    store,
    streamer,
    retrieveChannels: core.retrieveChannels,
    now: core.now,
    breaker: { baseInterval: TimeSpan.milliseconds(10) },
    readLatest: async (key, count, tr) => {
      await onRead();
      reads++;
      return core.readLatest([key], count, tr).get(key);
    },
  };
  const open = (
    count: number,
    notify: (l: Latest) => void = () => {},
    key = core.data[0].key,
    window?: Window,
  ): Latest => {
    const latest: Latest = new Latest(props, key, window, count, () => notify(latest));
    return latest;
  };
  return {
    streamer,
    open,
    reads: () => reads,
    close: async () => await streamer.close(),
  };
};

const createCore = (derived = false): SyntheticCore =>
  new SyntheticCore({
    rate: Rate.hz(100),
    start: START,
    now: START.add(TimeSpan.minutes(10)),
    derived,
  });

// Returns the timestamps of the snapshot's samples.
const timesOf = (latest: Latest): bigint[] =>
  Array.from(latest.value().series) as bigint[];

describe("Latest", () => {
  describe("snapshot", () => {
    it("should return the same snapshot while nothing changes", async () => {
      const core = createCore();
      const { open, close } = createLatest(core);
      const latest = open(100);
      try {
        await expect.poll(() => latest.value().status.variant, POLL).toBe("success");
        expect(latest.value()).toBe(latest.value());
      } finally {
        latest.close();
        await close();
      }
    });

    it("should keep the series and move the time as the stream grows", async () => {
      const core = createCore();
      const { open, close } = createLatest(core);
      const latest = open(100, undefined, core.index.key);
      try {
        await expect.poll(() => latest.value().status.variant, POLL).toBe("success");
        // The Core streams in a domain apart from the read's, so the first streamed
        // series stays apart and the next one starts the buffer.
        core.advance(TimeSpan.milliseconds(100));
        core.advance(TimeSpan.milliseconds(100));
        await expect.poll(() => latest.value().series.series.length, POLL).toBe(3);
        const before = latest.value();
        core.advance(TimeSpan.milliseconds(100));
        await expect.poll(() => latest.value().series.length, POLL).toBe(130);
        const after = latest.value();
        expect(after.series).toBe(before.series);
        expect(after.time?.valueOf()).toBe(
          before.time!.add(TimeSpan.milliseconds(100)).valueOf(),
        );
      } finally {
        latest.close();
        await close();
      }
    });
  });

  it("should serve every sample of a frame larger than the count", async () => {
    const core = createCore();
    const { open, close } = createLatest(core);
    const key = core.data[0].key;
    const latest = open(10_000, undefined, key);
    try {
      await expect.poll(() => latest.value().status.variant, POLL).toBe("success");
      const domain = domainOf(latest.value().series.series[0].alignment) + 1n;
      const start = core.now();
      const series = [0, 1].map(
        (i) =>
          new Series({
            data: new Float32Array(10_000).fill(i),
            dataType: DataType.FLOAT32,
            timeRange: new TimeRange(
              start.add(TimeSpan.seconds(i)),
              start.add(TimeSpan.seconds(i + 1)),
            ),
            alignment: (domain << 32n) + BigInt(i * 10_000),
          }),
      );
      core.stream(new framer.Frame([key, key], series));
      await expect.poll(() => latest.value().series.length, POLL).toBe(20_000);
    } finally {
      latest.close();
      await close();
    }
  });

  it("should hold about two windows of samples as the stream grows", async () => {
    const core = createCore();
    const { open, close } = createLatest(core);
    const window = dynamicWindow(TimeSpan.seconds(1));
    const latest = open(1000, undefined, core.index.key, window);
    try {
      await expect.poll(() => latest.value().status.variant, POLL).toBe("success");
      for (let i = 0; i < 30; i++) {
        core.advance(TimeSpan.milliseconds(100));
        const recent = core.now().sub(TimeSpan.milliseconds(10)).valueOf();
        await expect
          .poll(() => latest.value().time?.valueOf() ?? 0n, POLL)
          .toBeGreaterThan(recent);
      }
      expect(latest.value().series.length).toBeLessThanOrEqual(210);
    } finally {
      latest.close();
      await close();
    }
  });

  it("should drop the samples the window leaves with no stream", async () => {
    const core = createCore();
    const { open, close } = createLatest(core);
    const window = dynamicWindow(TimeSpan.seconds(1));
    const latest = open(1000, undefined, core.index.key, window);
    try {
      await expect.poll(() => latest.value().status.variant, POLL).toBe("success");
      expect(latest.value().series.length).toBe(100);
      core.advance(TimeSpan.seconds(2), false);
      expect(latest.value().series.length).toBe(0);
    } finally {
      latest.close();
      await close();
    }
  });

  it("should drop samples no value served once they pass the cap", async () => {
    const core = createCore();
    const { open, close } = createLatest(core);
    const latest = open(10, undefined, core.index.key);
    try {
      await expect.poll(() => latest.value().status.variant, POLL).toBe("success");
      for (let i = 0; i < 300; i++) core.advance(TimeSpan.seconds(5));
      const recent = core.now().sub(TimeSpan.milliseconds(10)).valueOf();
      await expect
        .poll(() => latest.value().time?.valueOf() ?? 0n, POLL)
        .toBeGreaterThan(recent);
      expect(latest.value().series.length).toBeLessThan(150_000);
    } finally {
      latest.close();
      await close();
    }
  });

  it("should serve streamed samples while the read waits", async () => {
    const core = createCore();
    let release: () => void = () => {};
    const released = new Promise<void>((resolve) => (release = resolve));
    let waiting = false;
    const { open, close } = createLatest(core, async () => {
      waiting = true;
      await released;
    });
    const latest = open(1000, undefined, core.index.key);
    try {
      await expect.poll(() => waiting, POLL).toBe(true);
      core.advance(TimeSpan.milliseconds(100));
      await expect.poll(() => latest.value().series.length, POLL).toBe(10);
      expect(latest.value().status.variant).toBe("loading");
      release();
      await expect.poll(() => latest.value().status.variant, POLL).toBe("success");
      const times = timesOf(latest);
      expect(times).toHaveLength(1000);
      const steps = new Set(times.slice(1).map((t, i) => t - times[i]));
      expect([...steps]).toEqual([Rate.hz(100).period.valueOf()]);
    } finally {
      latest.close();
      await close();
    }
  });

  it("should keep samples streamed before a drop after the new read", async () => {
    const core = createCore();
    let release: () => void = () => {};
    const released = new Promise<void>((resolve) => (release = resolve));
    let waiting = 0;
    const { open, close } = createLatest(core, async () => {
      waiting++;
      await released;
    });
    const latest = open(1000, undefined, core.index.key);
    try {
      await expect.poll(() => waiting, POLL).toBe(1);
      core.advance(TimeSpan.milliseconds(100));
      await expect.poll(() => latest.value().series.length, POLL).toBe(10);
      core.restart();
      release();
      // The reconnect replaces the read running at the drop with a read up to the
      // newest sample held, then a read fills the samples after it.
      await expect.poll(() => waiting, POLL).toBe(3);
      await expect.poll(() => latest.value().status.variant, POLL).toBe("success");
      const times = timesOf(latest);
      expect(times).toHaveLength(1000);
      const steps = new Set(times.slice(1).map((t, i) => t - times[i]));
      expect([...steps]).toEqual([Rate.hz(100).period.valueOf()]);
    } finally {
      latest.close();
      await close();
    }
  });

  it("should serve streamed samples while the read fails", async () => {
    const core = createCore();
    const { open, close } = createLatest(core, () => {
      throw new Error("stream closed unexpectedly");
    });
    const latest = open(1000, undefined, core.index.key);
    try {
      await expect
        .poll(() => latest.value().status.description, POLL)
        .toBe("stream closed unexpectedly");
      core.advance(TimeSpan.milliseconds(100));
      await expect.poll(() => latest.value().series.length, POLL).toBe(10);
      expect(latest.value().status).toMatchObject({
        variant: "error",
        message: "Failed to read channel data",
      });
    } finally {
      latest.close();
      await close();
    }
  });

  it("should keep samples it served under an error when a read lands", async () => {
    const core = createCore();
    let failed = false;
    let release: () => void = () => {};
    const released = new Promise<void>((resolve) => (release = resolve));
    const { open, close } = createLatest(core, async () => {
      if (!failed) {
        failed = true;
        throw new Error("stream closed unexpectedly");
      }
      await released;
    });
    const latest = open(1000, undefined, core.index.key);
    try {
      await expect.poll(() => latest.value().status.variant, POLL).toBe("error");
      core.advance(TimeSpan.milliseconds(100));
      await expect.poll(() => latest.value().series.length, POLL).toBe(10);
      const served = latest.value().series.series;
      release();
      await expect.poll(() => latest.value().status.variant, POLL).toBe("success");
      expect(latest.value().series.length).toBe(10);
      const { series } = latest.value().series;
      expect(series).toHaveLength(served.length);
      series.forEach((s, i) => expect(s).toBe(served[i]));
    } finally {
      latest.close();
      await close();
    }
  });

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

  it("should leave a calculated channel's streamed samples the read holds out by time", async () => {
    const core = createCore(true);
    let written = false;
    // The Core writes, and streams, samples the read then holds.
    const { open, close } = createLatest(core, () => {
      if (written) return;
      written = true;
      core.advance(TimeSpan.milliseconds(100));
    });
    const latest = open(100, undefined, core.calculated!.key);
    try {
      await expect.poll(() => latest.value().status.variant, POLL).toBe("success");
      await sleep.sleep(TimeSpan.milliseconds(50));
      expect(written).toBe(true);
      expect(latest.value().series.length).toBe(100);
    } finally {
      latest.close();
      await close();
    }
  });

  it("should keep serving a buffer it served after the buffer closes", async () => {
    const core = createCore(true);
    const { open, close } = createLatest(core);
    const key = core.virtual!.key;
    // 10,000 text samples make a 400 KB buffer, which a few entries leave nearly empty.
    const latest = open(10_000, undefined, key);
    const entry = (i: number): Series =>
      new Series({
        data: [`entry ${i}`],
        dataType: DataType.STRING,
        alignment: BigInt(i),
        timeRange: new TimeRange(
          START.add(TimeSpan.seconds(i)),
          START.add(TimeSpan.seconds(i + 1)),
        ),
      });
    try {
      await expect.poll(() => latest.value().status.variant, POLL).toBe("success");
      core.stream(new framer.Frame(key, entry(0)));
      core.stream(new framer.Frame(key, entry(1)));
      await expect.poll(() => latest.value().series.length, POLL).toBe(2);
      const served = latest.value().series.series[1];
      // A gap in the alignments closes the buffer.
      core.stream(new framer.Frame(key, entry(5)));
      await expect.poll(() => latest.value().series.length, POLL).toBe(3);
      expect(latest.value().series.series[1]).toBe(served);
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
