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
} from "@/channel/payload";

const reqZ = z.object({ channels: unkeyedChannelPayload.array() });

const resZ = z.object({ channels: channelPayload.array() });

export class ChannelCreator {
  private static readonly ENDPOINT = "/channel/create";
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(
    channels: UnparsedChannel | UnparsedChannel[]
  ): Promise<ChannelPayload[]> {
    const req = { channels: parseChannels(toArray(channels)) };
    const [res, err] = await this.client.send<typeof reqZ, typeof resZ>(
      ChannelCreator.ENDPOINT,
      req,
      resZ
    );
    if (err != null) throw err;
    return res.channels;
  }
}
