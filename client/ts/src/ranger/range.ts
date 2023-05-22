// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { LazyArray, TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
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
}

export class DynamicRange {
  span: TimeSpan;
  gcResolution: TimeSpan;
  private _streamer: Streamer | null;
  private readonly frameClient: FrameClient;

  constructor(
    span: TimeSpan,
    frameClient: FrameClient,
    gcResolution: TimeSpan = TimeSpan.seconds(30)
  ) {
    this.span = span;
    this.frameClient = frameClient;
    this.gcResolution = gcResolution;
    this._streamer = null;
    void this.start();
  }

  private async start(): Promise<void> {
    const from = TimeStamp.now().sub(this.span);
    this._streamer = await this.frameClient.newStreamer(from);
    for await (const fr of this._streamer) {
      console.log(fr);
    }
  }

  private get streamer(): Streamer {
    if (this._streamer == null) throw new Error("streamer is null");
    return this._streamer;
  }

  async update(...channels: ChannelParams[]): Promise<void> {
    await this.streamer.update(...channels);
  }

  close(): void {
    this.streamer.close();
  }
}
