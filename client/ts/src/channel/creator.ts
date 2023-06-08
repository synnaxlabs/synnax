// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { UnaryClient } from "@synnaxlabs/freighter";
import { toArray } from "@synnaxlabs/x";
import { z } from "zod";

import {
  ChannelPayload,
  channelPayload,
  parseChannels,
  unkeyedChannelPayload,
  UnparsedChannel,
} from "./payload";

import { Transport } from "@/transport";

const requestZ = z.object({
  channels: unkeyedChannelPayload.array(),
});

type Request = z.input<typeof requestZ>;

const responseZ = z.object({
  channels: channelPayload.array(),
});

type Response = z.output<typeof responseZ>;

export class ChannelCreator {
  private static readonly ENDPOINT = "/channel/create";
  private readonly client: UnaryClient;

  constructor(transport: Transport) {
    this.client = transport.postClient();
  }

  async create(channel: UnparsedChannel): Promise<ChannelPayload>;

  async create(
    channels: UnparsedChannel | UnparsedChannel[]
  ): Promise<ChannelPayload[]>;

  async create(
    channels: UnparsedChannel | UnparsedChannel[]
  ): Promise<ChannelPayload | ChannelPayload[]> {
    const single = !Array.isArray(channels);
    const { channels: ch_ } = await this.execute({
      channels: parseChannels(toArray(channels)),
    });
    return single ? ch_[0] : ch_;
  }

  private async execute(request: Request): Promise<Response> {
    const [res, err] = await this.client.send<typeof requestZ, typeof responseZ>(
      ChannelCreator.ENDPOINT,
      request,
      responseZ
    );
    if (err != null) throw err;
    return res;
  }
}
