import {
  ErrorPayloadSchema,
  Stream,
  StreamClient,
} from '@synnaxlabs/freighter';
import { z } from 'zod';

import { ChannelPayload } from '../channel/payload';
import Registry from '../channel/registry';
import { TimeRange } from '../telem';

import { SegmentPayload, SegmentPayloadSchema } from './payload';
import Sugared from './sugared';

enum Command {
  OPEN = 0,
  NEXT = 1,
  PREV = 2,
  FIRST = 3,
  LAST = 4,
  NEXT_SPAN = 5,
  PREV_SPAN = 6,
  NEXT_RANGE = 7,
  VALID = 8,
  ERROR = 9,
  SEEK_FIRST = 10,
  SEEK_LAST = 11,
  SEEK_LT = 12,
  SEEK_GE = 13,
}

enum ResponseVariant {
  NONE = 0,
  ACK = 1,
  DATA = 2,
}

const RequestSchema = z.object({
  command: z.nativeEnum(Command),
  span: z.number().optional(),
  range: z.instanceof(TimeRange).optional(),
  stamp: z.number().optional(),
  keys: z.string().array().optional(),
});

type Request = z.infer<typeof RequestSchema>;

const ResponseSchema = z.object({
  variant: z.nativeEnum(ResponseVariant),
  ack: z.boolean(),
  command: z.nativeEnum(Command),
  error: ErrorPayloadSchema.optional(),
  segments: SegmentPayloadSchema.array().nullable(),
});

type Response = z.infer<typeof ResponseSchema>;

export class CoreIterator {
  private static ENDPOINT = '/segment/iterate';
  private client: StreamClient;
  private stream: Stream<Request, Response> | undefined;
  private readonly aggregate: boolean = false;
  values: SegmentPayload[] = [];

  constructor(client: StreamClient, aggregate = false) {
    this.client = client;
    this.aggregate = aggregate;
  }

  async open(tr: TimeRange, keys: string[]) {
    this.stream = await this.client.stream(
      CoreIterator.ENDPOINT,
      RequestSchema,
      // eslint-disable-next-line @typescript-eslint/ban-ts-comment
      // @ts-ignore
      ResponseSchema
    );
    await this.execute({ command: Command.OPEN, keys, range: tr });
    this.values = [];
  }

  async next(): Promise<boolean> {
    return this.execute({ command: Command.NEXT });
  }

  async prev(): Promise<boolean> {
    return this.execute({ command: Command.PREV });
  }

  async first(): Promise<boolean> {
    return this.execute({ command: Command.FIRST });
  }

  async last(): Promise<boolean> {
    return this.execute({ command: Command.LAST });
  }

  async nextSpan(span: number): Promise<boolean> {
    return this.execute({ command: Command.NEXT_SPAN, span });
  }

  async prevSpan(span: number): Promise<boolean> {
    return this.execute({ command: Command.PREV_SPAN, span });
  }

  async nextRange(range: TimeRange): Promise<boolean> {
    return this.execute({ command: Command.NEXT_RANGE, range });
  }

  async seekFirst(): Promise<boolean> {
    return this.execute({ command: Command.SEEK_FIRST });
  }

  async seekLast(): Promise<boolean> {
    return this.execute({ command: Command.SEEK_LAST });
  }

  async seekLT(stamp: number): Promise<boolean> {
    return this.execute({ command: Command.SEEK_LT, stamp });
  }

  async seekGE(stamp: number): Promise<boolean> {
    return this.execute({ command: Command.SEEK_GE, stamp });
  }

  async valid(): Promise<boolean> {
    return this.execute({ command: Command.VALID });
  }

  async close() {
    if (!this.stream) return;
    this.stream?.closeSend();
    const [, exc] = await this.stream.receive();
    if (exc?.message != 'EOF') throw exc;
  }

  private async execute(request: Request): Promise<boolean> {
    if (!this.stream)
      throw new Error('iterator.open() must be called before any other method');
    const err = this.stream.send(request);
    if (err) throw err;
    if (!this.aggregate) this.values = [];
    for (;;) {
      const [res, err] = await this.stream.receive();
      if (err || !res) throw err;
      if (res.variant == ResponseVariant.ACK) return res.ack;
      if (res.segments) this.values.push(...res.segments);
    }
  }
}

export class SugaredIterator extends CoreIterator {
  channels: Registry;

  constructor(client: StreamClient, channels: Registry, aggregate = false) {
    super(client, aggregate);
    this.channels = channels;
  }

  async value(): Promise<Record<string, Sugared>> {
    const result: Record<string, Sugared> = {};
    this.values.sort((a, b) => a.start.valueOf() - b.start.valueOf());
    const keys = this.values.map((v) => v.channelKey);
    const channels = await this.channels.getN(...keys);
    this.values.forEach((v) => {
      const sugared = new Sugared(
        channels.find((c) => c.key == v.channelKey) as ChannelPayload,
        v
      );
      console.log(sugared.start, sugared.end);
      if (v.channelKey in result) {
        result[v.channelKey].extend(sugared);
      } else {
        result[v.channelKey] = sugared;
      }
    });
    return result;
  }
}
