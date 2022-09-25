import { UnaryClient } from '@synnaxlabs/freighter';
import { z } from 'zod';

import { DataType, Rate } from '../telem';
import Transport from '../transport';

import { ChannelPayload, ChannelPayloadSchema } from './ChannelPayload';

const RequestSchema = z.object({
  channel: ChannelPayloadSchema,
  count: z.number(),
});

type Request = z.infer<typeof RequestSchema>;

const ResponseSchema = z.object({
  channels: ChannelPayloadSchema.array(),
});

type Response = z.infer<typeof ResponseSchema>;

export default class ChannelCreator {
  private static ENDPOINT = '/channel/create';
  private client: UnaryClient;

  constructor(transport: Transport) {
    this.client = transport.getClient();
  }

  async create({
    name = '',
    nodeID = 0,
    rate = new Rate(0),
    dataType = DataType.Unknown,
  }: Omit<ChannelPayload, 'density' | 'key'>): Promise<ChannelPayload> {
    return (
      await this.execute({
        channel: { name, nodeID, rate, dataType },
        count: 1,
      })
    ).channels[0];
  }

  async createMany({
    name = '',
    nodeID = 0,
    rate = new Rate(0),
    dataType = DataType.Unknown,
    count = 1,
  }: Omit<ChannelPayload, 'density' | 'key'> & { count: number }): Promise<
    ChannelPayload[]
  > {
    return (
      await this.execute({
        channel: { name, nodeID, rate, dataType },
        count,
      })
    ).channels;
  }

  private async execute(request: Request): Promise<Response> {
    const [res, err] = await this.client.send(
      ChannelCreator.ENDPOINT,
      request,
      // eslint-disable-next-line @typescript-eslint/ban-ts-comment
      // @ts-ignore
      ResponseSchema
    );
    if (err) {
      throw err;
    }
    return res as Response;
  }
}
