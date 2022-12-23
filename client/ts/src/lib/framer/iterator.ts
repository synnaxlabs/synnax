import { EOF, ErrorPayloadSchema } from "@synnaxlabs/freighter";
import type { Stream, StreamClient } from "@synnaxlabs/freighter";
import { z } from "zod";

import Registry from "../channel/registry";
import {
  TimeRange,
  TimeSpan,
  TimeStamp,
  UnparsedTimeSpan,
  UnparsedTimeStamp,
} from "../telem";

import { ArrayPayload, FramePayload, framePayloadSchema } from "./payload";

export const AUTO_SPAN = new TimeSpan(-1);

enum Command {
  Open = 0,
  Next = 1,
  Prev = 2,
  SeekFirst = 3,
  SeekLast = 4,
  SeekLE = 5,
  SeekGE = 6,
  Valid = 7,
  Error = 8,
}

enum ResponseVariant {
  None = 0,
  Ack = 1,
  Data = 2,
}

const RequestSchema = z.object({
  command: z.nativeEnum(Command),
  span: z.instanceof(TimeSpan).optional(),
  range: z.instanceof(TimeRange).optional(),
  stamp: z.instanceof(TimeStamp).optional(),
  keys: z.string().array().optional(),
});

type Request = z.infer<typeof RequestSchema>;

const ResponseSchema = z.object({
  variant: z.nativeEnum(ResponseVariant),
  ack: z.boolean(),
  command: z.nativeEnum(Command),
  error: ErrorPayloadSchema.optional(),
  frame: framePayloadSchema.optional(),
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
  private static readonly ENDPOINT = "/frame/iterate";
  private readonly client: StreamClient;
  private stream: Stream<Request, Response> | undefined;
  private readonly aggregate: boolean = false;
  values: FramePayload[] = [];

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
  async open(tr: TimeRange, keys: string[]): Promise<void> {
    this.stream = await this.client.stream(
      CoreIterator.ENDPOINT,
      RequestSchema,
      // eslint-disable-next-line @typescript-eslint/ban-ts-comment
      // @ts-expect-error
      ResponseSchema
    );
    await this.execute({ command: Command.Open, keys, range: tr });
    this.values = [];
  }

  /**
   * Reads the next time span of telemetry for each channel in the iterator.
   *
   * @param span - The time span to read. A negative span is equivalent
   * to calling prev with the absolute value of the span. If the span is
   * AUTO_SPAN, the iterator will automatically determine the span to read.
   * This is useful for iterating over an entire range efficiently.
   *
   * @returns false if a segment satisfying the request can't be found for a
   * particular channel or the iterator has accumulated an error.
   */
  async next(span: UnparsedTimeSpan): Promise<boolean> {
    return await this.execute({ command: Command.Next, span: new TimeSpan(span) });
  }

  /**
   * Reads the previous time span of telemetry for each channel in the iterator.
   *
   * @param span - The time span to read. A negative span is equivalent
   * to calling next with the absolute value of the span. If the span is
   * AUTO_SPAN, the iterator will automatically determine the span to read.
   * This is useful for iterating over an entire range efficiently.
   *
   * @returns false if a segment satisfying the request can't be found for a particular
   * channel or the iterator has accumulated an error.
   */
  async prev(span: UnparsedTimeSpan): Promise<boolean> {
    return await this.execute({ command: Command.Prev, span: new TimeSpan(span) });
  }

  /**
   * Seeks the iterator to the first segment in the time range, but does not read
   * it. Also invalidates the iterator. The iterator will not be considered valid
   * until a call to next or prev.
   *
   * @returns false if the iterator is not pointing to a valid segment for a particular
   * channel or has accumulated an error.
   */
  async seekFirst(): Promise<boolean> {
    return await this.execute({ command: Command.SeekFirst });
  }

  /** Seeks the iterator to the last segment in the time range, but does not read it.
   * Also invalidates the iterator. The iterator will not be considered valid
   * until a call to next or prev.
   *
   * @returns false if the iterator is not pointing to a valid segment for a particular
   * channel or has accumulated an error.
   */
  async seekLast(): Promise<boolean> {
    return await this.execute({ command: Command.SeekLast });
  }

  /**
   * Seeks the iterator to the first segment whose start is less than or equal to
   * the provided timestamp. Also invalidates the iterator. The iterator will not be
   * considered valid until a call to next or prev.
   *
   * @returns false if the iterator is not pointing to a valid segment for a particular
   * channel or has accumulated an error.
   */
  async seekLE(stamp: UnparsedTimeStamp): Promise<boolean> {
    return await this.execute({ command: Command.SeekLE, stamp: new TimeStamp(stamp) });
  }

  /**
   * Seeks the iterator to the first segment whose start is greater than or equal to
   * the provided timestamp. Also invalidates the iterator. The iterator will not be
   * considered valid until a call to next or prev.
   *
   * @returns false if the iterator is not pointing to a valid segment for a particular
   * channel or has accumulated an error.
   */
  async seekGE(stamp: UnparsedTimeStamp): Promise<boolean> {
    return await this.execute({ command: Command.SeekGE, stamp: new TimeStamp(stamp) });
  }

  /**
   * @returns true if the iterator value contains a valid segment, and fale otherwise.
   * valid most commonly returns false when the iterator is exhausted or has
   * accumulated an error.
   */
  async valid(): Promise<boolean> {
    return await this.execute({ command: Command.Valid });
  }

  /**
   * Closes the iterator. An iterator MUST be closed after use, and this method
   * should probably be placed in a 'finally' block. If the iterator is not closed,
   * it may leak resources.
   */
  async close(): Promise<void> {
    if (this.stream == null)
      throw new Error("iterator.open() must be called before any other method");
    this.stream.closeSend();
    const [, exc] = await this.stream.receive();
    if (exc == null)
      throw new Error("received unexpected response from core's iterator");
    if (!(exc instanceof EOF)) throw exc;
  }

  private async execute(request: Request): Promise<boolean> {
    if (this.stream == null)
      throw new Error("iterator.open() must be called before any other method");
    const err = this.stream.send(request);
    if (err != null) throw err;
    if (!this.aggregate) this.values = [];
    for (;;) {
      const [res, err] = await this.stream.receive();
      // eslint-disable-next-line @typescript-eslint/no-throw-literal
      if (err != null || res == null) throw err;
      if (res.variant === ResponseVariant.Ack) return res.ack;
      if (res.frame != null) this.values.push(res.frame);
    }
  }
}

export class TypedIterator extends CoreIterator {
  channels: Registry;

  constructor(client: StreamClient, channels: Registry, aggregate = false) {
    super(client, aggregate);
    this.channels = channels;
  }

  async value(): Promise<Record<string, ArrayPayload>> {
    const result: Record<string, ArrayPayload> = {};
    this.values.forEach((frame) => {
      if (frame.keys == null) return;
      frame.keys.forEach((key, i) => {
        if (frame.arrays == null) return;
        const v = frame.arrays[i];
        if (key in result) {
          result[key] = concatTypedArrays(v, result[key]);
        } else {
          result[key] = v;
        }
      });
    });
    return result;
  }
}

const concatTypedArrays = (a: ArrayPayload, b: ArrayPayload): ArrayPayload => {
  if (a.dataType == null || b.dataType == null) {
    throw new Error("Cannot concat arrays with unknown data type");
  }
  if (a.dataType.valueOf() !== b.dataType.valueOf()) {
    throw new Error("Cannot concat arrays with different data types");
  }
  const c = new Uint8Array(a.data.length + b.data.length);
  c.set(a.data, 0);
  c.set(b.data, a.data.length);
  return { timeRange: a.timeRange, dataType: a.dataType, data: c };
};
