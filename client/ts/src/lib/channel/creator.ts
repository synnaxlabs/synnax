import type { UnaryClient } from "@synnaxlabs/freighter";
import { z } from "zod";

import { UnparsedDataType, UnparsedRate } from "../telem";
import Transport from "../transport";

import { ChannelPayload, channelPayloadSchema } from "./payload";

const RequestSchema = z.object({
  channels: channelPayloadSchema.array(),
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

export default class Creator {
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
      Creator.ENDPOINT,
      request,
      // eslint-disable-next-line @typescript-eslint/ban-ts-comment
      // @ts-expect-error
      ResponseSchema
    );
    if (err != null) throw err;
    return res as Response;
  }
}
