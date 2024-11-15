// Copyright 2024 Synnax Labs, Inc.
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
import { type TimeRange, TimeSpan } from "@synnaxlabs/x/telem";
import { Mutex } from "async-mutex";

import { type Cache } from "@/telem/client/cache/cache";
import { ReadResponse, responseDigests } from "@/telem/client/types";

interface PromiseFns<T> {
  resolve: (value: T | PromiseLike<T>) => void;
  reject: (reason?: any) => void;
}

type ToFetch = [TimeRange, Set<channel.Key>];

export type ReadRemoteFunc = (
  tr: TimeRange,
  keys: channel.Keys,
) => Promise<framer.Frame>;

interface ReaderProps {
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
  /** Used for logging, tracing, etc. */
  instrumentation?: alamos.Instrumentation;
}

interface ReaderMu {
  mu: Mutex;
  requests: Map<ToFetch, PromiseFns<void>>;
}

/**
 * Reader is used for reading telemetry data from the Synnax cluster. See the documentation
 * in the README for more.
 */
export class Reader {
  private readonly props: Required<ReaderProps>;

  private readonly debouncedRead: () => void;

  // Guards concurrency between the batchRead function and the read function.
  private readonly guarded: ReaderMu = {
    mu: new Mutex(),
    requests: new Map(),
  };

  constructor(props: ReaderProps) {
    this.props = {
      instrumentation: alamos.NOOP,
      batchDebounce: TimeSpan.milliseconds(50),
      ...props,
    };
    this.debouncedRead = debounce(() => {
      void this.batchRead();
    }, this.props.batchDebounce.milliseconds);
  }

  /** Implements ReadClient. */
  async read(
    tr: TimeRange,
    channels: channel.Keys,
  ): Promise<Record<channel.Key, ReadResponse>> {
    const {
      instrumentation: { L },
      cache,
    } = this.props;
    L.debug("starting read", { tr: tr.toPrettyString(), channels });
    const start = performance.now();
    // Instead of executing a fetch for each channel, we'll batch related time ranges
    // together to get the most out of each fetch.
    const toFetch = new Map<string, ToFetch>();
    const responses: Record<channel.Key, ReadResponse> = {};

    // If this is the first time we're fetching data for these channels, we'll need to
    // populate new entries in the cache.
    await cache.populateMissing(channels);

    try {
      for (const key of channels) {
        // Try fetching from the cache.
        const unary = cache.get(key);
        const { series, gaps } = unary.read(tr);
        // In this case we have all the data we need and don't need to execute a fetch
        // for this channel.
        if (gaps.length === 0) responses[key] = new ReadResponse(unary.channel, series);

        // For each gap in the data, add it in the fetch map.
        gaps.forEach((gap) => {
          const exists = toFetch.get(gap.toString());
          if (exists == null) toFetch.set(gap.toString(), [gap, new Set([key])]);
          else toFetch.set(gap.toString(), [gap, new Set([...exists[1], key])]);
        });
      }

      if (toFetch.size === 0) {
        L.debug("read satisfied by cache", () => ({
          tr: tr.toPrettyString(),
          channels,
          responses: responseDigests(Object.values(responses)),
          time: TimeSpan.milliseconds(performance.now() - start).toString(),
        }));
        return responses;
      }

      L.debug("read cache miss", () => ({
        tr: tr.toPrettyString(),
        channels,
        toFetch: Array.from(toFetch.values()).map(([r, k]) => ({
          timeRange: r.toPrettyString(),
          channels: k,
        })),
        responses: responseDigests(Object.values(responses)),
      }));

      // Fetch any missing gaps in the data using a batched read. This will automatically
      // populate the cache with the new data.
      const { mu, requests } = this.guarded;
      for (const [, [range, keys]] of toFetch)
        await new Promise<void>((resolve, reject) => {
          void mu.runExclusive(async () => {
            requests.set([range, keys], { resolve, reject });
            this.debouncedRead();
          });
        });
    } catch (e) {
      L.error("read failed", { tr: tr.toPrettyString(), channels, error: e });
      throw e;
    }

    // The cache has fetched all the data we need, so we just re-execute a cache read
    // to get the new data.
    for (const key of channels) {
      const unary = cache.get(key);
      const { series } = unary.read(tr);
      responses[key] = new ReadResponse(unary.channel, series);
    }

    L.debug("read satisfied by fetch", () => ({
      tr: tr.toPrettyString(),
      channels,
      responses: responseDigests(Object.values(responses)),
      time: TimeSpan.milliseconds(performance.now() - start).toString(),
    }));

    return responses;
  }

  private async batchRead(): Promise<void> {
    const {
      instrumentation: { L },
      readRemote,
      cache,
    } = this.props;
    const { mu, requests } = this.guarded;
    await mu.runExclusive(async () => {
      const compressedToFetch: ToFetch[] = [];
      try {
        requests.forEach((_, k) => {
          const [tr, keys] = k;
          const groupWith = compressedToFetch.find(
            ([r]) =>
              r.start.span(tr.start).lessThan(TimeSpan.milliseconds(5)) &&
              r.end.span(tr.end).lessThan(TimeSpan.milliseconds(5)),
          );
          if (groupWith == null) compressedToFetch.push([tr, keys]);
          else groupWith[1] = new Set([...groupWith[1], ...keys]);
        });
        L.debug("batch read", {
          toFetch: compressedToFetch.map(([r, k]) => ({
            timeRange: r.toPrettyString(),
            channels: k,
          })),
        });
        for (const [range, keys] of compressedToFetch) {
          const keysArray = Array.from(keys);
          const frame = await readRemote(range, keysArray);
          keysArray.forEach((key) => {
            const unary = cache.get(key);
            const data = frame.get(key);
            unary.writeStatic(data.series);
          });
        }
        requests.forEach((toFetch) => toFetch.resolve());
      } catch (e) {
        console.log(e);
        L.error("batch read failed", { error: JSON.stringify(e) }, true);
        requests.forEach((toFetch) => toFetch.reject(e));
      } finally {
        requests.clear();
      }
    });
  }
}
