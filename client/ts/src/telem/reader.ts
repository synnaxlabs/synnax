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
  resolve: () => void;
  reject: (reason?: unknown) => void;
}

interface BatchFetch {
  gap: TimeRange;
  channels: Set<channel.Key>;
  requests: Set<ReadRequest>;
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
 * Reads historical telemetry, serving from the cache and batch-filling gaps from
 * the server. A batch failure rejects only the reads whose gaps were in the failed
 * fetch.
 */
export class Reader {
  private readonly props: Required<ReaderProps>;
  private readonly debouncedRead: ReturnType<typeof debounce>;
  /** A mutex for serializing access to requests. */
  private readonly mu = sync.newMutex({
    requests: new Set<ReadRequest>(),
    closed: false,
  });

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
    this.debouncedRead = debounce(
      () => void this.batchRead(),
      this.props.batchDebounce,
    );
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
    const { mu } = this;
    await new Promise<void>((resolve, reject) => {
      void mu.runExclusive(async () => {
        if (mu.closed) return reject(new UnexpectedError("telemetry reader is closed"));
        mu.requests.add({ channel, gaps, resolve, reject });
        this.debouncedRead();
      });
    });
    return unary.read(tr).series;
  }

  private async batchRead(): Promise<void> {
    const { readRemote, cache, overlapThreshold } = this.props;
    const { mu } = this;
    await mu.runExclusive(async () => {
      if (mu.closed) return;
      const batched: BatchFetch[] = [];
      mu.requests.forEach((req) =>
        req.gaps.forEach((gap) => {
          const g = batched.find((r) => r.gap.equals(gap, overlapThreshold));
          if (g == null)
            batched.push({
              gap,
              channels: new Set([req.channel]),
              requests: new Set([req]),
            });
          else {
            g.channels.add(req.channel);
            g.gap = TimeRange.max(g.gap, gap);
            g.requests.add(req);
          }
        }),
      );
      const failures = new Map<ReadRequest, unknown>();
      await Promise.all(
        batched.map(async ({ gap, channels, requests }) => {
          try {
            const frame = await readRemote(gap, Array.from(channels));
            channels.forEach((key) => cache.get(key).writeStatic(frame.get(key)));
          } catch (err) {
            // Fail only the reads served by this batch; sibling batches settle on
            // their own results.
            requests.forEach((req) => {
              if (!failures.has(req)) failures.set(req, err);
            });
          }
        }),
      );
      mu.requests.forEach((req) => {
        const err = failures.get(req);
        if (err == null) req.resolve();
        else req.reject(err);
      });
      mu.requests.clear();
    });
  }

  async close(): Promise<void> {
    this.debouncedRead.cancel();
    await this.mu.runExclusive(async () => {
      this.mu.closed = true;
      const err = new UnexpectedError("telemetry reader is closed");
      this.mu.requests.forEach(({ reject }) => reject(err));
      this.mu.requests.clear();
    });
  }
}
