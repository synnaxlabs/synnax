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
  type Series,
  staticWindow,
  sync,
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
import { telem } from "@/telem/aether";

const MAX_ENTRIES = 100_000;
// Past maxEntries, a log evicts this share of maxEntries beyond the excess, so a full
// log moves its entries once per share instead of on every pull.
const EVICTION_HEADROOM = 0.1;
// The newest stored samples of each channel a log shows when it opens.
const PAGE_SIZE = 10_000;
// How often a log asks again for the times of its waiting samples. The client retries a
// failed read only when asked.
const RECHECK_INTERVAL = TimeSpan.seconds(5);
// How long a sample waits for its time before the log drops it.
const WAIT_LIMIT = TimeSpan.seconds(20);
// The stored samples of each channel that one older page reads.
const OLDER_PAGE_SIZE = 1_000;

const streamMultiChannelLogPropsZ = z.object({
  channels: z.array(channel.keyZ.or(z.string())),
});

export type StreamMultiChannelLogProps = z.input<typeof streamMultiChannelLogPropsZ>;

// A sample that waits for its time.
interface Sample {
  alignment: bigint;
  value: string;
  text: boolean;
  receivedAt: TimeStamp;
}

// The oldest sample the entries hold of a channel. A null alignment means the entries
// hold every sample of the channel at the time and none before it.
interface Oldest {
  time: bigint;
  alignment: bigint | null;
}

// An older page of a channel: its timed samples, oldest first.
interface Page {
  samples: Array<Omit<Sample, "receivedAt"> & { time: TimeStamp }>;
  // full is true when the read reached its count, so older samples can exist.
  full: boolean;
  // missed is true when the index read left out samples of the page.
  missed: boolean;
  status: cstatus.Status;
}

// A channel whose retrieval failed. A definitive failure is not retried.
interface Failure {
  status: cstatus.Status;
  definitive: boolean;
}

interface Meta {
  // The snapshot and the count of appended samples that entries were last read up to.
  snapshot?: framer.Snapshot;
  appended: number;
  // Whether the channel stores samples, so that older pages can read them.
  stored: boolean;
  // Samples that wait for their times, oldest first.
  pending: Sample[];
  // Whether the log has warned that it dropped samples that never got a time.
  dropWarned: boolean;
  oldest?: Oldest;
  // exhausted is true when the entries hold the channel's first stored sample, or the
  // channel stores none.
  exhausted: boolean;
  // The count of the next older page. It grows while pages hold only samples at the
  // oldest time, which the entries already hold.
  pageSize: number;
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
  private readonly recheckInterval: TimeSpan;
  private recheck: ReturnType<typeof setTimeout> | null = null;
  private entries: LogEntry[] = [];
  private status = telem.SUCCESS_STATUS;
  private valid = false;
  private generation = 0;
  // abort ends the running older page read, whose results go stale on a channel
  // change, a reset, or an eviction.
  private abort: AbortController | null = null;
  // paging is true from the start of a read until the next read may start, reading only
  // while the read runs.
  private paging = false;
  private reading = false;
  // full is true once an older page passed maxEntries.
  private full = false;
  private readonly pageBreaker: breaker.Breaker;
  private pageFailure?: cstatus.Status;
  private _channels: channel.Key[] = [];

  constructor(
    client: telem.Client | null,
    props: unknown,
    options?: telem.CreateOptions,
    now: () => TimeStamp = client?.now ?? (() => TimeStamp.now()),
    maxEntries: number = MAX_ENTRIES,
    breakerConfig?: breaker.Config,
    recheckInterval: TimeSpan = RECHECK_INTERVAL,
  ) {
    super(props);
    this.client = client;
    this.onStatusChange = options?.onStatusChange;
    this.failures = new telem.FailureReporter(this.onStatusChange);
    this.breaker = framer.retry.createBreaker(this.retryNotifier, breakerConfig);
    this.pageBreaker = framer.retry.createBreaker(this.retryNotifier, breakerConfig);
    this.now = now;
    this.maxEntries = maxEntries;
    this.recheckInterval = recheckInterval;
    this._channels = this.props.channels.filter(
      (ch): ch is number => typeof ch === "number",
    );
  }

  /** @returns the entries, with the worst status of the channels. */
  value(): telem.Result<LogEntry[]> {
    if (this._channels.length > 0 && !this.valid) void this.resolve();
    const status = this.reading
      ? cstatus.worst(this.status, telem.LOADING_STATUS)!
      : this.status;
    return { data: this.entries, status };
  }

  /** Channels that stay keep their entries and samples, so none show twice. */
  setChannels(channels: channel.Key[]): void {
    if (compare.primitiveArrays(this._channels, channels) === compare.EQUAL) return;
    this._channels = channels;
    this.endPaging();
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
   * Reads the next older page of each channel whose first stored sample the entries do
   * not hold. Does nothing while a page read runs or the entries are full.
   */
  loadOlder(): void {
    const { client } = this;
    if (this.paging || this.full || client == null) return;
    const metas = [...this.metas].filter(
      ([, m]) => !m.exhausted && m.oldest != null && m.stored,
    );
    if (metas.length > 0) void this.readOlder(client, metas);
  }

  /** Never rejects. A failed read waits under the breaker before the next one. */
  private async readOlder(
    client: telem.Client,
    metas: Array<[channel.Key, Meta]>,
  ): Promise<void> {
    const abort = new AbortController();
    [this.abort, this.paging, this.reading] = [abort, true, true];
    this.notify();
    const { signal } = abort;
    const oldests = metas.map(([, { oldest }]) => oldest!);
    const read = await Promise.all(
      metas.map(
        async ([key, { pageSize }], i) =>
          await readPage(
            client,
            key,
            oldests[i],
            pageSize,
            signal,
            this.recheckInterval,
          ),
      ),
    );
    if (signal.aborted) return;
    const failed = read.find((p) => p.status.variant === "error")?.status;
    if (failed != null) {
      if (failed.description !== this.pageFailure?.description)
        this.onStatusChange?.(failed);
      this.pageFailure = failed;
      this.reading = false;
      this.notify();
      if ((await this.pageBreaker.wait()) && !signal.aborted) this.paging = false;
      return;
    }
    [this.paging, this.reading, this.pageFailure] = [false, false, undefined];
    this.pageBreaker.reset();
    // A full page reaches back only to its oldest sample, or to the oldest time held if
    // it has none, so the entries keep only the times that every full page covers.
    let boundary: bigint | null = null;
    read.forEach(({ samples, full }, i) => {
      if (!full) return;
      const first = samples[0]?.time.valueOf() ?? oldests[i].time;
      if (boundary == null || first > boundary) boundary = first;
    });
    const items: Array<{ meta: Meta; p: Page["samples"][number]; group: LogEntry[] }> =
      [];
    metas.forEach(([key, meta], i) => {
      for (const p of read[i].samples)
        if (boundary == null || p.time.valueOf() >= boundary)
          items.push({ meta, p, group: toEntries(key, p, p.time) });
    });
    items.sort((x, y) => compareTimes(x.group[0], y.group[0]));
    // A page that passes maxEntries keeps its newest times that fit, and the log reads
    // no older page after it.
    let lines = items.reduce((n, { group }) => n + group.length, 0);
    let cut = 0;
    while (lines > this.maxEntries - this.entries.length && cut < items.length)
      lines -= items[cut++].group.length;
    while (
      cut > 0 &&
      cut < items.length &&
      items[cut].p.time.equals(items[cut - 1].p.time)
    )
      lines -= items[cut++].group.length;
    if (cut > 0) this.full = true;
    const shown = items.slice(cut);
    metas.forEach(([, meta], i) => {
      const { samples, full, missed } = read[i];
      const own = shown.filter((item) => item.meta === meta);
      if (own.length > 0) {
        const [{ time, alignment }] = own.map((item) => item.p);
        meta.oldest = { time: time.valueOf(), alignment };
      }
      meta.exhausted = !full && !missed && own.length === samples.length;
      // A full page of samples the entries hold, or a page whose index read missed
      // samples, reads more next time, up to maxEntries.
      if ((!full && !missed) || samples.length > 0) meta.pageSize = OLDER_PAGE_SIZE;
      else if (meta.pageSize >= this.maxEntries) meta.exhausted = true;
      else meta.pageSize *= 2;
    });
    if (shown.length > 0)
      this.insert(
        shown.map(({ group }) => group),
        true,
      );
    this.notify();
  }

  private cancelPage(): void {
    this.abort?.abort();
    [this.abort, this.paging, this.reading] = [null, false, false];
  }

  private endPaging(): void {
    this.cancelPage();
    this.full = false;
    this.pageBreaker.reset();
  }

  /**
   * Never rejects. Retrieves the channels the log has no metadata for, then subscribes
   * to them. A connectivity failure retries under the breaker.
   */
  private async resolve(): Promise<void> {
    const generation = ++this.generation;
    this.valid = true;
    const { client } = this;
    if (client == null) {
      this.onStatusChange?.(telem.DISCONNECTED_STATUS);
      return;
    }
    const isMissing = (k: channel.Key): boolean =>
      !this.metas.has(k) && this.unresolved.get(k)?.definitive !== true;
    if (this.subscription == null || this._channels.some(isMissing))
      this.status = telem.LOADING_STATUS;
    while (generation === this.generation) {
      const missing = this._channels.filter(isMissing);
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
    const request: framer.LatestRequest = {
      keys: [...this.metas.keys()],
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
  // the entries or the status changed. A log with samples and no entries is loading.
  private pull(subscription: framer.Subscription): boolean {
    let worst: cstatus.Status = telem.SUCCESS_STATUS;
    const groups: LogEntry[][] = [];
    const receivedAt = this.now();
    let waiting = false;
    this.metas.forEach((meta, key) => {
      const snapshot = subscription.value(key);
      worst = cstatus.worst(worst, snapshot.status)!;
      // Asking opens the stream of the channel's index before its first live sample.
      subscription.timeAt(key, 0n);
      if (snapshot.status.variant === "loading") return;
      this.append(key, meta, snapshot, receivedAt);
      this.drain(key, meta, subscription, groups, receivedAt);
      if (meta.pending.length > 0) waiting = true;
    });
    if (waiting && groups.length === 0 && this.entries.length === 0)
      worst = cstatus.worst(worst, telem.LOADING_STATUS)!;
    this.scheduleRecheck(subscription, waiting);
    this.unresolved.forEach((f) => (worst = cstatus.worst(worst, f.status)!));
    const statusChanged = worst !== this.status;
    this.status = worst;
    this.failures.observe(worst);
    if (groups.length === 0) return statusChanged;
    this.insert(groups);
    this.evict();
    return true;
  }

  private scheduleRecheck(subscription: framer.Subscription, waiting: boolean): void {
    if (!waiting) {
      this.stopRecheck();
      return;
    }
    if (this.recheck != null) return;
    this.recheck = setTimeout(() => {
      this.recheck = null;
      if (this.pull(subscription)) this.notify();
    }, this.recheckInterval.milliseconds);
  }

  private stopRecheck(): void {
    if (this.recheck != null) clearTimeout(this.recheck);
    this.recheck = null;
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
    if (
      meta.snapshot == null &&
      snapshot.status.variant === "success" &&
      series.length < PAGE_SIZE
    )
      meta.exhausted = true;
    let skip = Math.max(0, series.length - appended + meta.appended);
    for (const s of series.series) {
      for (let i = skip; i < s.length; i++)
        meta.pending.push({
          alignment: alignmentOf(s, i),
          value: s.asString(i, true),
          text: s.dataType.equals(DataType.STRING),
          receivedAt,
        });
      skip = Math.max(0, skip - s.length);
    }
    [meta.snapshot, meta.appended] = [snapshot, appended];
  }

  // Turns the queued samples that have times into entries. A sample that waits past
  // WAIT_LIMIT is dropped, as are the oldest samples that wait past what the log reads
  // when it opens plus what it can hold.
  private drain(
    key: channel.Key,
    meta: Meta,
    subscription: framer.Subscription,
    groups: LogEntry[][],
    now: TimeStamp,
  ): void {
    let dropped = 0;
    meta.pending = meta.pending.filter((sample) => {
      const time = subscription.timeAt(key, sample.alignment);
      if (time != null) {
        groups.push(toEntries(key, sample, time));
        if (meta.stored) meta.oldest = older(meta.oldest, sample, time);
        return false;
      }
      if (sample.receivedAt.span(now).lessThan(WAIT_LIMIT)) return true;
      dropped++;
      return false;
    });
    const excess = Math.max(0, meta.pending.length - PAGE_SIZE - this.maxEntries);
    meta.pending.splice(0, excess);
    // An older page can still read the dropped samples with their times.
    if (dropped > 0) meta.exhausted = false;
    if (dropped + excess === 0 || meta.dropWarned) return;
    meta.dropWarned = true;
    this.onStatusChange?.({
      variant: "warning",
      message: "Dropped log samples with no time",
      description: `The time of the samples of channel ${key} did not arrive.`,
    });
  }

  // Inserts each group of entries at its place in time order: after the entries that
  // share its time, or before them when prepended from an older page. groups holds the
  // entries of one sample per group.
  private insert(groups: LogEntry[][], prepended = false): void {
    groups.sort((a, b) => compareTimes(a[0], b[0]));
    const first = groups[0][0].timestamp.valueOf();
    const last = this.entries.at(-1)?.timestamp.valueOf();
    if (!prepended && (last == null || last <= first)) {
      groups.forEach((g) => append(this.entries, g));
      return;
    }
    const after = (t: bigint, e: LogEntry): boolean =>
      prepended ? e.timestamp.valueOf() < t : e.timestamp.valueOf() <= t;
    let i = bound(this.entries, first, prepended);
    const merged = this.entries.slice(0, i);
    for (const g of groups) {
      const t = g[0].timestamp.valueOf();
      while (i < this.entries.length && after(t, this.entries[i]))
        merged.push(this.entries[i++]);
      append(merged, g);
    }
    for (; i < this.entries.length; i++) merged.push(this.entries[i]);
    this.entries = merged;
  }

  // Evicts the oldest entries past maxEntries. The cut falls between two times, so the
  // entries keep every sample at or after it, and an older page reads before it. When
  // the newest time alone passes maxEntries, the cut falls between two of its samples.
  private evict(): void {
    const { entries } = this;
    if (entries.length <= this.maxEntries) return;
    let excess =
      entries.length -
      this.maxEntries +
      Math.floor(this.maxEntries * EVICTION_HEADROOM);
    const time = (i: number): bigint => entries[i].timestamp.valueOf();
    let next = excess;
    while (next < entries.length && time(next) === time(next - 1)) next++;
    const split = next >= entries.length;
    if (!split) excess = next;
    else
      while (excess < entries.length && entries[excess].continuation === true) excess++;
    const cut =
      excess < entries.length
        ? entries[excess].timestamp.valueOf()
        : entries[entries.length - 1].timestamp.valueOf() + 1n;
    entries.splice(0, excess);
    this.cancelPage();
    this.metas.forEach((m, key) => {
      if (m.oldest == null || m.oldest.time > cut || (!split && m.oldest.time === cut))
        return;
      m.exhausted = false;
      if (!split) {
        m.oldest = { time: cut, alignment: null };
        return;
      }
      // A split time keeps the channel's oldest held sample at it as the boundary, so
      // an older page reads the evicted samples of that time again.
      const kept = entries.find((e) => e.channelKey === key && e.alignment != null);
      m.oldest =
        kept != null && kept.timestamp.valueOf() === cut
          ? { time: cut, alignment: kept.alignment! }
          : { time: cut + 1n, alignment: null };
    });
  }

  private reset(): void {
    this.endPaging();
    this.generation++;
    this.retryNotifier.notify();
    this.stopRecheck();
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

// Returns the older of the oldest sample held and the sample at time.
const toMeta = (ch: channel.Channel): Meta => {
  const stored = ch.isIndex || (ch.index !== 0 && (!ch.virtual || ch.isCalculated));
  return {
    stored,
    appended: 0,
    pending: [],
    dropWarned: false,
    exhausted: !stored,
    pageSize: OLDER_PAGE_SIZE,
  };
};

const older = (
  oldest: Oldest | undefined,
  { alignment }: Sample,
  time: TimeStamp,
): Oldest | undefined => {
  const t = time.valueOf();
  if (oldest == null || t < oldest.time) return { time: t, alignment };
  if (t > oldest.time || oldest.alignment == null || alignment >= oldest.alignment)
    return oldest;
  return { time: t, alignment };
};

const domainOf = (alignment: bigint): bigint => alignment >> 32n;

// Returns whether sample p is older than the oldest sample held.
const before = (p: { time: TimeStamp; alignment: bigint }, oldest: Oldest): boolean => {
  const t = p.time.valueOf();
  if (t !== oldest.time) return t < oldest.time;
  return (
    oldest.alignment != null &&
    domainOf(p.alignment) === domainOf(oldest.alignment) &&
    p.alignment < oldest.alignment
  );
};

// Waits for the read of the channel to land, or returns null if the signal aborts.
const settled = async (
  subscription: framer.Subscription,
  key: channel.Key,
  signal: AbortSignal,
): Promise<framer.Snapshot | null> =>
  await new Promise((resolve) => {
    signal.addEventListener("abort", () => resolve(null));
    const check = (): void => {
      const snapshot = subscription.value(key);
      const { status } = snapshot;
      if (
        status.variant !== "success" &&
        status.variant !== "error" &&
        status.message !== framer.NO_HISTORY.message
      )
        return;
      stop();
      resolve(snapshot);
    };
    const stop = subscription.onChange(check);
    check();
  });

// Calls check on every change and every recheck until it returns true, the signal
// aborts, or the page's wait runs out.
const waitFor = async (
  subscription: framer.Subscription,
  check: () => boolean,
  signal: AbortSignal,
  recheckInterval: TimeSpan,
): Promise<void> =>
  await new Promise((resolve) => {
    let checks = Math.ceil(WAIT_LIMIT.milliseconds / recheckInterval.milliseconds) + 1;
    const done = (): void => {
      stop();
      clearInterval(timer);
      resolve();
    };
    const run = (): void => {
      if (check() || --checks <= 0) done();
    };
    const stop = subscription.onChange(run);
    const timer = setInterval(run, recheckInterval.milliseconds);
    signal.addEventListener("abort", done);
    run();
  });

// Reads the newest count stored samples of the channel up to the oldest one held, and
// returns those older than it. A sample that gets no time is left out. An aborted read
// returns an empty page.
const readPage = async (
  client: telem.Client,
  key: channel.Key,
  oldest: Oldest,
  count: number,
  signal: AbortSignal,
  recheckInterval: TimeSpan,
): Promise<Page> => {
  const end = oldest.alignment == null ? oldest.time : oldest.time + 1n;
  const subscription = client.feed.subscribe({
    keys: [key],
    window: staticWindow(new TimeRange(TimeStamp.MIN, new TimeStamp(end))),
    limit: { variant: "latest", count },
  });
  const data = await settled(subscription, key, signal);
  if (data == null) {
    subscription.close();
    return { samples: [], full: false, missed: false, status: telem.SUCCESS_STATUS };
  }
  const read: Array<Omit<Sample, "receivedAt">> = [];
  for (const s of data.series.series)
    for (let i = 0; i < s.length; i++)
      read.push({
        alignment: alignmentOf(s, i),
        value: s.asString(i, true),
        text: s.dataType.equals(DataType.STRING),
      });
  const timeOf = ({ alignment }: { alignment: bigint }): TimeStamp | null =>
    subscription.timeAt(key, alignment);
  await waitFor(
    subscription,
    () => read.every((p) => timeOf(p) != null),
    signal,
    recheckInterval,
  );
  const samples: Page["samples"] = [];
  let missed = false;
  for (const p of read) {
    const time = timeOf(p);
    if (time == null) missed = true;
    else if (before({ ...p, time }, oldest)) samples.push({ ...p, time });
  }
  const full = data.series.length >= count;
  subscription.close();
  return { samples, full, missed, status: data.status };
};

const alignmentOf = (s: Series, i: number): bigint =>
  s.alignment + BigInt(i) * s.alignmentMultiple;

// Returns the entries of a sample. A text sample splits into one entry per line.
const toEntries = (
  key: channel.Key,
  { value, text, alignment }: Pick<Sample, "value" | "text" | "alignment">,
  timestamp: TimeStamp,
): LogEntry[] => {
  if (!text || !value.includes("\n"))
    return [{ channelKey: key, timestamp, value, alignment }];
  return value.split(/\r?\n/).map((line, j) => ({
    channelKey: key,
    timestamp,
    value: line,
    alignment,
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

// Returns the position after the last entry before time, or at or before it unless
// prepended.
const bound = (entries: LogEntry[], time: bigint, prepended: boolean): number => {
  let [lo, hi] = [0, entries.length];
  while (lo < hi) {
    const mid = (lo + hi) >>> 1;
    const t = entries[mid].timestamp.valueOf();
    if (t < time || (!prepended && t === time)) lo = mid + 1;
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
