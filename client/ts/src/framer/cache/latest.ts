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
  type TimeStamp,
  type Window,
} from "@synnaxlabs/x";

import { type channel } from "@/channel";
import { NotFoundError } from "@/errors";
import { type ChannelRetriever } from "@/framer/adapter";
import { domainOf } from "@/framer/cache/block";
import {
  isDefinitive,
  LOADING,
  newBreaker,
  READ_FAILED,
  SUCCESS,
} from "@/framer/cache/status";
import { calculated } from "@/framer/cache/store";
import { type Demand, type MultiplexedStreamer } from "@/framer/cache/streamer";
import { type Snapshot } from "@/framer/subscription";
import { status } from "@/status";

/** Reads the newest count stored samples of a channel in the time range. */
export interface LatestRead {
  (key: channel.Key, count: number, tr: TimeRange): Promise<MultiSeries>;
}

export interface LatestProps {
  streamer: MultiplexedStreamer;
  readLatest: LatestRead;
  retrieveChannels: ChannelRetriever;
  now: () => TimeStamp;
  /** Retry behavior for failed reads. */
  breaker?: breaker.Config;
}

/**
 * Serves the newest samples of one channel: a read of its newest stored samples, then
 * every sample the stream delivers after them. Once loaded, the samples only grow at
 * their end and lose whole series from their front.
 */
export class Latest {
  // series holds the samples from front on, and total counts them.
  private series: Series[] = [];
  private front = 0;
  private total = 0;
  // held holds streamed series until the read lands.
  private held: Series[] | null = [];
  private readonly demand: Demand;
  private failure: status.Status | null = null;
  private loading = true;
  private generation = 0;
  // abort ends the load's wait for the stream's ack.
  private abort = new AbortController();
  private readonly notifier = new sync.Notifier();
  private readonly statuses = {
    loading: status.create(LOADING),
    success: status.create(SUCCESS),
  };

  constructor(
    private readonly props: LatestProps,
    readonly key: channel.Key,
    private window: Window | undefined,
    private count: number,
    private readonly notify: () => void,
  ) {
    this.demand = props.streamer.demand([key], (frame) => {
      const series = frame.get(key);
      if (series == null) return;
      this.push(series);
      this.notify();
    });
    this.demand.onStatusChange(() => this.notify());
    void this.load();
  }

  /**
   * Replaces the window and count, reading the newest samples again. An equal window
   * and count do nothing.
   */
  update(window: Window | undefined, count: number): void {
    if (count === this.count && deep.equal(window, this.window)) return;
    this.window = window;
    this.count = count;
    this.stop();
    this.clear();
    this.held = [];
    this.loading = true;
    void this.load();
  }

  /** @returns the newest samples, trimmed to the count and the window. */
  value(): Snapshot {
    this.trim();
    const last = this.series.at(-1);
    return {
      series: new MultiSeries(this.series.slice(this.front)),
      status: this.status(),
      time: last == null || last.timeRange.isZero ? undefined : last.timeRange.end,
    };
  }

  /** Releases the channel's stream. */
  close(): void {
    this.stop();
    this.demand.close();
    this.clear();
    this.held = null;
  }

  // Ends the running load.
  private stop(): void {
    this.generation++;
    this.abort.abort();
    this.abort = new AbortController();
    this.notifier.notify();
  }

  private clear(): void {
    this.series = [];
    this.front = 0;
    this.total = 0;
  }

  // Reads the newest stored samples after the stream acks the channel, so the stream
  // delivers every sample the read misses. A virtual channel has none, so it only
  // streams.
  private async load(): Promise<void> {
    const generation = this.generation;
    const { signal } = this.abort;
    const { retrieveChannels, streamer, readLatest, now } = this.props;
    const brk = newBreaker(this.notifier, this.props.breaker);
    while (generation === this.generation)
      try {
        const [payload] = await retrieveChannels([this.key]);
        if (payload == null) throw new NotFoundError(`channel ${this.key} not found`);
        let read: Series[] = [];
        if (!payload.virtual || calculated(payload)) {
          await streamer.waitAcked([this.key], signal);
          if (generation !== this.generation) return;
          const tr =
            this.window == null ? TimeRange.MAX : resolveWindow(this.window, now());
          read = (await readLatest(this.key, this.count, tr)).series;
        }
        if (generation !== this.generation) return;
        this.land(read);
        this.failure = null;
        return this.notify();
      } catch (e) {
        if (generation !== this.generation) return;
        this.failure = status.fromException(e, READ_FAILED);
        this.notify();
        if (isDefinitive(e)) return this.land([]);
        if (!(await brk.wait())) return;
      }
  }

  // Joins the read to the series streamed while it ran. A streamed sample at or before
  // the newest read sample is in the read, so it leaves.
  private land(read: Series[]): void {
    const held = this.held ?? [];
    this.held = null;
    this.loading = false;
    this.clear();
    this.push(read);
    const newest = read.findLast((s) => s.length > 0);
    if (newest == null) return this.push(held);
    const end = newest.alignment + BigInt(newest.length);
    const domain = domainOf(newest.alignment);
    this.push(
      held.flatMap((s) => {
        if (domainOf(s.alignment) === domain) {
          const overlap = Number(end - s.alignment);
          if (overlap >= s.length) return [];
          return overlap > 0 ? [s.sub(overlap)] : [s];
        }
        return s.timeRange.end.beforeEq(newest.timeRange.end) ? [] : [s];
      }),
    );
  }

  private push(series: Series[]): void {
    if (this.held != null) return void this.held.push(...series);
    for (const s of series) {
      if (s.length === 0) continue;
      // Series of two data types cannot share a snapshot.
      const first = this.series.at(this.front);
      if (first != null && !first.dataType.equals(s.dataType)) this.clear();
      this.series.push(s);
      this.total += s.length;
    }
    this.trim();
  }

  // Drops whole series from the front while the rest hold the count, and series that
  // end before the window.
  private trim(): void {
    const { series } = this;
    while (
      this.front < series.length &&
      this.total - series[this.front].length >= this.count
    )
      this.total -= series[this.front++].length;
    if (this.window != null) {
      const { start } = resolveWindow(this.window, this.props.now());
      while (
        this.front < series.length &&
        !series[this.front].timeRange.isZero &&
        series[this.front].timeRange.end.beforeEq(start)
      )
        this.total -= series[this.front++].length;
    }
    if (this.front > series.length / 2) {
      this.series = series.slice(this.front);
      this.front = 0;
    }
  }

  private status(): status.Status {
    const stream = this.demand.status(this.key);
    if (this.failure != null && status.moreSevere(this.failure.variant, stream.variant))
      return this.failure;
    if (stream.variant !== "success") return stream;
    return this.loading ? this.statuses.loading : this.statuses.success;
  }
}
