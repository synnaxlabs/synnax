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

import { ChannelPayload, channelPayloadSchema } from "./payload";

import { ValidationError } from "@/errors";
import { Transport } from "@/transport";

const requestSchema = z.object({
  nodeId: z.number().optional(),
  keysOrNames: z.array(z.union([z.string().array(), z.string()])).optional(),
});

type Request = z.infer<typeof requestSchema>;

const responseSchema = z.object({
  channels: channelPayloadSchema.array(),
});

export class ChannelRetriever {
  private static readonly ENDPOINT = "/channel/retrieve";
  private readonly client: UnaryClient;

  constructor(transport: Transport) {
    this.client = transport.getClient();
  }

  private async execute(request: Request): Promise<ChannelPayload[]> {
    const [res, err] = await this.client.send(
      ChannelRetriever.ENDPOINT,
      request,
      // eslint-disable-next-line @typescript-eslint/ban-ts-comment
      // @ts-expect-error
      responseSchema
    );
    if (err != null) throw err;
    return res?.channels as ChannelPayload[];
  }

  async retrieve(keyOrName: string): Promise<ChannelPayload>;

  async retrieve(...keysOrNames: Array<string | string[]>): Promise<ChannelPayload[]>;

  async retrieve(
    ...keysOrNames: Array<string | string[]>
  ): Promise<ChannelPayload | ChannelPayload[]> {
    const single = keysOrNames.length === 1 && typeof keysOrNames[0] === "string";
    const res = await this.execute(keysOrNames.length > 0 ? { keysOrNames } : {});
    if (!single) return res;
    if (res.length === 0) throw new ValidationError("Channel not found");
    if (res.length > 1) throw new ValidationError("Multiple channels found");
    return res[0];
  }
}
