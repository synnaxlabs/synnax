// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, type framer, status as cstatus } from "@synnaxlabs/client";
import {
  compare,
  DataType,
  type destructor,
  type MultiSeries,
  type Series,
  TimeRange,
  TimeStamp,
} from "@synnaxlabs/x";
import { z } from "zod";

import {
  type LogEntry,
  type LogSource,
  type LogSourceSpec,
} from "@/log/aether/telem/types";
import { type status } from "@/status/aether";
import { type CreateOptions } from "@/telem/aether/factory";
import { type Client, DISCONNECTED_STATUS } from "@/telem/aether/remote";
import { AbstractSource } from "@/telem/aether/telem";

const MAX_ENTRIES = 100_000;

// A sparse channel on a dense index needs many index samples to time its own. The index
// read stops at this multiple of the data read, and the untimed samples are left out.
const MAX_INDEX_READ_FACTOR = 4;

// A data allocation and its index allocation can arrive in different stream callbacks,
// so a few recent index allocations stay in reach.
const MAX_INDEX_BUFFERS = 4;

const streamMultiChannelLogPropsZ = z.object({
  channels: z.array(channel.keyZ.or(z.string())),
});

export type StreamMultiChannelLogProps = z.input<typeof streamMultiChannelLogPropsZ>;

interface ChannelMeta {
  key: channel.Key;
  dataType: DataType;
  // The channel holding the sample times, or 0 when there is none. Entries of a channel
  // without an index carry the time the log received them.
  index: channel.Key;
  // The streamed buffers not yet read in full, oldest first. The last one can still
  // grow. readCursor is the position in the first.
  pending: Series[];
  readCursor: number;
  skipInitialBatch: boolean;
  // Whether the log holds the channel's stored samples.
  historyRead: boolean;
  indexBuffers: Series[];
  // The latest sample time in the log. A streamed sample at or before it is a
  // duplicate. Sample time is the one identity the stream and stored reads share.
  highWater: bigint | null;
  // The first sample time the stream delivered. A stored sample at or after it is a
  // duplicate.
  streamStart: bigint | null;
  // Whether the log has reported dropping samples whose index samples never arrived.
  untimedDropReported: boolean;
}

interface PushResult {
  pushed: number;
  cursor: number;
}

/**
 * Shows the samples of several channels in one list, sorted by sample time. It reads the
 * latest stored samples, then streams new ones. The list keeps the newest maxEntries
 * entries.
 */
export class StreamMultiChannelLog
  extends AbstractSource<typeof streamMultiChannelLogPropsZ>
  implements LogSource
{
  static readonly TYPE = "stream-multi-channel-log";
  schema = streamMultiChannelLogPropsZ;

  private readonly client: Client | null;
  private readonly onStatusChange?: status.Adder;
  private readonly now: () => TimeStamp;
  private readonly maxEntries: number;
  private channelMeta: Map<channel.Key, ChannelMeta> = new Map();
  private entries: LogEntry[] = [];
  private stopStreaming?: destructor.Destructor;
  private valid = false;
  private _channels: channel.Key[] = [];
  private readGeneration = 0;

  constructor(
    client: Client | null,
    props: unknown,
    options?: CreateOptions,
    now: () => TimeStamp = () => client?.now() ?? TimeStamp.now(),
    maxEntries: number = MAX_ENTRIES,
  ) {
    super(props);
    this.client = client;
    this.onStatusChange = options?.onStatusChange;
    this.now = now;
    this.maxEntries = maxEntries;
    this._channels = this.props.channels.filter(
      (ch): ch is number => typeof ch === "number",
    );
  }

  value(): LogEntry[] {
    if (this._channels.length === 0) return this.entries;
    if (!this.valid) void this.read();
    return this.entries;
  }

  setChannels(channels: channel.Key[]): void {
    if (compare.primitiveArrays(this._channels, channels) === compare.EQUAL) return;
    this._channels = channels;
    this.valid = false;
    // A read for the old channel list must not land after this change.
    this.readGeneration++;
    if (channels.length === 0) {
      this.stopStreaming?.();
      this.stopStreaming = undefined;
      this.entries = [];
      this.channelMeta.clear();
    }
    this.notify();
  }

  /** Never rejects: a failure invalidates the read and reaches onStatusChange. */
  private async read(): Promise<void> {
    this.valid = true;
    const { client } = this;
    if (client == null) {
      this.onStatusChange?.(DISCONNECTED_STATUS);
      return;
    }
    // A newer read() or cleanup() bumps the generation; older reads bail after
    // awaiting.
    const generation = ++this.readGeneration;
    this.stopStreaming?.();
    this.stopStreaming = undefined;
    let unread: ChannelMeta[];
    try {
      const channels = await Promise.all(
        this._channels.map(async (ch) => await client.channels.retrieve(ch)),
      );
      if (generation !== this.readGeneration) return;
      unread = this.updateChannelMeta(channels);
      this.stream(client);
    } catch (e) {
      if (generation !== this.readGeneration) return;
      this.valid = false;
      this.onStatusChange?.(cstatus.fromException(e, "Failed to stream log channels"));
      return;
    }
    // The stream opens first, so a slow read never delays live entries.
    try {
      const stored = await Promise.all(
        unread
          .filter(({ index }) => index !== 0)
          .map(async (meta) => ({
            meta,
            ...(await readLatest(client, meta, this.maxEntries)),
          })),
      );
      if (generation !== this.readGeneration) return;
      const entries = stored.flatMap(({ meta, data, index }) => {
        meta.historyRead = true;
        return this.storedEntries(meta, data, index);
      });
      // One stable sort instead of an insert per entry.
      if (entries.length > 0) this.entries = [...this.entries, ...entries].sort(byTime);
      this.gcEntries();
    } catch (e) {
      if (generation !== this.readGeneration) return;
      this.onStatusChange?.(cstatus.fromException(e, "Failed to read log channels"));
    }
    this.notify();
  }

  // Keeps the state of the channels the log already shows, and returns the channels
  // whose stored samples it has not read.
  private updateChannelMeta(channels: channel.Channel[]): ChannelMeta[] {
    const prev = this.channelMeta;
    const newKeys = new Set(channels.map((ch) => ch.key));
    const removedKeys = new Set([...prev.keys()].filter((k) => !newKeys.has(k)));
    if (removedKeys.size > 0)
      this.entries = this.entries.filter((e) => !removedKeys.has(e.channelKey));
    // A restarted stream replays the buffered data of every channel. A channel without
    // an index cannot detect the replay by sample time, so it skips the first batch.
    const isRestart = prev.size > 0;
    this.channelMeta = new Map();
    const unread: ChannelMeta[] = [];
    for (const ch of channels) {
      const existing = prev.get(ch.key);
      const index = channelIndex(ch);
      const meta: ChannelMeta = {
        key: ch.key,
        dataType: new DataType(ch.dataType),
        index,
        pending: [],
        readCursor: 0,
        skipInitialBatch: isRestart && index === 0,
        historyRead: existing?.historyRead ?? false,
        indexBuffers: [],
        highWater: existing?.highWater ?? null,
        streamStart: existing?.streamStart ?? null,
        untimedDropReported: existing?.untimedDropReported ?? false,
      };
      this.channelMeta.set(ch.key, meta);
      if (!meta.historyRead) unread.push(meta);
    }
    return unread;
  }

  private stream(client: Client): void {
    const keys = new Set<channel.Key>();
    for (const { key, index } of this.channelMeta.values()) {
      keys.add(key);
      if (index !== 0) keys.add(index);
    }
    const handler: framer.StreamHandler = (res) => {
      let pushed = 0;
      for (const meta of this.channelMeta.values()) {
        const index = meta.index === 0 ? undefined : res.get(meta.index);
        if (index != null) meta.indexBuffers.push(...index.series);
        const allocated = res.get(meta.key);
        if (allocated != null && allocated.series.length > 0)
          // First callback after channel change: skip buffered data.
          if (meta.skipInitialBatch) {
            const last = allocated.series[allocated.series.length - 1];
            [meta.pending, meta.readCursor] = [[last], last.length];
            meta.skipInitialBatch = false;
          } else meta.pending.push(...allocated.series);
        pushed += this.drain(meta);
        this.trimPending(meta);
        // After the drain, so every index buffer of this callback times its data.
        meta.indexBuffers.splice(0, meta.indexBuffers.length - MAX_INDEX_BUFFERS);
      }
      this.gcEntries();
      if (pushed > 0) this.notify();
    };
    // The exact feed, since the feed rounds sample times to float32.
    this.stopStreaming = client.exactFeed.stream(handler, [...keys]).close;
  }

  // Pushes the pending samples in order, up to the first one whose index sample has not
  // arrived. Returns the number of entries added.
  private drain(meta: ChannelMeta): number {
    let pushed = 0;
    while (meta.pending.length > 0) {
      const [buf] = meta.pending;
      const result = this.pushStreamed(meta, buf, meta.readCursor);
      pushed += result.pushed;
      meta.readCursor = result.cursor;
      if (result.cursor < buf.length || meta.pending.length === 1) break;
      meta.pending.shift();
      meta.readCursor = 0;
    }
    return pushed;
  }

  // Drops the oldest held samples past the entry limit, so a channel whose index samples
  // never arrive cannot grow the log's memory without bound.
  private trimPending(meta: ChannelMeta): void {
    let held = meta.pending.reduce((n, s) => n + s.length, 0) - meta.readCursor;
    if (held <= this.maxEntries) return;
    while (held > this.maxEntries) {
      const [buf] = meta.pending;
      const excess = held - this.maxEntries;
      const remaining = buf.length - meta.readCursor;
      if (remaining > excess || meta.pending.length === 1) {
        meta.readCursor += Math.min(excess, remaining);
        break;
      }
      meta.pending.shift();
      meta.readCursor = 0;
      held -= remaining;
    }
    if (meta.untimedDropReported) return;
    meta.untimedDropReported = true;
    this.onStatusChange?.({
      variant: "warning",
      message: "Dropped log samples with no time",
      description: `The index samples of channel ${meta.key} did not arrive.`,
    });
  }

  // Pushes the samples of buf from start on, skipping samples whose index sample is
  // gone. The returned cursor stops at the first sample whose index has not arrived.
  private pushStreamed(meta: ChannelMeta, buf: Series, start: number): PushResult {
    let pushed = 0;
    const receivedAt = this.now().valueOf();
    for (let i = start; i < buf.length; i++) {
      if (meta.index === 0) {
        pushed += this.insert(meta, buf, i, receivedAt);
        continue;
      }
      const alignment = sampleAlignment(buf, i);
      const time = findTime(meta.indexBuffers, alignment);
      if (time == null) {
        // Index samples arrive in order, so a later one means this sample's is gone.
        if (meta.indexBuffers.some((s) => s.alignment > alignment)) continue;
        return { pushed, cursor: i };
      }
      if (meta.highWater != null && time <= meta.highWater) continue;
      meta.highWater = time;
      meta.streamStart ??= time;
      pushed += this.insert(meta, buf, i, time);
    }
    return { pushed, cursor: buf.length };
  }

  // Returns the entries of the stored samples the stream has not delivered.
  private storedEntries(
    meta: ChannelMeta,
    data: MultiSeries,
    index: MultiSeries,
  ): LogEntry[] {
    const entries: LogEntry[] = [];
    for (const s of data.series)
      for (let i = 0; i < s.length; i++) {
        const time = index.atAlignment(sampleAlignment(s, i)) as bigint | undefined;
        if (time == null) continue;
        if (meta.streamStart != null && time >= meta.streamStart) continue;
        entries.push(...toEntries(meta, s, i, time));
        if (meta.highWater == null || time > meta.highWater) meta.highWater = time;
      }
    return entries;
  }

  // Inserts sample i of buf at its place in time order. Returns the number of entries
  // added.
  private insert(meta: ChannelMeta, buf: Series, i: number, time: bigint): number {
    const group = toEntries(meta, buf, i, time);
    const last = this.entries.at(-1);
    if (last == null || last.timestamp.valueOf() <= time) this.entries.push(...group);
    else this.entries.splice(upperBound(this.entries, time), 0, ...group);
    return group.length;
  }

  // Evicts the oldest entries past maxEntries.
  private gcEntries(): void {
    if (this.entries.length <= this.maxEntries) return;
    let excess = this.entries.length - this.maxEntries;
    // Skip leading continuations so the new head isn't an orphan.
    while (excess < this.entries.length && this.entries[excess].continuation) excess++;
    this.entries.splice(0, excess);
  }

  cleanup(): void {
    this.readGeneration++;
    this.stopStreaming?.();
    this.stopStreaming = undefined;
    this.entries = [];
    this.channelMeta.clear();
    this.valid = false;
  }
}

const channelIndex = (ch: channel.Channel): channel.Key => {
  if (ch.isIndex) return ch.key;
  if (ch.virtual && !channel.isCalculated(ch)) return 0;
  return ch.index;
};

// Returns the entries of sample i of buf. A string sample splits into one entry per
// line.
const toEntries = (
  meta: ChannelMeta,
  buf: Series,
  i: number,
  time: bigint,
): LogEntry[] => {
  const timestamp = new TimeStamp(time);
  const value = buf.asString(i, true);
  if (!meta.dataType.equals(DataType.STRING) || !value.includes("\n"))
    return [{ channelKey: meta.key, timestamp, value }];
  return value.split(/\r?\n/).map((line, j) => ({
    channelKey: meta.key,
    timestamp,
    value: line,
    continuation: j > 0,
  }));
};

// Reads the latest n stored samples of the channel, and index samples to time them. The
// index can hold more samples than the channel, so its read doubles until it reaches
// the first data sample or MAX_INDEX_READ_FACTOR times n.
const readLatest = async (
  client: Client,
  meta: ChannelMeta,
  n: number,
): Promise<{ data: MultiSeries; index: MultiSeries }> => {
  const data = await client.readLatest(meta.key, n, TimeRange.MAX);
  if (meta.index === meta.key || data.series.length === 0) return { data, index: data };
  const first = data.series[0].alignment;
  for (let m = n; ; m *= 2) {
    const index = await client.readLatest(meta.index, m, TimeRange.MAX);
    if (
      index.length < m ||
      index.series[0].alignment <= first ||
      m >= n * MAX_INDEX_READ_FACTOR
    )
      return { data, index };
  }
};

const byTime = (a: LogEntry, b: LogEntry): number => {
  const [ta, tb] = [a.timestamp.valueOf(), b.timestamp.valueOf()];
  return ta < tb ? -1 : ta > tb ? 1 : 0;
};

const sampleAlignment = (s: Series, i: number): bigint =>
  s.alignment + BigInt(i) * s.alignmentMultiple;

const findTime = (buffers: Series[], alignment: bigint): bigint | null => {
  for (const s of buffers) {
    const time = s.atAlignment(alignment) as bigint | undefined;
    if (time != null) return time;
  }
  return null;
};

// Returns the position after the last entry at or before time.
const upperBound = (entries: LogEntry[], time: bigint): number => {
  let [lo, hi] = [0, entries.length];
  while (lo < hi) {
    const mid = (lo + hi) >>> 1;
    if (entries[mid].timestamp.valueOf() <= time) lo = mid + 1;
    else hi = mid;
  }
  return lo;
};

export const streamMultiChannelLog = (
  props: StreamMultiChannelLogProps,
): LogSourceSpec => ({
  type: StreamMultiChannelLog.TYPE,
  props,
  variant: "source",
  valueType: "log",
});
