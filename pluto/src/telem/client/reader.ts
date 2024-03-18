import { type alamos } from "@synnaxlabs/alamos";
import { type framer, type channel } from "@synnaxlabs/client";
import { TimeSpan, type Destructor, type TimeRange, debounce } from "@synnaxlabs/x";
import { Mutex } from "async-mutex";

import { type CacheManager } from "@/telem/client/cacheManager";
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

export class Reader {
  private readonly ins: alamos.Instrumentation;
  private readonly cache: CacheManager;
  private readonly readRemote: ReadRemoteFunc;

  private readonly mu = new Mutex();
  private readonly readRequests = new Map<ToFetch, PromiseFns<void>>();
  private readonly debouncedRead: () => void;

  constructor(
    cache: CacheManager,
    fetcher: ReadRemoteFunc,
    ins: alamos.Instrumentation,
  ) {
    this.cache = cache;
    this.readRemote = fetcher;
    this.ins = ins;
    this.debouncedRead = debounce(() => {
      void this.batchRead();
    }, 50);
  }

  private async batchRead(): Promise<void> {
    await this.mu.runExclusive(async () => {
      const compressedToFetch: ToFetch[] = [];

      try {
        this.readRequests.forEach((toFetch, k) => {
          const [tr, keys] = k;
          const groupWith = compressedToFetch.find(
            ([r]) =>
              r.start.span(tr.start).lessThan(TimeSpan.milliseconds(5)) &&
              r.end.span(tr.end).lessThan(TimeSpan.milliseconds(5)),
          );
          if (groupWith == null) compressedToFetch.push([tr, keys]);
          else {
            groupWith[1] = new Set([...groupWith[1], ...keys]);
          }
        });
        this.ins.L.debug("batch read", {
          toFetch: compressedToFetch.map(([r, k]) => ({
            timeRange: r.toPrettyString(),
            channels: k,
          })),
        });
        for (const [range, keys] of compressedToFetch) {
          const keysArray = Array.from(keys);
          const frame = await this.readRemote(range, keysArray);
          for (const key of keysArray) {
            const cache = this.cache.get(key);
            const data = frame.get(key);
            if (data.length > 0) {
              cache.writeStatic(data);
            }
          }
        }
        this.readRequests.forEach((toFetch) => toFetch.resolve());
      } catch (e) {
        this.ins.L.error("batch read failed", { error: e });
        this.readRequests.forEach((toFetch) => toFetch.reject(e));
      } finally {
        this.readRequests.clear();
      }
    });
  }

  /** Implements ReadClient. */
  async read(
    tr: TimeRange,
    channels: channel.Keys,
  ): Promise<Record<channel.Key, ReadResponse>> {
    this.ins.L.debug("starting read", { tr: tr.toPrettyString(), channels });
    const start = performance.now();
    // Instead of executing a fetch for each channel, we'll batch related time ranges
    // together to get the most out of each fetch.
    const toFetch = new Map<string, ToFetch>();
    const releasers: Destructor[] = [];
    const responses: Record<channel.Key, ReadResponse> = {};

    await this.cache.populateMissing(channels);

    try {
      for (const key of channels) {
        // Read from cache.
        const cache = this.cache.get(key);
        const { series, gaps } = cache.dirtyRead(tr);
        // In this case we have all the data we need and don't need to execute a fetch
        // for this channel.
        if (gaps.length === 0) responses[key] = new ReadResponse(cache.channel, series);

        // For each gap in the data, add it in the fetch map.
        gaps.forEach((gap) => {
          const exists = toFetch.get(gap.toString());
          if (exists == null) toFetch.set(gap.toString(), [gap, new Set([key])]);
          else toFetch.set(gap.toString(), [gap, new Set([...exists[1], key])]);
        });
      }

      if (toFetch.size === 0) {
        this.ins.L.debug("read satisfied by cache", {
          tr: tr.toPrettyString(),
          channels,
          responses: responseDigests(Object.values(responses)),
          time: TimeSpan.milliseconds(performance.now() - start).toString(),
        });
        return responses;
      }

      this.ins.L.debug("read cache miss", {
        tr: tr.toPrettyString(),
        channels,
        toFetch: Array.from(toFetch.values()).map(([r, k]) => ({
          timeRange: r.toPrettyString(),
          channels: k,
        })),
        responses: responseDigests(Object.values(responses)),
      });

      // Fetch any missing gaps in the data. Writing to the cache will automatically
      // order the data correctly.
      for (const [, [range, keys]] of toFetch) {
        await new Promise<void>((resolve, reject) => {
          void this.mu.runExclusive(async () => {
            this.readRequests.set([range, keys], { resolve, reject });
            this.debouncedRead();
          });
        });
      }
    } catch (e) {
      this.ins.L.error("read failed", { tr: tr.toPrettyString(), channels, error: e });
      throw e;
    } finally {
      releasers.forEach((r) => r());
    }

    // Re-read from cache so we get correct ordering.
    for (const key of channels) {
      const cache = this.cache.get(key);
      const { series } = cache.dirtyRead(tr);
      responses[key] = new ReadResponse(cache.channel, series);
    }

    this.ins.L.debug("read satisfied by fetch", {
      tr: tr.toPrettyString(),
      channels,
      responses: responseDigests(Object.values(responses)),
      time: TimeSpan.milliseconds(performance.now() - start).toString(),
    });

    return responses;
  }
}
