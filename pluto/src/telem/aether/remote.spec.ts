// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type channel,
  DataType,
  framer,
  type status as cstatus,
  ValidationError,
} from "@synnaxlabs/client";
import { SyntheticCore, type SyntheticCoreProps } from "@synnaxlabs/client/testutil";
import {
  bounds,
  type breaker,
  dynamicWindow,
  Rate,
  type Reduction,
  Series,
  sleep,
  staticWindow,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { afterEach, assert, describe, expect, it, vi } from "vitest";

import { telem } from "@/telem/aether";
import { createFactory } from "@/telem/aether/factory";
import {
  ChannelData,
  type ChannelDataProps,
  type Client,
  DISCONNECTED_STATUS,
  streamChannelStringValue,
  streamChannelValue,
} from "@/telem/aether/remote";
import {
  type NumberSource,
  type StringSource,
  type ValueProps,
} from "@/telem/aether/telem";
import { telemTest } from "@/telem/aether/testutil";
import { line } from "@/vis/line/aether";

const RATE = Rate.hz(100);
const START = TimeStamp.seconds(1_000_000);
const NOW = START.add(TimeSpan.minutes(10));
// The newest sample the Core holds at NOW.
const NEWEST = 59_999;
// A historical window: it ends before the samples the Core can still commit.
const HOME = new TimeRange(START, NOW.sub(TimeSpan.seconds(2)));
const POLL = { timeout: 5000, interval: 5 };
const BREAKER: breaker.Config = {
  baseInterval: TimeSpan.milliseconds(10),
  maxInterval: TimeSpan.milliseconds(50),
};
const OPTIONS: Partial<framer.FeedProps> = {
  batchDebounce: TimeSpan.milliseconds(5),
  removalDelay: TimeSpan.ZERO,
  breaker: BREAKER,
};

const createCore = (props: Partial<SyntheticCoreProps> = {}): SyntheticCore =>
  new SyntheticCore({ rate: RATE, start: START, now: NOW, ...props });

// The newest stored sample of a channel, as the Core reads it.
const newest = (core: SyntheticCore, key: channel.Key): Series =>
  core.readLatest([key], 1).get(key).series[0];

const ns = (t: TimeStamp): number => Number(t.valueOf());

describe("remote", () => {
  const closers: Array<() => Promise<void> | void> = [];
  afterEach(async () => {
    for (const close of closers.splice(0).reverse()) await close();
  });

  // Connects to the Core as the telemetry provider does, with fast batches and retries.
  const connect = (
    core: SyntheticCore,
    props: Partial<framer.FeedProps> = {},
  ): Client => {
    const client = telemTest.createClient(core, { ...OPTIONS, ...props });
    closers.push(async () => await client.feed.close());
    return client;
  };

  describe("channel values", () => {
    const createNumber = (client: Client | null, ch: channel.Key): NumberSource => {
      const spec = streamChannelValue({ channel: ch });
      const source = createFactory(client).create(spec) as NumberSource;
      closers.push(() => source.cleanup?.());
      return source;
    };

    const createString = (client: Client | null, ch: channel.Key): StringSource => {
      const spec = streamChannelStringValue({ channel: ch });
      const source = createFactory(client).create(spec) as StringSource;
      closers.push(() => source.cleanup?.());
      return source;
    };

    it("should read NaN for a zero channel", () => {
      const source = createNumber(connect(createCore()), 0);
      const { data, status } = source.value();
      expect(data).toBeNaN();
      expect(status.variant).toBe("success");
    });

    it("should read an empty string for a zero channel", () => {
      expect(createString(connect(createCore()), 0).value().data).toBe("");
    });

    it("should report loading until the newest stored sample lands", async () => {
      const core = createCore();
      const { key } = core.data[0];
      const source = createNumber(connect(core), key);
      const first = source.value();
      expect(first.data).toBeNaN();
      expect(first.status.variant).toBe("loading");
      await expect.poll(() => source.value().status.variant, POLL).toBe("success");
      expect(source.value().data).toBe(newest(core, key).at(-1));
    });

    it("should notify once for the stored sample and each live one", async () => {
      const core = createCore();
      const { key } = core.data[0];
      const source = createNumber(connect(core), key);
      const handler = vi.fn();
      source.onChange(handler);
      source.value();
      await expect.poll(() => handler.mock.calls.length, POLL).toBe(1);
      expect(source.value().data).toBe(newest(core, key).at(-1));
      core.advance(RATE.period);
      await expect.poll(() => handler.mock.calls.length, POLL).toBe(2);
      const latest = newest(core, key).at(-1);
      expect(source.value().data).toBe(latest);
      expect(latest).not.toBe(newest(createCore(), key).at(-1));
    });

    it("should notify a live sample that a read converted first", async () => {
      const core = createCore({ channels: 2 });
      const client = connect(core);
      const [a, b] = core.data.map(({ key }) => createNumber(client, key));
      const heard = [vi.fn(), vi.fn()];
      // Each notice reads the other source, as a pipeline over both does.
      a.onChange(() => {
        heard[0]();
        b.value();
      });
      b.onChange(() => {
        heard[1]();
        a.value();
      });
      a.value();
      b.value();
      const counts = () => heard.map((h) => h.mock.calls.length);
      await expect.poll(counts, POLL).toEqual([1, 1]);
      core.advance(RATE.period);
      await expect.poll(counts, POLL).toEqual([2, 2]);
    });

    it("should notify a rewrite of the newest sample at its time and count", async () => {
      const core = createCore();
      const { key } = core.data[0];
      let reads = 0;
      const client = connect(core, {
        readLatest: async (keys, count, tr) => {
          reads++;
          return core.readLatest(keys, count, tr);
        },
      });
      const source = createNumber(client, key);
      const handler = vi.fn();
      source.onChange(handler);
      source.value();
      await expect.poll(() => handler.mock.calls.length, POLL).toBe(1);
      const { time, appended } = source.value();
      await expect.poll(() => core.streams(core.historyChange.key), POLL).toBe(true);
      core.rewrite(key, new TimeRange(START, core.now()), 10);
      await expect.poll(() => reads, POLL).toBe(2);
      const rewritten = newest(core, key).at(-1);
      await expect.poll(() => source.value().data, POLL).toBe(rewritten);
      expect(handler).toHaveBeenCalledTimes(2);
      expect(source.value().time).toEqual(time);
      expect(source.value().appended).toBe(appended);
    });

    it("should clear a value once a history change deletes its samples", async () => {
      const core = createCore();
      const { key } = core.data[0];
      let deleted = false;
      const client = connect(core, {
        readLatest: async (keys, count, tr) =>
          deleted ? new framer.Frame() : core.readLatest(keys, count, tr),
      });
      const source = createNumber(client, key);
      const handler = vi.fn();
      source.onChange(handler);
      source.value();
      await expect.poll(() => handler.mock.calls.length, POLL).toBe(1);
      await expect.poll(() => core.streams(core.historyChange.key), POLL).toBe(true);
      // The Core deletes every sample, so its reads return none.
      deleted = true;
      core.rewrite(key, new TimeRange(START, core.now()), 0);
      await expect.poll(() => handler.mock.calls.length, POLL).toBe(2);
      const { data, status, time } = source.value();
      expect(data).toBeNaN();
      expect(status.variant).toBe("success");
      expect(time).toBeUndefined();
    });

    it("should count each streamed sample as appended", async () => {
      const core = createCore();
      const { key } = core.data[0];
      const source = createNumber(connect(core), key);
      await expect.poll(() => source.value().status.variant, POLL).toBe("success");
      const { appended = 0 } = source.value();
      await expect.poll(() => core.streams(key), POLL).toBe(true);
      core.advance(RATE.period);
      await expect.poll(() => source.value().appended, POLL).toBe(appended + 1);
    });

    it("should report the end of the newest sample as its time", async () => {
      const core = createCore();
      const { key } = core.data[0];
      const source = createNumber(connect(core), key);
      await expect.poll(() => source.value().status.variant, POLL).toBe("success");
      expect(source.value().time).toEqual(newest(core, key).timeRange.end);
      core.advance(RATE.period);
      const { end } = newest(core, key).timeRange;
      await expect.poll(() => source.value().time, POLL).toEqual(end);
    });

    it("should convert a sample once however often it is read", async () => {
      const core = createCore({ text: (i) => `sample ${i}` });
      const { key } = core.data[0];
      core.retype(key, DataType.STRING);
      const source = createString(connect(core), key);
      await expect.poll(() => source.value().data, POLL).toBe(`sample ${NEWEST}`);
      const asString = vi.spyOn(Series.prototype, "asString");
      try {
        for (let i = 0; i < 10; i++) source.value();
        expect(asString).not.toHaveBeenCalled();
        core.advance(RATE.period);
        await expect.poll(() => source.value().data, POLL).toBe(`sample ${NEWEST + 1}`);
        for (let i = 0; i < 10; i++) source.value();
        expect(asString).toHaveBeenCalledTimes(1);
      } finally {
        asString.mockRestore();
      }
    });

    it("should keep the spaces and punctuation of a string sample", async () => {
      const core = createCore({ text: (i) => `valve ${i}: open, 3.5 bar!` });
      const { key } = core.data[0];
      core.retype(key, DataType.STRING);
      const source = createString(connect(core), key);
      await expect
        .poll(() => source.value().data, POLL)
        .toBe(`valve ${NEWEST}: open, 3.5 bar!`);
    });

    it("should read a numeric channel as text", async () => {
      const core = createCore();
      const { key } = core.data[0];
      const source = createString(connect(core), key);
      await expect
        .poll(() => source.value().data, POLL)
        .toBe(newest(core, key).asString(-1));
    });

    it("should stop on cleanup and subscribe again on the next read", async () => {
      const core = createCore();
      const client = connect(core);
      const { key } = core.data[0];
      const source = createNumber(client, key);
      const witness = createNumber(client, key);
      await expect.poll(() => source.value().status.variant, POLL).toBe("success");
      await expect.poll(() => witness.value().status.variant, POLL).toBe("success");
      const handler = vi.fn();
      source.onChange(handler);
      source.cleanup?.();
      core.advance(RATE.period);
      const latest = newest(core, key).at(-1);
      // The witness shares the stream, so the sample reached every subscription.
      await expect.poll(() => witness.value().data, POLL).toBe(latest);
      expect(handler).not.toHaveBeenCalled();
      expect(source.value().status.variant).toBe("loading");
      await expect.poll(() => source.value().data, POLL).toBe(latest);
    });

    it("should show live samples while its history read fails", async () => {
      const core = createCore();
      const { key } = core.data[0];
      const client = connect(core, {
        readLatest: async () => {
          throw new Error("history unavailable");
        },
      });
      const source = createNumber(client, key);
      await expect
        .poll(() => source.value().status.description, POLL)
        .toBe("history unavailable");
      expect(source.value().data).toBeNaN();
      core.advance(RATE.period);
      const latest = newest(core, key).at(-1);
      await expect.poll(() => source.value().data, POLL).toBe(latest);
      expect(source.value().status.variant).toBe("error");
    });

    it("should notify with an error when the channel cannot be retrieved", async () => {
      const source = createNumber(connect(createCore()), 999);
      const handler = vi.fn();
      source.onChange(handler);
      source.value();
      await expect.poll(() => handler.mock.calls.length, POLL).toBe(1);
      expect(source.value().status).toMatchObject({
        variant: "error",
        message: "Failed to stream channel value",
        description: "channel 999 not found",
      });
    });

    it("should notify when only its status changes", async () => {
      const core = createCore();
      const { key } = core.data[0];
      const source = createNumber(connect(core), key);
      source.value();
      await expect.poll(() => source.value().status.variant, POLL).toBe("success");
      const variants: string[] = [];
      source.onChange(() => variants.push(source.value().status.variant));
      core.disconnect();
      await expect.poll(() => variants.at(-1), POLL).toBe("warning");
      core.reconnect();
      await expect.poll(() => source.value().status.variant, POLL).toBe("success");
      expect(variants.at(-1)).not.toBe("warning");
    });

    it("should notify with the error of a channel the Core denies", async () => {
      const core = createCore();
      const { key } = core.data[0];
      core.deny(key);
      const source = createNumber(connect(core), key);
      const statuses: cstatus.Status[] = [];
      source.onChange(() => statuses.push(source.value().status));
      source.value();
      await expect.poll(() => statuses.at(-1)?.variant, POLL).toBe("error");
      expect(statuses.at(-1)).toMatchObject({
        message: "Failed to stream channel",
        description: `You do not have permission to do that. access to channel ${key} denied`,
      });
      expect(source.value().data).toBeNaN();
    });

    it("should average from the first sample after a retrieval recovers", async () => {
      const core = createCore();
      const { key } = core.data[0];
      const client = connect(core);
      let failures = 1;
      const flaky = telemTest.withRetrieve(client, async (ch) => {
        if (failures-- > 0) throw new Error("connection refused");
        return await client.channels.retrieve(ch);
      });
      const source = createFactory(flaky).create(
        telem.sourcePipeline("number", {
          connections: [{ from: "valueStream", to: "average" }],
          segments: {
            valueStream: streamChannelValue({ channel: key }),
            average: telem.rollingAverage({ windowSize: 3 }),
          },
          outlet: "average",
        }),
      ) as NumberSource;
      closers.push(() => source.cleanup?.());
      const statuses: cstatus.Status[] = [];
      source.onChange(() => statuses.push(source.value().status));
      source.value();
      const latest = newest(core, key).at(-1);
      await expect.poll(() => source.value().data, POLL).toBe(latest);
      expect(statuses[0]).toMatchObject({
        variant: "error",
        description: "connection refused",
      });
    });
  });

  describe("ChannelData", () => {
    const create = (
      client: Client | null,
      props: Partial<ChannelDataProps>,
    ): ChannelData => {
      const source = new ChannelData(
        client,
        { channel: 0, window: staticWindow(HOME), ...props },
        undefined,
        BREAKER,
      );
      closers.push(() => source.cleanup());
      return source;
    };

    const settle = async (source: ChannelData, props?: ValueProps): Promise<void> =>
      await expect.poll(() => source.value(props).status.variant, POLL).toBe("success");

    // Counts the reductions of every read the Core serves.
    const connectCounting = (core: SyntheticCore) => {
      const reductions: Array<Reduction | undefined> = [];
      const client = connect(core, {
        readRemote: async (tr, keys, reduction) => {
          reductions.push(reduction);
          return core.read(tr, keys, reduction);
        },
      });
      return { client, reductions };
    };

    const view = (start: TimeSpan, end: TimeSpan, points = 1000): ValueProps => ({
      bounds: { lower: ns(START.add(start)), upper: ns(START.add(end)) },
      points,
    });

    it("should return nothing when no channel is set", () => {
      const source = create(connect(createCore()), {});
      const { data, status } = source.value();
      expect(data[0]).toEqual(bounds.INVALID);
      expect(data[1]).toHaveLength(0);
      expect(status.variant).toBe("success");
    });

    it("should draw the home view reduced to about its points", async () => {
      const core = createCore();
      const { client, reductions } = connectCounting(core);
      const y = create(client, { channel: core.data[0].key });
      expect(y.value().status.variant).toBe("loading");
      await settle(y);
      const [, series] = y.value().data;
      expect(series.series.every((s) => s.alignmentMultiple > 1n)).toBe(true);
      // Each min_max group holds two samples.
      expect(series.length / 2).toBeGreaterThan(900);
      expect(series.length / 2).toBeLessThan(1100);
      expect(series.series[0].timeRange.start).toEqual(START);
      expect(
        series.series.at(-1)?.timeRange.end.afterEq(HOME.end.sub(RATE.period)),
      ).toBe(true);
      expect(reductions.every((r) => r?.variant === "limit")).toBe(true);
    });

    it("should draw a zoomed view at full resolution", async () => {
      const core = createCore();
      const y = create(connect(core), { channel: core.data[0].key });
      await settle(y);
      const zoomed = view(TimeSpan.seconds(60), TimeSpan.seconds(65));
      y.value(zoomed);
      await expect
        .poll(() => {
          const { data, status } = y.value(zoomed);
          return (
            status.variant === "success" &&
            data[1].series.every((s) => s.alignmentMultiple === 1n)
          );
        }, POLL)
        .toBe(true);
      const [, series] = y.value(zoomed).data;
      const span = new TimeRange(
        START.add(TimeSpan.seconds(60)),
        START.add(TimeSpan.seconds(65)),
      );
      expect(series.series.some((s) => s.timeRange.overlapsWith(span))).toBe(true);
      expect(series.length).toBeGreaterThanOrEqual(500);
    });

    it("should draw a zoom out from the samples it holds before the zoom settles", async () => {
      const core = createCore();
      const { client, reductions } = connectCounting(core);
      const y = create(client, { channel: core.data[0].key });
      await settle(y);
      const zoomed = view(TimeSpan.seconds(60), TimeSpan.seconds(65));
      await expect
        .poll(() => {
          const { data, status } = y.value(zoomed);
          return (
            status.variant === "success" &&
            data[1].series.every((s) => s.alignmentMultiple === 1n)
          );
        }, POLL)
        .toBe(true);
      const reads = reductions.length;
      const { bounds: home } = view(TimeSpan.ZERO, HOME.span);
      const [, series] = y.value({ bounds: home }).data;
      expect(series.series[0].timeRange.start).toEqual(START);
      expect(
        series.series.at(-1)?.timeRange.end.afterEq(HOME.end.sub(RATE.period)),
      ).toBe(true);
      await sleep.sleep(TimeSpan.milliseconds(50));
      expect(reductions).toHaveLength(reads);
    });

    it("should read nothing for bounds given without points", async () => {
      const core = createCore();
      const { client, reductions } = connectCounting(core);
      const y = create(client, { channel: core.data[0].key });
      const { bounds: home } = view(TimeSpan.ZERO, HOME.span);
      // The line reads again on every change while its zoom settles.
      const stop = y.onChange(() => y.value({ bounds: home }));
      y.value({ bounds: home });
      await sleep.sleep(TimeSpan.milliseconds(50));
      stop();
      expect(reductions).toHaveLength(0);
      await settle(y, { bounds: home, points: 1000 });
      expect(reductions.length).toBeGreaterThan(0);
    });

    it("should release the series a new view no longer draws", async () => {
      const core = createCore();
      const y = create(connect(core), { channel: core.data[0].key });
      await settle(y);
      const home = y.value().data[1].series;
      expect(home.every((s) => s.refCount === 1)).toBe(true);
      const zoomed = view(TimeSpan.seconds(60), TimeSpan.seconds(65));
      await expect
        .poll(() => y.value(zoomed).data[1].series.some((s) => home.includes(s)), POLL)
        .toBe(false);
      expect(home.every((s) => s.refCount === 0)).toBe(true);
      expect(y.value(zoomed).data[1].series.every((s) => s.refCount === 1)).toBe(true);
    });

    it("should bound an average line by min_max samples of its home view", async () => {
      const core = createCore();
      const y = create(connect(core), {
        channel: core.data[0].key,
        aggregation: "average",
      });
      await settle(y);
      // The Core's min_max groups span each value plus and minus one.
      expect(y.value().data[0].upper).toBeCloseTo(1, 1);
      // The draw bounds the line until the min_max samples load.
      expect(y.value({ extremesKept: true }).data[0].upper).toBeCloseTo(1, 1);
      await expect
        .poll(() => y.value({ extremesKept: true }).data[0].upper, POLL)
        .toBeGreaterThan(1.5);
      const [kept, series] = y.value({ extremesKept: true }).data;
      expect(kept.upper).toBeCloseTo(2, 1);
      expect(kept.lower).toBeCloseTo(-2, 1);
      expect(series).not.toBe(y.value().data[1]);
    });

    it("should report the worse status of its draw and bounds reads", async () => {
      const core = createCore();
      const client = connect(core, {
        readRemote: async (tr, keys, reduction) => {
          if (reduction?.variant === "limit" && reduction.aggregation === "min_max")
            throw new ValidationError("min_max reads are down");
          return core.read(tr, keys, reduction);
        },
      });
      const y = create(client, { channel: core.data[0].key, aggregation: "average" });
      await settle(y);
      await expect
        .poll(() => y.value({ extremesKept: true }).status.variant, POLL)
        .toBe("error");
      expect(y.value().status).toMatchObject({
        message: "Failed to read channel data",
        description: "min_max reads are down",
      });
    });

    it("should bound a line by its draw while reopened bounds load", async () => {
      const core = createCore();
      const y = create(connect(core), {
        channel: core.data[0].key,
        aggregation: "average",
      });
      const b = { extremesKept: true };
      const draw = { points: 1000 };
      await settle(y, draw);
      await expect.poll(() => y.value(b).data[0].upper, POLL).toBeGreaterThan(1.5);
      // Draws with no bounds read between them close the bounds.
      y.value(draw);
      y.value(draw);
      const { data, status } = y.value(b);
      expect(status.variant).toBe("success");
      expect(data[0].upper).toBeCloseTo(1, 1);
      await expect.poll(() => y.value(b).data[0].upper, POLL).toBeGreaterThan(1.5);
    });

    it("should send each subscription only the requests that change", async () => {
      const core = createCore();
      const client = connect(core);
      const subscribe = vi.spyOn(client.feed, "subscribe");
      const y = create(client, { channel: core.data[0].key, aggregation: "average" });
      const draw = { points: 1000 };
      const b = { extremesKept: true };
      await settle(y, draw);
      y.value(b);
      const updates = subscribe.mock.results.map(({ value }) =>
        vi.spyOn(value as framer.Subscription, "update"),
      );
      const counts = () => updates.map((u) => u.mock.calls.length);
      y.value(b);
      y.value(draw);
      expect(counts()).toEqual([0, 0]);
      y.value({ ...b, points: 500 });
      y.value(view(TimeSpan.seconds(60), TimeSpan.seconds(65)));
      expect(counts()).toEqual([1, 1]);
    });

    it("should close bounds that no read used since the last draw", async () => {
      const core = createCore();
      const client = connect(core, {
        readRemote: async (tr, keys, reduction) => {
          if (reduction?.variant === "limit" && reduction.aggregation === "min_max")
            throw new ValidationError("min_max reads are down");
          return core.read(tr, keys, reduction);
        },
      });
      const y = create(client, { channel: core.data[0].key, aggregation: "average" });
      await settle(y);
      const readBounds = () => y.value({ extremesKept: true }).status.variant;
      await expect.poll(readBounds, POLL).toBe("error");
      const draw = { points: 1000 };
      expect(y.value(draw).status.variant).toBe("error");
      expect(y.value(draw).status.variant).toBe("success");
      await expect.poll(readBounds, POLL).toBe("error");
    });

    it("should report the cache's error for a virtual channel's index", async () => {
      const core = createCore({ derived: true });
      const x = create(connect(core), {
        channel: core.virtual!.key,
        useIndexOfChannel: true,
      });
      await expect.poll(() => x.value().status.variant, POLL).toBe("error");
      expect(x.value().status).toMatchObject({
        message: "Failed to read channel data",
        description: "virtual channel virtual has no samples to draw",
      });
    });

    it("should draw an xy line's home view with decimate at every zoom", async () => {
      const core = createCore({ channels: 2 });
      const [x, y] = core.data.map((c) => c.key);
      const { client, reductions } = connectCounting(core);
      const source = create(client, { channel: y, xChannel: x });
      const home = view(TimeSpan.ZERO, TimeSpan.minutes(10));
      await settle(source, home);
      expect(source.value(home).aggregation).toBe("decimate");
      const [b, series] = source.value(home).data;
      expect(b).toEqual(series.bounds);
      const zoomed = view(TimeSpan.seconds(60), TimeSpan.seconds(65));
      expect(source.value(zoomed).data[1].series).toEqual(series.series);
      expect(source.value({ extremesKept: true }).data[1].series).toEqual(
        series.series,
      );
      expect(reductions.length).toBeGreaterThan(0);
      expect(
        reductions.every((r) => r?.variant === "limit" && r.aggregation === "decimate"),
      ).toBe(true);
    });

    it("should draw a live xy line's history with decimate", async () => {
      const core = createCore({ channels: 2 });
      const [x, y] = core.data.map((c) => c.key);
      const { client, reductions } = connectCounting(core);
      const source = create(client, {
        channel: y,
        xChannel: x,
        window: dynamicWindow(TimeSpan.minutes(5)),
      });
      await settle(source);
      const [, series] = source.value().data;
      expect(series.series.at(-1)?.timeRange.end).toEqual(
        newest(core, y).timeRange.end,
      );
      const reduced = reductions.filter((r) => r != null);
      expect(reduced.length).toBeGreaterThan(0);
      expect(
        reduced.every((r) => r.variant === "limit" && r.aggregation === "decimate"),
      ).toBe(true);
    });

    it("should draw tiles before a live window and raw samples in it", async () => {
      const core = createCore();
      const { key } = core.data[0];
      const y = create(connect(core), {
        channel: key,
        window: dynamicWindow(TimeSpan.minutes(2)),
      });
      const zoomedOut = view(TimeSpan.minutes(2), TimeSpan.minutes(10));
      await settle(y, zoomedOut);
      const drawn = (): Series[] => y.value(zoomedOut).data[1].series;
      expect(drawn()[0].alignmentMultiple).toBeGreaterThan(1n);
      expect(drawn().at(-1)?.alignmentMultiple).toBe(1n);
      const last = (): unknown => y.value(zoomedOut).data[1].at(-1);
      expect(last()).toBe(newest(core, key).at(-1));
      core.advance(TimeSpan.seconds(1));
      await expect.poll(last, POLL).toBe(newest(core, key).at(-1));
    });

    it("should bound a static index by its home view", async () => {
      const core = createCore();
      const x = create(connect(core), {
        channel: core.data[0].key,
        useIndexOfChannel: true,
      });
      await settle(x);
      expect(x.value().data[0]).toEqual(HOME.numericBounds);
    });

    it("should bound a live index by the span before its newest sample", async () => {
      const core = createCore();
      const { key } = core.data[0];
      const span = TimeSpan.seconds(10);
      const x = create(connect(core), {
        channel: key,
        useIndexOfChannel: true,
        window: dynamicWindow(span),
      });
      await settle(x);
      const upper = ns(newest(core, key).timeRange.end);
      expect(x.value().data[0]).toEqual({
        lower: upper - ns(new TimeStamp(span)),
        upper,
      });
      core.advance(TimeSpan.seconds(1));
      const next = ns(newest(core, key).timeRange.end);
      await expect.poll(() => x.value().data[0].upper, POLL).toBe(next);
    });

    it("should release its series and stop notifying on cleanup", async () => {
      const core = createCore();
      const client = connect(core);
      const props = {
        channel: core.data[0].key,
        window: dynamicWindow(TimeSpan.minutes(1)),
      };
      const y = create(client, props);
      const witness = create(client, props);
      await settle(y);
      await settle(witness);
      // The witness draws some of the same series, so each holds a reference too.
      const drawn = y.value().data[1].series;
      const counts = drawn.map((s) => s.refCount);
      const handler = vi.fn();
      y.onChange(handler);
      y.cleanup();
      expect(drawn.map((s) => s.refCount)).toEqual(counts.map((n) => n - 1));
      core.advance(TimeSpan.seconds(1));
      // The witness shares the stream, so the samples reached every subscription.
      await expect
        .poll(() => witness.value().data[1].at(-1), POLL)
        .toBe(newest(core, props.channel).at(-1));
      expect(handler).not.toHaveBeenCalled();
    });

    it("should retry a channel retrieval that fails to connect", async () => {
      const core = createCore();
      const client = connect(core);
      let failures = 2;
      const flaky = telemTest.withRetrieve(client, async (ch) => {
        if (failures-- > 0) throw new Error("connection refused");
        return await client.channels.retrieve(ch);
      });
      const y = create(flaky, { channel: core.data[0].key });
      const statuses: cstatus.Status[] = [];
      y.onChange(() => statuses.push(y.value().status));
      await settle(y);
      // Both failures are equal, so only the first notifies.
      const errors = statuses.filter((s) => s.variant === "error");
      expect(errors).toHaveLength(1);
      expect(errors[0]).toMatchObject({
        message: "Failed to read channel data",
        description: "connection refused",
      });
    });

    it("should drop a channel retrieval that lands after cleanup", async () => {
      const core = createCore();
      const client = connect(core);
      let release: () => void = () => {};
      const released = new Promise<void>((resolve) => (release = resolve));
      let [started, landed] = [false, false];
      const gated = telemTest.withRetrieve(client, async (ch) => {
        started = true;
        await released;
        const retrieved = await client.channels.retrieve(ch);
        landed = true;
        return retrieved;
      });
      const y = create(gated, { channel: core.data[0].key });
      const handler = vi.fn();
      y.onChange(handler);
      y.value();
      await expect.poll(() => started, POLL).toBe(true);
      y.cleanup();
      release();
      await expect.poll(() => landed, POLL).toBe(true);
      expect(handler).not.toHaveBeenCalled();
    });

    it("should notify with an error and park on a missing channel", async () => {
      const y = create(connect(createCore()), { channel: 999 });
      const handler = vi.fn();
      y.onChange(handler);
      await expect.poll(() => y.value().status.variant, POLL).toBe("error");
      expect(y.value().status).toMatchObject({
        message: "Failed to read channel data",
        description: "channel 999 not found",
      });
      expect(handler).toHaveBeenCalledOnce();
    });
  });

  // A line pairs x and y by alignment, and the Core streams a sample with an alignment
  // that never pairs with the one it reads back.
  describe("live line", () => {
    const LIVE_START = TimeStamp.seconds(100);
    const PERIOD = TimeSpan.milliseconds(100);

    const createLive = (): { core: SyntheticCore; client: Client } => {
      const core = new SyntheticCore({
        rate: 10,
        start: LIVE_START,
        now: LIVE_START.add(TimeSpan.seconds(1)),
      });
      return { core, client: connect(core) };
    };

    // Subscribes to the channel as another view of it would, then streams the given
    // number of its samples.
    const stream = async (
      core: SyntheticCore,
      client: Client,
      key: channel.Key,
      count: number,
    ): Promise<void> => {
      const sub = client.feed.subscribe({
        keys: [key],
        limit: { variant: "latest", count: 1 },
      });
      closers.push(() => sub.close());
      // The read lands once the stream acks the channel, so later samples stream.
      await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
      for (let i = 0; i < count; i++) {
        core.advance(PERIOD);
        await sleep.sleep(TimeSpan.ZERO);
      }
    };

    // Opens the x and y sources of a live line of the data channel as a line plot does,
    // with the index as the line's X channel or with no X channel.
    const createLine = (
      core: SyntheticCore,
      client: Client,
      span: TimeSpan,
      indexAsX: boolean,
    ): [ChannelData, ChannelData] => {
      const yChannel = core.data[0].key;
      const xChannel = indexAsX ? core.index.key : 0;
      const create = (props: Partial<ChannelDataProps>): ChannelData => {
        const source = new ChannelData(
          client,
          { channel: yChannel, window: dynamicWindow(span), xChannel, ...props },
          core.now,
          BREAKER,
        );
        closers.push(() => source.cleanup());
        return source;
      };
      const x = create(indexAsX ? { channel: xChannel } : { useIndexOfChannel: true });
      return [x, create({})];
    };

    const sample = (t: TimeStamp): number =>
      Math.round(t.span(LIVE_START).seconds / PERIOD.seconds);

    const samples = (from: number, to: number): number[] =>
      Array.from({ length: to - from }, (_, i) => from + i);

    // Returns the index of every sample a line of x and y draws, in order.
    const drawn = (x: ChannelData, y: ChannelData, shown?: ValueProps): number[] =>
      line
        .buildDrawOperations(
          x.value(shown).data[1],
          y.value(shown).data[1],
          line.DEFAULT_OVERLAP_THRESHOLD,
        )
        .flatMap((op) =>
          Array.from({ length: op.count }, (_, i) => {
            const t = op.x.at(op.xOffset + i, true);
            assert(typeof t === "bigint");
            return sample(new TimeStamp(t));
          }),
        )
        .sort((a, b) => a - b);

    const STREAMING = [
      {
        streaming: "index",
        channels: (core: SyntheticCore) => [core.index.key],
        indexAsX: false,
      },
      {
        streaming: "data channel",
        channels: (core: SyntheticCore) => [core.data[0].key],
        indexAsX: false,
      },
      {
        streaming: "index and then its data channel",
        channels: (core: SyntheticCore) => [core.index.key, core.data[0].key],
        indexAsX: false,
      },
      {
        streaming: "index, its X channel,",
        channels: (core: SyntheticCore) => [core.index.key],
        indexAsX: true,
      },
    ];

    it.each(STREAMING)(
      "should draw every sample of a line added while its $streaming streams",
      async ({ channels, indexAsX }) => {
        const { core, client } = createLive();
        for (const key of channels(core)) await stream(core, client, key, 20);
        const [x, y] = createLine(core, client, TimeSpan.seconds(30), indexAsX);
        await expect
          .poll(() => drawn(x, y), POLL)
          .toEqual(samples(0, sample(core.now())));
      },
    );

    // A paused view stays put while the line moves on, so it soon draws from blocks
    // reduced for history.
    it.each(STREAMING)(
      "should keep every sample of a paused line added while its $streaming streams",
      async ({ channels, indexAsX }) => {
        const { core, client } = createLive();
        for (const key of channels(core)) await stream(core, client, key, 200);
        const span = TimeSpan.seconds(2);
        const [x, y] = createLine(core, client, span, indexAsX);
        const end = core.now();
        const [from, to] = [sample(end.sub(span)), sample(end)];
        await expect
          .poll(() => drawn(x, y), POLL)
          .toEqual(expect.arrayContaining(samples(from, to)));
        const paused: ValueProps = {
          bounds: { lower: ns(end.sub(span)), upper: ns(end) },
          points: 2000,
        };
        for (let i = 0; i < 10; i++) {
          core.advance(TimeSpan.seconds(1));
          x.value(paused);
          y.value(paused);
          await sleep.sleep(TimeSpan.milliseconds(20));
        }
        await expect
          .poll(() => drawn(x, y, paused).filter((i) => i >= from && i < to), POLL)
          .toEqual(samples(from, to));
      },
    );
  });

  describe("disconnected", () => {
    it("should still create remote sources from a null-client factory", () => {
      const factory = createFactory(null);
      expect(factory.create(streamChannelValue({ channel: 1 }))).not.toBeNull();
    });

    it("should report a disconnected status instead of throwing", () => {
      const source = createFactory(null).create(
        streamChannelValue({ channel: 1 }),
      ) as NumberSource;
      const { data, status } = source.value();
      expect(data).toBeNaN();
      expect(status).toBe(DISCONNECTED_STATUS);
    });

    it("should return empty data with a disconnected status from a line source", () => {
      const source = new ChannelData(null, { channel: 1, window: staticWindow(HOME) });
      const { data, status } = source.value();
      expect(data[0]).toStrictEqual(bounds.INVALID);
      expect(data[1]).toHaveLength(0);
      expect(status).toBe(DISCONNECTED_STATUS);
    });
  });
});
