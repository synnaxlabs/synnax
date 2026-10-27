// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  channel,
  framer,
  status as cstatus,
  UnexpectedError,
} from "@synnaxlabs/client";
import {
  type breaker,
  compare,
  DataType,
  type MultiSeries,
  type Series,
  sync,
  TimeStamp,
} from "@synnaxlabs/x";
import { z } from "zod";

import {
  type LogEntry,
  type LogSource,
  type LogSourceSpec,
} from "@/log/aether/telem/types";
import { type status } from "@/status/aether";
import { telem } from "@/telem/aether";

const MAX_ENTRIES = 100_000;
// Past maxEntries, a log evicts this share of maxEntries beyond the excess, so a full
// log moves its entries once per share instead of on every pull.
const EVICTION_HEADROOM = 0.1;
// The newest stored samples of each channel a log shows when it opens.
const PAGE_SIZE = 10_000;

const streamMultiChannelLogPropsZ = z.object({
  channels: z.array(channel.keyZ.or(z.string())),
});

export type StreamMultiChannelLogProps = z.input<typeof streamMultiChannelLogPropsZ>;

// A sample with no entries yet. time is set once known.
interface Sample {
  alignment: bigint;
  // The start of the sample's series, or null when the series has no time range, and
  // the sample's position in it.
  start: bigint | null;
  offset: number;
  value: string;
  text: boolean;
  time?: TimeStamp;
}

// A channel whose retrieval failed. A definitive failure is not retried.
interface Failure {
  status: cstatus.Status;
  definitive: boolean;
}

interface Meta {
  // The channel that holds the sample times: the channel itself for an index, or null
  // for a channel with no index, whose entries take the time the log receives them.
  index: channel.Key | null;
  // The snapshot and the count of appended samples that entries were last read up to.
  snapshot?: framer.Snapshot;
  appended: number;
  // Samples that wait for their index samples, oldest first.
  pending: Sample[];
  // Whether the log has warned that it dropped samples that never got a time.
  dropWarned: boolean;
}

/**
 * Shows the samples of several channels in one list, sorted by sample time. A log reads
 * the newest stored samples of each channel when it opens, then streams. The list keeps
 * the newest maxEntries entries.
 */
export class StreamMultiChannelLog
  extends telem.AbstractSource<typeof streamMultiChannelLogPropsZ>
  implements LogSource
{
  static readonly TYPE = "stream-multi-channel-log";
  schema = streamMultiChannelLogPropsZ;

  private readonly client: telem.Client | null;
  private readonly onStatusChange?: status.Adder;
  private readonly now: () => TimeStamp;
  private readonly maxEntries: number;
  private readonly failures: telem.FailureReporter;
  private readonly retryNotifier = new sync.Notifier();
  private readonly breaker: breaker.Breaker;
  private subscription: framer.Subscription | null = null;
  private readonly metas = new Map<channel.Key, Meta>();
  private readonly unresolved = new Map<channel.Key, Failure>();
  private entries: LogEntry[] = [];
  private status = telem.SUCCESS_STATUS;
  private valid = false;
  private generation = 0;
  private _channels: channel.Key[] = [];

  constructor(
    client: telem.Client | null,
    props: unknown,
    options?: telem.CreateOptions,
    now: () => TimeStamp = client?.now ?? (() => TimeStamp.now()),
    maxEntries: number = MAX_ENTRIES,
    breakerConfig?: breaker.Config,
  ) {
    super(props);
    this.client = client;
    this.onStatusChange = options?.onStatusChange;
    this.failures = new telem.FailureReporter(this.onStatusChange);
    this.breaker = framer.retry.createBreaker(this.retryNotifier, breakerConfig);
    this.now = now;
    this.maxEntries = maxEntries;
    this._channels = this.props.channels.filter(
      (ch): ch is number => typeof ch === "number",
    );
  }

  /** @returns the entries, with the worst status of the channels. */
  value(): telem.Result<LogEntry[]> {
    if (this._channels.length > 0 && !this.valid) void this.resolve();
    return { data: this.entries, status: this.status };
  }

  /** Channels that stay keep their entries and samples, so none show twice. */
  setChannels(channels: channel.Key[]): void {
    if (compare.primitiveArrays(this._channels, channels) === compare.EQUAL) return;
    this._channels = channels;
    const kept = new Set(channels);
    this.entries = this.entries.filter((e) => kept.has(e.channelKey));
    for (const key of [...this.metas.keys(), ...this.unresolved.keys()])
      if (!kept.has(key)) {
        this.metas.delete(key);
        this.unresolved.delete(key);
      }
    if (channels.length === 0) this.reset();
    else if (this.valid) void this.resolve();
    this.notify();
  }

  /**
   * Never rejects. Retrieves the channels the log has no metadata for, then subscribes
   * to them and their indexes. A connectivity failure retries under the breaker.
   */
  private async resolve(): Promise<void> {
    const generation = ++this.generation;
    this.valid = true;
    const { client } = this;
    if (client == null) {
      this.onStatusChange?.(telem.DISCONNECTED_STATUS);
      return;
    }
    if (this.subscription == null) this.status = telem.LOADING_STATUS;
    while (generation === this.generation) {
      const missing = this._channels.filter(
        (k) => !this.metas.has(k) && this.unresolved.get(k)?.definitive !== true,
      );
      const results = await Promise.allSettled(
        missing.map(async (k) => await client.channels.retrieve(k)),
      );
      if (generation !== this.generation) return;
      let retried = false;
      results.forEach((r, i) => {
        const key = missing[i];
        if (r.status === "fulfilled") {
          this.metas.set(key, toMeta(r.value));
          this.unresolved.delete(key);
          return;
        }
        const definitive = framer.retry.isDefinitive(r.reason);
        const status = cstatus.fromException(r.reason, "Failed to read channel data");
        this.unresolved.set(key, { status, definitive });
        if (!definitive) retried = true;
      });
      this.subscribe(client);
      if (!retried) return this.breaker.reset();
      if (!(await this.breaker.wait())) return;
    }
  }

  private subscribe(client: telem.Client): void {
    const keys = new Set<channel.Key>();
    this.metas.forEach(({ index }, key) => {
      keys.add(key);
      if (index != null) keys.add(index);
    });
    const request: framer.LatestRequest = {
      keys: [...keys],
      limit: { variant: "latest", count: PAGE_SIZE },
    };
    let { subscription } = this;
    if (subscription != null) subscription.update(request);
    else {
      subscription = client.feed.subscribe(request);
      this.subscription = subscription;
      const sub = subscription;
      subscription.onChange(() => {
        if (this.pull(sub)) this.notify();
      });
    }
    this.pull(subscription);
    this.notify();
  }

  // Adds the samples each channel received since the last pull, and returns whether
  // the entries changed. A channel's entries wait while it or its index loads, so its
  // history lands in one pull before its live samples. Reading a snapshot lets the feed
  // drop the samples it held unread, so a channel is not read while its index loads.
  private pull(subscription: framer.Subscription): boolean {
    let worst: cstatus.Status = telem.SUCCESS_STATUS;
    const snapshots = new Map<channel.Key, framer.Snapshot>();
    const snapshotOf = (key: channel.Key): framer.Snapshot => {
      let s = snapshots.get(key);
      if (s != null) return s;
      s = subscription.value(key);
      snapshots.set(key, s);
      worst = cstatus.worst(worst, s.status)!;
      return s;
    };
    const groups: LogEntry[][] = [];
    const receivedAt = this.now();
    this.metas.forEach((meta, key) => {
      const index = meta.index == null ? null : snapshotOf(meta.index);
      if (index?.status.variant === "loading") return;
      const snapshot = snapshotOf(key);
      if (snapshot.status.variant === "loading") return;
      this.append(key, meta, snapshot, receivedAt);
      this.drain(key, meta, index?.series ?? null, groups);
    });
    this.unresolved.forEach((f) => (worst = cstatus.worst(worst, f.status)!));
    this.status = worst;
    this.failures.observe(worst);
    if (groups.length === 0) return false;
    this.insert(groups);
    this.evict();
    return true;
  }

  // Queues the samples the channel appended since its last pull, which are the newest
  // samples of the snapshot.
  private append(
    key: channel.Key,
    meta: Meta,
    snapshot: framer.Snapshot,
    receivedAt: TimeStamp,
  ): void {
    if (meta.snapshot === snapshot) return;
    const { series, appended } = snapshot;
    if (appended == null)
      throw new UnexpectedError(`snapshot of channel ${key} has no appended count`);
    let skip = Math.max(0, series.length - appended + meta.appended);
    for (const s of series.series) {
      for (let i = skip; i < s.length; i++) {
        let time: TimeStamp | undefined;
        if (meta.index == null) time = receivedAt;
        else if (meta.index === key) time = new TimeStamp(s.at(i, true) as bigint);
        meta.pending.push({
          alignment: alignmentOf(s, i),
          start: s.timeRange.isZero ? null : s.timeRange.start.valueOf(),
          offset: i,
          value: s.asString(i, true),
          text: s.dataType.equals(DataType.STRING),
          time,
        });
      }
      skip = Math.max(0, skip - s.length);
    }
    [meta.snapshot, meta.appended] = [snapshot, appended];
  }

  // Turns the queued samples into entries, oldest first, up to the first sample whose
  // index sample has not arrived. A sample whose index sample the index no longer holds
  // never gets a time, so it is dropped, as are the oldest waiting samples past
  // maxEntries.
  private drain(
    key: channel.Key,
    meta: Meta,
    index: MultiSeries | null,
    groups: LogEntry[][],
  ): void {
    const newest = index?.series.at(-1);
    const reach = newest == null ? null : alignmentOf(newest, newest.length - 1);
    const first = index?.series[0];
    const from =
      first == null || first.length === 0 ? null : (first.at(0, true) as bigint);
    let dropped = 0;
    let i = 0;
    for (; i < meta.pending.length; i++) {
      const sample = meta.pending[i];
      const found = index == null ? undefined : indexTime(index, sample);
      const time = sample.time ?? (found == null ? null : new TimeStamp(found));
      if (time != null) groups.push(toEntries(key, sample, time));
      else if (
        (reach != null && reach > sample.alignment) ||
        (from != null && sample.start != null && sample.start < from)
      )
        dropped++;
      else break;
    }
    const excess = Math.max(0, meta.pending.length - i - this.maxEntries);
    meta.pending.splice(0, i + excess);
    if (dropped + excess === 0 || meta.dropWarned) return;
    meta.dropWarned = true;
    this.onStatusChange?.({
      variant: "warning",
      message: "Dropped log samples with no time",
      description: `The index samples of channel ${key} did not arrive.`,
    });
  }

  // Inserts each group of entries at its place in time order, after the entries that
  // share its time. groups holds the entries of one sample per group.
  private insert(groups: LogEntry[][]): void {
    groups.sort((a, b) => compareTimes(a[0], b[0]));
    const first = groups[0][0].timestamp.valueOf();
    const last = this.entries.at(-1)?.timestamp.valueOf();
    if (last == null || last <= first) {
      groups.forEach((g) => append(this.entries, g));
      return;
    }
    let i = upperBound(this.entries, first);
    const merged = this.entries.slice(0, i);
    for (const g of groups) {
      const t = g[0].timestamp.valueOf();
      while (i < this.entries.length && this.entries[i].timestamp.valueOf() <= t)
        merged.push(this.entries[i++]);
      append(merged, g);
    }
    for (; i < this.entries.length; i++) merged.push(this.entries[i]);
    this.entries = merged;
  }

  // Evicts the oldest entries past maxEntries, never leaving a continuation first.
  private evict(): void {
    if (this.entries.length <= this.maxEntries) return;
    let excess =
      this.entries.length -
      this.maxEntries +
      Math.floor(this.maxEntries * EVICTION_HEADROOM);
    while (excess < this.entries.length && this.entries[excess].continuation) excess++;
    this.entries.splice(0, excess);
  }

  private reset(): void {
    this.generation++;
    this.retryNotifier.notify();
    this.subscription?.close();
    this.subscription = null;
    this.metas.clear();
    this.unresolved.clear();
    this.entries = [];
    this.status = telem.SUCCESS_STATUS;
    this.valid = false;
  }

  cleanup(): void {
    this.reset();
    this.failures.reset();
  }
}

const toMeta = (ch: channel.Channel): Meta => {
  let index: channel.Key | null = ch.index;
  if (ch.isIndex) index = ch.key;
  else if (index === 0 || (ch.virtual && !ch.isCalculated)) index = null;
  return {
    index,
    appended: 0,
    pending: [],
    dropWarned: false,
  };
};

const alignmentOf = (s: Series, i: number): bigint =>
  s.alignment + BigInt(i) * s.alignmentMultiple;

// Returns the time of the sample's index sample, or undefined if the index does not
// hold it. The stream and a read place one sample at different alignments, so a
// sample the index holds at another alignment is the offset-th index sample from the
// start of the sample's series.
const indexTime = (index: MultiSeries, p: Sample): bigint | undefined => {
  const time = index.atAlignment(p.alignment) as bigint | undefined;
  if (time != null || p.start == null) return time;
  let remaining = -1;
  for (const s of index.series) {
    if (s.length === 0) continue;
    if (remaining < 0) {
      if ((s.at(-1, true) as bigint) < p.start) continue;
      const j = lowerBound(s, p.start);
      if ((s.at(j, true) as bigint) !== p.start) return undefined;
      remaining = j + p.offset;
    }
    if (remaining < s.length) return s.at(remaining, true) as bigint;
    remaining -= s.length;
  }
  return undefined;
};

// Returns the position of the first time in s at or after time.
const lowerBound = (s: Series, time: bigint): number => {
  let [lo, hi] = [0, s.length];
  while (lo < hi) {
    const mid = (lo + hi) >>> 1;
    if ((s.at(mid, true) as bigint) < time) lo = mid + 1;
    else hi = mid;
  }
  return lo;
};

// Returns the entries of a sample. A text sample splits into one entry per line.
const toEntries = (
  key: channel.Key,
  { value, text }: Sample,
  timestamp: TimeStamp,
): LogEntry[] => {
  if (!text || !value.includes("\n")) return [{ channelKey: key, timestamp, value }];
  return value.split(/\r?\n/).map((line, j) => ({
    channelKey: key,
    timestamp,
    value: line,
    continuation: j > 0,
  }));
};

// Appends in a loop, since a spread of a sample with many lines passes the argument
// limit.
const append = (to: LogEntry[], from: LogEntry[]): void => {
  for (const e of from) to.push(e);
};

const compareTimes = (a: LogEntry, b: LogEntry): number => {
  const [x, y] = [a.timestamp.valueOf(), b.timestamp.valueOf()];
  return x < y ? -1 : x > y ? 1 : 0;
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
