// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos } from "@synnaxlabs/alamos";
import { type channel, type framer } from "@synnaxlabs/client";
import { debounce, MultiSeries, sync, TimeRange, TimeSpan } from "@synnaxlabs/x";

import { NATIVE_FIDELITY } from "@/telem/client/cache/unary";
import { type Cache } from "@/telem/client/cache/cache";
import { type ReadClient } from "@/telem/client/client";

/**
 * A function that reads a telemetry frame from the Synnax cluster at the
 * requested fidelity. A fidelity of NATIVE_FIDELITY (1n) means raw data; higher
 * values indicate the caller is willing to receive decimated data with each
 * output sample representing `fidelity` native samples. Implementations may
 * clamp the actual served fidelity and report it back via the
 * `alignmentMultiple` field on the returned Series.
 */
export interface ReadRemoteFunc {
  (tr: TimeRange, keys: channel.Key[], fidelity: bigint): Promise<framer.Frame>;
}

interface ReadRequest {
  channel: channel.Key;
  fidelity: bigint;
  gaps: TimeRange[];
  resolve: () => void;
  reject: (reason?: unknown) => void;
}

interface BatchFetch {
  gap: TimeRange;
  fidelity: bigint;
  channels: Set<channel.Key>;
}

export interface ReaderArgs {
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
export class Reader implements ReadClient {
  /** Parsed and validated arguments passed to the constructor. */
  private readonly args: Required<ReaderArgs>;
  /**
   * A function that debounced reads to the cluster by the debounce interval
   * specified in args.batchDebounce.
   */
  private readonly debouncedRead: () => void;
  /** A mutex for serializing access to requests. */
  private readonly mu = sync.newMutex({
    requests: new Set<ReadRequest>(),
    closed: false,
  });

  constructor(args: ReaderArgs) {
    this.args = {
      ...args,
      instrumentation: args.instrumentation ?? alamos.NOOP,
      batchDebounce: args.batchDebounce ?? TimeSpan.milliseconds(50),
      overlapThreshold: args.overlapThreshold ?? TimeSpan.milliseconds(5),
    };
    const deb = this.args.batchDebounce.milliseconds;
    this.debouncedRead = debounce(() => void this.batchRead(), deb);
  }

  /** Implements ReadClient. */
  async read(
    tr: TimeRange,
    channel: channel.Key,
    fidelity: bigint = NATIVE_FIDELITY,
  ): Promise<MultiSeries> {
    const { cache } = this.args;
    await cache.populateMissing([channel]);
    const unary = cache.get(channel);
    const normalized = fidelity === 0n ? NATIVE_FIDELITY : fidelity;
    const { series, gaps } = unary.dirtyReadAtFidelity(tr, normalized);
    if (gaps.length === 0) return series;
    const { mu } = this;
    await new Promise<void>((resolve, reject) => {
      void mu.runExclusive(async () => {
        if (mu.closed) return;
        mu.requests.add({ channel, fidelity: normalized, gaps, resolve, reject });
      });
      this.debouncedRead();
    });
    return unary.dirtyReadAtFidelity(tr, normalized).series;
  }

  private async batchRead(): Promise<void> {
    const { readRemote, cache, overlapThreshold } = this.args;
    const { mu } = this;
    await mu.runExclusive(async () => {
      const finish = (err?: unknown) =>
        mu.requests.forEach(({ resolve, reject }) =>
          err == null ? resolve() : reject(err),
        );
      try {
        if (mu.closed) return finish();
        const batched: BatchFetch[] = [];
        // Group requests by (gap, fidelity). Requests at different fidelities
        // cannot share a batch because they produce series with different
        // alignmentMultiple values and must land in different cache tiers.
        mu.requests.forEach(({ channel, fidelity, gaps }) =>
          gaps.forEach((gap) => {
            const g = batched.find(
              (r) => r.fidelity === fidelity && r.gap.equals(gap, overlapThreshold),
            );
            if (g == null)
              batched.push({ gap, fidelity, channels: new Set([channel]) });
            else {
              g.channels.add(channel);
              g.gap = TimeRange.max(g.gap, gap);
            }
          }),
        );
        await Promise.all(
          batched.map(async ({ gap, fidelity, channels }) => {
            const frame = await readRemote(gap, Array.from(channels), fidelity);
            channels.forEach((key) => {
              const series = frame.get(key);
              // Key each written series by the fidelity the server actually
              // served (the Series.alignmentMultiple field), not by what we
              // requested; these may differ if the server clamped.
              series.series.forEach((s) =>
                cache
                  .get(key)
                  .writeStaticAtFidelity(
                    new MultiSeries([s]),
                    s.alignmentMultiple,
                  ),
              );
            });
          }),
        );
        finish();
      } catch (err) {
        finish(err);
      } finally {
        mu.requests.clear();
      }
    });
  }

  async close(): Promise<void> {
    await this.mu.runExclusive(async () => {
      this.mu.closed = true;
    });
  }
}
