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
  LazyArray,
  Streamer as CoreStreamer,
  Frame,
} from "@synnaxlabs/client";
import { Destructor } from "@synnaxlabs/x";

export class Streamer {
  private readonly wrap: CoreStreamer;
  readonly listeners: Map<(fr: Frame) => void, ChannelKeys>;

  constructor(wrap: CoreStreamer) {
    this.wrap = wrap;
    this.listeners = new Map();
    void this.start();
  }

  listen(keys: ChannelKeys, f: (fr: Frame) => void): Destructor {
    this.listeners.set(f, keys);
    this.update();
    return () => this.listeners.delete(f);
  }

  private update(): void {
    const keys = new Set<ChannelKey>();
    this.listeners.forEach((v) => v.forEach((k) => keys.add(k)));
    this.wrap.update(...keys);
  }

  private async start(): Promise<void> {
    for await (const frame of this.wrap) {
      this.listeners.forEach((_, f) => f(frame));
    }
  }

  stop(): void {
    this.wrap.close();
  }
}

class ChannelCache {
  private readonly channel: Channel;
  private readonly entries: ChannelCacheEntry[];
  private readonly handlers: Map<(res: ChannelCacheResult) => void, undefined>;
  private readonly streamer: Streamer;
  private streamerDestructor: Destructor | null;

  constructor(channel: Channel, streamer: Streamer) {
    this.channel = channel;
    this.handlers = new Map();
    this.entries = [];
    this.streamer = streamer;
    this.streamerDestructor = null;
  }

  private handler(fr: Frame): void {}

  stream(handler: (res: ChannelCacheResult) => void): Destructor {
    if (this.streamerDestructor == null)
      this.streamerDestructor = this.streamer.listen([this.channel.key], this.handler);
    this.handlers.set(handler, undefined);
    return () => {
      this.handlers.delete(handler);
      if (this.handlers.size === 0) this.streamerDestructor?.();
    };
  }
}

interface ChannelCacheEntry {
  demand: number;
  gl: WebGLBuffer;
  arr: LazyArray;
}

class ChannelCacheResult {
  entries: ChannelCacheEntry[];
  release(): void {}
}
