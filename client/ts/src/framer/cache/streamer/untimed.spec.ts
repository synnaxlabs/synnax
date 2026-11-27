// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { disposable } from "@synnaxlabs/x";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import type { channel } from "@/channel";
import type { framer } from "@/framer";
import { join } from "@/framer/cache/join";
import { store as cacheStore } from "@/framer/cache/store";
import { streamer as cacheStreamer } from "@/framer/cache/streamer";

// A streamer whose reads block until close, and whose updates wait until the spec
// acks them.
class HeldStreamer implements framer.Streamer {
  declare [Symbol.dispose]: () => void;
  declare [Symbol.asyncDispose]: () => Promise<void>;
  static {
    disposable.implement(this);
  }

  keys: channel.Key[];
  readonly untimed: (channel.Key[] | undefined)[] = [];
  private readonly acks: (() => void)[] = [];
  private closed = false;
  private wake: (() => void) | null = null;

  constructor(keys: channel.Key[]) {
    this.keys = keys;
  }

  get pending(): number {
    return this.acks.length;
  }

  ack(): void {
    this.acks.shift()?.();
  }

  async update(
    keys: channel.Params,
    options?: framer.StreamerUpdateOptions,
  ): Promise<void> {
    this.keys = keys as channel.Key[];
    this.untimed.push(options?.untimedKeys);
    await new Promise<void>((resolve) => this.acks.push(resolve));
  }

  close(): void {
    this.closed = true;
    this.wake?.();
    this.acks.splice(0).forEach((resolve) => resolve());
  }

  async next(): Promise<IteratorResult<framer.Frame>> {
    if (!this.closed) await new Promise<void>((resolve) => (this.wake = resolve));
    return { done: true, value: undefined };
  }

  async read(): Promise<framer.Frame> {
    throw new Error("unused");
  }

  [Symbol.asyncIterator](): AsyncIterator<framer.Frame> {
    return this;
  }
}

describe("Multiplexed untimed demands", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("should make a timed demand on an untimed key wait for the ack of its time", async () => {
    const store = new cacheStore.Store();
    const ms = new HeldStreamer([1]);
    const configs: { untimedKeys: channel.Key[] }[] = [];
    const streamer = new cacheStreamer.Multiplexed({
      store,
      joins: new join.Joins(store),
      onWait: () => {},
      openStreamer: async (config) => {
        configs.push(config);
        return ms;
      },
    });
    const untimed = streamer.demand([1], () => {}, { untimed: true });
    await vi.advanceTimersByTimeAsync(200);
    expect(configs[0].untimedKeys).toEqual([1]);
    await streamer.waitAcked([1]);
    const timed = streamer.demand([1], () => {});
    await vi.advanceTimersByTimeAsync(200);
    // The update that asks for the time of key 1 waits for its ack.
    expect(ms.untimed.at(-1)).toEqual([]);
    expect(ms.pending).toBe(1);
    const waited = vi.fn();
    void streamer.waitAcked([1]).then(waited);
    await vi.advanceTimersByTimeAsync(10);
    expect(waited).not.toHaveBeenCalled();
    ms.ack();
    await vi.advanceTimersByTimeAsync(10);
    expect(waited).toHaveBeenCalled();
    untimed.close();
    timed.close();
    await streamer.close();
  });
});
