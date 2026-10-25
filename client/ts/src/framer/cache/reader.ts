// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Unreachable } from "@synnaxlabs/freighter";
import {
  debounce,
  MultiSeries,
  type Reduction,
  type Series,
  sync,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";

import { type channel } from "@/channel";
import { UnexpectedError } from "@/errors";
import { type Cache } from "@/framer/cache/cache";
import { type Frame } from "@/framer/frame";

/** Reads a telemetry frame from the Core, reduced when a reduction is given. */
export interface RemoteReader {
  (tr: TimeRange, keys: channel.Key[], reduction?: Reduction): Promise<Frame>;
}

export const DEFAULT_BATCH_DEBOUNCE = TimeSpan.milliseconds(50);

// Samples stamped this close to a fetch may commit after it answers, so a fetch never
// marks them as covered.
const COMMIT_MARGIN = TimeSpan.seconds(1);

interface ReadRequest {
  channel: channel.Key;
  gaps: TimeRange[];
}

// Tracks one read's outstanding fetches, so the read settles when its last fetch lands.
// The first failure among them wins.
interface Tracker {
  entry: debounce.Entry<ReadRequest>;
  remaining: number;
  failure?: unknown;
}

interface BatchFetch {
  gap: TimeRange;
  channels: Set<channel.Key>;
  trackers: Set<Tracker>;
}

export interface ReaderProps {
  /** Function used to read remote data from the server. */
  readRemote: RemoteReader;
  /** Will read from and populate the given cache with fetched data. */
  cache: Cache;
  /**
   * Used to batch read requests to the server to minimize traffic. The window is not
   * extended by later reads, so it is also the maximum wait before a batch fires.
   * @default TimeSpan.milliseconds(50)
   */
  batchDebounce?: TimeSpan;
  /**
   * Reads whose gap time ranges are within this threshold merge into one request.
   * @default TimeSpan.milliseconds(5)
   */
  overlapThreshold?: TimeSpan;
  /**
   * Deadline for a single remote fetch. A fetch that does not settle within it rejects
   * the reads it serves with an Unreachable error.
   * @default TimeSpan.seconds(30)
   */
  fetchTimeout?: TimeSpan;
  /**
   * The current time on the Core's clock, which stamps every sample.
   * @default TimeStamp.now
   */
  now?: () => TimeStamp;
}

/**
 * Reads historical telemetry, serving from the cache and batch-filling gaps from the
 * server. A fetch failure rejects only the reads whose gaps were in the failed fetch.
 */
export class Reader {
  private readonly props: Required<Omit<ReaderProps, "batchDebounce">>;
  private readonly batcher: debounce.Batcher<ReadRequest>;
  private closed = false;

  constructor(props: ReaderProps) {
    const {
      readRemote,
      cache,
      batchDebounce = DEFAULT_BATCH_DEBOUNCE,
      overlapThreshold = TimeSpan.milliseconds(5),
      fetchTimeout = TimeSpan.seconds(30),
      now = () => TimeStamp.now(),
    } = props;
    this.props = { readRemote, cache, overlapThreshold, fetchTimeout, now };
    this.batcher = new debounce.Batcher({
      interval: batchDebounce,
      exec: async (entries) => await this.batchRead(entries),
    });
  }

  /**
   * Reads the given time range for the given channel.
   * @throws {UnexpectedError} if the reader is closed while the read is pending.
   * @throws {Unreachable} if a fetch serving the read exceeds the fetch timeout.
   */
  async read(tr: TimeRange, channel: channel.Key): Promise<MultiSeries> {
    const { cache } = this.props;
    const unary = cache.get(channel);
    const { series, gaps } = unary.read(tr);
    if (gaps.length === 0) return series;
    if (this.closed) throw new UnexpectedError("telemetry reader is closed");
    await this.batcher.enqueue({ channel, gaps });
    return unary.read(tr).series;
  }

  private async batchRead(entries: Array<debounce.Entry<ReadRequest>>): Promise<void> {
    const { cache, overlapThreshold } = this.props;
    const batched: BatchFetch[] = [];
    const trackers = new Map<debounce.Entry<ReadRequest>, Tracker>();
    entries.forEach((entry) => {
      const unary = cache.get(entry.req.channel);
      entry.req.gaps.forEach((rawGap) => {
        // An earlier batch or a streaming write may have filled part of the gap while
        // this entry sat in the debounce window.
        unary.read(rawGap).gaps.forEach((gap) => {
          let tracker = trackers.get(entry);
          if (tracker == null) {
            tracker = { entry, remaining: 0 };
            trackers.set(entry, tracker);
          }
          const g = batched.find((r) => r.gap.equals(gap, overlapThreshold));
          if (g == null) {
            batched.push({
              gap,
              channels: new Set([entry.req.channel]),
              trackers: new Set([tracker]),
            });
            tracker.remaining++;
          } else {
            g.channels.add(entry.req.channel);
            g.gap = TimeRange.max(g.gap, gap);
            if (!g.trackers.has(tracker)) {
              g.trackers.add(tracker);
              tracker.remaining++;
            }
          }
        });
      });
    });
    entries.forEach((entry) => {
      if (!trackers.has(entry)) entry.resolve();
    });
    await Promise.all(batched.map(async (b) => await this.fetchGap(b)));
  }

  // Never throws: every outcome, including a deadline expiry, settles through the
  // trackers.
  private async fetchGap({ gap, channels, trackers }: BatchFetch): Promise<void> {
    const { cache, now, readRemote, fetchTimeout } = this.props;
    let failure: unknown;
    const answered = new TimeRange(
      gap.start,
      TimeStamp.min(gap.end, now().sub(COMMIT_MARGIN)),
    );
    try {
      const frame = await sync.withTimeout(
        readRemote(gap, Array.from(channels)),
        fetchTimeout,
        () =>
          new Unreachable({
            message: `gap read for ${gap.toString()} timed out after ${fetchTimeout.toString()}`,
          }),
      );
      if (!this.closed) {
        // Group series by key in one pass; per-key get() scans the whole frame.
        const grouped = new Map<channel.Key, Series[]>();
        frame.forEach((k, s) => {
          const key = k as channel.Key;
          const existing = grouped.get(key);
          if (existing == null) grouped.set(key, [s]);
          else existing.push(s);
        });
        channels.forEach((key) => {
          const unary = cache.get(key);
          unary.writeStatic(new MultiSeries(grouped.get(key) ?? []));
          unary.markFetched(answered);
        });
      }
    } catch (err) {
      failure = err;
    }
    trackers.forEach((tracker) => {
      if (failure != null) tracker.failure ??= failure;
      tracker.remaining--;
      if (tracker.remaining > 0) return;
      if (tracker.failure == null) tracker.entry.resolve();
      else tracker.entry.reject(tracker.failure);
    });
  }

  /**
   * Closes the reader, rejecting every pending read with an UnexpectedError. In-flight
   * fetch results are discarded when they arrive.
   */
  async close(): Promise<void> {
    this.closed = true;
    this.batcher.close(new UnexpectedError("telemetry reader is closed"));
  }
}
