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
  Open = 0,
  Next = 1,
  Prev = 2,
  First = 3,
  Last = 4,
  NextSpan = 5,
  PrevSpan = 6,
  NextRange = 7,
  Valid = 8,
  Error = 9,
  SeekFirst = 10,
  SeekLast = 11,
  SeekLT = 12,
  SeekGE = 13,
}

enum ResponseVariant {
  None = 0,
  Ack = 1,
  Data = 2,
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
    await this.execute({ command: Command.Open, keys, range: tr });
    this.values = [];
  }

  async next(): Promise<boolean> {
    return this.execute({ command: Command.Next });
  }

  async prev(): Promise<boolean> {
    return this.execute({ command: Command.Prev });
  }

  async first(): Promise<boolean> {
    return this.execute({ command: Command.First });
  }

  async last(): Promise<boolean> {
    return this.execute({ command: Command.Last });
  }

  async nextSpan(span: number): Promise<boolean> {
    return this.execute({ command: Command.NextSpan, span });
  }

  async prevSpan(span: number): Promise<boolean> {
    return this.execute({ command: Command.PrevSpan, span });
  }

  async nextRange(range: TimeRange): Promise<boolean> {
    return this.execute({ command: Command.NextRange, range });
  }

  async seekFirst(): Promise<boolean> {
    return this.execute({ command: Command.SeekFirst });
  }

  async seekLast(): Promise<boolean> {
    return this.execute({ command: Command.SeekLast });
  }

  async seekLT(stamp: number): Promise<boolean> {
    return this.execute({ command: Command.SeekLT, stamp });
  }

  async seekGE(stamp: number): Promise<boolean> {
    return this.execute({ command: Command.SeekGE, stamp });
  }

  async valid(): Promise<boolean> {
    return this.execute({ command: Command.Valid });
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
      if (res.variant == ResponseVariant.Ack) return res.ack;
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
      if (v.channelKey in result) {
        result[v.channelKey].extend(sugared);
      } else {
        result[v.channelKey] = sugared;
      }
    });
    return result;
  }
}
