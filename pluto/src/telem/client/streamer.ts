// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos } from "@synnaxlabs/alamos";
import { type channel, type framer, type Synnax } from "@synnaxlabs/client";
import { EOF } from "@synnaxlabs/freighter";
import { type AsyncDestructor, compare, errors, type Required } from "@synnaxlabs/x";
import { Mutex } from "async-mutex";

import { type Cache } from "@/telem/client/cache/cache";
import { ReadResponse } from "@/telem/client/types";

export type StreamHandler = (data: Record<channel.Key, ReadResponse>) => void;

interface ListenerEntry {
  valid: boolean;
  keys: channel.Keys;
}

interface StreamerProps {
  core: Synnax;
  cache: Cache;
  instrumentation?: alamos.Instrumentation;
}

export class Streamer {
  private readonly props: Required<StreamerProps>;

  private readonly mu: Mutex = new Mutex();
  private readonly listeners = new Map<StreamHandler, ListenerEntry>();
  private streamerRunLoop: Promise<void> | null = null;
  private streamer: framer.Streamer | null = null;
  private closed = false;

  constructor(props: StreamerProps) {
    this.props = {
      instrumentation: alamos.NOOP,
      ...props,
    };
  }

  /** Implements StreamClient. */
  async stream(handler: StreamHandler, keys: channel.Keys): Promise<AsyncDestructor> {
    const {
      cache,
      instrumentation: { L },
    } = this.props;
    await cache.populateMissing(keys);
    return await this.mu.runExclusive(async () => {
      L.debug("adding stream handler", { keys });
      this.listeners.set(handler, { valid: true, keys });
      const dynamicBuffs: Record<channel.Key, ReadResponse> = {};
      for (const key of keys) {
        const unary = cache.get(key);
        const bufs = unary.leadingBuffer != null ? [unary.leadingBuffer] : [];
        dynamicBuffs[key] = new ReadResponse(unary.channel, bufs);
      }
      handler(dynamicBuffs);
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
    }, 5000);
  }

  private async updateStreamer(): Promise<void> {
    if (this.closed) return;
    const {
      instrumentation: { L },
      core,
    } = this.props;
    // Assemble the set of keys we need to stream.
    const keys = new Set<channel.Key>();
    this.listeners.forEach((v) => v.keys.forEach((k) => keys.add(k)));

    // If we have no keys to stream, close the streamer to save network chatter.
    if (keys.size === 0) {
      L.info("no keys to stream, closing streamer");
      this.streamer?.close();
      if (this.streamerRunLoop != null) await this.streamerRunLoop;
      this.streamer = null;
      L.info("streamer closed successfully");
      return;
    }

    const arrKeys = Array.from(keys);
    if (compare.primitiveArrays(arrKeys, this.streamer?.keys ?? []) === compare.EQUAL) {
      L.debug("streamer keys unchanged", { keys: arrKeys });
      return;
    }

    // Update or create the streamer.
    if (this.streamer == null) {
      L.info("creating new streamer", { keys: arrKeys });
      this.streamer = await core.openStreamer(arrKeys);
      this.streamerRunLoop = this.runStreamer(this.streamer);
    }

    L.debug("updating streamer", { prev: this.streamer.keys, next: arrKeys });

    try {
      await this.streamer.update(arrKeys);
    } catch (e) {
      L.error("failed to update streamer", { error: e });
      if (EOF.matches(e)) {
        L.warn("streamer closed unexpectedly, resetting");
        this.streamer = null;
        await this.updateStreamer();
        return;
      }
      throw e;
    }
  }

  private async runStreamer(streamer: framer.Streamer): Promise<void> {
    const {
      cache,
      instrumentation: { L },
    } = this.props;
    try {
      for await (const frame of streamer) {
        const changed: ReadResponse[] = [];
        for (const k of frame.keys) {
          const series = frame.get(k);
          const unary = cache.get(k);
          const out = unary.writeDynamic(series.series);
          changed.push(new ReadResponse(unary.channel, out));
        }
        this.listeners.forEach((entry, handler) => {
          if (!entry.valid) return;
          const notify = changed.filter((r) => entry.keys.includes(r.channel.key));
          if (notify.length === 0) return;
          const d = Object.fromEntries(notify.map((r) => [r.channel.key, r]));
          handler(d);
        });
      }
    } catch (e) {
      console.error("streamer run loop failed", e);
      L.error("streamer run loop failed", { error: e }, true);
      throw e;
    }
  }

  async close(): Promise<void> {
    this.streamer?.close();
    if (this.streamerRunLoop != null) await this.streamerRunLoop;
    this.closed = true;
  }
}
