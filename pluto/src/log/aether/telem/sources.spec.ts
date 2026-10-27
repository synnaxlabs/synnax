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
} from "@synnaxlabs/client";
import { SyntheticCore, type SyntheticCoreProps } from "@synnaxlabs/client/testutil";
import { Rate, Series, TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { afterEach, describe, expect, it, vi } from "vitest";

import {
  StreamMultiChannelLog,
  type StreamMultiChannelLogProps,
} from "@/log/aether/telem/sources";
import { type LogEntry } from "@/log/aether/telem/types";
import { telem } from "@/telem/aether";
import { telemTest } from "@/telem/aether/testutil";

const RATE = Rate.hz(10);
const START = TimeStamp.seconds(1_000_000);
// Every stored sample, since a log reads up to PAGE stored samples of any age.
const HISTORY = 300;
const NOW = START.add(TimeSpan.seconds(HISTORY / 10));
// The newest stored samples of each channel a log shows when it opens.
const PAGE = 10_000;
const POLL = { timeout: 5000, interval: 5 };
const OPTIONS: Partial<framer.FeedProps> = {
  batchDebounce: TimeSpan.milliseconds(5),
  removalDelay: TimeSpan.ZERO,
  breaker: { baseInterval: TimeSpan.milliseconds(10) },
};
// Streamed samples sit in a leading alignment domain, apart from stored ones.
const LEADING = ((1n << 31n) + 7n) << 32n;

const createCore = (props: Partial<SyntheticCoreProps> = {}): SyntheticCore =>
  new SyntheticCore({ rate: RATE, start: START, now: NOW, channels: 2, ...props });

const samplesOf = (core: SyntheticCore, key: channel.Key, count: number): Series[] =>
  core.readLatest([key], count).get(key).series;

// The text of the newest count samples of a channel, as the Core reads them.
const textOf = (core: SyntheticCore, key: channel.Key, count: number): string[] =>
  samplesOf(core, key, count).flatMap((s) =>
    Array.from({ length: s.length }, (_, i) => s.asString(i, true)),
  );

// The times of the newest count samples of the index.
const timesOf = (core: SyntheticCore, count: number): bigint[] =>
  samplesOf(core, core.index.key, count).flatMap((s) =>
    Array.from({ length: s.length }, (_, i) => s.at(i, true) as bigint),
  );

const valuesOf = (entries: LogEntry[], key: channel.Key): string[] =>
  entries.filter((e) => e.channelKey === key).map((e) => e.value);

const stampsOf = (entries: LogEntry[]): bigint[] =>
  entries.map((e) => e.timestamp.valueOf());

const sorted = (entries: LogEntry[]): boolean =>
  entries.every(
    (e, i) => i === 0 || entries[i - 1].timestamp.valueOf() <= e.timestamp.valueOf(),
  );

// Returns the index series that stamps count samples from alignment at time.
const indexSeries = (alignment: bigint, times: bigint[]): Series =>
  new Series({
    data: new BigInt64Array(times),
    dataType: DataType.TIMESTAMP,
    timeRange: new TimeRange(times[0], times[times.length - 1] + 1n),
    alignment,
  });

describe("StreamMultiChannelLog", () => {
  const closers: Array<() => Promise<void> | void> = [];
  afterEach(async () => {
    for (const close of closers.splice(0).reverse()) await close();
  });

  const connect = (
    core: SyntheticCore,
    props: Partial<framer.FeedProps> = {},
  ): telem.Client => {
    const client = telemTest.createClient(core, { ...OPTIONS, ...props });
    closers.push(async () => await client.feed.close());
    return client;
  };

  // Opens a log that stamps the entries of channels with no index on the Core's clock.
  const open = (
    core: SyntheticCore,
    client: telem.Client | null,
    props: Partial<StreamMultiChannelLogProps> = {},
    options?: telem.CreateOptions,
    maxEntries?: number,
  ): StreamMultiChannelLog => {
    const log = new StreamMultiChannelLog(
      client,
      { channels: [], ...props },
      options,
      core.now,
      maxEntries,
      OPTIONS.breaker,
    );
    closers.push(() => log.cleanup());
    return log;
  };

  const entriesOf = (log: StreamMultiChannelLog): LogEntry[] => log.value().data;

  const settle = async (log: StreamMultiChannelLog, count: number): Promise<void> =>
    await expect.poll(() => entriesOf(log).length, POLL).toBe(count);

  it("should show no entries when no channels are set", () => {
    const core = createCore();
    expect(entriesOf(open(core, connect(core)))).toEqual([]);
  });

  it("should show the stored samples at their sample times, then each live sample", async () => {
    const core = createCore();
    const [a] = core.data.map((c) => c.key);
    const log = open(core, connect(core), { channels: [a] });
    await settle(log, HISTORY);
    expect(valuesOf(entriesOf(log), a)).toEqual(textOf(core, a, HISTORY));
    expect(stampsOf(entriesOf(log))).toEqual(timesOf(core, HISTORY));
    core.advance(TimeSpan.seconds(1));
    await settle(log, HISTORY + 10);
    const live = entriesOf(log).slice(HISTORY);
    expect(live.map((e) => e.value)).toEqual(textOf(core, a, 10));
    expect(stampsOf(live)).toEqual(timesOf(core, 10));
  });

  it("should show stored samples of any age", async () => {
    const core = createCore({ now: START.add(TimeSpan.days(2)), rate: Rate.hz(0.01) });
    const [a] = core.data.map((c) => c.key);
    const log = open(core, connect(core), { channels: [a] });
    const count = timesOf(core, PAGE).length;
    await settle(log, count);
    expect(entriesOf(log)[0].timestamp.valueOf()).toBe(START.valueOf());
  });

  it("should sort the entries of all channels by sample time", async () => {
    const core = createCore();
    const [a, b] = core.data.map((c) => c.key);
    const log = open(core, connect(core), { channels: [a] });
    await settle(log, HISTORY);
    log.setChannels([a, b]);
    await settle(log, 2 * HISTORY);
    core.advance(TimeSpan.seconds(1));
    await settle(log, 2 * HISTORY + 20);
    const entries = entriesOf(log);
    expect(sorted(entries)).toBe(true);
    expect(entries.slice(0, 4).map((e) => e.channelKey)).toEqual([a, b, a, b]);
    expect(valuesOf(entries, b)).toEqual(textOf(core, b, HISTORY + 10));
  });

  it("should show a sample once its index sample arrives", async () => {
    const core = createCore();
    const [a] = core.data.map((c) => c.key);
    const client = connect(core);
    const log = open(core, client, { channels: [a] });
    await settle(log, HISTORY);
    const time = core.now().valueOf();
    const witness = client.feed.subscribe({
      keys: [a],
      limit: { variant: "latest", count: 1 },
    });
    closers.push(() => witness.close());
    await expect.poll(() => witness.value(a).status.variant, POLL).toBe("success");
    core.stream(
      new framer.Frame({ [a]: new Series({ data: [7], alignment: LEADING }) }),
    );
    core.stream(
      new framer.Frame({ [a]: new Series({ data: [8], alignment: LEADING + 1n }) }),
    );
    await expect.poll(() => witness.value(a).series.at(-1), POLL).toBe(8);
    expect(entriesOf(log)).toHaveLength(HISTORY);
    core.stream(
      new framer.Frame({ [core.index.key]: indexSeries(LEADING, [time, time + 1n]) }),
    );
    await settle(log, HISTORY + 2);
    const live = entriesOf(log).slice(HISTORY);
    expect(live.map((e) => e.value)).toEqual(["7", "8"]);
    expect(stampsOf(live)).toEqual([time, time + 1n]);
  });

  it("should time samples at alignments that none of their index samples have", async () => {
    const core = createCore({ dataMisaligned: true });
    const [a] = core.data.map((c) => c.key);
    const log = open(core, connect(core), { channels: [a] });
    await settle(log, HISTORY);
    core.advance(TimeSpan.seconds(1));
    await settle(log, HISTORY + 10);
    expect(stampsOf(entriesOf(log).slice(HISTORY))).toEqual(timesOf(core, 10));
  });

  it("should show samples streamed apart from their index samples", async () => {
    const core = createCore({ dataFramedApart: true });
    const [a] = core.data.map((c) => c.key);
    const log = open(core, connect(core), { channels: [a] });
    await settle(log, HISTORY);
    core.advance(TimeSpan.seconds(1));
    await settle(log, HISTORY + 10);
    expect(stampsOf(entriesOf(log).slice(HISTORY))).toEqual(timesOf(core, 10));
  });

  it("should stamp an index channel's entries with its own samples", async () => {
    const core = createCore();
    const log = open(core, connect(core), { channels: [core.index.key] });
    await settle(log, HISTORY);
    expect(stampsOf(entriesOf(log))).toEqual(timesOf(core, HISTORY));
  });

  it("should stamp the entries of a virtual channel with the time it receives them", async () => {
    const core = createCore({ derived: true });
    const { key } = core.virtual!;
    const log = open(core, connect(core), { channels: [key] });
    log.value();
    await expect.poll(() => core.streams(key), POLL).toBe(true);
    core.advance(TimeSpan.seconds(1));
    await settle(log, 10);
    expect(entriesOf(log).every((e) => e.timestamp.equals(core.now()))).toBe(true);
    expect(log.value().status.variant).not.toBe("error");
  });

  it("should drop the stored samples its index read does not reach, and warn once", async () => {
    const core = createCore();
    const [a] = core.data.map((c) => c.key);
    const statuses: cstatus.Crude[] = [];
    const client = connect(core, {
      readLatest: async (keys, count, tr) => {
        const frame = new framer.Frame();
        core
          .readLatest(keys, count, tr)
          .get(a)
          .series.forEach((s) => frame.push(a, s));
        const { key } = core.index;
        if (keys.includes(key))
          core
            .readLatest([key], HISTORY / 2, tr)
            .get(key)
            .series.forEach((s) => frame.push(key, s));
        return frame;
      },
    });
    const log = open(
      core,
      client,
      { channels: [a] },
      {
        onStatusChange: (s) => statuses.push(s),
      },
    );
    await settle(log, HISTORY / 2);
    expect(valuesOf(entriesOf(log), a)).toEqual(textOf(core, a, HISTORY / 2));
    expect(statuses).toEqual([
      expect.objectContaining({
        variant: "warning",
        message: "Dropped log samples with no time",
      }),
    ]);
  });

  it("should drop the oldest samples that wait past maxEntries", async () => {
    const core = createCore({ dataMisaligned: true, dataUntimed: true });
    const [a] = core.data.map((c) => c.key);
    const statuses: cstatus.Crude[] = [];
    const log = open(
      core,
      connect(core),
      { channels: [a] },
      { onStatusChange: (s) => statuses.push(s) },
      50,
    );
    await expect.poll(() => entriesOf(log).length, POLL).toBeGreaterThan(0);
    core.advance(TimeSpan.seconds(6));
    await expect.poll(() => statuses.length, POLL).toBe(1);
    expect(statuses[0]).toMatchObject({ variant: "warning" });
  });

  it("should notify once per pull that adds entries", async () => {
    const core = createCore();
    const [a] = core.data.map((c) => c.key);
    const log = open(core, connect(core), { channels: [a] });
    const handler = vi.fn();
    log.onChange(handler);
    await settle(log, HISTORY);
    const loaded = handler.mock.calls.length;
    core.advance(TimeSpan.seconds(1));
    await settle(log, HISTORY + 10);
    expect(handler).toHaveBeenCalledTimes(loaded + 1);
  });

  describe("history", () => {
    // Returns a client whose history reads wait for release, and whose first fails when
    // failed is true.
    const gate = (core: SyntheticCore, failed: boolean) => {
      let release: () => void = () => {};
      const released = new Promise<void>((resolve) => (release = resolve));
      let reads = 0;
      let open = false;
      const client = connect(core, {
        readLatest: async (keys, count, tr) => {
          reads++;
          if (failed && !open) throw new Error("history unavailable");
          await released;
          return core.readLatest(keys, count, tr);
        },
      });
      const releaseAll = (): void => {
        open = true;
        release();
      };
      return { client, release: releaseAll, reads: () => reads };
    };

    it("should show history before the samples streamed while it loads", async () => {
      const core = createCore();
      const [a] = core.data.map((c) => c.key);
      const { client, release, reads } = gate(core, false);
      const log = open(core, client, { channels: [a] });
      log.value();
      await expect.poll(reads, POLL).toBe(1);
      const witness = client.feed.subscribe({
        keys: [a],
        limit: { variant: "latest", count: 10 },
      });
      closers.push(() => witness.close());
      core.advance(TimeSpan.seconds(1));
      await expect.poll(() => witness.value(a).series.length, POLL).toBe(10);
      expect(entriesOf(log)).toEqual([]);
      release();
      await settle(log, HISTORY + 10);
      expect(valuesOf(entriesOf(log), a)).toEqual(textOf(core, a, HISTORY + 10));
      expect(stampsOf(entriesOf(log))).toEqual(timesOf(core, HISTORY + 10));
    });

    it("should show live samples while its history read fails, each once", async () => {
      const core = createCore();
      const [a] = core.data.map((c) => c.key);
      const { client, release } = gate(core, true);
      const log = open(core, client, { channels: [a] });
      await expect
        .poll(() => log.value().status, POLL)
        .toMatchObject({
          variant: "error",
          message: "Failed to read channel data",
          description: "history unavailable",
        });
      core.advance(TimeSpan.seconds(1));
      await settle(log, 10);
      expect(valuesOf(entriesOf(log), a)).toEqual(textOf(core, a, 10));
      release();
      await expect.poll(() => log.value().status.variant, POLL).toBe("success");
      core.advance(TimeSpan.seconds(1));
      await settle(log, 20);
      expect(valuesOf(entriesOf(log), a)).toEqual(textOf(core, a, 20));
    });
  });

  it("should time samples streamed while its index history loads", async () => {
    const core = createCore();
    const [a] = core.data.map((c) => c.key);
    const { promise: released, resolve: release } = Promise.withResolvers<void>();
    let calls = 0;
    const client = connect(core, {
      readLatest: async (keys, count, tr) => {
        // A failed read of both channels reads each alone, so the data lands first.
        if (calls++ === 0) throw new Error("read each channel alone");
        if (keys.includes(core.index.key)) await released;
        return core.readLatest(keys, count, tr);
      },
    });
    const log = open(core, client, { channels: [a] });
    log.value();
    await expect.poll(() => calls, POLL).toBe(3);
    await new Promise((resolve) => setTimeout(resolve, 50));
    const streamed = 1_000;
    core.advance(TimeSpan.seconds(streamed / 10));
    release();
    await settle(log, HISTORY + streamed);
    expect(valuesOf(entriesOf(log), a)).toEqual(textOf(core, a, HISTORY + streamed));
    expect(stampsOf(entriesOf(log))).toEqual(timesOf(core, HISTORY + streamed));
  });

  it("should show the samples written while its stream was down", async () => {
    const core = createCore();
    const [a] = core.data.map((c) => c.key);
    const log = open(core, connect(core), { channels: [a] });
    await settle(log, HISTORY);
    core.disconnect();
    await expect.poll(() => log.value().status.variant, POLL).toBe("warning");
    // The stream never delivers the samples written while it is down.
    core.advance(TimeSpan.seconds(1));
    core.reconnect();
    await settle(log, HISTORY + 10);
    expect(valuesOf(entriesOf(log), a)).toEqual(textOf(core, a, HISTORY + 10));
    expect(sorted(entriesOf(log))).toBe(true);
  });

  it("should show each sample once across a history change", async () => {
    const core = createCore();
    const [a] = core.data.map((c) => c.key);
    let reads = 0;
    const client = connect(core, {
      readLatest: async (keys, count, tr) => {
        reads++;
        return core.readLatest(keys, count, tr);
      },
    });
    const log = open(core, client, { channels: [a] });
    await settle(log, HISTORY);
    await expect.poll(() => core.streams(core.historyChange.key), POLL).toBe(true);
    const shown = textOf(core, a, HISTORY);
    core.rewrite(a, new TimeRange(START, core.now()), 10);
    await expect.poll(() => reads, POLL).toBe(2);
    await expect.poll(() => log.value().status.variant, POLL).toBe("success");
    core.advance(TimeSpan.seconds(1));
    await settle(log, HISTORY + 10);
    expect(valuesOf(entriesOf(log), a)).toEqual([...shown, ...textOf(core, a, 10)]);
  });

  it("should show every sample of a frame larger than its page", async () => {
    const core = createCore();
    const [a] = core.data.map((c) => c.key);
    const log = open(core, connect(core), { channels: [a] });
    await settle(log, HISTORY);
    const start = core.now().valueOf();
    const times = Array.from({ length: 2 * PAGE }, (_, i) => start + BigInt(i));
    const frame = new framer.Frame();
    [0, 1].forEach((i) => {
      const alignment = LEADING + BigInt(i * PAGE);
      frame.push(a, new Series({ data: new Float32Array(PAGE).fill(i), alignment }));
      frame.push(
        core.index.key,
        indexSeries(alignment, times.slice(i * PAGE, (i + 1) * PAGE)),
      );
    });
    core.stream(frame);
    await settle(log, HISTORY + 2 * PAGE);
    const live = valuesOf(entriesOf(log), a).slice(HISTORY);
    expect(live.filter((v) => v === "0")).toHaveLength(PAGE);
    expect(live.filter((v) => v === "1")).toHaveLength(PAGE);
  });

  describe("setChannels", () => {
    it("should keep the entries and samples of kept channels", async () => {
      const core = createCore();
      const [a, b] = core.data.map((c) => c.key);
      const log = open(core, connect(core), { channels: [a] });
      await settle(log, HISTORY);
      log.setChannels([a, b]);
      await settle(log, 2 * HISTORY);
      expect(valuesOf(entriesOf(log), a)).toEqual(textOf(core, a, HISTORY));
      expect(valuesOf(entriesOf(log), b)).toEqual(textOf(core, b, HISTORY));
      core.advance(TimeSpan.seconds(1));
      await settle(log, 2 * HISTORY + 20);
      expect(valuesOf(entriesOf(log), a)).toEqual(textOf(core, a, HISTORY + 10));
    });

    it("should do nothing for equal channels", async () => {
      const core = createCore();
      const [a] = core.data.map((c) => c.key);
      const log = open(core, connect(core), { channels: [a] });
      await settle(log, HISTORY);
      const handler = vi.fn();
      log.onChange(handler);
      log.setChannels([a]);
      expect(handler).not.toHaveBeenCalled();
    });

    it("should drop the entries of a removed channel", async () => {
      const core = createCore();
      const [a, b] = core.data.map((c) => c.key);
      const log = open(core, connect(core), { channels: [a, b] });
      await settle(log, 2 * HISTORY);
      log.setChannels([b]);
      expect(valuesOf(entriesOf(log), a)).toEqual([]);
      expect(valuesOf(entriesOf(log), b)).toEqual(textOf(core, b, HISTORY));
      core.advance(TimeSpan.seconds(1));
      await settle(log, HISTORY + 10);
      expect(valuesOf(entriesOf(log), a)).toEqual([]);
    });

    it("should show no entries once every channel is removed", async () => {
      const core = createCore();
      const client = connect(core);
      const [a] = core.data.map((c) => c.key);
      const log = open(core, client, { channels: [a] });
      const witness = open(core, client, { channels: [a] });
      await settle(log, HISTORY);
      await settle(witness, HISTORY);
      log.setChannels([]);
      expect(entriesOf(log)).toEqual([]);
      core.advance(TimeSpan.seconds(1));
      // The witness shares the stream, so the samples reached every subscription.
      await settle(witness, HISTORY + 10);
      expect(entriesOf(log)).toEqual([]);
    });
  });

  describe("newline handling", () => {
    const cases: Array<[string, string, string[]]> = [
      ["\\n", "a\nb", ["a", "b"]],
      ["\\r\\n without a stray \\r", "a\r\nb", ["a", "b"]],
      ["consecutive newlines as blank entries", "a\n\nb", ["a", "", "b"]],
      ["no newline into one entry", "a b", ["a b"]],
      ["an empty sample into one entry", "", [""]],
    ];

    it.each(cases)("should split %s", async (_, text, lines) => {
      const core = createCore({ text: () => text });
      const [a] = core.data.map((c) => c.key);
      core.retype(a, DataType.STRING);
      const log = open(core, connect(core), { channels: [a] });
      await settle(log, HISTORY * lines.length);
      const first = entriesOf(log).slice(0, lines.length);
      expect(first.map((e) => e.value)).toEqual(lines);
      expect(first.map((e) => e.continuation ?? false)).toEqual(
        lines.map((_, i) => i > 0),
      );
    });

    it("should show a sample with more lines than the argument limit", async () => {
      const lines = 200_000;
      const core = createCore({
        text: (i) => (i === 0 ? `${"x\n".repeat(lines - 1)}x` : "y"),
      });
      const [a] = core.data.map((c) => c.key);
      core.retype(a, DataType.STRING);
      const log = open(core, connect(core), { channels: [a] }, undefined, 2 * lines);
      await settle(log, lines + HISTORY - 1);
    });

    // History reads hand the log series of any data type, stamped by the index.
    const historyOf = (core: SyntheticCore, key: channel.Key, series: Series) =>
      connect(core, {
        readLatest: async () =>
          new framer.Frame({
            [key]: series,
            [core.index.key]: indexSeries(
              series.alignment,
              Array.from(
                { length: series.length },
                (_, i) => START.valueOf() + BigInt(i),
              ),
            ),
          }),
      });

    const typed: Array<[string, Series, string[]]> = [
      ["float64 samples exactly", new Series([0.1 + 0.2]), ["0.30000000000000004"]],
      ["int64 samples", new Series(new BigInt64Array([42n, -7n])), ["42", "-7"]],
      [
        "json samples without splitting them",
        new Series({ data: [{ note: "a\nb" }], dataType: DataType.JSON }),
        ['{"note":"a\\nb"}'],
      ],
    ];

    it.each(typed)("should show %s", async (_, series, values) => {
      const core = createCore();
      const [a] = core.data.map((c) => c.key);
      const log = open(core, historyOf(core, a, series), { channels: [a] });
      await settle(log, values.length);
      expect(entriesOf(log).map((e) => e.value)).toEqual(values);
      expect(entriesOf(log).every((e) => e.continuation !== true)).toBe(true);
    });
  });

  describe("eviction", () => {
    it("should keep entries of any age", async () => {
      const core = createCore();
      const [a] = core.data.map((c) => c.key);
      const log = open(core, connect(core), { channels: [a] });
      await settle(log, HISTORY);
      core.advance(TimeSpan.days(8), false);
      core.advance(TimeSpan.seconds(1));
      await settle(log, HISTORY + 10);
    });

    it("should evict past maxEntries, never leaving a continuation first", async () => {
      const core = createCore({ text: () => "a\nb\nc" });
      const [a] = core.data.map((c) => c.key);
      core.retype(a, DataType.STRING);
      const log = open(core, connect(core), { channels: [a] }, undefined, 5);
      await settle(log, 3);
      expect(entriesOf(log).map((e) => e.value)).toEqual(["a", "b", "c"]);
    });

    it("should evict a tenth of maxEntries beyond the excess", async () => {
      const core = createCore();
      const [a] = core.data.map((c) => c.key);
      const log = open(core, connect(core), { channels: [a] }, undefined, 100);
      await settle(log, 90);
      core.advance(TimeSpan.seconds(1));
      await settle(log, 100);
      core.advance(TimeSpan.seconds(1));
      await settle(log, 90);
      expect(valuesOf(entriesOf(log), a)).toEqual(textOf(core, a, 90));
    });
  });

  describe("older pages", () => {
    // More stored samples than a log reads when it opens.
    const STORED = PAGE + 2_500;
    const deep = (): SyntheticCore =>
      createCore({ now: START.add(TimeSpan.seconds(STORED / 10)) });

    // Asks for older entries until the log shows count entries.
    const page = async (log: StreamMultiChannelLog, count: number): Promise<void> =>
      await expect
        .poll(() => {
          log.loadOlder();
          return entriesOf(log).length;
        }, POLL)
        .toBe(count);

    it("should read older pages until it shows every stored sample", async () => {
      const core = deep();
      const [a] = core.data.map((c) => c.key);
      const log = open(core, connect(core), { channels: [a] });
      await settle(log, PAGE);
      log.loadOlder();
      // The page reads up to and including the oldest sample held.
      await settle(log, PAGE + 999);
      await page(log, STORED);
      const { reads } = core;
      log.loadOlder();
      await new Promise((resolve) => setTimeout(resolve, 50));
      expect(core.reads).toBe(reads);
      expect(valuesOf(entriesOf(log), a)).toEqual(textOf(core, a, STORED));
      expect(stampsOf(entriesOf(log))).toEqual(timesOf(core, STORED));
    });

    it("should page every channel back together", async () => {
      const core = deep();
      const [a, b] = core.data.map((c) => c.key);
      const log = open(core, connect(core), { channels: [a, b] });
      await settle(log, 2 * PAGE);
      await page(log, 2 * STORED);
      expect(sorted(entriesOf(log))).toBe(true);
      expect(valuesOf(entriesOf(log), b)).toEqual(textOf(core, b, STORED));
    });

    // Returns a client whose reads serve STORED samples of a at the times of time.
    const served = (core: SyntheticCore, time: (i: number) => bigint) => {
      const [a] = core.data.map((c) => c.key);
      return connect(core, {
        readLatest: async (_, count, tr) => {
          let hi = STORED;
          while (hi > 0 && time(hi - 1) >= tr.end.valueOf()) hi--;
          let lo = Math.max(0, hi - count);
          while (lo < hi && time(lo) < tr.start.valueOf()) lo++;
          const frame = new framer.Frame();
          if (hi <= lo) return frame;
          const alignment = (1n << 32n) + BigInt(lo);
          const indices = Array.from({ length: hi - lo }, (_, j) => lo + j);
          frame.push(a, new Series({ data: new Float32Array(indices), alignment }));
          frame.push(core.index.key, indexSeries(alignment, indices.map(time)));
          return frame;
        },
      });
    };

    const all = Array.from({ length: STORED }, (_, i) => String(i));

    it("should keep every sample of a time that a page splits", async () => {
      const core = createCore();
      const [a] = core.data.map((c) => c.key);
      // Seven samples share each time, so pages split runs of one time.
      const time = (i: number): bigint => START.valueOf() + BigInt(Math.floor(i / 7));
      const log = open(core, served(core, time), { channels: [a] });
      await settle(log, PAGE);
      await page(log, STORED);
      expect(valuesOf(entriesOf(log), a)).toEqual(all);
    });

    it("should read past more samples at the oldest time than a page holds", async () => {
      const core = createCore();
      const [a] = core.data.map((c) => c.key);
      // Samples 1,000 to 3,999 share one time, and the open read holds 1,500 of them.
      const time = (i: number): bigint =>
        START.valueOf() + BigInt(i < 1_000 ? i : i < 4_000 ? 1_000 : i - 3_000);
      const log = open(core, served(core, time), { channels: [a] });
      await settle(log, PAGE);
      await page(log, STORED);
      expect(valuesOf(entriesOf(log), a)).toEqual(all);
    });

    it("should drop a page that an eviction made stale", async () => {
      const core = deep();
      const [a] = core.data.map((c) => c.key);
      let held = false;
      const { promise: released, resolve: release } = Promise.withResolvers<void>();
      let waiting = false;
      const client = connect(core, {
        readLatest: async (keys, count, tr) => {
          if (held && tr.end.valueOf() !== TimeStamp.MAX.valueOf()) {
            waiting = true;
            await released;
          }
          return core.readLatest(keys, count, tr);
        },
      });
      const log = open(core, client, { channels: [a] }, undefined, PAGE + 500);
      await settle(log, PAGE);
      held = true;
      log.loadOlder();
      await expect.poll(() => waiting, POLL).toBe(true);
      core.advance(TimeSpan.seconds(60));
      await expect.poll(() => entriesOf(log).length, POLL).toBeLessThan(PAGE);
      release();
      held = false;
      await expect
        .poll(() => {
          log.loadOlder();
          return entriesOf(log).length;
        }, POLL)
        .toBe(PAGE + 500);
      expect(valuesOf(entriesOf(log), a)).toEqual(textOf(core, a, PAGE + 500));
    });

    it("should stop reading older pages once a page passes maxEntries", async () => {
      const core = deep();
      const [a] = core.data.map((c) => c.key);
      const log = open(core, connect(core), { channels: [a] }, undefined, PAGE + 1_500);
      await settle(log, PAGE);
      await page(log, PAGE + 1_500);
      const { reads } = core;
      log.loadOlder();
      await new Promise((resolve) => setTimeout(resolve, 50));
      expect(core.reads).toBe(reads);
      expect(valuesOf(entriesOf(log), a)).toEqual(textOf(core, a, PAGE + 1_500));
    });

    it("should read the samples before the entries it evicted", async () => {
      const core = deep();
      const [a] = core.data.map((c) => c.key);
      const log = open(core, connect(core), { channels: [a] }, undefined, 100);
      await settle(log, 90);
      await page(log, 100);
      expect(valuesOf(entriesOf(log), a)).toEqual(textOf(core, a, 100));
    });

    it("should post a failed page read once, then read the page again", async () => {
      const core = deep();
      const [a] = core.data.map((c) => c.key);
      let failed = true;
      const client = connect(core, {
        readLatest: async (keys, count, tr) => {
          if (failed && tr.end.valueOf() !== TimeStamp.MAX.valueOf())
            throw new Error("page unavailable");
          return core.readLatest(keys, count, tr);
        },
      });
      const statuses: cstatus.Crude[] = [];
      const log = open(
        core,
        client,
        { channels: [a] },
        {
          onStatusChange: (s) => statuses.push(s),
        },
      );
      await settle(log, PAGE);
      log.loadOlder();
      await expect.poll(() => statuses.length, POLL).toBe(1);
      expect(statuses[0]).toMatchObject({
        variant: "error",
        description: "page unavailable",
      });
      failed = false;
      await page(log, STORED);
      expect(statuses).toHaveLength(1);
    });

    it("should wait for a page while its stream is down", async () => {
      const core = deep();
      const [a] = core.data.map((c) => c.key);
      const retries: Error[] = [];
      const client = connect(core, { onRetry: (e) => retries.push(e) });
      const log = open(core, client, { channels: [a] });
      await settle(log, PAGE);
      core.disconnect();
      await expect.poll(() => log.value().status.variant, POLL).toBe("warning");
      log.loadOlder();
      await expect.poll(() => retries.length, POLL).toBeGreaterThan(0);
      core.reconnect();
      await page(log, STORED);
    });

    it("should read no older page of a virtual channel", async () => {
      const core = createCore({ derived: true });
      const { key } = core.virtual!;
      const log = open(core, connect(core), { channels: [key] });
      log.value();
      await expect.poll(() => core.streams(key), POLL).toBe(true);
      core.advance(TimeSpan.seconds(1));
      await settle(log, 10);
      const { reads } = core;
      log.loadOlder();
      await new Promise((resolve) => setTimeout(resolve, 50));
      expect(core.reads).toBe(reads);
    });
  });

  it("should clear its entries and stop on cleanup", async () => {
    const core = createCore();
    const client = connect(core);
    const [a] = core.data.map((c) => c.key);
    const log = open(core, client, { channels: [a] });
    const witness = open(core, client, { channels: [a] });
    await settle(log, HISTORY);
    await settle(witness, HISTORY);
    const handler = vi.fn();
    log.onChange(handler);
    log.cleanup();
    core.advance(TimeSpan.seconds(1));
    // The witness shares the stream, so the samples reached every subscription.
    await settle(witness, HISTORY + 10);
    expect(handler).not.toHaveBeenCalled();
  });

  it("should post the error of a channel it cannot retrieve", async () => {
    const core = createCore();
    const statuses: cstatus.Crude[] = [];
    const log = open(
      core,
      connect(core),
      { channels: [999] },
      { onStatusChange: (s) => statuses.push(s) },
    );
    log.value();
    await expect.poll(() => statuses.length, POLL).toBe(1);
    expect(statuses[0]).toMatchObject({
      variant: "error",
      message: "Failed to read channel data",
      description: "channel 999 not found",
    });
  });

  it("should report the worst status of its channels", async () => {
    const core = createCore();
    const [a, b] = core.data.map((c) => c.key);
    const log = open(core, connect(core), { channels: [a, 999, b] });
    expect(log.value().status.variant).toBe("loading");
    await expect
      .poll(() => log.value().status, POLL)
      .toMatchObject({
        variant: "error",
        message: "Failed to read channel data",
        description: "channel 999 not found",
      });
    await settle(log, 2 * HISTORY);
    log.setChannels([a, b]);
    await expect.poll(() => log.value().status.variant, POLL).toBe("success");
  });

  it("should report a disconnected status once instead of throwing", () => {
    const statuses: cstatus.Crude[] = [];
    const log = open(
      createCore(),
      null,
      { channels: [1] },
      { onStatusChange: (s) => statuses.push(s) },
    );
    expect(entriesOf(log)).toEqual([]);
    expect(entriesOf(log)).toEqual([]);
    expect(statuses).toEqual([telem.DISCONNECTED_STATUS]);
  });
});
