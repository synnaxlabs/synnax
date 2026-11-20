// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { EOF } from "@synnaxlabs/freighter";
import { disposable, sleep, TimeSpan } from "@synnaxlabs/x";

import type { channel } from "@/channel";
import type { framer } from "@/framer";

/** A change stream that delivers the frames a spec pushes, in push order. */
export class PushStreamer implements framer.Streamer {
  declare [Symbol.dispose]: () => void;
  declare [Symbol.asyncDispose]: () => Promise<void>;
  static {
    disposable.implement(this);
  }

  keys: channel.Key[] = [];
  private readonly frames: framer.Frame[] = [];
  private done = false;
  private dropped = false;

  push(frame: framer.Frame): void {
    this.frames.push(frame);
  }

  /** Fails the read after the pushed frames, so the cache opens a new stream. */
  drop(): void {
    this.dropped = true;
  }

  async update(): Promise<void> {}

  close(): void {
    this.done = true;
  }

  async next(): Promise<IteratorResult<framer.Frame>> {
    while (!this.done) {
      const frame = this.frames.shift();
      if (frame != null) return { done: false, value: frame };
      if (this.dropped) throw new Error("stream dropped");
      await sleep.sleep(TimeSpan.milliseconds(5));
    }
    return { done: true, value: undefined };
  }

  async read(): Promise<framer.Frame> {
    const res = await this.next();
    if (res.done) throw new EOF();
    return res.value;
  }

  [Symbol.asyncIterator](): AsyncIterator<framer.Frame> {
    return this;
  }
}
