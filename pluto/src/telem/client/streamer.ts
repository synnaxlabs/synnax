// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos } from "@synnaxlabs/alamos";
import { type channel, type framer } from "@synnaxlabs/client";
import {
  type AsyncDestructor,
  breaker,
  compare,
  type CrudeTimeSpan,
  nullToArr,
  type Optional,
  type Required,
  type Series,
  TimeSpan,
} from "@synnaxlabs/x";
import { Mutex } from "async-mutex";

import { type Cache } from "@/telem/client/cache/cache";
import { ReadResponse } from "@/telem/client/types";

export type StreamHandler = (data: Record<channel.Key, ReadResponse>) => void;

interface ListenerEntry {
  valid: boolean;
  keys: channel.Keys;
}

interface StreamerProps {
  cache: Cache;
  instrumentation?: alamos.Instrumentation;
  core: StreamClient;
  streamUpdateDelay?: CrudeTimeSpan;
  breakerConfig?: breaker.Config;
}

export interface CoreStreamer {
  update: (keys: channel.Keys) => Promise<void>;
  close: () => void;
  keys: channel.Keys;
  next: () => Promise<IteratorResult<framer.Frame>>;
  [Symbol.asyncIterator](): AsyncIterator<framer.Frame>;
}

export interface StreamClient {
  openStreamer: (keys: channel.Keys) => Promise<CoreStreamer>;
}

export class Streamer {
  private readonly props: Omit<
    Optional<Required<StreamerProps>, "breakerConfig">,
    "streamUpdateDelay"
  > & {
    streamUpdateDelay: TimeSpan;
  };

  private readonly mu: Mutex = new Mutex();
  private readonly listeners = new Map<StreamHandler, ListenerEntry>();
  private streamerRunLoop: Promise<void> | null = null;
  private streamer: CoreStreamer | null = null;
  private closed = false;

  constructor(props: StreamerProps) {
    this.props = {
      instrumentation: alamos.NOOP,
      ...props,
      streamUpdateDelay: new TimeSpan(props.streamUpdateDelay ?? TimeSpan.seconds(5)),
    };
  }

  /** Implements StreamClient. */
  async stream(handler: StreamHandler, keys: channel.Keys): Promise<AsyncDestructor> {
    const { cache, instrumentation: ins } = this.props;

    // Make sure that the cache has entries for all relevant channels. This will also
    // do a check to make sure that the channels actually exist.
    await cache.populateMissing(keys);

    return await this.mu.runExclusive(async () => {
      ins.L.debug("adding stream handler", { keys });

      // Bind a new listener.
      this.listeners.set(handler, { valid: true, keys });

      // Pull any existing dynamic buffers from the cache so that the caller has
      // access to them as they get filled.
      const dynamicBuffers: Record<channel.Key, ReadResponse> = Object.fromEntries(
        keys.map((key) => {
          const unary = cache.get(key);
          return [
            key,
            new ReadResponse(unary.channel, nullToArr<Series>(unary.leadingBuffer)),
          ];
        }),
      );
      handler(dynamicBuffers);

      // Update the remote streamer to start streaming the new channels.
      await this.updateStreamer();

      return async () => await this.removeStreamHandler(handler);
    });
  }

  private async removeStreamHandler(handler: StreamHandler): Promise<void> {
    const {
      instrumentation: { L },
    } = this.props;
    await this.mu.runExclusive(() => {
      const entry = this.listeners.get(handler);
      if (entry == null) return;
      entry.valid = false;
    });
    setTimeout(() => {
      void this.mu.runExclusive(async () => {
        L.debug("removing stream handler");
        if (this.listeners.delete(handler)) return await this.updateStreamer();
        L.warn("attempted to remove non-existent stream handler");
      });
    }, this.props.streamUpdateDelay.milliseconds);
  }

  private async updateStreamer(): Promise<void> {
    if (this.closed) return;
    const { instrumentation: ins } = this.props;
    // Assemble the set of keys we need to stream.
    const keys = new Set<channel.Key>();
    this.listeners.forEach((v) => v.keys.forEach((k) => keys.add(k)));

    // If we have no keys to stream, close the streamer to save network chatter.
    if (keys.size === 0) {
      ins.L.info("no keys to stream, closing streamer");
      this.streamer?.close();
      if (this.streamerRunLoop != null) await this.streamerRunLoop;
      this.streamer = null;
      ins.L.info("streamer closed successfully");
      return;
    }

    const arrKeys = Array.from(keys);
    if (compare.primitiveArrays(arrKeys, this.streamer?.keys ?? []) === compare.EQUAL) {
      ins.L.debug("streamer keys unchanged", { keys: arrKeys });
      return;
    }

    // Update or create the streamer.
    if (this.streamer == null) {
      ins.L.info("creating new streamer", { keys: arrKeys });
      this.streamer = await this.props.core.openStreamer(arrKeys);
      this.streamerRunLoop = this.runStreamer(this.streamer);
    }

    ins.L.debug("updating streamer", { prev: this.streamer.keys, next: arrKeys });

    try {
      await this.streamer.update(arrKeys);
    } catch (e) {
      ins.L.error("failed to update streamer", { error: e });
      throw e;
    }
  }

  private notifyListeners(changed: ReadResponse[]): void {
    if (changed.length === 0) return;
    this.listeners.forEach((entry, handler) => {
      if (!entry.valid) return;
      const notify = changed.filter((r) => entry.keys.includes(r.channel.key));
      if (notify.length === 0) return;
      const d = Object.fromEntries(notify.map((r) => [r.channel.key, r]));
      handler(d);
    });
  }

  private async runStreamer(streamer: CoreStreamer): Promise<void> {
    const { cache, instrumentation: ins } = this.props;
    try {
      for await (const frame of streamer) {
        const changed: ReadResponse[] = [];
        for (const k of frame.keys) {
          const series = frame.get(k);
          const unary = cache.get(k);
          const out = unary.writeDynamic(series.series);
          changed.push(new ReadResponse(unary.channel, out));
        }
        this.notifyListeners(changed);
      }
    } catch (e) {
      console.error("streamer run loop failed", { error: e }, true);
      throw e;
    }
  }

  async close(): Promise<void> {
    this.streamer?.close();
    if (this.streamerRunLoop != null) await this.streamerRunLoop;
    this.closed = true;
  }
}

class HardenedStreamer implements CoreStreamer {
  private wrapped_: CoreStreamer | null = null;
  private readonly breaker: breaker.Breaker;
  private readonly client: StreamClient;
  keys: channel.Keys;

  constructor(
    client: StreamClient,
    keys: channel.Keys,
    breakerConfig?: breaker.Config,
  ) {
    this.client = client;
    this.breaker = new breaker.Breaker(breakerConfig);
    this.keys = keys;
  }

  static async open(
    client: StreamClient,
    keys: channel.Keys,
    config?: breaker.Config,
  ): Promise<HardenedStreamer> {
    const h = new HardenedStreamer(client, keys, {
      baseInterval: TimeSpan.seconds(1),
      maxRetries: 1000,
      scale: 1,
    });
    await h.runStreamer();
    return h;
  }

  private async runStreamer(): Promise<void> {
    while (true)
      try {
        this.wrapped_ = await this.client.openStreamer(this.keys);
        return;
      } catch (e) {
        if (!(await this.breaker.wait())) throw e;
        continue;
      }
  }

  private get wrapped(): CoreStreamer {
    if (this.wrapped_ == null) throw new Error("stream closed");
    return this.wrapped_;
  }

  async update(keys: channel.Keys): Promise<void> {
    this.keys = keys;
    await this.runStreamer();
    try {
      await this.wrapped.update(keys);
    } catch (e) {
      if (!(await this.breaker.wait())) throw e;
      await this.runStreamer();
      throw e;
    }
  }

  async next(): Promise<IteratorResult<framer.Frame>> {
    try {
      return await this.wrapped.next();
    } catch (e) {
      await this.runStreamer();
      return await this.next();
    }
  }

  close(): void {
    this.wrapped.close();
  }

  [Symbol.asyncIterator](): AsyncIterator<framer.Frame> {
    return this;
  }
}
