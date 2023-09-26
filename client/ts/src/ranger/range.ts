// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Series, TimeRange } from "@synnaxlabs/x";

import { type Key, type Params, type Name } from "@/channel/payload";
import { type framer } from "@/framer";

export class Range {
  key: string;
  name: string;
  readonly timeRange: TimeRange;
  private readonly frameClient: framer.Client;

  constructor(
    name: string,
    timeRange: TimeRange = TimeRange.ZERO,
    key: string,
    _frameClient: framer.Client,
  ) {
    this.key = key;
    this.name = name;
    this.timeRange = timeRange;
    this.frameClient = _frameClient;
  }

  async read(channel: Key | Name): Promise<Series>;

  async read(channels: Params): Promise<framer.Frame>;

  async read(channels: Params): Promise<Series | framer.Frame> {
    return await this.frameClient.read(this.timeRange, channels);
  }
}
