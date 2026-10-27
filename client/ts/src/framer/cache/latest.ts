// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type breaker,
  deep,
  MultiSeries,
  resolveWindow,
  type Series,
  sync,
  TimeRange,
  TimeStamp,
  type Window,
} from "@synnaxlabs/x";

import { channel } from "@/channel";
import { NotFoundError, UnsupportedError } from "@/errors";
import { type ChannelRetriever } from "@/framer/adapter";
import { domainOf } from "@/framer/cache/block";
import { Dynamic } from "@/framer/cache/dynamic";
import { LOADING, NO_HISTORY, READ_FAILED, SUCCESS } from "@/framer/cache/status";
import { type Store } from "@/framer/cache/store";
import { type Demand, type MultiplexedStreamer } from "@/framer/cache/streamer";
import { createBreaker, isDefinitive } from "@/framer/retry";
import { type Snapshot } from "@/framer/subscription";
import { status } from "@/status";

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
  // dropped is true when the stream dropped while the load ran.
  dropped?: boolean;
}

/**
 * Serves the newest samples of one channel: a read of its newest stored samples, and
 * every newer sample the stream delivers. Streamed samples serve while the read waits
 * or fails. When the stream drops, it reads the newest samples again once the Core acks
 * the channel. The samples grow at their end and lose whole series from their front,
 * unless their data type changes. A series leaves only once a value served it.
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
  // landed is true once a read of the request lands.
  private landed = false;
  private running: Load | null;
  private readonly demand: Demand;
  private readonly stopInvalidations: () => void;
  // dedupe is how fresh finds repeated samples: by Core alignment for a stored channel,
  // and by time for a calculated one, whose alignments depend on how its inputs split
  // into series. A channel that only streams repeats no sample.
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

  constructor(
    private readonly props: LatestProps,
    readonly key: channel.Key,
    private window: Window | undefined,
    private count: number,
    private readonly notify: () => void,
  ) {
    this.buffer = this.createBuffer();
    this.demand = props.streamer.demand([key], (frame) => {
      const series = frame.get(key);
      if (series == null) return;
      this.push(series);
      this.notify();
    });
    this.demand.onStatusChange(() => this.notify());
    this.demand.onDrop(() => this.reload());
    this.stopInvalidations = props.store.onInvalidate((ch, tr) => {
      if (ch === key) this.changed(tr);
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
    this.landed = false;
    this.clear();
    this.restart(0);
  }

  /**
   * @returns the newest samples, trimmed to the count and the window. The snapshot
   * stays the same object until its samples or status change. A stream that only grows
   * the newest series keeps the snapshot's series and moves its time.
   */
  value(): Snapshot {
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

  /** Releases the channel's stream. */
  close(): void {
    this.restart(null);
    this.stopInvalidations();
    this.demand.close();
    this.clear();
  }

  private rebuild(): Snapshot {
    this.trim();
    this.unserved = 0;
    const series = [...this.series];
    const leading = this.buffer.leadingBuffer;
    if (leading != null) series.push(leading);
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
    const repeated = this.landed && end != null;
    // Until a read lands, the samples dropped count again when the read returns them.
    if (!repeated) this.appended -= this.total;
    this.clear();
    this.restart(0, { end, repeated });
    this.notify();
  }

  // Ends the running load, then starts one whose streamed series start at mark, unless
  // mark is null.
  private restart(
    mark: number | null,
    read: Pick<Load, "end" | "repeated" | "after"> = {},
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
    this.snapshot = null;
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
  // delivers every sample the read misses. A virtual channel has none, so it only
  // streams.
  private async load(run: Load): Promise<void> {
    const { signal } = run.abort;
    const { retrieveChannels, streamer, readLatest, now } = this.props;
    const brk = createBreaker(this.notifier, this.props.breaker);
    while (this.running === run)
      try {
        const [payload] = await retrieveChannels([this.key]);
        if (payload == null) throw new NotFoundError(`channel ${this.key} not found`);
        let read: Series[] = [];
        if (!payload.virtual) this.dedupe = "alignment";
        else this.dedupe = channel.isCalculated(payload) ? "time" : null;
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
        }
        if (this.running !== run) return;
        this.land(read);
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
    const repeated = this.running?.repeated ?? false;
    const dropped = this.running?.dropped ?? false;
    this.landed = true;
    this.closeLeading();
    const streamed = this.series.splice(this.running?.mark ?? 0);
    streamed.forEach((s) => {
      this.closedTotal -= s.length;
      this.appended -= s.length;
    });
    this.running = null;
    const { appended } = this;
    this.push(read);
    if (repeated) this.appended = appended;
    this.push(streamed);
    this.unserved = this.total;
    this.snapshot = null;
    if (dropped) this.reload();
  }

  private push(series: Series[]): void {
    for (const s of series) {
      const fresh = this.fresh(s);
      if (fresh == null) continue;
      // Series of two data types cannot share a snapshot.
      const first = this.series.at(0) ?? this.buffer.leadingBuffer;
      if (first != null && !first.dataType.equals(fresh.dataType)) this.clear();
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
