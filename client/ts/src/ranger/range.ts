// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Series, TimeRange, TimeSpan } from "@synnaxlabs/x";
import { z } from "zod";

import { ChannelKeyOrName, ChannelParams } from "@/channel";
import { Frame, FrameClient } from "@/framer";

const rangeVariants = z.enum(["static", "dynamic"]);
type RangeVariant = z.infer<typeof rangeVariants>;

export class Range {
  key: string;
  name: string;
  open: boolean;
  variant: RangeVariant;
  readonly timeRange: TimeRange;
  readonly span: TimeSpan;
  private readonly frameClient: FrameClient;

  constructor(
    key: string,
    name: string,
    open: boolean,
    variant: RangeVariant,
    timeRange: TimeRange = TimeRange.ZERO,
    span: TimeSpan = TimeSpan.ZERO,
    _frameClient: FrameClient
  ) {
    this.key = key;
    this.name = name;
    this.timeRange = timeRange;
    this.span = span;
    this.variant = variant;
    this.open = open;
    this.frameClient = _frameClient;
  }

  static readonly z = z.object({
    key: z.string(),
    name: z.string(),
    open: z.boolean(),
    timeRange: TimeRange.z,
    span: TimeSpan.z,
    variant: z.enum(["static", "dynamic"]),
  });

  toPayload(): RangePayload {
    return {
      key: this.key,
      name: this.name,
      open: this.open,
      timeRange: this.timeRange,
      span: this.span,
      variant: this.variant,
    };
  }

  read(channel: ChannelKeyOrName): Promise<Series>;

  read(channels: ChannelParams): Promise<Frame>;

  async read(channels: ChannelParams): Promise<Series | Frame> {
    return await this.frameClient.read(this.timeRange, channels);
  }
}

export type RangePayload = z.infer<typeof Range.z>;
