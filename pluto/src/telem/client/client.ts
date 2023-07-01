// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Channel,
  ChannelKey,
  ChannelKeys,
  QueryError,
  Streamer,
  Synnax,
} from "@synnaxlabs/client";
import { Compare, Series, TimeRange } from "@synnaxlabs/x";

import { ChannelCache } from "@/telem/client/cache";

export type StreamHandler = (data: Record<ChannelKey, ReadResponse> | null) => void;

export interface Client {
  readonly core: Synnax;
  read: (
    tr: TimeRange,
    channels: ChannelKeys
  ) => Promise<Record<ChannelKey, ReadResponse>>;
  setStreamHandler: (handler: StreamHandler, keys: ChannelKeys) => void;
  removeStreamHandler: (handler: StreamHandler) => void;
  close: () => void;
}

export class ClientProxy implements Client {
  private _client: Client | null;

  constructor() {
    this._client = null;
  }

  swap(client: Client): void {
    this._client?.close();
    this._client = client;
  }

  get core(): Synnax {
    return this.client.core;
  }

  private get client(): Client {
    if (this._client == null) throw new QueryError("Client is not initialized");
    return this._client;
  }

  async read(
    tr: TimeRange,
    channels: ChannelKeys
  ): Promise<Record<ChannelKey, ReadResponse>> {
    return await this.client.read(tr, channels);
  }

  setStreamHandler(handler: StreamHandler, keys: ChannelKeys): void {
    this.client.setStreamHandler(handler, keys);
  }

  removeStreamHandler(handler: StreamHandler): void {
    this.client.removeStreamHandler(handler);
  }

  close(): void {
    this.client.close();
  }
}

export class BaseClient implements Client {
  core: Synnax;
  private _streamer: Streamer | null;
  private readonly cache: Map<ChannelKey, ChannelCache>;
  private readonly listeners: Map<StreamHandler, ChannelKeys>;

  constructor(wrap: Synnax) {
    this.core = wrap;
    this._streamer = null;
    this.cache = new Map();
    this.listeners = new Map();
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
    channels: ChannelKeys
  ): Promise<Record<ChannelKey, ReadResponse>> {
    // Instead of executing a fetch for each channel, we'll batch related time ranges
    // together to get the most out of each fetch.
    const toFetch = new Map<string, [TimeRange, ChannelKeys]>();
    const responses: Record<ChannelKey, ReadResponse> = {};

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

  async setStreamHandler(handler: StreamHandler, keys: ChannelKeys): Promise<void> {
    this.listeners.set(handler, keys);
    const dynamicBuffs: Record<ChannelKey, ReadResponse> = {};
    for (const key of keys) {
      const c = await this.getCache(key);
      dynamicBuffs[key] = new ReadResponse(c.channel, [c.dynamic.buffer]);
    }
    handler(dynamicBuffs);
    void this.updateStreamer();
  }

  removeStreamHandler(handler: StreamHandler): void {
    this.listeners.delete(handler);
    void this.updateStreamer();
  }

  private async getCache(key: ChannelKey): Promise<ChannelCache> {
    const c = this.cache.get(key);
    if (c != null) return c;
    const channel = await this.core.channels.retrieve(key);
    const cache = new ChannelCache(10000, channel);
    this.cache.set(key, cache);
    return cache;
  }

  private async updateStreamer(): Promise<void> {
    // Assemble the set of keys we need to stream.
    const keys = new Set<ChannelKey>();
    this.listeners.forEach((v) => v.forEach((k) => keys.add(k)));

    // If we have no keys to stream, close the streamer to save network chatter.
    if (keys.size === 0) {
      this._streamer = null;
    }

    const arrKeys = Array.from(keys);
    if (Compare.primitiveArrays(arrKeys, this._streamer?.keys ?? []) === 0) return;

    // Update or create the streamer.
    if (this._streamer != null) return await this._streamer.update(arrKeys);
    this._streamer = await this.core.telem.newStreamer(arrKeys);

    void this.start(this._streamer);
  }

  private async start(streamer: Streamer): Promise<void> {
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
    this.core.close();
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
