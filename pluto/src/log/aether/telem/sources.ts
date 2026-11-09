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
  deep,
  type Series,
  staticWindow,
  sync,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { z } from "zod";

import {
  type Filter,
  type LogEntry,
  type LogSource,
  type LogSourceSpec,
  ZERO_FILTER,
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
// How long a log waits to read its oldest time again while the Core has none.
const START_RECHECK_INTERVAL = TimeSpan.seconds(30);
// The stored samples of each channel that one older page reads.
const OLDER_PAGE_SIZE = 1_000;

const streamMultiChannelLogPropsZ = z.object({
  channels: z.array(channel.keyZ),
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

// A page of a channel: its timed samples, oldest first.
interface Page {
  samples: Array<Omit<Sample, "receivedAt"> & { time: TimeStamp }>;
  // full is true when the read reached its count, so more samples can exist.
  full: boolean;
  // missed is true when the index read left out samples of the page.
  missed: boolean;
  status: cstatus.Status;
}

// A span of time newer than the entries, after start up to and including end. counts
// holds how many stored samples each channel has in it, which is what a read needs.
interface Unread {
  start: bigint;
  end: bigint;
  counts: Map<channel.Key, number>;
  // The lines of the samples that streamed into the span.
  lines?: number;
  // Whether a read of the span missed times, which a second read does not wait out.
  missed?: boolean;
}

// A channel whose retrieval failed. A definitive failure is not retried.
interface Failure {
  status: cstatus.Status;
  definitive: boolean;
}

interface Meta {
  name: string;
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
 * the newest stored samples of each channel when it opens, then streams. The list holds
 * at most maxEntries entries of a window that moves over the stored samples: reading in
 * one direction evicts entries from the other, and a later read finds them again.
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
  // abort ends the running page read, whose results go stale on a channel change, a
  // reset, or an eviction.
  private abort: AbortController | null = null;
  // paging is true from the start of a read until the next read may start, reading only
  // while the read runs.
  private paging = false;
  private reading = false;
  // unread holds the spans newer than the entries, oldest first. The log is live while
  // it is empty. Otherwise the last span counts the samples that stream past the log.
  private unread: Unread[] = [];
  // start is the oldest time the channels store, once known.
  private start: TimeStamp | null = null;
  // startRead counts the reads of start, and the newest one sets it.
  private startRead = 0;
  private startPending = false;
  private startReadAt = TimeStamp.ZERO;
  // skipped holds the streamed lines of channels that store nothing, which no read can
  // find again, while the log is not live.
  private skipped: LogEntry[][] = [];
  private skippedLines = 0;
  private readonly pageBreaker: breaker.Breaker;
  private readonly startBreaker: breaker.Breaker;
  private pageFailure?: cstatus.Status;
  private _channels: channel.Key[] = [];
  private filter: Filter = ZERO_FILTER;
  private hidden = new Set<channel.Key>();

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
    this.startBreaker = framer.retry.createBreaker(this.retryNotifier, breakerConfig);
    this.now = now;
    this.maxEntries = maxEntries;
    this.recheckInterval = recheckInterval;
    this._channels = this.props.channels;
  }

  /** @returns the entries, with the worst status of the channels. */
  value(): telem.Result<LogEntry[]> {
    if (this._channels.length > 0 && !this.valid) void this.resolve();
    const status = this.reading
      ? cstatus.worst(this.status, telem.LOADING_STATUS)!
      : this.status;
    return { data: this.entries, status };
  }

  /**
   * Channels that stay keep their entries and samples while the log is live, so none
   * show twice. A log that is not live returns to the newest samples.
   */
  setChannels(channels: channel.Key[]): void {
    if (compare.primitiveArrays(this._channels, channels) === compare.EQUAL) return;
    this._channels = channels;
    this.start = null;
    this.startRead++;
    this.startPending = false;
    this.endPaging();
    const kept = new Set(channels);
    this.entries = this.entries.filter((e) => kept.has(e.channelKey));
    for (const key of [...this.metas.keys(), ...this.unresolved.keys()])
      if (!kept.has(key)) {
        this.metas.delete(key);
        this.unresolved.delete(key);
      }
    if (channels.length === 0) this.reset();
    else {
      this.resume();
      if (this.valid) void this.resolve();
    }
    this.notify();
  }

  /**
   * The log holds only the entries the filter keeps, so a change reads them again from
   * the newest samples.
   */
  setFilter(filter: Filter): void {
    if (deep.equal(this.filter, filter)) return;
    this.filter = filter;
    this.hidden = new Set(filter.hiddenChannels);
    this.cancelPage();
    this.reload();
  }

  // The metas of the channels the filter keeps. No read asks for a hidden channel.
  private shownMetas(): Array<[channel.Key, Meta]> {
    return [...this.metas].filter(([key]) => !this.hidden.has(key));
  }

  /** The times the channels store, or null until the oldest is known. */
  get timeRange(): TimeRange | null {
    const end = this.unread.at(-1)?.end ?? this.entries.at(-1)?.timestamp.valueOf();
    if (this.start == null || end == null) return null;
    // Samples written after the read of start can be older than it.
    const start = TimeStamp.min(this.start, this.entries[0]?.timestamp ?? this.start);
    return new TimeRange(start, new TimeStamp(end));
  }

  /** Whether the entries hold the oldest stored sample of each shown channel. */
  get exhausted(): boolean {
    const metas = this.shownMetas().map(([, m]) => m);
    return (
      metas.length > 0 && metas.every((m) => m.exhausted && m.pending.length === 0)
    );
  }

  /** Whether the entries end at the newest samples, so that new samples join them. */
  get live(): boolean {
    return this.unread.length === 0;
  }

  /** Moves a log that is not live back to the newest samples. */
  resume(): void {
    // A late older page could evict the newest entries of a live log.
    this.cancelPage();
    if (this.live) return this.notify();
    this.reload();
  }

  // Clears the entries and pulls the newest samples again.
  private reload(): void {
    this.unread = [];
    this.skipped = [];
    this.skippedLines = 0;
    this.entries = [];
    this.metas.forEach((meta) => {
      Object.assign(meta, {
        snapshot: undefined,
        appended: 0,
        pending: [],
        oldest: undefined,
        exhausted: !meta.stored,
        pageSize: OLDER_PAGE_SIZE,
      });
    });
    if (this.subscription != null) this.pull(this.subscription);
    this.notify();
  }

  /**
   * Reads the next span of newer samples that the entries do not hold. Does nothing
   * while a page read runs or the log is live.
   */
  loadNewer(): void {
    const { client } = this;
    const [span] = this.unread;
    if (this.paging || client == null || span == null) return;
    if (this.unread.length === 1)
      this.unread.push({ start: span.end, end: span.end, counts: new Map() });
    void this.readNewer(client, span);
  }

  /**
   * Reads the next older page of each channel whose first stored sample the entries do
   * not hold. Does nothing while a page read runs.
   */
  loadOlder(): void {
    const { client } = this;
    if (this.paging || client == null) return;
    const metas = this.shownMetas().filter(
      ([, m]) => !m.exhausted && m.oldest != null && m.stored,
    );
    if (metas.length > 0) void this.readOlder(client, metas);
  }

  /**
   * Never rejects. Returns the pages, or null when the read aborts or fails. A failed
   * read waits under the breaker before the next one.
   */
  private async readPages(
    read: (signal: AbortSignal) => Array<Promise<Page>>,
  ): Promise<Page[] | null> {
    const abort = new AbortController();
    [this.abort, this.paging, this.reading] = [abort, true, true];
    this.notify();
    const { signal } = abort;
    const pages = await Promise.all(read(signal));
    if (signal.aborted) return null;
    const failed = pages.find((p) => p.status.variant === "error")?.status;
    if (failed != null) {
      if (failed.description !== this.pageFailure?.description)
        this.onStatusChange?.(failed);
      this.pageFailure = failed;
      this.reading = false;
      this.notify();
      if ((await this.pageBreaker.wait()) && !signal.aborted) this.paging = false;
      return null;
    }
    [this.paging, this.reading, this.pageFailure] = [false, false, undefined];
    this.pageBreaker.reset();
    return pages;
  }

  private async readNewer(client: telem.Client, span: Unread): Promise<void> {
    // A span counts what the log saw arrive, and an outage can hide more. A page that
    // fills its count may hide samples, so the span splits, and one time reads more.
    const middle = (span.start + span.end) / 2n;
    const limit = middle > span.start ? PAGE_SIZE : Number.MAX_SAFE_INTEGER;
    const counts = this.shownMetas()
      .filter(([, { stored }]) => stored)
      .map(([key]): [channel.Key, number] => [
        key,
        Math.min(Math.max(span.counts.get(key) ?? 0, OLDER_PAGE_SIZE), limit) + 1,
      ]);
    const size = (): number => [...span.counts.values()].reduce((a, b) => a + b, 0);
    const before = size();
    const range = new TimeRange(
      new TimeStamp(span.start + 1n),
      new TimeStamp(span.end + 1n),
    );
    const read = await this.readPages((signal) =>
      counts.map(
        async ([key, count]) =>
          await readPage(
            client,
            key,
            range,
            count,
            () => true,
            signal,
            this.recheckInterval,
          ),
      ),
    );
    if (read == null) return;
    // A span stays when a late sample changed the spans during the read, or its times
    // did not all arrive. A second read that misses times leaves those samples out, so
    // the log can move past them.
    const missed = read.some(({ missed }) => missed);
    if (
      this.unread[0] !== span ||
      size() !== before ||
      (missed && span.missed !== true)
    ) {
      if (this.unread[0] === span) span.missed = missed;
      this.notify();
      return;
    }
    const split = (): void => {
      const half = (start: bigint, end: bigint): Unread => ({
        start,
        end,
        counts: new Map(span.counts),
      });
      if (middle > span.start)
        this.unread.splice(0, 1, half(span.start, middle), half(middle, span.end));
      else counts.forEach(([key, count]) => span.counts.set(key, count * 2));
      this.notify();
    };
    if (read.some(({ full }) => full)) return split();
    const groups = counts.flatMap(([key], i) =>
      read[i].samples.map((p) => toEntries(key, p, p.time)),
    );
    // The entries keep this many lines after an eviction, so a longer read would evict
    // lines of its own before they show. Skipped lines count, since they join in time.
    const kept = this.keptLines;
    const { skipped } = this;
    // A pull adds its groups one channel at a time.
    skipped.sort((a, b) => compareTimes(a[0], b[0]));
    let lines = groups.reduce((total, group) => total + group.length, 0);
    let joined = 0;
    while (
      joined < skipped.length &&
      skipped[joined][0].timestamp.valueOf() <= span.end
    )
      lines += skipped[joined++].length;
    if (middle > span.start && lines > kept) return split();
    counts.forEach(([key], i) => {
      const meta = this.metas.get(key);
      if (meta == null) return;
      for (const p of read[i].samples) meta.oldest = older(meta.oldest, p, p.time);
    });
    this.unread.shift();
    // Only the span of streamed samples is left. Reading it would never catch a fast
    // stream, so the subscription gives it back once it fits in the entries and in the
    // page each channel holds. Skipped lines past it are newer than the entries, and
    // the log stays not live until they all join.
    const [tail] = this.unread;
    const drained =
      this.unread.length === 1 &&
      lines + (tail.lines ?? 0) <= kept &&
      [...tail.counts.values()].every((count) => count <= PAGE_SIZE);
    if (drained)
      while (
        joined < skipped.length &&
        (lines === 0 || lines + skipped[joined].length <= kept)
      )
        lines += skipped[joined++].length;
    for (const group of skipped.splice(0, joined)) {
      this.skippedLines -= group.length;
      groups.push(group);
    }
    const live = drained && skipped.length === 0;
    if (live) this.unread = [];
    if (groups.length > 0) this.insert(groups);
    this.evict();
    if (live) this.rejoin(tail);
    this.notify();
  }

  // The lines the entries keep after an eviction.
  private get keptLines(): number {
    return this.maxEntries - Math.floor(this.maxEntries * EVICTION_HEADROOM);
  }

  // Pulls the samples of a span the log only counted again, now that it is live.
  private rejoin({ counts }: Unread): void {
    counts.forEach((count, key) => {
      const meta = this.metas.get(key);
      if (meta == null) return;
      meta.appended -= count + meta.pending.length;
      meta.pending = [];
    });
    if (this.subscription != null) this.pull(this.subscription);
  }

  private async readOlder(
    client: telem.Client,
    metas: Array<[channel.Key, Meta]>,
  ): Promise<void> {
    const oldests = metas.map(([, { oldest }]) => oldest!);
    const read = await this.readPages((signal) =>
      metas.map(
        async ([key, { pageSize }], i) =>
          await readPage(
            client,
            key,
            new TimeRange(
              TimeStamp.MIN,
              // A null alignment excludes the samples at the time, which could fill
              // the page.
              new TimeStamp(oldests[i].time + (oldests[i].alignment == null ? 0n : 1n)),
            ),
            pageSize,
            (p) => before(p, oldests[i]),
            signal,
            this.recheckInterval,
          ),
      ),
    );
    if (read == null) return;
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
    // An eviction keeps the oldest lines, so a page longer than half of them could
    // evict the lines the view sits on. The next page reads what this one leaves out.
    const limit = Math.floor(this.keptLines / 2);
    let start = items.length - 1;
    let lines = items.at(-1)?.group.length ?? 0;
    while (start > 0 && lines + items[start - 1].group.length <= limit)
      lines += items[--start].group.length;
    items.splice(0, start);
    metas.forEach(([, meta], i) => {
      const { samples, full, missed } = read[i];
      const own = items.filter((item) => item.meta === meta);
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
    if (items.length > 0)
      this.insert(
        items.map(({ group }) => group),
        true,
      );
    this.evictNewest();
    this.notify();
  }

  private cancelPage(): void {
    this.abort?.abort();
    [this.abort, this.paging, this.reading] = [null, false, false];
  }

  private endPaging(): void {
    this.cancelPage();
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
      void this.readStart(client);
      if (!retried) return this.breaker.reset();
      if (!(await this.breaker.wait())) return;
    }
  }

  // Never rejects. A newer read replaces this one. A connectivity failure retries under
  // the breaker. Any other failure leaves start unknown and startPending set, so only a
  // change of channels reads again.
  private async readStart(client: telem.Client): Promise<void> {
    const read = ++this.startRead;
    this.startReadAt = this.now();
    const keys = [...this.metas].filter(([, { stored }]) => stored).map(([key]) => key);
    this.startBreaker.reset();
    this.startPending = true;
    while (read === this.startRead)
      try {
        const start = keys.length === 0 ? null : await client.feed.firstTime(keys);
        if (read !== this.startRead) return;
        this.start = start;
        this.startPending = false;
        return this.notify();
      } catch (e) {
        if (framer.retry.isDefinitive(e)) return;
        if (!(await this.startBreaker.wait())) {
          if (read === this.startRead) this.startPending = false;
          return;
        }
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
      if (snapshot.status.variant === "loading" || this.hidden.has(key)) return;
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
    // A channel that was empty when the log read start can have one now. Samples that
    // stream without being stored find none, so the reads stay far apart.
    if (
      this.start == null &&
      !this.startPending &&
      this.client != null &&
      receivedAt.span(this.startReadAt).greaterThan(START_RECHECK_INTERVAL)
    )
      void this.readStart(this.client);
    if (!this.live) {
      const [{ start }] = this.unread;
      const held = groups.filter(([{ timestamp }]) => timestamp.valueOf() <= start);
      this.skip(groups.filter(([{ timestamp }]) => timestamp.valueOf() > start));
      // The skipped samples move the end of the time range.
      if (held.length === 0) return true;
      this.insert(held);
      this.evictNewest();
      return true;
    }
    this.insert(groups);
    this.evict();
    return true;
  }

  // Counts the streamed samples that a log which is not live leaves out. Most join the
  // last span, which a read splits when it holds more than one page.
  private skip(groups: LogEntry[][]): void {
    const last = this.unread.at(-1)!;
    for (const group of groups) {
      const [{ channelKey, timestamp }] = group;
      if (this.metas.get(channelKey)?.stored !== true) {
        this.skipped.push(group);
        this.skippedLines += group.length;
        continue;
      }
      const time = timestamp.valueOf();
      const span =
        time > last.start ? last : this.unread.find(({ end }) => time <= end)!;
      if (time > span.end) span.end = time;
      span.counts.set(channelKey, (span.counts.get(channelKey) ?? 0) + 1);
      span.lines = (span.lines ?? 0) + group.length;
    }
    let excess = 0;
    while (this.skippedLines > this.maxEntries)
      this.skippedLines -= this.skipped[excess++].length;
    this.skipped.splice(0, excess);
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
    if (meta.snapshot === snapshot && meta.appended === snapshot.appended) return;
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
      name: meta.name,
      message: "Dropped log samples with no time",
      description:
        "The times of these samples did not arrive. Check that what writes to this " +
        "channel also writes to its index channel.",
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

  // Evicts the newest entries past maxEntries, which a later read finds again. The cut
  // falls between two times.
  private evictNewest(): void {
    const { entries } = this;
    if (entries.length <= this.maxEntries) return;
    const headroom = Math.floor(this.maxEntries * EVICTION_HEADROOM);
    let at = this.maxEntries - headroom;
    const time = (i: number): bigint => entries[i].timestamp.valueOf();
    while (at > 0 && time(at) === time(at - 1)) at--;
    // One time that fills the entries stays whole, since a read finds a time whole.
    if (at === 0) at = bound(entries, time(0), false);
    if (at === entries.length) return;
    const evicted = entries.splice(at);
    const end = evicted[evicted.length - 1].timestamp.valueOf();
    const counts = new Map<channel.Key, number>();
    for (const { channelKey, continuation } of evicted)
      if (continuation !== true && this.metas.get(channelKey)?.stored === true)
        counts.set(channelKey, (counts.get(channelKey) ?? 0) + 1);
    const start = time(at - 1);
    // A channel with no entries left reads older samples from below the unread span.
    this.metas.forEach((m) => {
      if (m.oldest != null && m.oldest.time > start)
        m.oldest = { time: start + 1n, alignment: null };
    });
    if (this.live) this.unread.push({ start: end, end, counts: new Map() });
    this.unread.unshift({ start, end, counts });
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
    this.unread = [];
    this.skipped = [];
    this.skippedLines = 0;
    this.entries = [];
    this.start = null;
    this.startRead++;
    this.startPending = false;
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
    name: ch.name,
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
  { alignment }: Pick<Sample, "alignment">,
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

// Reads the newest count stored samples of the channel in the range, and returns those
// that keep accepts. A sample that gets no time is left out. An aborted read returns an
// empty page.
const readPage = async (
  client: telem.Client,
  key: channel.Key,
  range: TimeRange,
  count: number,
  keep: (sample: { time: TimeStamp; alignment: bigint }) => boolean,
  signal: AbortSignal,
  recheckInterval: TimeSpan,
): Promise<Page> => {
  const subscription = client.feed.subscribe({
    keys: [key],
    window: staticWindow(range),
    limit: { variant: "latest", count },
  });
  const aborted = (): Page => {
    subscription.close();
    return { samples: [], full: false, missed: false, status: telem.SUCCESS_STATUS };
  };
  const data = await settled(subscription, key, signal);
  if (data == null) return aborted();
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
  // The feed can close the subscription just before the abort.
  if (signal.aborted) return aborted();
  const samples: Page["samples"] = [];
  let missed = false;
  for (const p of read) {
    const time = timeOf(p);
    if (time == null) missed = true;
    else if (keep({ ...p, time })) samples.push({ ...p, time });
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
