// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  ChannelKey,
  ChannelKeys,
  Streamer as CoreStreamer,
  Frame,
} from "@synnaxlabs/client";

export class Streamer {
  private readonly wrap: CoreStreamer;
  readonly listeners: Map<(fr: Frame) => void, ChannelKeys>;

  constructor(wrap: CoreStreamer) {
    this.wrap = wrap;
    this.listeners = new Map();
    void this.start();
  }

  async set(keys: ChannelKeys, f: (fr: Frame) => void): Promise<void> {
    this.listeners.set(f, keys);
    await this.update();
  }

  async delete(f: (fr: Frame) => void): Promise<void> {
    this.listeners.delete(f);
    await this.update();
  }

  private async update(): Promise<void> {
    const keys = new Set<ChannelKey>();
    this.listeners.forEach((v) => v.forEach((k) => keys.add(k)));
    await this.wrap.update(...keys);
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
