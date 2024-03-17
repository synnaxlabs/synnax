import { type alamos } from "@synnaxlabs/alamos";
import { type framer, type channel, type Synnax } from "@synnaxlabs/client";
import { compare, type AsyncDestructor } from "@synnaxlabs/x";
import { Mutex } from "async-mutex";

import { type CacheManager } from "@/telem/client/cacheManager";
import { ReadResponse } from "@/telem/client/types";

export type StreamHandler = (data: Record<channel.Key, ReadResponse>) => void;

interface ListenerEntry {
  valid: boolean;
  keys: channel.Keys;
}

export class Streamer {
  private readonly core: Synnax;
  private readonly ins: alamos.Instrumentation;
  private readonly mu: Mutex = new Mutex();
  private readonly listeners = new Map<StreamHandler, ListenerEntry>();
  private readonly cache: CacheManager;
  private streamerRunLoop: Promise<void> | null = null;
  private streamer: framer.Streamer | null = null;

  constructor(cache: CacheManager, core: Synnax, ins: alamos.Instrumentation) {
    this.core = core;
    this.ins = ins;
    this.cache = cache;
  }

  /** Implements StreamClient. */
  async stream(handler: StreamHandler, keys: channel.Keys): Promise<AsyncDestructor> {
    await this.cache.populateMissing(keys);
    return await this.mu.runExclusive(async () => {
      this.ins.L.debug("adding stream handler", { keys });
      this.listeners.set(handler, { valid: true, keys });
      const dynamicBuffs: Record<channel.Key, ReadResponse> = {};
      for (const key of keys) {
        const c = this.cache.get(key);
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

  private async removeStreamHandler(handler: StreamHandler): Promise<void> {
    await this.mu.runExclusive(() => {
      const entry = this.listeners.get(handler);
      if (entry == null) return;
      entry.valid = false;
    });
    setTimeout(() => {
      void this.mu.runExclusive(async () => {
        this.ins.L.debug("removing stream handler");
        if (this.listeners.delete(handler)) return await this.updateStreamer();
        this.ins.L.warn("attempted to remove non-existent stream handler");
      });
    }, 5000);
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
        const cache = this.cache.get(k);
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

  async close(): Promise<void> {
    this.streamer?.close();
    if (this.streamerRunLoop != null) await this.streamerRunLoop;
  }
}
