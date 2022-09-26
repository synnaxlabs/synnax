import {
  EOF,
  ErrorPayloadSchema,
  Stream,
  StreamClient,
} from '@synnaxlabs/freighter';
import { z } from 'zod';

import { ChannelPayload } from '../channel/payload';
import Registry from '../channel/registry';
import { TimeRange } from '../telem';

import { SegmentPayload, SegmentPayloadSchema } from './payload';
import TypedSegment from './typed';

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

/**
 * Used to iterate over a clusters telemetry in time-order. It should not be
 * instantiated directly, and should instead be instantiated via the SegmentClient.
 *
 * Using an iterator is ideal when querying/processing large ranges of data, but
 * is relatively complex and difficult to use. If you're looking to retrieve
 *  telemetry between two timestamps, see the SegmentClient.read method.
 */
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

  /**
   * Opens the iterator, configuring it to iterate over the telemetry in the
   * channels with the given keys within the provided time range.
   *
   * @param tr - The time range to iterate over.
   * @param keys - The keys of the channels to iterate over.
   */
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

  /**
   * Reads the next segment for each channel in the iterator.
   *
   * @returns false if the next segment can't be found for one or more channels or
   * the iterator has accumulated an error.
   */
  async next(): Promise<boolean> {
    return this.execute({ command: Command.Next });
  }

  /**
   * Reads the previous segment for each channel in the iterator.
   *
   * @returns false if the next segment can't be found for one or more channels or
   * the iterator has accumulated an error.
   */
  async prev(): Promise<boolean> {
    return this.execute({ command: Command.Prev });
  }

  /**
   * Seeks to the beginning of the time range and reads the first segment of each
   * channel in the iterator.
   *
   * @returns false if no segments exists in the time range for a particular channel
   * or the iterator has accumulated an error.
   */
  async first(): Promise<boolean> {
    return this.execute({ command: Command.First });
  }

  /**
   * Seeks to the end of the time range and reads the last segment of each channel
   * in the iterator.
   *
   * @returns false if no segments exists in the time range for a particular channel,
   * or the iterator has accumulated an error.
   */
  async last(): Promise<boolean> {
    return this.execute({ command: Command.Last });
  }

  /**
   * Reads the next time span of telemetry for each channel in the iterator.
   *
   * @returns false if a segment satisfying the request can't be found for a
   * particular channel or the iterator has accumulated an error.
   */
  async nextSpan(span: number): Promise<boolean> {
    return this.execute({ command: Command.NextSpan, span });
  }

  /**
   * Reads the previous time span of telemetry for each channel in the iterator.
   *
   * @returns false if a segment satisfying the request can't be found for a particular
   * channel or the iterator has accumulated an error.
   */
  async prevSpan(span: number): Promise<boolean> {
    return this.execute({ command: Command.PrevSpan, span });
  }

  /**
   * Seeks the iterator to the start of the time range and reads the telemetry within
   * the range for each channel.
   *
   * @returns: False if a segment satisfying the request can't be found for a particular
   * channel or the iterator has accumulated an error.
   */
  async nextRange(range: TimeRange): Promise<boolean> {
    return this.execute({ command: Command.NextRange, range });
  }

  /**
   * Seeks the iterator to the first segment in the time range, but does not read
   * it. Also invalidates the iterator. The iterator will not be considered valid
   * until a call to first, last, next, prev, prev_span, next_span, or next_range.
   *
   * @returns false if the iterator is not pointing to a valid segment for a particular
   * channel or has accumulated an error.
   */
  async seekFirst(): Promise<boolean> {
    return this.execute({ command: Command.SeekFirst });
  }

  /** Seeks the iterator to the last segment in the time range, but does not read it.
   * Also invalidates the iterator. The iterator will not be considered valid
   * until a call to first, last, next, prev, prev_span, next_span, or next_range.
   *
   * @returns false if the iterator is not pointing to a valid segment for a particular
   * channel or has accumulated an error.
   */
  async seekLast(): Promise<boolean> {
    return this.execute({ command: Command.SeekLast });
  }

  /**
   * Seeks the iterator to the first segment whose start is less than or equal to
   * the provided timestamp. Also invalidates the iterator. The iterator will not be
   * considered valid until a call to first, last, next, prev, prev_span, next_span, or next_range.
   *
   * @returns false if the iterator is not pointing to a valid segment for a particular
   * channel or has accumulated an error.
   */
  async seekLT(stamp: number): Promise<boolean> {
    return this.execute({ command: Command.SeekLT, stamp });
  }

  /**
   * Seeks the iterator to the first segment whose start is greater than or equal to
   * the provided timestamp. Also invalidates the iterator. The iterator will not be
   * considered valid until a call to first, last, next, prev, prev_span, next_span, or next_range.
   *
   * @returns false if the iterator is not pointing to a valid segment for a particular
   * channel or has accumulated an error.
   */
  async seekGE(stamp: number): Promise<boolean> {
    return this.execute({ command: Command.SeekGE, stamp });
  }

  /**
   * @returns true if the iterator value contains a valid segment, and fale otherwise.
   * valid most commonly returns false when the iterator is exhausted or has
   * accumulated an error.
   */
  async valid(): Promise<boolean> {
    return this.execute({ command: Command.Valid });
  }

  /**
   * Closes the iterator. An iterator MUST be closed after use, and this method
   * should probably be placed in a 'finally' block. If the iterator is not closed,
   * it may leak resources.
   */
  async close() {
    if (!this.stream)
      throw new Error('iterator.open() must be called before any other method');
    this.stream.closeSend();
    const [, exc] = await this.stream.receive();
    if (!(exc instanceof EOF)) throw exc;
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

export class TypedIterator extends CoreIterator {
  channels: Registry;

  constructor(client: StreamClient, channels: Registry, aggregate = false) {
    super(client, aggregate);
    this.channels = channels;
  }

  async value(): Promise<Record<string, TypedSegment>> {
    const result: Record<string, TypedSegment> = {};
    this.values.sort((a, b) => a.start.valueOf() - b.start.valueOf());
    const keys = this.values.map((v) => v.channelKey);
    const channels = await this.channels.getN(...keys);
    this.values.forEach((v) => {
      const sugared = new TypedSegment(
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
