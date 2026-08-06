// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos } from "@synnaxlabs/alamos";
import { debounce, type MultiSeries, sync, TimeRange, TimeSpan } from "@synnaxlabs/x";

import { type channel } from "@/channel";
import { UnexpectedError } from "@/errors";
import { type framer } from "@/framer";
import { type Cache } from "@/telem/cache/cache";

/** A function that reads a telemetry frame from the Synnax cluster. */
export interface ReadRemoteFunc {
  (tr: TimeRange, keys: channel.Key[]): Promise<framer.Frame>;
}

interface ReadRequest {
  channel: channel.Key;
  gaps: TimeRange[];
}

interface BatchFetch {
  gap: TimeRange;
  channels: Set<channel.Key>;
  entries: Set<debounce.Entry<ReadRequest>>;
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
   * Used to batch read requests to the server to minimize traffic. Larger
   * values mean slower response times but less traffic. Smaller values mean faster
   * response times but more traffic. The window is not extended by later reads, so
   * it is also the maximum wait before a batch fires.
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
 * Reads historical telemetry, serving from the cache and batch-filling gaps from
 * the server. A batch failure rejects only the reads whose gaps were in the failed
 * fetch.
 */
export class Reader {
  private readonly props: Required<ReaderProps>;
  private readonly batcher: debounce.Batcher<ReadRequest>;
  // Serializes batch execution against close, so close waits for the batch in
  // flight.
  private readonly mu = sync.newMutex({ closed: false });

  constructor(props: ReaderProps) {
    const {
      readRemote,
      cache,
      instrumentation = alamos.NOOP,
      batchDebounce = TimeSpan.milliseconds(50),
      overlapThreshold = TimeSpan.milliseconds(5),
    } = props;
    this.props = {
      readRemote,
      cache,
      instrumentation,
      batchDebounce,
      overlapThreshold,
    };
    this.batcher = new debounce.Batcher({
      interval: batchDebounce,
      exec: async (entries) =>
        await this.mu.runExclusive(async () => {
          if (this.mu.closed) throw new UnexpectedError("telemetry reader is closed");
          await this.batchRead(entries);
        }),
    });
  }

  /**
   * Reads the given time range for the given channel.
   * @throws {UnexpectedError} if the reader is closed while the read is pending.
   */
  async read(tr: TimeRange, channel: channel.Key): Promise<MultiSeries> {
    const { cache } = this.props;
    const unary = cache.get(channel);
    const { series, gaps } = unary.read(tr);
    if (gaps.length === 0) return series;
    if (this.mu.closed) throw new UnexpectedError("telemetry reader is closed");
    await this.batcher.enqueue({ channel, gaps });
    return unary.read(tr).series;
  }

  private async batchRead(entries: Array<debounce.Entry<ReadRequest>>): Promise<void> {
    const { readRemote, cache, overlapThreshold } = this.props;
    const batched: BatchFetch[] = [];
    entries.forEach((entry) =>
      entry.req.gaps.forEach((gap) => {
        const g = batched.find((r) => r.gap.equals(gap, overlapThreshold));
        if (g == null)
          batched.push({
            gap,
            channels: new Set([entry.req.channel]),
            entries: new Set([entry]),
          });
        else {
          g.channels.add(entry.req.channel);
          g.gap = TimeRange.max(g.gap, gap);
          g.entries.add(entry);
        }
      }),
    );
    const failures = new Map<debounce.Entry<ReadRequest>, unknown>();
    await Promise.all(
      batched.map(async ({ gap, channels, entries: served }) => {
        try {
          const frame = await readRemote(gap, Array.from(channels));
          channels.forEach((key) => cache.get(key).writeStatic(frame.get(key)));
        } catch (err) {
          // Fail only the reads served by this batch; sibling batches settle on
          // their own results.
          served.forEach((entry) => {
            if (!failures.has(entry)) failures.set(entry, err);
          });
        }
      }),
    );
    entries.forEach((entry) => {
      const err = failures.get(entry);
      if (err == null) entry.resolve();
      else entry.reject(err);
    });
  }

  async close(): Promise<void> {
    this.batcher.close(new UnexpectedError("telemetry reader is closed"));
    await this.mu.runExclusive(async () => {
      this.mu.closed = true;
    });
  }
}
