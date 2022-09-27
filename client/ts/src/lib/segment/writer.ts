import {
  decodeError,
  EOF,
  ErrorPayloadSchema,
  Stream,
  StreamClient,
} from '@synnaxlabs/freighter';
import { z } from 'zod';

import ChannelRegistry from '../channel/registry';
import { GeneralError, UnexpectedError, ValidationError } from '../errors';
import { Size, TimeStamp, TypedArray, UnparsedTimeStamp } from '../telem';

import { SegmentPayload, SegmentPayloadSchema } from './payload';
import Splitter from './splitter';
import TypedSegment from './typed';
import { ContiguityValidator, ScalarTypeValidator } from './validator';

const RequestSchema = z.object({
  openKeys: z.string().array().optional(),
  segments: SegmentPayloadSchema.array().optional(),
});

type Request = z.infer<typeof RequestSchema>;

const ResponseSchema = z.object({
  ack: z.boolean(),
  error: ErrorPayloadSchema.optional(),
});

type Response = z.infer<typeof ResponseSchema>;

const NOT_OPEN = new GeneralError(
  'Writer has not been opened. Please open before calling write() or close().'
);

/**
 * CoreWriter is used to write telemetry to a set of channels in time-order. It
 * should not be instantiated directly, but rather through a {@link SegmentClient}.
 *
 * Using a writer is ideal when writing large volumes of data (such as recording
 * telemetry from a sensor), but it is relatively complex and challenging to use.
 * If you're looking to write a contiguous block of telemetry, see the {@link SegmentClient}
 * write() method.
 */
export class CoreWriter {
  private static ENDPOINT = '/segment/write';
  private client: StreamClient;
  private stream: Stream<Request, Response> | undefined;
  private keys: string[];

  constructor(client: StreamClient) {
    this.client = client;
    this.keys = [];
  }

  /**
   * Opens the writer, acquiring an exclusive lock on the given channels for
   * the duration of the writer's lifetime. open must be called before any other
   * writer methods.
   *
   * @param keys - A list of keys representing the channels the writer will write
   * to.
   */
  async open(keys: string[]) {
    this.keys = keys;
    this.stream = await this.client.stream(
      CoreWriter.ENDPOINT,
      // eslint-disable-next-line @typescript-eslint/ban-ts-comment
      // @ts-ignore
      RequestSchema,
      ResponseSchema
    );
    this.stream.send({ openKeys: keys });
    const [res, err] = await this.stream.receive();
    if (err) throw err;
    if (!res?.ack)
      throw new UnexpectedError(
        'Writer failed to positively acknowledge open request. This is a bug. Please report it.'
      );
  }

  /**
   * Validates and writes the given segments to the database. The provided segments
   * must:
   *
   *   1. Be in time order (on a per-channel basis)
   *   2. Have channel keys in the set of keys this writer was opened with.
   *   3. Have non-zero length data with the correct data type.
   *
   * @param segments - A list of segments to write to the database.
   * @returns false if the writer has accumulated an error. In this case,
   * the caller should stop executing requests and close the writer.
   */
  async write(segments: SegmentPayload[]): Promise<boolean> {
    if (!this.stream) throw NOT_OPEN;
    if (this.stream.received()) return false;

    this.checkKeys(segments);
    const err = this.stream.send({ segments });
    if (err) throw err;
    return true;
  }

  /**
   * Closes the writer, raising any accumulated error encountered during operation.
   * A writer MUST be closed after use, and this method should probably be placed
   * in a 'finally' block. If the writer is not closed, the database will not release
   * the exclusive lock on the channels, preventing any other callers from
   * writing to them. It also might leak resources and threads.
   */
  async close() {
    if (!this.stream) throw NOT_OPEN;
    this.stream.closeSend();
    const [res, err] = await this.stream.receive();
    if (!err && res?.error) throw decodeError(res.error);
    if (!(err instanceof EOF)) throw err;
  }

  private checkKeys(segments: SegmentPayload[]) {
    // check that the channel key of each segment is in the open keys
    segments
      .map((segment) => segment.channelKey)
      .forEach((key) => {
        if (!this.keys.includes(key))
          throw new ValidationError({
            field: 'channelKey',
            message: `Channel key ${key} is not in the list of keys this writer was opened with.`,
          });
      });
  }
}

/**
 * TypedWriter is used to write telemetry to a set of channels in time-order. It
 * should not be instantiated directly, but rather through a {@link SegmentClient}.
 *
 * Using a writer is ideal when writing large volumes of data (such as recording
 * telemetry from a sensor), but it is relatively complex and challenging to use.
 * If you're looking to write a contiguous block of telemetry, see the {@link SegmentClient}
 * write() method.
 */
export class TypedWriter {
  private core: CoreWriter;
  private splitter: Splitter;
  private channels: ChannelRegistry;
  private scalarTypeValidator: ScalarTypeValidator;
  private contiguityValidator: ContiguityValidator;

  constructor(client: StreamClient, channels: ChannelRegistry) {
    this.core = new CoreWriter(client);
    this.channels = channels;
    this.scalarTypeValidator = new ScalarTypeValidator();
    this.contiguityValidator = new ContiguityValidator({
      allowNoHighWaterMark: true,
      allowGaps: false,
      allowOverlap: false,
    });
    this.splitter = new Splitter(Size.Megabytes(4));
  }

  /**
   * Opens the writer, acquiring an exclusive lock on the given channels for
   * the duration of the writer's lifetime. open must be called before any other
   * writer methods.
   *
   * @param keys - A list of keys representing the channels the writer will write
   * to.
   */
  async open(keys: string[]) {
    await this.core.open(keys);
  }

  /**
   * Writes the given telemetry to the database.
   *
   * @param to - They key of the channel to write to. This must be in the set of
   * keys this writer was opened with.
   * @param start - The start time of the telemetry. This must be equal to
   * the end of the previous segment written to the channel (unless it's the first
   * write to that channel).
   * @param data - The telemetry to write. This must be a valid type for the channel.
   * @returns false if the writer has accumulated an error. In this case,
   * the caller should stop executing requests and close the writer.
   */
  async write(
    to: string,
    start: UnparsedTimeStamp,
    data: TypedArray
  ): Promise<boolean> {
    const ch = await this.channels.get(to);
    this.scalarTypeValidator.validate(data, ch.dataType);
    const pld: SegmentPayload = {
      channelKey: to,
      start: new TimeStamp(start),
      data: new Uint8Array(data.buffer),
    };
    const segment = new TypedSegment(ch, pld);
    this.contiguityValidator.validate(segment);
    const segments = this.splitter.split(segment);
    return this.core.write(segments.map((s) => s.payload));
  }

  /**
   * Closes the writer, raising any accumulated error encountered during operation.
   * A writer MUST be closed after use, and this method should probably be placed
   * in a 'finally' block. If the writer is not closed, the database will not release
   * the exclusive lock on the channels, preventing any other callers from
   * writing to them. It also might leak resources and threads.
   */
  async close() {
    await this.core.close();
  }
}
