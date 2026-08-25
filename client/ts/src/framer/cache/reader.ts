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
  errors,
  MultiSeries,
  type Series,
  TimeRange,
  TimeSpan,
} from "@synnaxlabs/x";

import { type channel } from "@/channel";
import { UnexpectedError } from "@/errors";
import { type Cache } from "@/framer/cache/cache";
import { type Frame } from "@/framer/frame";

/** A function that reads a telemetry frame from the Synnax cluster. */
export interface RemoteReader {
  (tr: TimeRange, keys: channel.Key[]): Promise<Frame>;
}

interface ReadRequest {
  channel: channel.Key;
  gaps: TimeRange[];
}

// Tracks one read's outstanding fetches, so the read settles when its last fetch
// lands. The first failure among them wins.
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
   * Deadline for a single remote fetch. A fetch that does not settle within it
   * rejects the reads it serves with an Unreachable error.
   * @default TimeSpan.seconds(30)
   */
  fetchTimeout?: TimeSpan;
}

/**
 * Reads historical telemetry, serving from the cache and batch-filling gaps from the
 * server. A fetch failure rejects only the reads whose gaps were in the failed fetch.
 */
export class Reader {
  private readonly props: Required<Omit<ReaderProps, "batchDebounce">>;
  private readonly batcher: debounce.Batcher<ReadRequest>;
  private closed = false;
  // Reads whose fetches are in flight, so close can reject them without waiting
  // on the fetches themselves.
  private readonly inFlight = new Set<Tracker>();

  constructor(props: ReaderProps) {
    const {
      readRemote,
      cache,
      batchDebounce = TimeSpan.milliseconds(50),
      overlapThreshold = TimeSpan.milliseconds(5),
      fetchTimeout = TimeSpan.seconds(30),
    } = props;
    this.props = { readRemote, cache, overlapThreshold, fetchTimeout };
    this.batcher = new debounce.Batcher({
      interval: batchDebounce,
      exec: async (entries) => {
        if (this.closed) throw new UnexpectedError("telemetry reader is closed");
        await this.batchRead(entries);
      },
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
        // An earlier batch or a streaming write may have filled part of the gap
        // while this entry sat in the debounce window.
        unary.read(rawGap).gaps.forEach((gap) => {
          let tracker = trackers.get(entry);
          if (tracker == null) {
            tracker = { entry, remaining: 0 };
            trackers.set(entry, tracker);
            this.inFlight.add(tracker);
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

  // Never throws: every outcome, including a deadline expiry and a post-close
  // arrival, settles through the trackers.
  private async fetchGap({ gap, channels, trackers }: BatchFetch): Promise<void> {
    const { cache } = this.props;
    let failure: unknown;
    try {
      const frame = await this.fetchWithDeadline(gap, Array.from(channels));
      if (!this.closed) {
        // Group series by key in one pass; per-key get() scans the whole frame.
        const grouped = new Map<channel.Key, Series[]>();
        frame.forEach((k, s) => {
          const key = k as channel.Key;
          const existing = grouped.get(key);
          if (existing == null) grouped.set(key, [s]);
          else existing.push(s);
        });
        channels.forEach((key) =>
          cache.get(key).writeStatic(new MultiSeries(grouped.get(key) ?? [])),
        );
      }
    } catch (err) {
      failure = err;
    }
    trackers.forEach((tracker) => {
      // close() already settled it.
      if (!this.inFlight.has(tracker)) return;
      if (failure != null) tracker.failure ??= failure;
      tracker.remaining--;
      if (tracker.remaining > 0) return;
      this.inFlight.delete(tracker);
      if (tracker.failure == null) tracker.entry.resolve();
      else tracker.entry.reject(tracker.failure);
    });
  }

  private async fetchWithDeadline(
    gap: TimeRange,
    channels: channel.Key[],
  ): Promise<Frame> {
    const { readRemote, fetchTimeout } = this.props;
    let timer: ReturnType<typeof setTimeout> | undefined;
    const deadline = new Promise<never>((_, reject) => {
      timer = setTimeout(() => {
        const timeout = fetchTimeout.toString();
        const message = `gap read for ${gap.toString()} timed out after ${timeout}`;
        reject(new Unreachable({ message }));
      }, fetchTimeout.milliseconds);
    });
    const fetched = readRemote(gap, channels);
    try {
      return await Promise.race([fetched, deadline]);
    } catch (err) {
      // The read already failed for its callers; a late settle of the losing
      // fetch must not surface as an unhandled rejection.
      fetched.catch(() => {});
      throw errors.fromUnknown(err);
    } finally {
      clearTimeout(timer);
    }
  }

  /**
   * Closes the reader, rejecting every pending read with an UnexpectedError.
   * In-flight fetch results are discarded when they arrive.
   */
  async close(): Promise<void> {
    this.closed = true;
    const err = new UnexpectedError("telemetry reader is closed");
    this.batcher.close(err);
    this.inFlight.forEach(({ entry }) => entry.reject(err));
    this.inFlight.clear();
  }
}
