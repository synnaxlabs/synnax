// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { UnaryClient } from "@synnaxlabs/freighter";
import { z } from "zod";

import {
  ChannelPayload,
  channelPayload,
  parseChannels,
  unkeyedChannelPayload,
  UnparsedChannel,
} from "./payload";

import { Transport } from "@/transport";

const RequestSchema = z.object({
  channels: unkeyedChannelPayload.array(),
});

type Request = z.infer<typeof RequestSchema>;

const responseZ = z.object({
  channels: channelPayload.array(),
});

type Response = z.infer<typeof responseZ>;

export class ChannelCreator {
  private static readonly ENDPOINT = "/channel/create";
  private readonly client: UnaryClient;

  constructor(transport: Transport) {
    this.client = transport.postClient();
  }

  async create(channel: UnparsedChannel): Promise<ChannelPayload>;

  async create(
    ...channels: Array<UnparsedChannel | UnparsedChannel[]>
  ): Promise<ChannelPayload[]>;

  async create(
    ...channels: Array<UnparsedChannel | UnparsedChannel[]>
  ): Promise<ChannelPayload | ChannelPayload[]> {
    const single = channels.length === 1 && !Array.isArray(channels[0]);
    const { channels: ch_ } = await this.execute({
      channels: parseChannels(channels.flat()),
    });
    return single ? ch_[0] : ch_;
  }

  private async execute(request: Request): Promise<Response> {
    const [res, err] = await this.client.send(
      ChannelCreator.ENDPOINT,
      request,
      responseZ
    );
    if (err != null) throw err;
    return res as Response;
  }
}
