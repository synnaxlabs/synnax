// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type Channel,
  type channel,
  QueryError,
  type framer,
  type Synnax,
} from "@synnaxlabs/client";
import { Compare, type Destructor, type Series, TimeRange } from "@synnaxlabs/x";
import { Mutex } from "async-mutex";
import { nanoid } from "nanoid";

import { cache } from "@/telem/client/cache";

export const CACHE_BUFFER_SIZE = 10000;

export type StreamHandler = (data: Record<channel.Key, ReadResponse>) => void;

export interface ChannelClient {
  retrieveChannel: (key: channel.Key) => Promise<Channel>;
}

export interface StaticClient {
  read: (
    tr: TimeRange,
    keys: channel.Keys,
  ) => Promise<Record<channel.Key, ReadResponse>>;
}

export interface StreamClient {
  stream: (handler: StreamHandler, keys: channel.Keys) => Promise<Destructor>;
}

export interface Client extends ChannelClient, StaticClient, StreamClient {
  close: () => void;
}

export class Proxy implements Client {
  _client: Client | null;

  constructor() {
    this._client = null;
  }

  async retrieveChannel(key: channel.Key): Promise<Channel> {
    return await this.client.retrieveChannel(key);
  }

  swap(client: Client | null): void {
    this._client?.close();
    this._client = client;
  }

  private get client(): Client {
    if (this._client == null) throw new QueryError("No cluster has been connected");
    return this._client;
  }

  async read(
    tr: TimeRange,
    channels: channel.Keys,
  ): Promise<Record<channel.Key, ReadResponse>> {
    return await this.client.read(tr, channels);
  }

  async stream(handler: StreamHandler, keys: channel.Keys): Promise<Destructor> {
    return await this.client.stream(handler, keys);
  }

  close(): void {
    this.client.close();
  }
}

export class Core implements Client {
  core: Synnax;
  private _streamer: framer.Streamer | null;
  private readonly cache: Map<channel.Key, cache.Cache>;
  private readonly listeners: Map<StreamHandler, channel.Keys>;
  key: string;
  mutex: Mutex;

  constructor(wrap: Synnax) {
    this.key = nanoid();
    this.core = wrap;
    this._streamer = null;
    this.cache = new Map();
    this.listeners = new Map();
    this.mutex = new Mutex();
  }

  async retrieveChannel(key: channel.Key): Promise<Channel> {
    return await this.core.channels.retrieve(key);
  }

  /**
   * Reads telemetry from the given channels for the given time range.
   *
   * @param tr
   * @param channels
   * @returns a record with the read response for each channel. Each channel is guaranteed
   * to have a response, regardless of whether or not data was found. Responses without
   * data will have no Seriess. Responses are NOT guaranteed to have the same topology
   * i.e. the same number of Seriess where each Series has the same length.
   * It's up to the caller to normalize the data shape if necessary.
   */

  async read(
    tr: TimeRange,
    channels: channel.Keys,
  ): Promise<Record<channel.Key, ReadResponse>> {
    // Instead of executing a fetch for each channel, we'll batch related time ranges
    // together to get the most out of each fetch.
    const toFetch = new Map<string, [TimeRange, channel.Keys]>();
    const responses: Record<channel.Key, ReadResponse> = {};

    for (const key of channels) {
      // Read from cache.
      const cache = await this.getCache(key);
      const [data, gaps] = cache.read(tr);
      // In this case we have all the data we need and don't need to execute a fetch
      // for this channel.
      if (gaps.length === 0) responses[key] = new ReadResponse(cache.channel, data);

      // For each gap in the data, add it in the fetch map.
      gaps.forEach((gap) => {
        const exists = toFetch.get(gap.toString());
        if (exists == null) toFetch.set(gap.toString(), [gap, [key]]);
        else toFetch.set(gap.toString(), [gap, [...exists[1], key]]);
      });
    }

    // Fetch any missing gaps in the data. Writing to the cache will automatically
    // order the data correctly.
    for (const [, [range, keys]] of toFetch) {
      const frame = await this.core.telem.read(range, keys);
      for (const key of keys) {
        const cache = await this.getCache(key);
        cache.writeStatic(range, frame.get(key));
      }
    }

    // Re-read from cache so we get correct ordering.
    for (const key of channels) {
      const cache = await this.getCache(key);
      const [data] = cache.read(tr);
      responses[key] = new ReadResponse(cache.channel, data);
    }

    return responses;
  }

  async stream(handler: StreamHandler, keys: channel.Keys): Promise<Destructor> {
    return await this.mutex.runExclusive(async () => {
      this.listeners.set(handler, keys);
      const dynamicBuffs: Record<channel.Key, ReadResponse> = {};
      for (const key of keys) {
        const c = await this.getCache(key);
        dynamicBuffs[key] = new ReadResponse(
          c.channel,
          c.dynamic.buffer != null ? [c.dynamic.buffer] : [],
        );
      }
      handler(dynamicBuffs);
      await this.updateStreamer();
      return () => this.removeStreamHandler(handler);
    });
  }

  private removeStreamHandler(handler: StreamHandler): void {
    this.listeners.delete(handler);
    void this.updateStreamer();
  }

  private async getCache(key: channel.Key): Promise<cache.Cache> {
    const c = this.cache.get(key);
    if (c != null) return c;
    const channel = await this.core.channels.retrieve(key);
    const cache_ = new cache.Cache(CACHE_BUFFER_SIZE, channel);
    this.cache.set(key, cache_);
    return cache_;
  }

  private async updateStreamer(): Promise<void> {
    // Assemble the set of keys we need to stream.
    const keys = new Set<channel.Key>();
    this.listeners.forEach((v) => v.forEach((k) => keys.add(k)));

    // If we have no keys to stream, close the streamer to save network chatter.
    if (keys.size === 0) {
      this._streamer?.close();
      this._streamer = null;
      return;
    }

    const arrKeys = Array.from(keys);
    if (Compare.primitiveArrays(arrKeys, this._streamer?.keys ?? []) === Compare.EQUAL)
      return;

    // Update or create the streamer.
    if (this._streamer == null) {
      this._streamer = await this.core.telem.newStreamer(arrKeys);
      void this.start(this._streamer);
    }

    await this._streamer.update(arrKeys);
  }

  private async start(streamer: framer.Streamer): Promise<void> {
    for await (const frame of streamer) {
      const changed: ReadResponse[] = [];
      for (const k of frame.keys) {
        const series = frame.get(k);
        const cache = await this.getCache(k);
        const out = cache.writeDynamic(series);
        changed.push(new ReadResponse(cache.channel, out));
      }
      this.listeners.forEach((keys, handler) => {
        const notify = changed.filter((r) => keys.includes(r.channel.key));
        if (notify.length === 0) return;
        const d = Object.fromEntries(notify.map((r) => [r.channel.key, r]));
        handler(d);
      });
    }
  }

  close(): void {
    this.cache.clear();
    this._streamer?.close();
  }
}

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
}
