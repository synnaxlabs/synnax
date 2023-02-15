// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { UnaryClient } from "@synnaxlabs/freighter";
import { UnparsedDataType, UnparsedRate } from "@synnaxlabs/x";
import { z } from "zod";

import {
  ChannelPayload,
  channelPayloadSchema,
  unkeyedChannelPayloadSchema,
} from "./payload";

import { Transport } from "@/transport";

const RequestSchema = z.object({
  channels: unkeyedChannelPayloadSchema.array(),
});

type Request = z.infer<typeof RequestSchema>;

const ResponseSchema = z.object({
  channels: channelPayloadSchema.array(),
});

type Response = z.infer<typeof ResponseSchema>;

export interface CreateChannelProps {
  rate: UnparsedRate;
  dataType: UnparsedDataType;
  name?: string;
  nodeId?: number;
  index?: string;
  isIndex?: boolean;
}

export class ChannelCreator {
  private static readonly ENDPOINT = "/channel/create";
  private readonly client: UnaryClient;

  constructor(transport: Transport) {
    this.client = transport.postClient();
  }

  async create(props: CreateChannelProps): Promise<ChannelPayload> {
    const [channel] = await this.createMany([props]);
    return channel;
  }

  async createMany(channels: CreateChannelProps[]): Promise<ChannelPayload[]> {
    return (await this.execute({ channels: channels as ChannelPayload[] })).channels;
  }

  private async execute(request: Request): Promise<Response> {
    const [res, err] = await this.client.send(
      ChannelCreator.ENDPOINT,
      request,
      // eslint-disable-next-line @typescript-eslint/ban-ts-comment
      // @ts-expect-error
      ResponseSchema
    );
    if (err != null) throw err;
    return res as Response;
  }
}
