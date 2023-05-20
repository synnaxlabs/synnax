// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { LazyArray, TimeRange } from "@synnaxlabs/x";
import { Observable, Subscriber } from "rxjs";
import { z } from "zod";

import { ChannelKeyOrName, ChannelParams } from "@/channel";
import { Frame, FrameClient } from "@/framer";
import { Streamer } from "@/framer/streamer";

export class Range {
  key: string;
  name: string;
  open: boolean;
  readonly timeRange: TimeRange;
  private readonly frameClient: FrameClient;

  constructor(
    key: string,
    name: string,
    timeRange: TimeRange,
    open: boolean,
    _frameClient: FrameClient
  ) {
    this.key = key;
    this.name = name;
    this.timeRange = timeRange;
    this.open = open;
    this.frameClient = _frameClient;
  }

  static readonly z = z.object({
    key: z.string(),
    name: z.string(),
    open: z.boolean(),
    timeRange: TimeRange.z,
  });

  read(channel: ChannelKeyOrName): Promise<LazyArray>;

  read(...channels: ChannelParams[]): Promise<Frame>;

  async read(...channels: ChannelParams[]): Promise<LazyArray | Frame> {
    return await this.frameClient.read(this.timeRange, ...channels);
  }

  // async stream(...channels: ChannelParams[]): Promise<FrameStreamReader> {
  //   return await this.frameClient.stream(this.timeRange.start, ...channels);
  // }
}

export interface RangeCache {
  read: (...channels: number[]) => Promise<LazyArray>;
}

export class DynamicRangeCache {
  range: Range;
  cache: Map<number, ChannelCache>;
  stream: Streamer;

  constructor(range: Range, stream: Streamer) {
    this.range = range;
    this.cache = new Map();
    this.stream = stream;
    new Observable<Frame>((s) => {
      void this.readLoop(s);
    }).subscribe((fr) => this.process(fr));
  }

  async readLoop(sub: Subscriber<Frame>): Promise<void> {
    while (true) sub.next(await this.stream.read());
  }

  async acquire(...channels: number[]): Promise<void> {
    for (const channel of channels) {
      const cache = this.cache.get(channel);
      if (cache != null) cache.acquire();
      else {
        this.cache.set(channel, new ChannelCache());
        await this.stream.update(channels);
      }
    }
  }

  async release(...channels: number[]): Promise<void> {
    for (const channel of channels) {
      const cache = this.cache.get(channel);
      if (cache != null) cache.release();
    }
  }

  private process(fr: Frame): void {
    fr.channelKeys.forEach((key) => {
      const cache = this.cache.get(key);
      if (cache != null) cache.write(fr.arrays[key]);
    });
  }
}

class ChannelCache {
  demand: number;
  arrays: LazyArray[];

  constructor() {
    this.demand = 0;
    this.arrays = [];
  }

  write(array: LazyArray): void {
    this.arrays.push(array);
  }

  acquire(): void {
    this.demand++;
  }

  release(): void {
    this.demand--;
  }
}
