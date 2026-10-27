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
  TimeSpan,
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

// The samples of each channel that one stored read returns.
const PAGE_SIZE = 1_000;

const HISTORY_RETRY_DELAY = TimeSpan.seconds(5);

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
  // Whether the log has reported dropping samples whose index samples never arrived.
  untimedDropReported: boolean;
}

// How a stored read places its page: the first page of a channel set, the latest page
// of channels added to it, or a page past either end of the list.
type PageKind = "initial" | "added" | "older" | "newer";

interface PushResult {
  pushed: number;
  cursor: number;
}

/**
 * Shows the samples of several channels in one list, sorted by sample time. It reads the
 * latest stored samples, then streams new ones. The list holds at most maxEntries
 * entries, and loadOlder and loadNewer move it through the stored samples. A list that
 * moves off the newest entries is detached until loadNewer reaches them again.
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
  private readonly pageSize: number;
  private channelMeta: Map<channel.Key, ChannelMeta> = new Map();
  private entries: LogEntry[] = [];
  private stopStreaming?: destructor.Destructor;
  private valid = false;
  private _channels: channel.Key[] = [];
  private readGeneration = 0;
  private _detached = false;
  // The entries streamed while the list is detached, newest last.
  private tail: LogEntry[] = [];
  // The sample time from which the tail holds every sample.
  private tailStart: bigint | null = null;
  // The sample time from which the list holds every stored sample, or null before the
  // first read. 0n means the list reaches the first stored sample.
  private start: bigint | null = null;
  // The sample time before which a detached list holds every stored sample.
  private end = 0n;
  private pageCount = 0;
  private pageInFlight: { id: number; kind: PageKind } | null = null;
  private retryTimer?: ReturnType<typeof setTimeout>;

  get detached(): boolean {
    return this._detached;
  }

  constructor(
    client: Client | null,
    props: unknown,
    options?: CreateOptions,
    now: () => TimeStamp = () => client?.now() ?? TimeStamp.now(),
    maxEntries: number = MAX_ENTRIES,
    pageSize: number = PAGE_SIZE,
  ) {
    super(props);
    this.client = client;
    this.onStatusChange = options?.onStatusChange;
    this.now = now;
    this.maxEntries = maxEntries;
    this.pageSize = pageSize;
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
    clearTimeout(this.retryTimer);
    if (channels.length === 0) {
      this.stopStreaming?.();
      this.stopStreaming = undefined;
      this.reset();
    }
    this.notify();
  }

  /** Reads the page of stored entries before the oldest entry in the list. */
  loadOlder(): void {
    const { client, start } = this;
    if (client == null || start == null || start === 0n || this.pageInFlight != null)
      return;
    const tr = new TimeRange(TimeStamp.MIN, new TimeStamp(start));
    void this.readPage(client, "older", this.indexedMeta(), tr);
  }

  /** Reads the page of stored entries after the newest entry of a detached list. */
  loadNewer(): void {
    const { client } = this;
    if (client == null || !this._detached || this.pageInFlight != null) return;
    const tr = new TimeRange(new TimeStamp(this.end), TimeStamp.MAX);
    void this.readPage(client, "newer", this.indexedMeta(), tr);
  }

  /** Replaces a detached list with the newest entries. */
  resume(): void {
    this.cancelScrollPage();
    if (!this._detached) return;
    this.attach();
    this.notify();
  }

  private cancelScrollPage(): void {
    const kind = this.pageInFlight?.kind;
    if (kind === "older" || kind === "newer") this.pageInFlight = null;
  }

  private attach(): void {
    this.cancelScrollPage();
    this.entries = this.tail;
    this.tail = [];
    this.tailStart = null;
    this._detached = false;
    this.start = this.entries[0]?.timestamp.valueOf() ?? null;
  }

  private reset(): void {
    this.pageInFlight = null;
    this.entries = [];
    this.tail = [];
    this.tailStart = null;
    this._detached = false;
    this.start = null;
    this.channelMeta.clear();
  }

  private indexedMeta(): ChannelMeta[] {
    return [...this.channelMeta.values()].filter(({ index }) => index !== 0);
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
    clearTimeout(this.retryTimer);
    this.stopStreaming?.();
    this.stopStreaming = undefined;
    let unread: ChannelMeta[];
    try {
      const channels = await Promise.all(
        this._channels.map(async (ch) => await client.channels.retrieve(ch)),
      );
      if (generation !== this.readGeneration) return;
      unread = this.updateChannelMeta(channels);
      if (this._detached) this.attach();
      this.stream(client);
    } catch (e) {
      if (generation !== this.readGeneration) return;
      this.valid = false;
      this.onStatusChange?.(cstatus.fromException(e, "Failed to stream log channels"));
      return;
    }
    // The stream opens first, so a slow read never delays live entries.
    await this.readHistory(client, unread);
  }

  // Reads the latest page of the indexed channels in metas.
  private async readHistory(client: Client, metas: ChannelMeta[]): Promise<void> {
    const { start } = this;
    const indexed = metas.filter(({ index }) => index !== 0);
    if (start == null) await this.readPage(client, "initial", indexed, TimeRange.MAX);
    else {
      const tr = new TimeRange(new TimeStamp(start), TimeStamp.MAX);
      await this.readPage(client, "added", indexed, tr);
    }
  }

  /**
   * Reads a page of each channel within tr and merges it into the list. Never rejects:
   * a failure reaches onStatusChange. Channels whose initial or added page fails read
   * again after HISTORY_RETRY_DELAY. A failed older or newer page merges nothing.
   */
  private async readPage(
    client: Client,
    kind: PageKind,
    metas: ChannelMeta[],
    tr: TimeRange,
  ): Promise<void> {
    const generation = this.readGeneration;
    const id = ++this.pageCount;
    this.pageInFlight = { id, kind };
    const stale = (): boolean =>
      generation !== this.readGeneration || this.pageInFlight?.id !== id;
    const edge = kind === "newer" ? "first" : "last";
    try {
      const results = await Promise.allSettled(
        metas.map(
          async (meta) => await readEdge(client, meta, this.pageSize, tr, edge),
        ),
      );
      if (stale()) return;
      const failed = metas.filter((_, i) => results[i].status === "rejected");
      const rejection = results.find((r) => r.status === "rejected");
      const scroll = kind === "older" || kind === "newer";
      if (rejection != null) {
        this.onStatusChange?.(
          cstatus.fromException(rejection.reason, "Failed to read log channels"),
        );
        if (!scroll) this.scheduleRetry(client, generation, failed);
      }
      if (failed.length === metas.length || (scroll && failed.length > 0)) return;
      // A full page stops short of the samples beyond it. The page is complete only up
      // to the boundary that the shortest reach of a full page sets.
      let boundary: bigint | null = null;
      const incoming: LogEntry[] = [];
      results.forEach((res, i) => {
        if (res.status === "rejected") return;
        const meta = metas[i];
        const { data, index } = res.value;
        meta.historyRead = true;
        const entries = this.storedEntries(meta, data, index);
        incoming.push(...entries);
        if (data.length < this.pageSize || entries.length === 0) return;
        const reach = (edge === "last" ? entries[0] : entries.at(-1))?.timestamp;
        const t = reach?.valueOf();
        if (t == null) return;
        if (boundary == null || (edge === "last" ? t > boundary : t < boundary))
          boundary = t;
      });
      // One stable sort instead of an insert per entry.
      incoming.sort(byTime);
      if (kind === "newer") this.mergeNewer(incoming, boundary);
      else this.mergeOlder(kind, incoming, boundary);
    } finally {
      const current = !stale();
      if (this.pageInFlight?.id === id) this.pageInFlight = null;
      if (current) this.notify();
    }
  }

  private scheduleRetry(
    client: Client,
    generation: number,
    metas: ChannelMeta[],
  ): void {
    const retry = (): void => {
      if (generation !== this.readGeneration) return;
      // An added page assumes the list holds the newest entries.
      if (this._detached)
        this.retryTimer = setTimeout(retry, HISTORY_RETRY_DELAY.milliseconds);
      else void this.readHistory(client, metas);
    };
    this.retryTimer = setTimeout(retry, HISTORY_RETRY_DELAY.milliseconds);
  }

  private mergeOlder(
    kind: PageKind,
    incoming: LogEntry[],
    boundary: bigint | null,
  ): void {
    this.entries = merge(this.entries, incoming);
    if (boundary != null) this.entries.splice(0, lowerBound(this.entries, boundary));
    // An added channel can only narrow the times the list holds in full.
    if (boundary != null)
      this.start = kind === "added" ? max(this.start, boundary) : boundary;
    else if (kind !== "added") this.start = 0n;
    // Only a page the user scrolled to gives up the newest entries for older ones.
    if (kind !== "older") {
      this.gcEntries();
      return;
    }
    if (this.entries.length <= this.maxEntries) return;
    let cut = this.maxEntries;
    // Keeps a multi-line sample whole.
    while (cut > 0 && this.entries[cut].continuation) cut--;
    const evicted = this.entries.splice(cut);
    const last = this.entries.at(-1)?.timestamp.valueOf() ?? 0n;
    // A cut between entries of one sample time leaves that time incomplete.
    this.end = evicted[0].timestamp.valueOf() === last ? last : last + 1n;
    // The tail must hold every sample from tailStart on, so only a list that held the
    // newest entries hands its evicted ones over.
    if (!this._detached) {
      this.tail = evicted;
      this.tailStart = evicted[0].timestamp.valueOf();
    }
    this._detached = true;
  }

  private mergeNewer(incoming: LogEntry[], boundary: bigint | null): void {
    const { tailStart } = this;
    const reached = boundary == null || (tailStart != null && boundary >= tailStart);
    const kept = incoming.filter((e) => {
      const t = e.timestamp.valueOf();
      if (boundary != null && t > boundary) return false;
      return !reached || tailStart == null || t < tailStart;
    });
    this.entries = merge(this.entries, kept);
    if (boundary != null) this.end = boundary + 1n;
    if (reached) {
      this.entries = merge(this.entries, this.tail);
      this.tail = [];
      this.tailStart = null;
      this._detached = false;
    }
    this.gcEntries();
  }

  // Keeps the state of the channels the log already shows, and returns the channels
  // whose stored samples it has not read.
  private updateChannelMeta(channels: channel.Channel[]): ChannelMeta[] {
    const prev = this.channelMeta;
    const newKeys = new Set(channels.map((ch) => ch.key));
    const removedKeys = new Set([...prev.keys()].filter((k) => !newKeys.has(k)));
    if (removedKeys.size > 0) {
      this.entries = this.entries.filter((e) => !removedKeys.has(e.channelKey));
      this.tail = this.tail.filter((e) => !removedKeys.has(e.channelKey));
    }
    // A restarted stream replays the buffered data of every channel. A channel without
    // an index cannot detect the replay by sample time, so it skips the first batch.
    const isRestart = prev.size > 0;
    this.channelMeta = new Map();
    const unread: ChannelMeta[] = [];
    for (const ch of channels) {
      const existing = prev.get(ch.key);
      const index = channelIndex(ch);
      // The restarted stream replays only the latest buffer, so samples still waiting
      // for their index samples stay pending.
      const kept = existing?.index === index ? existing : undefined;
      const meta: ChannelMeta = {
        key: ch.key,
        dataType: new DataType(ch.dataType),
        index,
        pending: kept?.pending ?? [],
        readCursor: kept?.readCursor ?? 0,
        skipInitialBatch: isRestart && index === 0,
        historyRead: existing?.historyRead ?? false,
        indexBuffers: kept?.indexBuffers ?? [],
        highWater: existing?.highWater ?? null,
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
      if (this._detached) this.trimTail();
      else this.gcEntries();
      if (pushed > 0 && !this._detached) this.notify();
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
      pushed += this.insert(meta, buf, i, time);
    }
    return { pushed, cursor: buf.length };
  }

  // Returns the entries of the stored samples that index samples time.
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
        entries.push(...toEntries(meta, s, i, time));
        if (meta.highWater == null || time > meta.highWater) meta.highWater = time;
      }
    return entries;
  }

  // Inserts sample i of buf at its place in time order. Returns the number of entries
  // added.
  private insert(meta: ChannelMeta, buf: Series, i: number, time: bigint): number {
    const group = toEntries(meta, buf, i, time);
    const list = this._detached ? this.tail : this.entries;
    const last = list.at(-1);
    if (last == null || last.timestamp.valueOf() <= time) list.push(...group);
    else list.splice(upperBound(list, time), 0, ...group);
    return group.length;
  }

  // Evicts the oldest entries past maxEntries.
  private gcEntries(): void {
    const evicted = evictOldest(this.entries, this.maxEntries);
    const first = this.entries[0]?.timestamp.valueOf();
    if (evicted == null || first == null) return;
    // An eviction between entries of one sample time leaves that time incomplete.
    this.start = max(
      this.start,
      evicted.timestamp.valueOf() === first ? first + 1n : first,
    );
  }

  private trimTail(): void {
    const evicted = evictOldest(this.tail, this.maxEntries);
    const first = this.tail[0]?.timestamp.valueOf();
    if (evicted == null || first == null) return;
    // An eviction between entries of one sample time leaves that time incomplete.
    this.tailStart = max(
      this.tailStart,
      evicted.timestamp.valueOf() === first ? first + 1n : first,
    );
  }

  cleanup(): void {
    this.readGeneration++;
    clearTimeout(this.retryTimer);
    this.stopStreaming?.();
    this.stopStreaming = undefined;
    this.reset();
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

// Reads n stored samples of the channel from the first or last sample within tr, and
// index samples to time them. The index read stays within the time range of the data.
// The index can hold more samples than the channel in that range, so its read doubles
// until it reaches the far end of the data.
const readEdge = async (
  client: Client,
  meta: ChannelMeta,
  n: number,
  tr: TimeRange,
  edge: "first" | "last",
): Promise<{ data: MultiSeries; index: MultiSeries }> => {
  const read = edge === "first" ? client.readEarliest : client.readLatest;
  const data = await read(meta.key, n, tr);
  if (meta.index === meta.key || data.series.length === 0) return { data, index: data };
  const covers = (index: MultiSeries): boolean => {
    if (edge === "last") return index.series[0].alignment <= data.series[0].alignment;
    const [i, d] = [index.series.at(-1), data.series.at(-1)];
    if (i == null || d == null) return false;
    return sampleAlignment(i, i.length - 1) >= sampleAlignment(d, d.length - 1);
  };
  for (let m = n; ; m *= 2) {
    const index = await read(meta.index, m, data.timeRange);
    if (index.length < m || covers(index)) return { data, index };
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

// Merges sorted incoming entries into the sorted list, leaving out samples the list
// already holds.
const merge = (list: LogEntry[], incoming: LogEntry[]): LogEntry[] => {
  if (incoming.length === 0) return list;
  const from = lowerBound(list, incoming[0].timestamp.valueOf());
  const to = upperBound(list, incoming[incoming.length - 1].timestamp.valueOf());
  const held = new Set(list.slice(from, to).map(sampleKey));
  const fresh = incoming.filter((e) => !held.has(sampleKey(e)));
  if (fresh.length === 0) return list;
  const [first, last] = [list[0], list.at(-1)];
  if (
    first == null ||
    fresh[fresh.length - 1].timestamp.valueOf() < first.timestamp.valueOf()
  )
    return [...fresh, ...list];
  if (last != null && fresh[0].timestamp.valueOf() >= last.timestamp.valueOf())
    return [...list, ...fresh];
  return [...list, ...fresh].sort(byTime);
};

// The lines of a multi-line sample share its key.
const sampleKey = (e: LogEntry): string => `${e.channelKey}:${e.timestamp.valueOf()}`;

// Evicts the oldest entries past max, keeping multi-line samples whole. Returns the
// newest evicted entry, or undefined when none was evicted.
const evictOldest = (list: LogEntry[], max: number): LogEntry | undefined => {
  if (list.length <= max) return undefined;
  let excess = list.length - max;
  while (excess < list.length && list[excess].continuation) excess++;
  return list.splice(0, excess).at(-1);
};

const max = (a: bigint | null, b: bigint): bigint => (a == null || b > a ? b : a);

// Returns the position of the first entry at or after time.
const lowerBound = (entries: LogEntry[], time: bigint): number => {
  let [lo, hi] = [0, entries.length];
  while (lo < hi) {
    const mid = (lo + hi) >>> 1;
    if (entries[mid].timestamp.valueOf() < time) lo = mid + 1;
    else hi = mid;
  }
  return lo;
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
