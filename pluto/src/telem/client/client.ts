// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type alamos } from "@synnaxlabs/alamos";
import {
  type Channel,
  type channel,
  QueryError,
  type framer,
  type Synnax,
  UnexpectedError,
} from "@synnaxlabs/client";
import {
  compare,
  type Destructor,
  type Series,
  TimeRange,
  type SeriesDigest,
  TimeSpan,
  type AsyncDestructor,
  debounce,
} from "@synnaxlabs/x";
import { Mutex } from "async-mutex";
import { nanoid } from "nanoid/non-secure";

import { cache } from "@/telem/client/cache";

export const CACHE_BUFFER_SIZE = 10000;

export type StreamHandler = (data: Record<channel.Key, ReadResponse>) => void;

/**
 * A client that can be used to retrieve a channel from the Synnax cluster
 * by its key.
 */
export interface ChannelClient {
  /**
   * Retrieves a channel from the Synnax cluster by its key.
   *
   * @param key - The key of the channel to retrieve.
   * @returns the channel with the given key.
   * @throws QueryError if the channel does not exist.
   */
  retrieveChannel: (key: channel.Key) => Promise<Channel>;
}

/** A client that can be used to read telemetry from the Synnax cluster. */
export interface ReadClient {
  /**
   * Reads telemetry from the given channels for the given time range.
   *
   * @param tr  - The time range to read from.
   * @param keys - The keys of the channels to read from.
   * @returns a record with a read response for each channel in keys, regardless of
   * whether or not data was found for the given chnannel. NOTE: Responses are not
   * guaranteed to have the same topology i.e each response may contain a different
   * number of Series with different lengths. It's up to the caller to use the
   * 'alignment' field of the Series to normalize the data shape if necessary.
   */
  read: (
    tr: TimeRange,
    keys: channel.Keys,
  ) => Promise<Record<channel.Key, ReadResponse>>;
}

/** A client that can be used to stream telemetry from the Synnax cluster. */
export interface StreamClient {
  stream: (handler: StreamHandler, keys: channel.Keys) => Promise<AsyncDestructor>;
}

/**
 * Client provides an interface for performing basic telemetry operations
 * against a Synnax cluster. This interface is a simplification of the Synnax
 * client to make it easy to stub out for testing.
 */
export interface Client extends ChannelClient, ReadClient, StreamClient {
  key: string;
  /** Close closes the client, releasing all resources from the cache. */
  close: () => void;
}

/**
 * Proxy is a Client implementation that proxies all operations to another Client,
 * allowing the underlying Client to be swapped out at runtime. If no Client is
 * set, all operations will throw an error.
 */
export class Proxy implements Client {
  key: string = nanoid();
  _client: Client | null = null;

  swap(client: Client | null): void {
    this.key = nanoid();
    this._client?.close();
    this._client = client;
  }

  /** Implements ChannelClient. */
  async retrieveChannel(key: channel.Key): Promise<Channel> {
    return await this.client.retrieveChannel(key);
  }

  /** Implements ReadClient. */
  async read(
    tr: TimeRange,
    channels: channel.Keys,
  ): Promise<Record<channel.Key, ReadResponse>> {
    return await this.client.read(tr, channels);
  }

  /** Stream implements StreamClient. */
  async stream(handler: StreamHandler, keys: channel.Keys): Promise<AsyncDestructor> {
    return await this.client.stream(handler, keys);
  }

  /** Close implements CLient. */
  close(): void {
    this.client.close();
  }

  private get client(): Client {
    if (this._client == null) throw new QueryError("No cluster has been connected");
    return this._client;
  }
}

interface ListenerEntry {
  valid: boolean;
  keys: channel.Keys;
}

/**
 * Core wraps a Synnax client to implement the pluto telemetry Client interface,
 * adding a transparent caching layer.
 */
export class Core implements Client {
  readonly key: string = nanoid();
  private readonly core: Synnax;
  private readonly ins: alamos.Instrumentation;
  private readonly cache = new Map<channel.Key, cache.Cache>();
  private readonly listeners = new Map<StreamHandler, ListenerEntry>();
  private streamerRunLoop: Promise<void> | null = null;
  private readonly streamerMu: Mutex = new Mutex();
  private streamer: framer.Streamer | null = null;
  private readonly channelRetriever: AsyncBatchRetriever;

  constructor(wrap: Synnax, ins: alamos.Instrumentation) {
    this.core = wrap;
    this.ins = ins;
    this.channelRetriever = new AsyncBatchRetriever(
      async (keys) => await this.core.channels.retrieve(keys),
    );
  }

  async populateMissingCacheEntries(keys: channel.Keys): Promise<void> {
    const toFetch: channel.Keys = [];
    for (const key of keys) {
      if (this.cache.has(key)) continue;
      toFetch.push(key);
    }
    if (toFetch.length === 0) return;
    const channels = await this.channelRetriever.retrieve(toFetch);
    for (const c of channels) {
      const ins = this.ins.child(`cache-${c.name}-${c.key}`);
      const cache_ = new cache.Cache(CACHE_BUFFER_SIZE, c, ins);
      if (!this.cache.has(c.key)) this.cache.set(c.key, cache_);
    }
  }

  /** Implements ChannelClient. */
  async retrieveChannel(key: channel.Key): Promise<Channel> {
    return await this.core.channels.retrieve(key);
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
    const toFetch = new Map<string, [TimeRange, channel.Keys]>();
    const releasers: Destructor[] = [];
    const responses: Record<channel.Key, ReadResponse> = {};

    await this.populateMissingCacheEntries(channels);

    try {
      for (const key of channels) {
        // Read from cache.
        const cache = this.getCache(key);
        const { series, gaps, done } = await cache.dirtyReadForStaticWrite(tr);
        releasers.push(done);
        // In this case we have all the data we need and don't need to execute a fetch
        // for this channel.
        if (gaps.length === 0) responses[key] = new ReadResponse(cache.channel, series);

        // For each gap in the data, add it in the fetch map.
        gaps.forEach((gap) => {
          const exists = toFetch.get(gap.toString());
          if (exists == null) toFetch.set(gap.toString(), [gap, [key]]);
          else toFetch.set(gap.toString(), [gap, [...exists[1], key]]);
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
        const frame = await this.core.telem.read(range, keys);
        for (const key of keys) {
          const cache = this.getCache(key);
          const data = frame.get(key);
          if (data.length > 0) cache.writeStatic(data);
        }
      }
    } catch (e) {
      this.ins.L.error("read failed", { tr: tr.toPrettyString(), channels, error: e });
      throw e;
    } finally {
      releasers.forEach((r) => r());
    }

    // Re-read from cache so we get correct ordering.
    for (const key of channels) {
      const cache = this.getCache(key);
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

  /** Implements StreamClient. */
  async stream(handler: StreamHandler, keys: channel.Keys): Promise<AsyncDestructor> {
    await this.populateMissingCacheEntries(keys);
    return await this.streamerMu.runExclusive(async () => {
      this.ins.L.debug("adding stream handler", { keys });
      this.listeners.set(handler, { valid: true, keys });
      const dynamicBuffs: Record<channel.Key, ReadResponse> = {};
      for (const key of keys) {
        const c = this.getCache(key);
        dynamicBuffs[key] = new ReadResponse(
          c.channel,
          c.dynamic.buffer != null ? [c.dynamic.buffer] : [],
        );
      }
      handler(dynamicBuffs);
      await this.updateStreamer();
      return async () => await this.removeStreamHandler(handler);
    });
  }

  /** Implements Client. */
  close(): void {
    this.ins.L.info("closing client", { key: this.key });
    this.cache.clear();
    this.streamer?.close();
  }

  private async removeStreamHandler(handler: StreamHandler): Promise<void> {
    await this.streamerMu.runExclusive(() => {
      const entry = this.listeners.get(handler);
      if (entry == null) return;
      entry.valid = false;
    });
    setTimeout(() => {
      void this.streamerMu.runExclusive(async () => {
        this.ins.L.debug("removing stream handler");
        if (this.listeners.delete(handler)) return await this.updateStreamer();
        this.ins.L.warn("attempted to remove non-existent stream handler");
      });
    }, 5000);
  }

  private getCache(key: channel.Key): cache.Cache {
    const c = this.cache.get(key);
    if (c != null) return c;
    throw new UnexpectedError(`cache entry for ${key} not found`);
  }

  private async updateStreamer(): Promise<void> {
    // Assemble the set of keys we need to stream.
    const keys = new Set<channel.Key>();
    this.listeners.forEach((v) => v.keys.forEach((k) => keys.add(k)));

    // If we have no keys to stream, close the streamer to save network chatter.
    if (keys.size === 0) {
      this.ins.L.info("no keys to stream, closing streamer");
      this.streamer?.close();
      if (this.streamerRunLoop != null) await this.streamerRunLoop;
      this.streamer = null;
      this.ins.L.info("streamer closed successfully");
      return;
    }

    const arrKeys = Array.from(keys);
    if (compare.primitiveArrays(arrKeys, this.streamer?.keys ?? []) === compare.EQUAL) {
      this.ins.L.debug("streamer keys unchanged", { keys: arrKeys });
      return;
    }

    // Update or create the streamer.
    if (this.streamer == null) {
      this.ins.L.info("creating new streamer", { keys: arrKeys });
      this.streamer = await this.core.telem.newStreamer(arrKeys);
      this.streamerRunLoop = this.runStreamer(this.streamer);
    }

    this.ins.L.debug("updating streamer", { prev: this.streamer.keys, next: arrKeys });

    await this.streamer.update(arrKeys);
  }

  private async runStreamer(streamer: framer.Streamer): Promise<void> {
    for await (const frame of streamer) {
      const changed: ReadResponse[] = [];
      for (const k of frame.keys) {
        const series = frame.get(k);
        const cache = this.getCache(k);
        const out = cache.writeDynamic(series);
        changed.push(new ReadResponse(cache.channel, out));
      }
      this.listeners.forEach((entry, handler) => {
        const notify = changed.filter((r) => entry.keys.includes(r.channel.key));
        if (notify.length === 0) return;
        const d = Object.fromEntries(notify.map((r) => [r.channel.key, r]));
        if (entry.valid) handler(d);
      });
    }
  }
}

export interface ReadResponseDigest {
  channel: channel.Key;
  timeRange: string;
  series: SeriesDigest[];
}

export const responseDigests = (responses: ReadResponse[]): ReadResponseDigest[] =>
  responses.map((r) => r.digest);

export class ReadResponse {
  channel: Channel;
  data: Series[];

  constructor(channel: Channel, data: Series[]) {
    this.channel = channel;
    this.data = data;
  }

  get timeRange(): TimeRange {
    if (this.data.length === 0) return TimeRange.ZERO;
    const first = this.data[0].timeRange;
    const last = this.data[this.data.length - 1].timeRange;
    return new TimeRange(first.start, last.end);
  }

  get digest(): ReadResponseDigest {
    return {
      channel: this.channel.key,
      timeRange: this.timeRange.toPrettyString(),
      series: this.data.map((s) => s.digest),
    };
  }
}

interface PromiseFns<T> {
  resolve: (value: T) => void;
  reject: (reason?: any) => void;
}

// no interval
class AsyncBatchRetriever {
  private readonly mu = new Mutex();
  private readonly requests = new Map<channel.Keys, PromiseFns<channel.Channel[]>>();
  private readonly fn: (batch: channel.Keys) => Promise<channel.Channel[]>;
  private readonly debouncedRun: () => void;

  constructor(fn: (batch: channel.Key[]) => Promise<channel.Channel[]>) {
    this.fn = fn;
    this.debouncedRun = debounce(() => {
      void this.run();
    }, 10);
  }

  async retrieve(t: channel.Key[]): Promise<channel.Channel[]> {
    // eslint-disable-next-line @typescript-eslint/promise-function-async
    const a = new Promise<channel.Channel[]>((resolve, reject) => {
      void this.mu.runExclusive(() => {
        this.requests.set(t, { resolve, reject });
        this.debouncedRun();
      });
    });
    return await a;
  }

  async run(): Promise<void> {
    await this.mu.runExclusive(async () => {
      const keys = Array.from(this.requests.keys()).flat();
      try {
        const channels = await this.fn(keys);
        this.requests.forEach((fns) => {
          fns.resolve(channels.filter((c) => keys.includes(c.key)));
        });
      } catch (e) {
        this.requests.forEach((fns) => fns.reject(e));
      }
      this.requests.clear();
    });
  }
}
