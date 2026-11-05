// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  bounds,
  type breaker,
  DataType,
  deep,
  MultiSeries,
  resolveWindow,
  Series,
  sync,
  TimeRange,
  TimeSpan,
  TimeStamp,
  type Window,
} from "@synnaxlabs/x";

import { type channel } from "@/channel";
import { NotFoundError, UnsupportedError } from "@/errors";
import { type ChannelRetriever } from "@/framer/adapter";
import { domainOf } from "@/framer/cache/block";
import { Dynamic } from "@/framer/cache/dynamic";
import { IndexStamps, Stamps } from "@/framer/cache/stamps";
import { LOADING, NO_HISTORY, READ_FAILED, SUCCESS } from "@/framer/cache/status";
import { calculated, type Store } from "@/framer/cache/store";
import { type Demand, type MultiplexedStreamer } from "@/framer/cache/streamer";
import { createBreaker, isDefinitive } from "@/framer/retry";
import { type Snapshot } from "@/framer/subscription";
import { status } from "@/status";

// Returns the timestamps as one series at alignment.
const realign = (times: MultiSeries, alignment: bigint, tr: TimeRange): Series => {
  const data = new BigInt64Array(times.length);
  let i = 0;
  for (const t of times) data[i++] = t as bigint;
  return new Series({ data, dataType: DataType.TIMESTAMP, alignment, timeRange: tr });
};

export interface LatestProps {
  /** Reports the history changes that make the samples read go out of date. */
  store: Store;
  streamer: MultiplexedStreamer;
  /** Reads the newest count stored samples of the channel in the time range. */
  readLatest: (key: channel.Key, count: number, tr: TimeRange) => Promise<MultiSeries>;
  retrieveChannels: ChannelRetriever;
  now: () => TimeStamp;
  /** Retry behavior for failed reads. */
  breaker?: breaker.Config;
}

// Past this many samples, Latest drops samples that no value served.
const UNSERVED_CAP = 100_000;
// Bounds on the samples of a buffer that streamed series grow into.
const MIN_BUFFER = 1_024;
const MAX_BUFFER = 65_536;
// How long a series waits to read its index again after a read that did not stamp it.
const INDEX_RETRY = TimeSpan.seconds(5);
// The most reads of the index that one series gets.
const INDEX_TRIES = 3;

// The reads of the index for the held series over span. A read that does not stamp the
// series counts a try and waits until retryAt, which is unset while a read runs.
interface IndexRead {
  span: bounds.Bounds<bigint>;
  tries: number;
  retryAt?: TimeStamp;
}

// A running read of the newest stored samples.
interface Load {
  // abort ends the load's wait for the stream's ack.
  abort: AbortController;
  // mark is where the series streamed while the load runs start.
  mark: number;
  // end bounds a read after a history change, so every sample after it streams.
  end?: TimeStamp;
  // repeated is true when the read returns samples that already counted as appended,
  // so they do not count again.
  repeated?: boolean;
  // after starts the read after the samples held, since they already counted.
  after?: TimeStamp;
  // dropped is true when frames may have dropped before the load lands, so a read of
  // the samples after its read follows.
  dropped?: boolean;
}

/**
 * Serves the newest samples of one channel: a read of its newest stored samples, and
 * every newer sample the stream delivers. Streamed samples serve while the read waits
 * or fails, and a stream drop reads the newest samples again.
 */
export class Latest {
  // series holds the closed series, oldest first. The buffer's leading buffer holds the
  // newest samples after them, while streamed series continue it.
  private series: Series[] = [];
  private closedTotal = 0;
  private buffer: Dynamic;
  // unserved counts the newest samples that no value served.
  private unserved = 0;
  private appended = 0;
  // taken is true once consumers may have taken the samples held: a read of the
  // request landed, or a value served streamed samples under an error or warning.
  private taken = false;
  private running: Load | null;
  private readonly demand: Demand;
  private readonly stopInvalidations: () => void;
  // timeSource is null until the channel is found.
  private timeSource: { index: channel.Key } | "self" | "stamps" | null = null;
  // indexStamps holds the index series that stamp the samples held, from the first
  // timeAt that needs them.
  private indexStamps: IndexStamps | null = null;
  private indexDemand: Demand | null = null;
  // indexReads holds the index reads in flight and the ones that wait to run again.
  private indexReads: IndexRead[] = [];
  // indexRefused is true once the Core refuses a read of the index, which it then
  // refuses every time.
  private indexRefused = false;
  // stamps records each write until the channel turns out to have an index, since
  // streamed series can arrive before it is found.
  private stamps: Stamps | null;
  // seedable is true for a channel the Core stores nothing of, null until known.
  private seedable: boolean | null = null;
  // The newest series streamed before the channel was known.
  private pending: Series | null = null;
  // dedupe is how fresh finds repeated samples: by Core alignment for a stored channel,
  // and by time for one a calculation writes, whose alignments depend on how its inputs
  // split into series. A channel that only streams repeats no sample.
  private dedupe: "alignment" | "time" | null = null;
  private failure: status.Status | null = null;
  private readonly notifier = new sync.Notifier();
  private readonly statuses = {
    loading: status.create(LOADING),
    success: status.create(SUCCESS),
  };
  // snapshot is the value last served, or null after a change. It also goes out of date
  // when a series leaves the window. A growing leading buffer leaves it in date.
  private snapshot: Snapshot | null = null;
  // shown is true when a value served samples streamed since the running load began,
  // under an error or warning. Consumers may have taken them, so the read must not land
  // before them.
  private shown = false;

  constructor(
    private readonly props: LatestProps,
    readonly key: channel.Key,
    private window: Window | undefined,
    private count: number,
    private readonly notify: () => void,
  ) {
    this.buffer = this.createBuffer();
    this.stamps = this.createStamps();
    this.demand = props.streamer.demand([key], (frame) => {
      const series = frame.get(key);
      if (series == null) return;
      const last = series.at(-1);
      if (last != null && last.length > 0)
        if (this.seedable === true) props.store.recordStreamed(key, last);
        else if (this.seedable == null) this.pending = last;
      this.push(series, true);
      this.notify();
    });
    this.demand.onStatusChange((_, s) => {
      // A reconnect resets the tries of the series still missing stamps.
      if (s.variant === "success")
        this.indexReads = this.indexReads.filter((r) => r.retryAt == null);
      this.notify();
    });
    this.demand.onDrop(() => this.reload());
    this.stopInvalidations = props.store.onInvalidate((ch, tr) => {
      if (ch == null || ch === key) this.changed(tr);
    });
    this.running = { abort: new AbortController(), mark: 0 };
    void this.load(this.running);
  }

  /**
   * Replaces the window and count, reading the newest samples again. An equal window
   * and count do nothing.
   */
  update(window: Window | undefined, count: number): void {
    if (count === this.count && deep.equal(window, this.window)) return;
    this.window = window;
    this.count = count;
    this.taken = false;
    this.clear();
    this.restart(0);
  }

  /**
   * @returns the newest samples, trimmed to the count and the window. The snapshot
   * stays the same object until its samples or status change. A stream that only grows
   * the newest series keeps the snapshot's series and moves its time.
   */
  value(): Snapshot {
    const snapshot = this.current();
    const { running } = this;
    if (
      running != null &&
      snapshot.status.variant !== "loading" &&
      (this.buffer.leadingBuffer != null || this.series.length > running.mark)
    )
      [this.shown, this.taken] = [true, true];
    return snapshot;
  }

  /**
   * @returns the Core's stamp of the held sample at alignment, read from the channel's
   * index or, for a channel with no index, from the write that carried it. It answers
   * for at least the newest count samples. The first call for a channel with an index
   * starts streaming the index, and a miss reads the index for the sample's series. The
   * latest notifies once the stamps arrive. Null until then, until the channel is
   * found, and when no sample it answers for has the alignment.
   */
  timeAt(alignment: bigint): TimeStamp | null {
    const { timeSource } = this;
    if (timeSource == null) return null;
    if (timeSource === "self") return this.sampleAt(alignment);
    if (timeSource === "stamps") return this.stamps?.at(alignment) ?? null;
    this.indexStamps ??= this.streamIndex(timeSource.index);
    const stamp = this.indexStamps.at(alignment);
    if (stamp == null) void this.readIndex(timeSource.index, alignment);
    return stamp;
  }

  /** Releases the channel's stream. */
  close(): void {
    this.indexDemand?.close();
    this.indexDemand = null;
    this.indexStamps = null;
    this.indexReads = [];
    this.timeSource = null;
    this.restart(null);
    this.stopInvalidations();
    this.demand.close();
    this.clear();
  }

  private current(): Snapshot {
    if (this.snapshot == null || this.props.now().valueOf() >= this.leavesAt())
      return this.rebuild();
    const status = this.status();
    if (this.unserved === 0 && status === this.snapshot.status) return this.snapshot;
    this.unserved = 0;
    this.snapshot = {
      series: this.snapshot.series,
      status,
      time: this.time(),
      appended: this.appended,
    };
    return this.snapshot;
  }

  private rebuild(): Snapshot {
    this.trim();
    this.unserved = 0;
    const series = this.held();
    const spans = series.map((s) => s.alignmentBounds);
    this.indexStamps?.retain(spans);
    this.indexReads = this.indexReads.filter((r) =>
      spans.some((s) => bounds.overlapsWith(s, r.span)),
    );
    this.snapshot = {
      series: new MultiSeries(series),
      status: this.status(),
      time: this.time(),
      appended: this.appended,
    };
    return this.snapshot;
  }

  // Returns when the window's start passes the front series' end or the leading
  // buffer's start, which trim then drops or closes.
  private leavesAt(): bigint {
    if (this.window?.variant !== "dynamic") return TimeStamp.MAX.valueOf();
    const span = BigInt(this.window.timeSpan);
    const first = this.series.at(0);
    const leading = this.buffer.leadingBuffer;
    let at = TimeStamp.MAX.valueOf();
    if (first != null && !first.timeRange.isZero) at = first.timeRange.end.valueOf();
    else if (leading != null) at = leading.timeRange.start.valueOf();
    return at + span;
  }

  // Reads the samples written after the newest one held, since frames may have dropped.
  // A drop while a load runs reads them once that load lands.
  private reload(): void {
    this.closeLeading();
    if (this.running != null) this.running.dropped = true;
    else this.restart(this.series.length, { after: this.time() });
    this.snapshot = null;
    this.notify();
  }

  // Drops the samples held and reads them again up to the newest, since a history
  // change can delete them. A change that ends before every sample held cannot reach
  // the newest count. A change while an earlier change's read waits keeps its bound.
  private changed(tr: TimeRange): void {
    const first = this.series.at(0) ?? this.buffer.leadingBuffer;
    if (
      this.total >= this.count &&
      first != null &&
      !first.timeRange.isZero &&
      !tr.end.after(first.timeRange.start)
    )
      return;
    const end = this.time() ?? this.running?.end;
    const repeated = this.taken && end != null;
    // The read of the samples after a drop still runs, once this read lands.
    const dropped = this.running?.dropped === true || this.running?.after != null;
    // Samples no consumer took count again when the read returns them.
    if (!repeated) this.appended -= this.total;
    this.clear();
    this.restart(0, { end, repeated, dropped });
    this.notify();
  }

  // Ends the running load, then starts one whose streamed series start at mark, unless
  // mark is null.
  private restart(
    mark: number | null,
    read: Pick<Load, "end" | "repeated" | "after" | "dropped"> = {},
  ): void {
    this.running?.abort.abort();
    this.notifier.notify();
    this.running =
      mark == null ? null : { abort: new AbortController(), mark, ...read };
    if (this.running != null) void this.load(this.running);
  }

  private clear(): void {
    this.buffer.close();
    this.buffer = this.createBuffer();
    this.series = [];
    this.closedTotal = 0;
    this.unserved = 0;
    this.shown = false;
    this.snapshot = null;
    if (this.stamps != null) this.stamps = this.createStamps();
  }

  // Each write holds at least one sample, so count records cover the newest count
  // samples.
  private createStamps(): Stamps {
    return new Stamps(Math.max(this.count, 1));
  }

  private findTimeSource(p: channel.Payload): void {
    if (this.timeSource != null) return;
    if (!p.isIndex && p.index === 0) {
      this.timeSource = "stamps";
      return;
    }
    this.timeSource = p.isIndex ? "self" : { index: p.index };
    this.stamps = null;
  }

  // Returns the timestamp sample at alignment, for an index channel.
  private sampleAt(alignment: bigint): TimeStamp | null {
    for (const s of this.held()) {
      const value = s.atAlignment(alignment);
      if (value != null) return new TimeStamp(value as bigint);
    }
    return null;
  }

  // Returns the closed series and the leading buffer, oldest first.
  private held(): Series[] {
    const leading = this.buffer.leadingBuffer;
    return leading == null ? [...this.series] : [...this.series, leading];
  }

  // Streams the index, keeping the index series written with this channel's samples.
  private streamIndex(index: channel.Key): IndexStamps {
    const stamps = new IndexStamps();
    this.indexDemand = this.props.streamer.demand([index], (frame) => {
      const data = frame.get(this.key);
      const written = frame.get(index);
      if (data == null || written == null) return;
      stamps.add(
        written,
        data.map((s) => s.alignmentBounds),
      );
      this.notify();
    });
    return stamps;
  }

  // Reads the index over the held series with the sample at alignment. The read pairs
  // by position, since a streamed series' alignments differ from its read's.
  private async readIndex(index: channel.Key, alignment: bigint): Promise<void> {
    const series = this.held().find((s) =>
      bounds.contains(s.alignmentBounds, alignment),
    );
    const stamps = this.indexStamps;
    if (series == null || stamps == null || this.indexRefused) return;
    const prev = this.indexReads.find((r) => bounds.contains(r.span, alignment));
    const running = prev != null && prev.retryAt == null;
    const waiting = prev?.retryAt?.after(this.props.now()) ?? false;
    if (running || waiting || (prev?.tries ?? 0) >= INDEX_TRIES) return;
    const tr =
      series === this.buffer.leadingBuffer
        ? this.buffer.dataTimeRange
        : series.timeRange;
    if (tr == null || tr.isZero) return;
    // The leading buffer can grow while the read waits.
    const { length } = series;
    const read: IndexRead = { span: series.alignmentBounds, tries: prev?.tries ?? 0 };
    this.indexReads = [...this.indexReads.filter((r) => r !== prev), read];
    try {
      const times = await this.props.readLatest(index, length, tr);
      if (this.indexStamps !== stamps) return;
      if (times.length === length) {
        stamps.add([realign(times, read.span.lower, tr)], [read.span]);
        this.indexReads = this.indexReads.filter((r) => r !== read);
        return this.notify();
      }
      read.tries++;
    } catch (e) {
      this.indexRefused = isDefinitive(e) || UnsupportedError.matches(e);
      read.tries++;
    }
    read.retryAt = this.props.now().add(INDEX_RETRY);
  }

  private createBuffer(): Dynamic {
    const size = Math.min(Math.max(this.count, MIN_BUFFER), MAX_BUFFER);
    // A value keeps every series it served, so a closed buffer stays the same object.
    return new Dynamic({
      dynamicBufferSize: size,
      now: this.props.now,
      compactionDisabled: true,
    });
  }

  private get total(): number {
    return this.closedTotal + (this.buffer.leadingBuffer?.length ?? 0);
  }

  // Reads the newest stored samples after the stream acks the channel, so the stream
  // delivers every sample the read misses. A virtual channel has none, so it starts
  // from the newest series the stream carried for another request, and streams.
  private async load(run: Load): Promise<void> {
    const { signal } = run.abort;
    const { retrieveChannels, streamer, readLatest, now } = this.props;
    const brk = createBreaker(this.notifier, this.props.breaker);
    while (this.running === run)
      try {
        const [payload] = await retrieveChannels([this.key]);
        if (payload == null) throw new NotFoundError(`channel ${this.key} not found`);
        if (this.running === run) this.findTimeSource(payload);
        let read: Series[] = [];
        let seed: Series | null = null;
        if (!payload.virtual) this.dedupe = "alignment";
        else this.dedupe = calculated(payload) ? "time" : null;
        this.seedable = this.dedupe == null;
        if (this.seedable && this.pending != null)
          this.props.store.recordStreamed(this.key, this.pending);
        this.pending = null;
        if (this.dedupe != null) {
          await streamer.waitAcked([this.key], signal);
          if (this.running !== run) return;
          let tr =
            this.window == null ? TimeRange.MAX : resolveWindow(this.window, now());
          if (run.end != null)
            tr = new TimeRange(tr.start, TimeStamp.min(tr.end, run.end));
          if (run.after != null)
            tr = new TimeRange(
              TimeStamp.min(TimeStamp.max(tr.start, run.after), tr.end),
              tr.end,
            );
          read = (await readLatest(this.key, this.count, tr)).series;
        } else if (this.total === 0) {
          seed = this.props.store.newestStreamed(this.key);
          if (seed != null) read = [seed];
        }
        if (this.running !== run) return;
        this.land(read);
        if (seed != null) this.stamps?.record(seed);
        this.failure = null;
        return this.notify();
      } catch (e) {
        if (this.running !== run) return;
        this.snapshot = null;
        // The Core stores no samples for a calculation over virtual channels.
        if (UnsupportedError.matches(e)) {
          this.failure = status.create(NO_HISTORY);
          this.land([]);
          return this.notify();
        }
        this.failure = status.fromException(e, READ_FAILED);
        this.notify();
        if (isDefinitive(e)) return this.land([]);
        if (!(await brk.wait())) return;
      }
  }

  // Puts the read before the series streamed while it ran, keeping each sample once.
  // The streamed samples count again as they land after the read.
  private land(read: Series[]): void {
    this.taken = true;
    const dropped = this.running?.dropped ?? false;
    if (this.shown) {
      [this.running, this.shown] = [null, false];
      this.push(read, false);
      if (dropped) this.reload();
      return;
    }
    const repeated = this.running?.repeated ?? false;
    this.closeLeading();
    const streamed = this.series.splice(this.running?.mark ?? 0);
    streamed.forEach((s) => {
      this.closedTotal -= s.length;
      this.appended -= s.length;
    });
    this.running = null;
    const { appended } = this;
    this.push(read, false);
    if (repeated) this.appended = appended;
    // Frames may have dropped between the read and the streamed series, so the next
    // read starts after the newest sample of this read, not after a streamed one.
    if (dropped) {
      this.closeLeading();
      this.restart(this.series.length, { after: this.time() });
    }
    this.push(streamed, false);
    this.unserved = this.total;
    this.snapshot = null;
  }

  // Appends the series the latest accepts. A live write records its stamp once, before
  // the buffer joins it to others.
  private push(series: Series[], live: boolean): void {
    for (const s of series) {
      const fresh = this.fresh(s);
      if (fresh == null) continue;
      // Series of two data types cannot share a snapshot.
      const first = this.series.at(0) ?? this.buffer.leadingBuffer;
      if (first != null && !first.dataType.equals(fresh.dataType)) this.clear();
      if (live) this.stamps?.record(fresh);
      this.unserved += fresh.length;
      this.appended += fresh.length;
      this.append(fresh);
    }
    this.trim();
  }

  // Grows the leading buffer with a series that continues the newest one, and closes a
  // series that does not as it is.
  private append(s: Series): void {
    const newest = this.newest();
    if (
      newest == null ||
      s.timeRange.isZero ||
      s.alignment !== newest.alignment + BigInt(newest.length)
    ) {
      this.closeLeading();
      return this.closeSeries(s);
    }
    const { flushed, allocated } = this.buffer.write(new MultiSeries([s]));
    flushed.series.forEach((f) => this.closeSeries(f));
    if (flushed.series.length > 0 || allocated.series.length > 0) this.snapshot = null;
  }

  private closeLeading(): void {
    const flushed = this.buffer.flush();
    if (flushed != null) this.closeSeries(flushed);
  }

  private closeSeries(s: Series): void {
    this.series.push(s);
    this.closedTotal += s.length;
    this.snapshot = null;
  }

  private newest(): Series | undefined {
    return this.buffer.leadingBuffer ?? this.series.at(-1);
  }

  // Returns the samples of s newer than the newest sample held, or null if it has none.
  // Under alignment dedupe, alignments decide in the newest sample's domain. Otherwise,
  // the ends of the time ranges decide, and a series with no time range stays. A series
  // that starts at or after the end of a static window never enters.
  private fresh(s: Series): Series | null {
    if (s.length === 0) return null;
    const timed = !s.timeRange.isZero;
    if (
      timed &&
      this.window?.variant === "static" &&
      !s.timeRange.start.before(resolveWindow(this.window, this.props.now()).end)
    )
      return null;
    const newest = this.newest();
    if (newest == null || this.dedupe == null) return s;
    if (
      this.dedupe === "alignment" &&
      domainOf(s.alignment) === domainOf(newest.alignment)
    ) {
      const overlap = newest.alignment + BigInt(newest.length) - s.alignment;
      if (overlap >= BigInt(s.length)) return null;
      return overlap > 0n ? s.sub(Number(overlap)) : s;
    }
    const end = this.time();
    if (!timed || end == null) return s;
    return s.timeRange.end.after(end) ? s : null;
  }

  // Closes the leading buffer once the window starts after its first sample, and drops
  // whole series from the front that end before the window or that the rest of the
  // samples replace under the count. A series no value served stays until the samples
  // pass the cap.
  private trim(): void {
    if (this.window != null) {
      const { start } = resolveWindow(this.window, this.props.now());
      const leading = this.buffer.leadingBuffer;
      if (leading != null && !leading.timeRange.start.after(start)) this.closeLeading();
      while (this.series.length > 0) {
        const { timeRange } = this.series[0];
        if (timeRange.isZero || timeRange.end.after(start)) break;
        this.drop();
      }
    }
    const cap = Math.max(this.count, UNSERVED_CAP);
    while (this.series.length > 0) {
      const { length } = this.series[0];
      const { total } = this;
      if (total - length < this.count) break;
      if (total - this.unserved < length && total <= cap) break;
      this.drop();
    }
  }

  private drop(): void {
    const { length } = this.series.shift()!;
    if (this.running != null) this.running.mark = Math.max(0, this.running.mark - 1);
    this.closedTotal -= length;
    this.unserved = Math.min(this.unserved, this.total);
    this.snapshot = null;
  }

  // Returns the end of the newest sample's span, or undefined without a time range.
  private time(): TimeStamp | undefined {
    const newest = this.newest();
    if (newest == null) return undefined;
    if (newest === this.buffer.leadingBuffer) return this.buffer.dataTimeRange?.end;
    return newest.timeRange.isZero ? undefined : newest.timeRange.end;
  }

  private status(): status.Status {
    const worst = status.worst(this.demand.status(this.key), this.failure)!;
    if (worst.variant !== "success") return worst;
    return this.running != null ? this.statuses.loading : this.statuses.success;
  }
}
