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
  array,
  compare,
  type CrudeTimeSpan,
  debounce,
  type destructor,
  MultiSeries,
  type Series,
  TimeSpan,
} from "@synnaxlabs/x";
import { Mutex } from "async-mutex";

import { type Cache } from "@/telem/client/cache/cache";

export type StreamHandler = (data: Map<channel.Key, MultiSeries>) => void;

interface ListenerEntry {
  valid: boolean;
  keys: Set<channel.Key>;
}

interface StreamerProps {
  cache: Cache;
  openStreamer: framer.StreamOpener;
  instrumentation?: alamos.Instrumentation;
  streamUpdateDelay?: CrudeTimeSpan;
}

// Introduce a slight debounce into stream start requests so that rapid streaming
// request don't slam the socket with lots of updates.
const STREAM_DEBOUNCE = TimeSpan.milliseconds(100).milliseconds;

export class Streamer {
  private readonly props: Omit<Required<StreamerProps>, "streamUpdateDelay"> & {
    streamUpdateDelay: TimeSpan;
  };

  private readonly mu: Mutex = new Mutex();
  private readonly listeners = new Map<StreamHandler, ListenerEntry>();
  private readonly debouncedUpdateStreamer: () => void;
  private streamerRunLoop: Promise<void> | null = null;
  private streamer: framer.Streamer | null = null;
  private closed = false;

  constructor(props: StreamerProps) {
    this.props = {
      instrumentation: alamos.NOOP,
      ...props,
      streamUpdateDelay: new TimeSpan(props.streamUpdateDelay ?? TimeSpan.seconds(5)),
    };
    this.debouncedUpdateStreamer = debounce(
      () => void this.updateStreamer(),
      STREAM_DEBOUNCE,
    );
  }

  /** Implements StreamClient. */
  async stream(
    handler: StreamHandler,
    keys: channel.Keys,
  ): Promise<destructor.Destructor> {
    const { cache, instrumentation: ins } = this.props;
    if (this.closed) return () => {};
    // Make sure that the cache has entries for all relevant channels. This will also
    // do a check to make sure that the channels actually exist.
    await cache.populateMissing(keys);

    return await this.mu.runExclusive(async () => {
      ins.L.debug("adding stream handler", { keys });

      // Bind a new listener.
      this.listeners.set(handler, { valid: true, keys: new Set(keys) });

      // Pull any existing dynamic buffers from the cache so that the caller has
      // access to them as they get filled.
      const dynamicBuffers: Map<channel.Key, MultiSeries> = new Map(
        keys.map((key) => {
          const unary = cache.get(key);
          return [key, new MultiSeries(array.toArray<Series>(unary.leadingBuffer))];
        }),
      );
      handler(dynamicBuffers);

      // Update the remote streamer to start streaming the new channels.
      this.debouncedUpdateStreamer();

      return () => this.removeStreamHandler(handler);
    });
  }

  private removeStreamHandler(handler: StreamHandler): void {
    const { instrumentation: ins } = this.props;
    void this.mu.runExclusive(() => {
      const entry = this.listeners.get(handler);
      if (entry == null) return;
      entry.valid = false;
    });
    setTimeout(() => {
      void this.mu.runExclusive(async () => {
        ins.L.debug("removing stream handler");
        if (this.listeners.delete(handler)) return this.debouncedUpdateStreamer();
        ins.L.warn("attempted to remove non-existent stream handler");
      });
    }, this.props.streamUpdateDelay.milliseconds);
  }

  private async updateStreamer(): Promise<void> {
    if (this.closed) return;
    const { instrumentation: ins } = this.props;
    try {
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
      const valuesEqual =
        compare.primitiveArrays(arrKeys, this.streamer?.keys ?? []) === compare.EQUAL;
      if (valuesEqual) {
        ins.L.debug("streamer keys unchanged", { keys: arrKeys });
        return;
      }

      // Update or create the streamer.
      if (this.streamer == null) {
        ins.L.info("creating new streamer", { keys: arrKeys });
        this.streamer = await this.props.openStreamer({ channels: arrKeys });
        this.streamerRunLoop = this.runStreamer(this.streamer);
      }

      ins.L.debug("updating streamer", { prev: this.streamer.keys, next: arrKeys });

      await this.streamer.update(arrKeys);
    } catch (e) {
      ins.L.error("failed to update streamer", { error: e });
      throw e;
    }
  }

  private async runStreamer(streamer: framer.Streamer): Promise<void> {
    const { cache, instrumentation: ins } = this.props;
    try {
      for await (const frame of streamer) {
        const changed: Map<channel.Key, MultiSeries> = new Map();
        for (const k of frame.keys) {
          const series = frame.get(k);
          const unary = cache.get(k);
          const out = unary.writeDynamic(series);
          changed.set(k, out);
        }
        if (changed.size !== 0)
          this.listeners.forEach(({ valid }, handler) => valid && handler(changed));
      }
    } catch (e) {
      ins.L.error("streamer run loop failed", { error: e }, true);
      throw e;
    }
  }

  async close(): Promise<void> {
    const { instrumentation: ins } = this.props;
    try {
      this.streamer?.close();
      if (this.streamerRunLoop != null) await this.streamerRunLoop;
    } catch (e) {
      ins.L.error("failed to close streamer", { error: e });
    }
    this.closed = true;
  }
}
