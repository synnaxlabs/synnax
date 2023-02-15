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
  keys: z.string().array().optional(),
  nodeId: z.number().optional(),
  names: z.string().array().optional(),
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

  async retrieve({
    key,
    name,
  }: {
    key?: string;
    name?: string;
  }): Promise<ChannelPayload> {
    if (key == null && name == null)
      // eslint-disable-next-line @typescript-eslint/no-throw-literal
      throw new ValidationError("Must provide either key or name");
    const req: Request = {
      keys: key != null ? [key] : undefined,
      names: name != null ? [name] : undefined,
    };
    const res = await this.execute(req);
    if (res.length === 0) throw new ValidationError("Channel not found");
    if (res.length > 1) throw new ValidationError("Multiple channels found");
    return res[0];
  }

  async filter({
    keys,
    nodeId,
    names,
  }: {
    keys?: string[];
    nodeId?: number;
    names?: string[];
  }): Promise<ChannelPayload[]> {
    return await this.execute({ keys, nodeId, names });
  }

  async retrieveAll(): Promise<ChannelPayload[]> {
    return await this.execute({});
  }
}
