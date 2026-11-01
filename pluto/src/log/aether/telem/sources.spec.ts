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
const NOW = START.add(TimeSpan.minutes(10));
const SPAN = TimeSpan.seconds(30);
// The samples a log shows from the span before NOW.
const HISTORY = 300;
// The newest stored samples of each channel a log shows when it opens.
const PAGE = 10_000;
const POLL = { timeout: 5000, interval: 5 };
const OPTIONS: Partial<framer.FeedProps> = {
  batchDebounce: TimeSpan.milliseconds(5),
  removalDelay: TimeSpan.ZERO,
  breaker: { baseInterval: TimeSpan.milliseconds(10) },
};

const createCore = (props: Partial<SyntheticCoreProps> = {}): SyntheticCore =>
  new SyntheticCore({ rate: RATE, start: START, now: NOW, channels: 2, ...props });

// The text of the newest count samples of a channel, as the Core reads them.
const textOf = (core: SyntheticCore, key: channel.Key, count: number): string[] =>
  core
    .readLatest([key], count)
    .get(key)
    .series.flatMap((s) =>
      Array.from({ length: s.length }, (_, i) => s.asString(i, true)),
    );

const valuesOf = (entries: LogEntry[], key: channel.Key): string[] =>
  entries.filter((e) => e.channelKey === key).map((e) => e.value);

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

  // Opens a log that stamps entries on the Core's clock.
  const open = (
    core: SyntheticCore,
    client: telem.Client | null,
    props: Partial<StreamMultiChannelLogProps> = {},
    options?: telem.CreateOptions,
    maxEntries?: number,
  ): StreamMultiChannelLog => {
    const log = new StreamMultiChannelLog(
      client,
      { channels: [], timeSpan: SPAN, ...props },
      options,
      core.now,
      maxEntries,
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

  it("should show the stored samples of its span, then each live sample", async () => {
    const core = createCore();
    const [a] = core.data.map((c) => c.key);
    const log = open(core, connect(core), { channels: [a] });
    await settle(log, HISTORY);
    expect(valuesOf(entriesOf(log), a)).toEqual(textOf(core, a, HISTORY));
    // History lands in one pull, so it takes the time it loaded.
    expect(entriesOf(log).every((e) => e.timestamp.equals(NOW))).toBe(true);
    core.advance(TimeSpan.seconds(1));
    await settle(log, HISTORY + 10);
    const live = entriesOf(log).slice(HISTORY);
    expect(live.map((e) => e.value)).toEqual(textOf(core, a, 10));
    expect(live.every((e) => e.timestamp.equals(core.now()))).toBe(true);
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

  it("should notify when a channel loads no samples", async () => {
    const core = createCore({ start: NOW });
    const [a] = core.data.map((c) => c.key);
    const log = open(core, connect(core), { channels: [a] });
    const notified: cstatus.Variant[] = [];
    log.onChange(() => notified.push(log.value().status.variant));
    expect(log.value().status.variant).toBe("loading");
    await expect.poll(() => notified.at(-1), POLL).toBe("success");
    expect(entriesOf(log)).toEqual([]);
  });

  describe("history", () => {
    // Returns a client whose history reads wait for release, and whose first fails when
    // failed is true.
    const gate = (core: SyntheticCore, failed: boolean) => {
      let release: () => void = () => {};
      const released = new Promise<void>((resolve) => (release = resolve));
      let reads = 0;
      const client = connect(core, {
        readLatest: async (keys, count, tr) => {
          if (reads++ === 0 && failed) throw new Error("history unavailable");
          await released;
          return core.readLatest(keys, count, tr);
        },
      });
      return { client, release, reads: () => reads };
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

    it("should show each live sample once when history changes after a failed read", async () => {
      const core = createCore();
      const [a] = core.data.map((c) => c.key);
      const { client, release, reads } = gate(core, true);
      const log = open(core, client, { channels: [a] });
      await expect.poll(() => log.value().status.variant, POLL).toBe("error");
      await expect.poll(() => core.streams(core.historyChange.key), POLL).toBe(true);
      core.advance(TimeSpan.seconds(1));
      await settle(log, 10);
      const shown = textOf(core, a, 10);
      await expect.poll(reads, POLL).toBe(2);
      core.rewrite(a, new TimeRange(START, core.now()), 10);
      await expect.poll(reads, POLL).toBe(3);
      release();
      await expect.poll(() => log.value().status.variant, POLL).toBe("success");
      core.advance(TimeSpan.seconds(1));
      await settle(log, 20);
      expect(valuesOf(entriesOf(log), a)).toEqual([...shown, ...textOf(core, a, 10)]);
    });
  });

  it("should show the samples written while its stream was down", async () => {
    const core = createCore();
    const [a] = core.data.map((c) => c.key);
    let held = false;
    let waiting = false;
    const { promise: released, resolve: release } = Promise.withResolvers<void>();
    const client = connect(core, {
      readLatest: async (keys, count, tr) => {
        if (held) {
          waiting = true;
          await released;
        }
        return core.readLatest(keys, count, tr);
      },
    });
    const log = open(core, client, { channels: [a] });
    await settle(log, HISTORY);
    held = true;
    core.disconnect();
    await expect.poll(() => log.value().status.variant, POLL).toBe("warning");
    // The stream never delivers the samples written while it is down.
    core.advance(TimeSpan.seconds(1));
    core.reconnect();
    await expect.poll(() => waiting, POLL).toBe(true);
    const witness = client.feed.subscribe({
      keys: [a],
      limit: { variant: "latest", count: 10 },
    });
    closers.push(() => witness.close());
    core.advance(TimeSpan.seconds(1));
    await expect.poll(() => witness.value(a).series.length, POLL).toBe(10);
    release();
    await settle(log, HISTORY + 20);
    expect(valuesOf(entriesOf(log), a)).toEqual(textOf(core, a, HISTORY + 20));
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
    const [last] = core.readLatest([a], 1).get(a).series;
    const start = core.now();
    const series = [0, 1].map(
      (i) =>
        new Series({
          data: new Float32Array(PAGE).fill(i),
          timeRange: new TimeRange(
            start.add(TimeSpan.milliseconds(i)),
            start.add(TimeSpan.milliseconds(i + 1)),
          ),
          alignment: last.alignment + 1n + BigInt(i * PAGE),
        }),
    );
    core.stream(new framer.Frame([a, a], series));
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

    it("should report loading until an added channel loads", async () => {
      const core = createCore();
      const [a, b] = core.data.map((c) => c.key);
      const log = open(core, connect(core), { channels: [a] });
      await settle(log, HISTORY);
      log.setChannels([a, b]);
      expect(log.value().status.variant).toBe("loading");
      await expect.poll(() => log.value().status.variant, POLL).toBe("success");
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

    // History reads hand the log series of any data type.
    const historyOf = (core: SyntheticCore, key: channel.Key, series: Series) =>
      connect(core, {
        readLatest: async () => new framer.Frame({ [key]: series }),
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
    it("should evict entries older than keepFor", async () => {
      const core = createCore();
      const [a] = core.data.map((c) => c.key);
      const log = open(core, connect(core), {
        channels: [a],
        keepFor: TimeSpan.seconds(2),
      });
      await settle(log, HISTORY);
      core.advance(TimeSpan.seconds(3));
      await settle(log, 30);
      core.advance(TimeSpan.seconds(1));
      await settle(log, 40);
      expect(valuesOf(entriesOf(log), a)).toEqual(textOf(core, a, 40));
    });

    it("should notify when a pull evicts entries and adds none", async () => {
      const core = createCore();
      const [a] = core.data.map((c) => c.key);
      const log = open(core, connect(core), {
        channels: [a],
        keepFor: TimeSpan.seconds(2),
      });
      await settle(log, HISTORY);
      core.advance(TimeSpan.seconds(1));
      await settle(log, HISTORY + 10);
      const handler = vi.fn();
      log.onChange(handler);
      core.advance(TimeSpan.seconds(2), false);
      // A stream drop changes only the status, so the pull it causes adds no entries.
      core.disconnect();
      await settle(log, 10);
      expect(handler).toHaveBeenCalled();
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

  it("should post the error of a channel it cannot read", async () => {
    const core = createCore();
    const statuses: cstatus.Crude[] = [];
    const log = open(
      core,
      connect(core),
      { channels: [999] },
      {
        onStatusChange: (s) => statuses.push(s),
      },
    );
    log.value();
    await expect.poll(() => statuses.length, POLL).toBe(1);
    expect(statuses[0]).toMatchObject({
      variant: "error",
      message: "Failed to read channel data",
      description: "channel 999 not found",
    });
  });

  it("should report the worst status of its channels' snapshots", async () => {
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
    log.setChannels([a, b]);
    await expect.poll(() => log.value().status.variant, POLL).toBe("success");
  });

  it("should report a disconnected status once instead of throwing", () => {
    const statuses: cstatus.Crude[] = [];
    const log = open(
      createCore(),
      null,
      { channels: [1] },
      {
        onStatusChange: (s) => statuses.push(s),
      },
    );
    expect(entriesOf(log)).toEqual([]);
    expect(entriesOf(log)).toEqual([]);
    expect(statuses).toEqual([telem.DISCONNECTED_STATUS]);
  });
});
