// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos } from "@synnaxlabs/alamos";
import { type channel, type framer } from "@synnaxlabs/client";
import { debounce } from "@synnaxlabs/x/debounce";
import { MultiSeries, TimeRange, TimeSpan } from "@synnaxlabs/x/telem";
import { Mutex } from "async-mutex";

import { type Cache } from "@/telem/client/cache/cache";

/** A function that reads a telemetry frame from the Synnax cluster. */
export interface ReadRemoteFunc {
  (tr: TimeRange, keys: channel.Keys): Promise<framer.Frame>;
}

interface ReadRequest {
  channel: channel.Key;
  timeRanges: TimeRange[];
  resolve: () => void;
}

interface BatchFetch {
  timeRange: TimeRange;
  channels: Set<channel.Key>;
}

export interface ReaderProps {
  /**
   * Function used to read remote data from the server. Used instead of
   * passing in a Synnax client directly to make testing easier.
   */
  readRemote: ReadRemoteFunc;
  /** Will read from and populate the given cache with fetched data. */
  cache: Cache;
  /**
   * Used to batch read request to the server to minimize traffic. Larger
   * values mean slower response times but less traffic. Smaller values mean faster
   * response times but more traffic.
   * @default TimeSpan.milliseconds(50)
   */
  batchDebounce?: TimeSpan;
  /**
   * A threshold for overlap between time ranges in order for them to be batched into
   * a single request to the server. For example, a read on channel one for time range
   * [1ms, 5ms] and a read for channel two for time range [4ms, 6ms] would be batched
   * under an overlap threshold of 2ms into a single request for time range [1ms, 6ms]
   * for the channels [one, two].
   * @default TimeSpan.milliseconds(5)
   */
  overlapThreshold?: TimeSpan;
  /** Used for logging, tracing, etc. */
  instrumentation?: alamos.Instrumentation;
}

/**
 * Reader is used for reading telemetry data from the Synnax cluster. See the documentation
 * in the README for more.
 */
export class Reader {
  private readonly props: Required<ReaderProps>;
  private readonly debouncedRead: () => void;
  private closed: boolean = false;
  private readonly mu: Mutex = new Mutex();
  private readonly requests: Set<ReadRequest> = new Set();

  constructor(props: ReaderProps) {
    this.props = props as Required<ReaderProps>;
    this.props.instrumentation ??= alamos.NOOP;
    this.props.batchDebounce ??= TimeSpan.milliseconds(50);
    this.props.overlapThreshold ??= TimeSpan.milliseconds(5);
    this.debouncedRead = debounce(
      () => void this.batchRead(),
      this.props.batchDebounce.milliseconds,
    );
  }

  /** Implements ReadClient. */
  async read(tr: TimeRange, channel: channel.Key): Promise<MultiSeries> {
    const { cache } = this.props;
    if (!(await cache.populateMissing([channel])) || this.closed)
      return new MultiSeries([]);
    const unary = cache.get(channel);
    const { series, gaps } = unary.read(tr);
    if (gaps.length === 0) return series;
    const { mu, requests } = this;
    await new Promise<void>((resolve) => {
      void mu.runExclusive(async () => {
        requests.add({ channel, timeRanges: gaps, resolve });
        this.debouncedRead();
      });
    });
    return unary.read(tr).series;
  }

  private async batchRead(): Promise<void> {
    const { readRemote, cache, overlapThreshold } = this.props;
    const { mu, requests } = this;
    await mu.runExclusive(async () => {
      const resolve = () => requests.forEach(({ resolve }) => resolve());
      try {
        if (this.closed) return resolve();
        const batched: BatchFetch[] = [];
        requests.forEach(({ channel, timeRanges }) =>
          timeRanges.forEach((tr) => {
            const g = batched.find((r) => r.timeRange.equals(tr, overlapThreshold));
            if (g == null)
              batched.push({ timeRange: tr, channels: new Set([channel]) });
            else {
              g.channels.add(channel);
              g.timeRange = TimeRange.max(g.timeRange, tr);
            }
          }),
        );
        await Promise.all(
          batched.map(async ({ timeRange, channels }) => {
            const frame = await readRemote(timeRange, Array.from(channels));
            if (this.closed) return;
            channels.forEach((key) => cache.get(key).writeStatic(frame.get(key)));
          }),
        );
      } finally {
        resolve();
        requests.clear();
      }
    });
  }

  async close(): Promise<void> {
    this.closed = true;
    await this.mu.waitForUnlock();
  }
}
