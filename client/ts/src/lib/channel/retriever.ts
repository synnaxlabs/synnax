import type { UnaryClient } from "@synnaxlabs/freighter";
import { z } from "zod";

import { ValidationError } from "../errors";
import Transport from "../transport";

import { ChannelPayload, channelPayloadSchema } from "./payload";

const requestSchema = z.object({
  keys: z.string().array().optional(),
  nodeId: z.number().optional(),
  names: z.string().array().optional(),
});

type Request = z.infer<typeof requestSchema>;

const responseSchema = z.object({
  channels: channelPayloadSchema.array(),
});

export default class Retriever {
  private static readonly ENDPOINT = "/channel/retrieve";
  private readonly client: UnaryClient;

  constructor(transport: Transport) {
    this.client = transport.getClient();
  }

  private async execute(request: Request): Promise<ChannelPayload[]> {
    const [res, err] = await this.client.send(
      Retriever.ENDPOINT,
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
