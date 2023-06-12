// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Channel, ChannelKey, ChannelKeys, Streamer, Synnax } from "@synnaxlabs/client";
import { LazyArray, TimeRange } from "@synnaxlabs/x";

import { Cache } from "@/telem/cache";
import { convertArrays } from "@/telem/convertArrays";

export type StreamHandler = (data: Record<ChannelKey, ReadResponse> | null) => void;

export class Client {
  readonly core: Synnax | null;
  private _streamer: Streamer | null;
  private readonly cache: Map<ChannelKey, Cache>;
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
   * data will have no LazyArrays. Responses are NOT guaranteed to have the same topology
   * i.e. the same number of LazyArrays where each LazyArray has the same length.
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
        cache.writeStatic(range, convertArrays(frame.get(key)));
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

  setStreamhandler(handler: StreamHandler, keys: ChannelKeys): void {
    this.listeners.set(handler, keys);
    void this.updateStreamer();
  }

  removeStreamHandler(handler: StreamHandler): void {
    this.listeners.delete(handler);
    void this.updateStreamer();
  }

  private async getCache(key: ChannelKey): Promise<Cache> {
    const c = this.cache.get(key);
    if (c != null) return c;
    const channel = await this.core.channels.retrieve(key);
    const cache = new Cache(1000, channel);
    this.cache.set(key, cache);
    return cache;
  }

  private async updateStreamer(): Promise<void> {
    // Assemble the set of keys we need to stream.
    const keys = new Set<ChannelKey>();
    this.listeners.forEach((v) => v.forEach((k) => keys.add(k)));

    // If we have no keys to stream, close the streamer to save network chatter.
    if (keys.size === 0) {
      this._streamer?.close();
      this._streamer = null;
    }

    // Update or create the streamer.
    const arrKeys = Array.from(keys);
    if (this._streamer != null) return await this._streamer.update(arrKeys);
    this._streamer = await this.core.telem.newStreamer(arrKeys);

    void this.start(this._streamer);
  }

  private async start(streamer: Streamer): Promise<void> {
    for await (const frame of streamer) {
      const changed = new Map<ChannelKey, [Channel, LazyArray[]]>();
      for (const k of frame.keys) {
        const arrays = frame.get(k);
        const cache = await this.getCache(k);
        const out = cache.writeDynamic(convertArrays(arrays));
        if (out.length > 0) changed.set(k, [cache.channel, out]);
      }
      this.listeners.forEach((v, k) => {
        const notify = v
          .map((c) => {
            const change = changed.get(c);
            if (change == null) return null;
            const [ch, arrays] = change;
            return new ReadResponse(ch, arrays);
          })
          .filter((e) => e != null) as ReadResponse[];
        if (notify.length > 0) k(notify);
      });
    }
  }
}

export class ReadResponse {
  channel: Channel;
  data: LazyArray[];

  constructor(channel: Channel, data: LazyArray[]) {
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
