// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Series, TimeRange } from "@synnaxlabs/x";

import { ChannelKeyOrName, ChannelParams } from "@/channel";
import { Frame, FrameClient } from "@/framer";

export class Range {
  key: string;
  name: string;
  readonly timeRange: TimeRange;
  private readonly frameClient: FrameClient;

  constructor(
    name: string,
    timeRange: TimeRange = TimeRange.ZERO,
    key: string,
    _frameClient: FrameClient
  ) {
    this.key = key;
    this.name = name;
    this.timeRange = timeRange;
    this.frameClient = _frameClient;
  }

  async read(channel: ChannelKeyOrName): Promise<Series>;

  async read(channels: ChannelParams): Promise<Frame>;

  async read(channels: ChannelParams): Promise<Series | Frame> {
    return await this.frameClient.read(this.timeRange, channels);
  }
}
