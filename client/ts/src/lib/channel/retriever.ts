import { UnaryClient } from '@synnaxlabs/freighter';
import { z } from 'zod';
import { tokenMiddleware } from '../auth';

import Transport from '../transport';

import { ChannelPayload, ChannelPayloadSchema } from './payload';

const RequestSchema = z.object({
  keys: z.string().array().optional(),
  nodeId: z.number().optional(),
  names: z.string().array().optional(),
});

type Request = z.infer<typeof RequestSchema>;

const ResponseSchema = z.object({
  channels: ChannelPayloadSchema.array(),
});

export default class Retriever {
  private static ENDPOINT = '/channel/retrieve';
  private client: UnaryClient;

  constructor(transport: Transport) {
    this.client = transport.getClient();
  }

  private async execute(request: Request): Promise<ChannelPayload[]> {
    const [res, err] = await this.client.send(
      Retriever.ENDPOINT,
      request,
      // eslint-disable-next-line @typescript-eslint/ban-ts-comment
      // @ts-ignore
      ResponseSchema
    );
    if (err) {
      throw err;
    }
    return res?.channels as ChannelPayload[];
  }

  async retrieveByKeys(...keys: string[]): Promise<ChannelPayload[]> {
    return await this.execute({ keys });
  }

  async retrieveByNames(...names: string[]): Promise<ChannelPayload[]> {
    return await this.execute({ names });
  }

  async retrieveByNodeID(nodeId: number): Promise<ChannelPayload[]> {
    return await this.execute({ nodeId: nodeId });
  }
}
