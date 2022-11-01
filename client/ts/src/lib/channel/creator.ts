import { UnaryClient } from '@synnaxlabs/freighter';
import { z } from 'zod';

import { DataType, Rate, UnparsedDataType, UnparsedRate } from '../telem';
import Transport from '../transport';

import { ChannelPayload, channelPayloadSchema } from './payload';

const RequestSchema = z.object({
  channel: channelPayloadSchema,
  count: z.number(),
});

type Request = z.infer<typeof RequestSchema>;

const ResponseSchema = z.object({
  channels: channelPayloadSchema.array(),
});

type Response = z.infer<typeof ResponseSchema>;

export type CreateChannelProps = {
  rate: UnparsedRate;
  dataType: UnparsedDataType;
  name?: string;
  nodeId?: number;
};

export default class Creator {
  private static ENDPOINT = '/channel/create';
  private client: UnaryClient;

  constructor(transport: Transport) {
    this.client = transport.postClient();
  }

  async create(props: CreateChannelProps): Promise<ChannelPayload> {
    const [channel] = await this.createMany({ ...props, count: 1 });
    return channel;
  }

  async createMany({
    rate,
    dataType,
    name = '',
    nodeId = 0,
    count = 1,
  }: CreateChannelProps & { count: number }): Promise<ChannelPayload[]> {
    return (
      await this.execute({
        channel: {
          name,
          nodeId,
          rate: new Rate(rate),
          dataType: new DataType(dataType),
        },
        count,
      })
    ).channels;
  }

  private async execute(request: Request): Promise<Response> {
    const [res, err] = await this.client.send(
      Creator.ENDPOINT,
      request,
      // eslint-disable-next-line @typescript-eslint/ban-ts-comment
      // @ts-ignore
      ResponseSchema
    );
    if (err) throw err;
    return res as Response;
  }
}
