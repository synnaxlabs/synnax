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
import { GLBufferController } from "@/telem/cache/bufferController";
import { ChannelRange, Range } from "@/telem/range";

export type StreamHandler = (data: ReadResponse[] | null) => void;

export class Client {
  private _streamer: Streamer | null;
  readonly core: Synnax;
  private readonly cache: Map<ChannelKey, Cache>;
  private readonly listeners: Map<StreamHandler, ChannelRange>;
  private readonly gl: GLBufferController;

  constructor(wrap: Synnax, gl: GLBufferController) {
    this.core = wrap;
    this._streamer = null;
    this.gl = gl;
    this.cache = new Map();
    this.listeners = new Map();
  }

  setStreamhandler(handler: StreamHandler, range: ChannelRange): void {
    this.listeners.set(handler, range);
    void this.updateStreamer();
  }

  removeStreamHandler(handler: StreamHandler): void {
    this.listeners.delete(handler);
    void this.updateStreamer();
  }

  async read(range: ChannelRange): Promise<ReadResponse[]> {
    const toFetch = new Map<string, [TimeRange, ChannelKeys]>();
    const responses: Record<ChannelKey, [Channel, LazyArray[]]> = {};
    for (const key of range.channels) {
      const cache = await this.getCache(key);
      const [data, gaps] = cache.read(range.timeRange);
      responses[key] = [cache.channel, data];
      for (const gap of gaps) {
        const exists = toFetch.get(gap.toString());
        if (exists == null) toFetch.set(gap.toString(), [gap, [key]]);
        else toFetch.set(gap.toString(), [gap, [...exists[1], key]]);
      }
    }
    for (const [, [range, keys]] of toFetch) {
      const frame = await this.core.telem.read(range, ...keys);
      for (const key of keys) {
        const cache = await this.getCache(key);
        const arrays = frame.get(key);
        cache.writeStatic(range, frame.get(key));
        responses[key] = [cache.channel, responses[key][1].concat(arrays)];
      }
    }
    return Object.values(responses).map(
      ([channel, data]) => new ReadResponse(channel, range, data)
    );
  }

  private async getCache(key: ChannelKey): Promise<Cache> {
    const c = this.cache.get(key);
    if (c != null) return c;
    const channel = await this.core.channels.retrieve(key);
    const cache = new Cache(this.gl, 1000, channel);
    this.cache.set(key, cache);
    return cache;
  }

  private async updateStreamer(): Promise<void> {
    const keys = new Set<ChannelKey>();
    this.listeners.forEach((v) => v.channels.forEach((k) => keys.add(k)));
    if (keys.size === 0) {
      this._streamer?.close();
      this._streamer = null;
    }
    if (this._streamer != null) return await this._streamer.update(...keys);
    this._streamer = await this.core.telem.newStreamer(...keys);
    void this.start(this._streamer);
  }

  private async start(streamer: Streamer): Promise<void> {
    const changed = new Map<ChannelKey, [Channel, LazyArray[]]>();
    for await (const frame of streamer) {
      for (const k of frame.channelKeys) {
        const arrays = frame.get(k);
        const cache = await this.getCache(k);
        const out = cache.writeDynamic(arrays);
        if (out.length > 0) changed.set(k, [cache.channel, out]);
      }
    }
    this.listeners.forEach((v, k) => {
      const notify = v.channels
        .map((c) => {
          const change = changed.get(c);
          if (change == null) return null;
          const [ch, arrays] = change;
          return new ReadResponse(ch, v, arrays);
        })
        .filter((e) => e != null) as ReadResponse[];
      if (notify.length > 0) k(notify);
    });
  }
}

export class ReadResponse {
  channel: Channel;
  range: Range;
  data: LazyArray[];

  constructor(channel: Channel, range: Range, data: LazyArray[]) {
    this.channel = channel;
    this.range = range;
    this.data = data;
  }

  get timeRange(): TimeRange {
    if (this.data.length === 0) return TimeRange.ZERO;
    const first = this.data[0].timeRange;
    const last = this.data[this.data.length - 1].timeRange;
    return new TimeRange(first.start, last.end);
  }
}
