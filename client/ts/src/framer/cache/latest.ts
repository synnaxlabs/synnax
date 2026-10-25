// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  breaker,
  MultiSeries,
  resolveWindow,
  type Series,
  sync,
  TimeRange,
  TimeSpan,
  type TimeStamp,
  type Window,
} from "@synnaxlabs/x";

import { channel } from "@/channel";
import { NotFoundError } from "@/errors";
import { type ChannelRetriever } from "@/framer/adapter";
import { isDefinitive } from "@/framer/cache/joiner";
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

const LOADING: status.Crude = { variant: "loading", message: "Loading channel data" };
const SUCCESS: status.Crude = { variant: "success", message: "Loaded channel data" };

const DOMAIN_SHIFT = 32n;

/**
 * Serves the newest samples of one channel: a read of its newest stored samples, then
 * every sample the stream delivers after them. Once loaded, the samples only grow at
 * their end and lose whole series from their front.
 */
export class Latest {
  private series: Series[] = [];
  // held holds streamed series until the read lands.
  private held: Series[] | null = [];
  private readonly demand: Demand;
  private failure: status.Status | null = null;
  private loading = true;
  private generation = 0;
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

  /** Replaces the window and count, reading the newest samples again. */
  update(window: Window | undefined, count: number): void {
    this.window = window;
    this.count = count;
    this.generation++;
    this.notifier.notify();
    this.series = [];
    this.held = [];
    this.loading = true;
    void this.load();
  }

  /** @returns the newest samples, trimmed to the count and the window. */
  value(): Snapshot {
    this.trim();
    const last = this.series.at(-1);
    return {
      // The stream appends to the held list, so the snapshot takes a copy.
      series: new MultiSeries([...this.series]),
      status: this.status(),
      time: last == null || last.timeRange.isZero ? undefined : last.timeRange.end,
    };
  }

  /** Releases the channel's stream. */
  close(): void {
    this.generation++;
    this.notifier.notify();
    this.demand.close();
    this.series = [];
    this.held = null;
  }

  // Reads the newest stored samples after the stream acks the channel, so the stream
  // delivers every sample the read misses. A virtual channel has none, so it only
  // streams.
  private async load(): Promise<void> {
    const generation = this.generation;
    const { retrieveChannels, streamer, readLatest, now } = this.props;
    const brk = new breaker.Breaker({
      baseInterval: TimeSpan.seconds(1),
      maxInterval: TimeSpan.seconds(30),
      maxRetries: Infinity,
      scale: 2,
      jitter: 0.25,
      sleepFn: async (span) => await this.notifier.wait(span),
      ...this.props.breaker,
    });
    while (generation === this.generation)
      try {
        const [payload] = await retrieveChannels([this.key]);
        if (payload == null) throw new NotFoundError(`channel ${this.key} not found`);
        let read: Series[] = [];
        if (!payload.virtual || channel.isCalculated(payload)) {
          await streamer.waitAcked([this.key]);
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
        this.failure = status.fromException(e, "Failed to read channel data");
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
    this.series = read.filter((s) => s.length > 0);
    const newest = this.series.at(-1);
    if (newest == null) return this.push(held);
    const end = newest.alignment + BigInt(newest.length);
    const domain = newest.alignment >> DOMAIN_SHIFT;
    this.push(
      held.flatMap((s) => {
        if (s.alignment >> DOMAIN_SHIFT === domain) {
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
      if (this.series.length > 0 && !this.series[0].dataType.equals(s.dataType))
        this.series = [];
      this.series.push(s);
    }
    this.trim();
  }

  // Drops whole series from the front while the rest hold the count, and series that
  // end before the window.
  private trim(): void {
    let total = this.series.reduce((sum, s) => sum + s.length, 0);
    let drop = 0;
    while (drop < this.series.length && total - this.series[drop].length >= this.count)
      total -= this.series[drop++].length;
    if (this.window != null) {
      const { start } = resolveWindow(this.window, this.props.now());
      while (
        drop < this.series.length &&
        !this.series[drop].timeRange.isZero &&
        this.series[drop].timeRange.end.beforeEq(start)
      )
        drop++;
    }
    if (drop > 0) this.series = this.series.slice(drop);
  }

  private status(): status.Status {
    const stream = this.demand.status(this.key);
    if (this.failure != null && status.moreSevere(this.failure.variant, stream.variant))
      return this.failure;
    if (stream.variant !== "success") return stream;
    return this.loading ? this.statuses.loading : this.statuses.success;
  }
}
