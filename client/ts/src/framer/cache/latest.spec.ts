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
  MultiSeries,
  Rate,
  Series,
  sleep,
  staticWindow,
  TimeRange,
  TimeSpan,
  TimeStamp,
  type Window,
} from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import { type channel } from "@/channel";
import { UnsupportedError } from "@/errors";
import { framer } from "@/framer";
import { domainOf } from "@/framer/cache/block";
import { Joins } from "@/framer/cache/join";
import { Latest, type LatestProps } from "@/framer/cache/latest";
import { Store } from "@/framer/cache/store";
import { MultiplexedStreamer } from "@/framer/cache/streamer";
import { SyntheticCore } from "@/testutil";

const START = TimeStamp.seconds(1_000_000);
const POLL = { timeout: 5000, interval: 5 };

// Wires a latest to the Core as a feed does, counting its reads. Each read waits for
// onRead and each lookup for onRetrieve, and readOf replaces what a read returns.
const createLatest = (
  core: SyntheticCore,
  onRead: (key: channel.Key) => void | Promise<void> = () => {},
  readOf: (key: channel.Key, read: MultiSeries) => MultiSeries = (_, read) => read,
  onRetrieve: () => void | Promise<void> = () => {},
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
    retrieveChannels: async (keys) => {
      await onRetrieve();
      return await core.retrieveChannels(keys);
    },
    now: core.now,
    breaker: { baseInterval: TimeSpan.milliseconds(10) },
    readLatest: async (key, count, tr) => {
      await onRead(key);
      reads++;
      return readOf(key, core.readLatest([key], count, tr).get(key));
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

  describe("a channel the Core stores nothing of", () => {
    const write = (alignment: bigint, stamp: TimeStamp): Series =>
      new Series({
        data: new Float32Array([Number(alignment)]),
        dataType: DataType.FLOAT32,
        alignment,
        timeRange: stamp.spanRange(TimeSpan.NANOSECOND),
      });

    it("should start a later request from the newest streamed sample", async () => {
      const core = createCore(true);
      const { open, close } = createLatest(core);
      const key = core.virtual!.key;
      const stamp = START.add(TimeSpan.SECOND);
      const first = open(1, undefined, key);
      try {
        await expect.poll(() => core.streams(key), POLL).toBe(true);
        core.stream(new framer.Frame(key, write(10n, START)));
        core.stream(new framer.Frame(key, write(11n, stamp)));
        await expect
          .poll(() => first.value().time, POLL)
          .toEqual(stamp.add(TimeSpan.NANOSECOND));
        const second = open(1, undefined, key);
        try {
          await expect.poll(() => second.value().series.length, POLL).toBe(1);
          expect(second.value().series.at(0)).toEqual(11);
          expect(second.value().time).toEqual(stamp.add(TimeSpan.NANOSECOND));
          expect(second.timeAt(11n)).toEqual(stamp);
        } finally {
          second.close();
        }
      } finally {
        first.close();
        await close();
      }
    });

    it("should start empty when nothing streamed the channel", async () => {
      const core = createCore(true);
      const { open, close } = createLatest(core);
      const latest = open(1, undefined, core.virtual!.key);
      try {
        await expect.poll(() => latest.value().status.variant, POLL).toBe("success");
        expect(latest.value().series.length).toBe(0);
        expect(latest.value().time).toBeUndefined();
      } finally {
        latest.close();
        await close();
      }
    });
  });

  describe("timeAt", () => {
    // Returns the alignment of every sample the latest holds.
    const alignmentsOf = (latest: Latest): bigint[] =>
      latest
        .value()
        .series.series.flatMap((s) =>
          Array.from({ length: s.length }, (_, i) => s.alignment + BigInt(i)),
        );

    // Returns the time of each held sample, or null for a sample timeAt misses.
    const timedOf = (latest: Latest): (bigint | null)[] =>
      alignmentsOf(latest).map((a) => latest.timeAt(a)?.valueOf() ?? null);

    // Returns true when every held sample is timed, one period apart, and the newest
    // ends the snapshot.
    const timedInOrder = (latest: Latest): boolean => {
      const period = Rate.hz(100).period.valueOf();
      const times = timedOf(latest);
      const end = latest.value().time?.valueOf();
      return (
        times.length > 0 &&
        times.every((t, i) => t != null && (i === 0 || t - times[i - 1]! === period)) &&
        times.at(-1)! + 1n === end
      );
    };

    // A write the Core stamps 1ns per sample, at alignments Cesium gives in a row.
    const write = (alignment: bigint, length: number, stamp: TimeStamp): Series =>
      new Series({
        data: new Float32Array(length),
        dataType: DataType.FLOAT32,
        alignment,
        timeRange: stamp.spanRange(TimeSpan.nanoseconds(length)),
      });

    it("should time a stored channel's read and streamed samples from its index", async () => {
      const core = createCore();
      const { open, close } = createLatest(core);
      const latest = open(100);
      try {
        await expect.poll(() => timedInOrder(latest), POLL).toBe(true);
        core.advance(TimeSpan.milliseconds(100));
        await expect.poll(() => latest.value().series.series.length, POLL).toBe(2);
        await expect.poll(() => timedInOrder(latest), POLL).toBe(true);
      } finally {
        latest.close();
        await close();
      }
    });

    it("should time samples streamed before the index streams", async () => {
      const core = createCore();
      const { open, close } = createLatest(core);
      const latest = open(100);
      try {
        await expect.poll(() => latest.value().status.variant, POLL).toBe("success");
        core.advance(TimeSpan.milliseconds(100));
        await expect.poll(() => latest.value().series.series.length, POLL).toBe(2);
        expect(core.streams(core.index.key)).toBe(false);
        await expect.poll(() => timedInOrder(latest), POLL).toBe(true);
      } finally {
        latest.close();
        await close();
      }
    });

    it("should time a channel's samples while another writer writes its index", async () => {
      const core = createCore();
      const { open, close } = createLatest(core);
      const { key: index } = core.index;
      const { key } = core.data[0];
      const latest = open(10, undefined, key);
      // Past the Core's clock, so a read of the index finds nothing there.
      const start = core.now().add(TimeSpan.hours(1));
      const own = 7n << 32n;
      const other = 8n << 32n;
      // Returns n index samples from start, one millisecond apart.
      const times = (alignment: bigint, from: TimeStamp, n: number): Series =>
        new Series({
          data: BigInt64Array.from({ length: n }, (_, i) =>
            from.add(TimeSpan.milliseconds(i)).valueOf(),
          ),
          dataType: DataType.TIMESTAMP,
          alignment,
          timeRange: from.range(from.add(TimeSpan.milliseconds(n))),
        });
      try {
        await expect.poll(() => latest.value().status.variant, POLL).toBe("success");
        latest.timeAt(0n);
        await expect.poll(() => core.streams(index), POLL).toBe(true);
        const written = times(own, start, 2);
        const data = new Series({
          data: new Float32Array(2),
          dataType: DataType.FLOAT32,
          alignment: own,
          timeRange: written.timeRange,
        });
        core.stream(new framer.Frame([index, key], [written, data]));
        core.stream(
          new framer.Frame(index, times(other, start.add(TimeSpan.seconds(1)), 1_000)),
        );
        await expect.poll(() => latest.timeAt(own), POLL).toEqual(start);
        expect(latest.timeAt(own + 1n)).toEqual(start.add(TimeSpan.milliseconds(1)));
      } finally {
        latest.close();
        await close();
      }
    });

    it("should forget the stamps of the samples it drops", async () => {
      const core = createCore();
      const { open, close } = createLatest(core);
      const latest = open(10);
      try {
        await expect.poll(() => timedInOrder(latest), POLL).toBe(true);
        const [{ alignment }] = latest.value().series.series;
        core.advance(TimeSpan.seconds(1));
        await expect
          .poll(() => latest.value().series.series[0].alignment, POLL)
          .not.toBe(alignment);
        expect(latest.timeAt(alignment)).toBeNull();
      } finally {
        latest.close();
        await close();
      }
    });

    it("should wait to read the index again after a read that does not stamp", async () => {
      const core = createCore();
      let short = true;
      const { open, close, reads } = createLatest(core, undefined, (key, read) =>
        short && key === core.index.key ? new MultiSeries([]) : read,
      );
      const latest = open(100);
      try {
        await expect.poll(() => latest.value().status.variant, POLL).toBe("success");
        const [{ alignment }] = latest.value().series.series;
        const before = reads();
        expect(latest.timeAt(alignment)).toBeNull();
        await expect.poll(reads, POLL).toBe(before + 1);
        await sleep.sleep(TimeSpan.milliseconds(20));
        expect(latest.timeAt(alignment)).toBeNull();
        await sleep.sleep(TimeSpan.milliseconds(20));
        expect(reads()).toBe(before + 1);
        short = false;
        core.advance(TimeSpan.seconds(4), false);
        expect(latest.timeAt(alignment)).toBeNull();
        await sleep.sleep(TimeSpan.milliseconds(20));
        expect(reads()).toBe(before + 1);
        core.advance(TimeSpan.seconds(1), false);
        await expect.poll(() => latest.timeAt(alignment), POLL).not.toBeNull();
        expect(reads()).toBe(before + 2);
      } finally {
        latest.close();
        await close();
      }
    });

    it("should read the index for a series at most three times", async () => {
      const core = createCore();
      const { open, close, reads } = createLatest(core, undefined, (key, read) =>
        key === core.index.key ? new MultiSeries([]) : read,
      );
      const latest = open(100);
      try {
        await expect.poll(() => latest.value().status.variant, POLL).toBe("success");
        const [{ alignment }] = latest.value().series.series;
        const before = reads();
        for (let i = 0; i < 4; i++) {
          expect(latest.timeAt(alignment)).toBeNull();
          await sleep.sleep(TimeSpan.milliseconds(20));
          core.advance(TimeSpan.seconds(5), false);
        }
        expect(reads()).toBe(before + 3);
      } finally {
        latest.close();
        await close();
      }
    });

    it("should stop reading the index once the Core refuses it", async () => {
      const core = createCore();
      const { open, close, reads } = createLatest(core, undefined, (key, read) => {
        if (key === core.index.key) throw new UnsupportedError("index read refused");
        return read;
      });
      const latest = open(100);
      try {
        await expect.poll(() => latest.value().status.variant, POLL).toBe("success");
        const [{ alignment }] = latest.value().series.series;
        const before = reads();
        expect(latest.timeAt(alignment)).toBeNull();
        await expect.poll(reads, POLL).toBe(before + 1);
        await sleep.sleep(TimeSpan.milliseconds(20));
        core.advance(TimeSpan.seconds(5), false);
        expect(latest.timeAt(alignment)).toBeNull();
        await sleep.sleep(TimeSpan.milliseconds(20));
        expect(reads()).toBe(before + 1);
      } finally {
        latest.close();
        await close();
      }
    });

    it("should time an index channel's samples from their own values", async () => {
      const core = createCore();
      const { open, close } = createLatest(core);
      const latest = open(100, undefined, core.index.key);
      try {
        await expect.poll(() => latest.value().status.variant, POLL).toBe("success");
        expect(timedOf(latest)).toEqual(timesOf(latest));
      } finally {
        latest.close();
        await close();
      }
    });

    it("should time the samples of each write to a channel with no index 1ns apart", async () => {
      const core = createCore(true);
      const { open, close } = createLatest(core);
      const key = core.virtual!.key;
      const latest = open(100, undefined, key);
      const stamps = [0, 1, 2].map((i) => START.add(TimeSpan.seconds(i)));
      try {
        await expect.poll(() => latest.value().status.variant, POLL).toBe("success");
        await expect.poll(() => core.streams(key), POLL).toBe(true);
        core.stream(new framer.Frame(key, write(10n, 2, stamps[0])));
        core.stream(new framer.Frame(key, write(12n, 1, stamps[1])));
        core.stream(new framer.Frame(key, write(13n, 2, stamps[2])));
        await expect.poll(() => latest.value().series.length, POLL).toBe(5);
        // The second and third writes share one buffer, and keep their own stamps.
        expect(latest.value().series.series).toHaveLength(2);
        const [t0, t1, t2] = stamps.map((t) => t.valueOf());
        expect(timedOf(latest)).toEqual([t0, t0 + 1n, t1, t2, t2 + 1n]);
      } finally {
        latest.close();
        await close();
      }
    });

    it("should stream the index only once timeAt needs it", async () => {
      const core = createCore();
      const { open, close } = createLatest(core);
      const latest = open(100);
      try {
        await expect.poll(() => latest.value().status.variant, POLL).toBe("success");
        expect(core.streams(core.index.key)).toBe(false);
        latest.timeAt(0n);
        await expect.poll(() => core.streams(core.index.key), POLL).toBe(true);
      } finally {
        latest.close();
        await close();
      }
    });

    it("should release the index's stream on close", async () => {
      const core = createCore();
      const { open, close } = createLatest(core);
      const latest = open(100);
      try {
        await expect.poll(() => latest.value().status.variant, POLL).toBe("success");
        latest.timeAt(0n);
        await expect.poll(() => core.streams(core.index.key), POLL).toBe(true);
        latest.close();
        await expect.poll(() => core.streams(core.index.key), POLL).toBe(false);
      } finally {
        await close();
      }
    });

    it("should time the samples an update adds", async () => {
      const core = createCore();
      const { open, close } = createLatest(core);
      const latest = open(10);
      // Returns true once the latest holds at least count samples, each one timed.
      const timed = (count: number): boolean => {
        const times = timedOf(latest);
        return times.length >= count && times.every((t) => t != null);
      };
      try {
        await expect.poll(() => timed(10), POLL).toBe(true);
        latest.update(undefined, 200);
        await expect.poll(() => timed(200), POLL).toBe(true);
      } finally {
        latest.close();
        await close();
      }
    });

    it("should time every write of a count past the buffer size", async () => {
      const core = createCore(true);
      const { open, close } = createLatest(core);
      const key = core.virtual!.key;
      const count = 70_000;
      const latest = open(count, undefined, key);
      const stampOf = (i: number): TimeStamp => START.add(TimeSpan.milliseconds(i));
      try {
        await expect.poll(() => latest.value().status.variant, POLL).toBe("success");
        await expect.poll(() => core.streams(key), POLL).toBe(true);
        const writes = Array.from({ length: count }, (_, i) =>
          write(BigInt(i), 1, stampOf(i)),
        );
        core.stream(new framer.Frame(new Array(count).fill(key), writes));
        await expect.poll(() => latest.value().series.length, POLL).toBe(count);
        expect(latest.timeAt(0n)).toEqual(stampOf(0));
        expect(latest.timeAt(BigInt(count - 1))).toEqual(stampOf(count - 1));
      } finally {
        latest.close();
        await close();
      }
    });

    it("should keep each write's stamp when the load lands after it", async () => {
      const core = createCore(true);
      let release: () => void = () => {};
      const released = new Promise<void>((resolve) => (release = resolve));
      const { open, close } = createLatest(core, undefined, undefined, async () => {
        await released;
      });
      const key = core.virtual!.key;
      const latest = open(100, undefined, key);
      const stamps = [0, 1, 2].map((i) => START.add(TimeSpan.seconds(i)));
      try {
        await expect.poll(() => core.streams(key), POLL).toBe(true);
        stamps.forEach((t, i) =>
          core.stream(new framer.Frame(key, write(10n + 2n * BigInt(i), 2, t))),
        );
        await expect.poll(() => latest.value().series.length, POLL).toBe(6);
        // The last two writes share one buffer, which the load closes as it lands.
        expect(latest.value().series.series).toHaveLength(2);
        release();
        await expect.poll(() => latest.value().status.variant, POLL).toBe("success");
        const [t0, t1, t2] = stamps.map((t) => t.valueOf());
        expect(timedOf(latest)).toEqual([t0, t0 + 1n, t1, t1 + 1n, t2, t2 + 1n]);
      } finally {
        latest.close();
        await close();
      }
    });

    // Returns a gate that holds the index reads while closed, and counts them.
    const createGate = (core: SyntheticCore) => {
      let release: () => void = () => {};
      let held: Promise<void> | null = null;
      const gate = {
        calls: 0,
        close: () => {
          held = new Promise((resolve) => (release = resolve));
        },
        open: () => release(),
        onRead: async (key: channel.Key) => {
          if (key !== core.index.key) return;
          gate.calls++;
          if (held != null) await held;
        },
      };
      return gate;
    };

    it("should stamp a live buffer that grows while its index read runs", async () => {
      const core = createCore();
      const gate = createGate(core);
      const { open, close } = createLatest(core, gate.onRead);
      const latest = open(1_000);
      try {
        await expect.poll(() => latest.value().status.variant, POLL).toBe("success");
        // The first streamed series stays apart from the read's, and the next one
        // starts the leading buffer.
        core.advance(TimeSpan.milliseconds(100));
        core.advance(TimeSpan.milliseconds(100));
        await expect.poll(() => latest.value().series.series.length, POLL).toBe(3);
        const leading = latest.value().series.series[2];
        const { alignment, length } = leading;
        gate.close();
        expect(latest.timeAt(alignment)).toBeNull();
        await expect.poll(() => gate.calls, POLL).toBe(1);
        core.advance(TimeSpan.milliseconds(100));
        await expect.poll(() => leading.length, POLL).toBeGreaterThan(length);
        gate.open();
        await expect.poll(() => latest.timeAt(alignment), POLL).not.toBeNull();
        await expect.poll(() => timedInOrder(latest), POLL).toBe(true);
      } finally {
        latest.close();
        await close();
      }
    });

    it("should read the index once while a read of it runs", async () => {
      const core = createCore();
      const gate = createGate(core);
      const { open, close } = createLatest(core, gate.onRead);
      const latest = open(100);
      try {
        await expect.poll(() => latest.value().status.variant, POLL).toBe("success");
        const [{ alignment }] = latest.value().series.series;
        gate.close();
        expect(latest.timeAt(alignment)).toBeNull();
        expect(latest.timeAt(alignment + 1n)).toBeNull();
        expect(latest.timeAt(alignment)).toBeNull();
        await sleep.sleep(TimeSpan.milliseconds(20));
        expect(gate.calls).toBe(1);
        gate.open();
        await expect.poll(() => latest.timeAt(alignment), POLL).not.toBeNull();
        expect(gate.calls).toBe(1);
      } finally {
        latest.close();
        await close();
      }
    });

    it("should ignore an index read that lands after close", async () => {
      const core = createCore();
      const gate = createGate(core);
      const { open, close } = createLatest(core, gate.onRead);
      let notified = 0;
      const latest = open(100, () => notified++);
      try {
        await expect.poll(() => latest.value().status.variant, POLL).toBe("success");
        const [{ alignment }] = latest.value().series.series;
        gate.close();
        latest.timeAt(alignment);
        await expect.poll(() => gate.calls, POLL).toBe(1);
        latest.close();
        const before = notified;
        gate.open();
        await sleep.sleep(TimeSpan.milliseconds(20));
        expect(notified).toBe(before);
        expect(latest.timeAt(alignment)).toBeNull();
      } finally {
        await close();
      }
    });

    it("should not read the index for samples with no time range", async () => {
      const core = new SyntheticCore({
        rate: Rate.hz(100),
        start: START,
        now: START.add(TimeSpan.minutes(10)),
        dataUntimed: true,
      });
      const gate = createGate(core);
      const { open, close } = createLatest(core, gate.onRead);
      const latest = open(1_000);
      try {
        await expect.poll(() => latest.value().status.variant, POLL).toBe("success");
        core.advance(TimeSpan.milliseconds(100));
        await expect.poll(() => latest.value().series.series.length, POLL).toBe(2);
        const untimed = latest.value().series.series[1];
        expect(untimed.timeRange.isZero).toBe(true);
        expect(latest.timeAt(untimed.alignment)).toBeNull();
        await sleep.sleep(TimeSpan.milliseconds(20));
        expect(gate.calls).toBe(0);
      } finally {
        latest.close();
        await close();
      }
    });

    it("should read the index again after a read fails", async () => {
      const core = createCore();
      let failing = true;
      const { open, close, reads } = createLatest(core, undefined, (key, read) => {
        if (failing && key === core.index.key) throw new Error("connection lost");
        return read;
      });
      const latest = open(100);
      try {
        await expect.poll(() => latest.value().status.variant, POLL).toBe("success");
        const [{ alignment }] = latest.value().series.series;
        const before = reads();
        expect(latest.timeAt(alignment)).toBeNull();
        await expect.poll(reads, POLL).toBe(before + 1);
        await sleep.sleep(TimeSpan.milliseconds(20));
        failing = false;
        core.advance(TimeSpan.seconds(5), false);
        await expect.poll(() => latest.timeAt(alignment), POLL).not.toBeNull();
        expect(reads()).toBe(before + 2);
      } finally {
        latest.close();
        await close();
      }
    });

    it("should keep its write stamps while a static window rejects later writes", async () => {
      const core = createCore(true);
      const { open, close } = createLatest(core);
      const key = core.virtual!.key;
      const end = START.add(TimeSpan.seconds(10));
      const latest = open(2, undefined, key, staticWindow(new TimeRange(START, end)));
      const [t0, t1] = [1, 2].map((i) => START.add(TimeSpan.seconds(i)));
      try {
        await expect.poll(() => latest.value().status.variant, POLL).toBe("success");
        await expect.poll(() => core.streams(key), POLL).toBe(true);
        core.stream(new framer.Frame(key, write(10n, 1, t0)));
        core.stream(new framer.Frame(key, write(11n, 1, t1)));
        await expect.poll(() => latest.value().series.length, POLL).toBe(2);
        for (let i = 0; i < 5; i++)
          core.stream(
            new framer.Frame(
              key,
              write(12n + BigInt(i), 1, end.add(TimeSpan.seconds(i))),
            ),
          );
        await sleep.sleep(TimeSpan.milliseconds(20));
        expect(latest.value().series.length).toBe(2);
        expect(timedOf(latest)).toEqual([t0.valueOf(), t1.valueOf()]);
      } finally {
        latest.close();
        await close();
      }
    });

    it("should stamp the writes after a data type change", async () => {
      const core = createCore(true);
      const { open, close } = createLatest(core);
      const key = core.virtual!.key;
      const latest = open(100, undefined, key);
      const [t0, t1] = [0, 1].map((i) => START.add(TimeSpan.seconds(i)));
      try {
        await expect.poll(() => latest.value().status.variant, POLL).toBe("success");
        await expect.poll(() => core.streams(key), POLL).toBe(true);
        core.stream(new framer.Frame(key, write(10n, 2, t0)));
        await expect.poll(() => latest.value().series.length, POLL).toBe(2);
        const retyped = new Series({
          data: new Float64Array(2),
          dataType: DataType.FLOAT64,
          alignment: 12n,
          timeRange: t1.spanRange(TimeSpan.nanoseconds(2)),
        });
        core.stream(new framer.Frame(key, retyped));
        await expect
          .poll(() => latest.value().series.series[0]?.dataType.toString(), POLL)
          .toBe(DataType.FLOAT64.toString());
        expect(timedOf(latest)).toEqual([t1.valueOf(), t1.valueOf() + 1n]);
        expect(latest.timeAt(10n)).toBeNull();
      } finally {
        latest.close();
        await close();
      }
    });

    it("should return null for an alignment it does not hold", async () => {
      const core = createCore(true);
      const { open, close } = createLatest(core);
      const stored = open(100);
      const key = core.virtual!.key;
      const unindexed = open(100, undefined, key);
      try {
        await expect
          .poll(() => {
            const times = timedOf(stored);
            return times.length > 0 && times.every((t) => t != null);
          }, POLL)
          .toBe(true);
        expect(stored.timeAt(0n)).toBeNull();
        await expect.poll(() => unindexed.value().status.variant, POLL).toBe("success");
        await expect.poll(() => core.streams(key), POLL).toBe(true);
        core.stream(new framer.Frame(key, write(10n, 2, START)));
        await expect.poll(() => unindexed.timeAt(10n), POLL).toEqual(START);
        expect(unindexed.timeAt(11n)).toEqual(START.add(TimeSpan.NANOSECOND));
        expect(unindexed.timeAt(9n)).toBeNull();
        expect(unindexed.timeAt(12n)).toBeNull();
      } finally {
        stored.close();
        unindexed.close();
        await close();
      }
    });
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
