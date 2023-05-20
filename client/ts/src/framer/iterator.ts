// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { EOF, ErrorPayloadSchema } from "@synnaxlabs/freighter";
import type { Stream, StreamClient } from "@synnaxlabs/freighter";
import {
  TimeRange,
  TimeSpan,
  TimeStamp,
  UnparsedTimeSpan,
  UnparsedTimeStamp,
} from "@synnaxlabs/x";
import { z } from "zod";

import { Frame } from "./frame";
import { framePayload } from "./payload";

import { ChannelKeys } from "@/channel/payload";
import { GeneralError, UnexpectedError } from "@/errors";

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

const NOT_OPEN = new GeneralError(
  "iterator.open() must be called before any other method"
);

const request = z.object({
  command: z.nativeEnum(Command),
  span: z.instanceof(TimeSpan).optional(),
  range: z.instanceof(TimeRange).optional(),
  stamp: z.instanceof(TimeStamp).optional(),
  keys: z.number().array().optional(),
});

type Request = z.infer<typeof request>;

const response = z.object({
  variant: z.nativeEnum(ResponseVariant),
  ack: z.boolean(),
  command: z.nativeEnum(Command),
  error: ErrorPayloadSchema.optional(),
  frame: framePayload.optional(),
});

type Response = z.infer<typeof response>;

/**
 * Used to iterate over a clusters telemetry in time-order. It should not be
 * instantiated directly, and should instead be instantiated via the SegmentClient.
 *
 * Using an iterator is ideal when querying/processing large ranges of data, but
 * is relatively complex and difficult to use. If you're looking to retrieve
 *  telemetry between two timestamps, see the SegmentClient.read method.
 */
export class Iterator {
  private static readonly ENDPOINT = "/frame/iterate";
  private readonly client: StreamClient;
  private stream: Stream<Request, Response> | undefined;
  private readonly aggregate: boolean = false;
  value: Frame;

  constructor(client: StreamClient, aggregate = false) {
    this.client = client;
    this.aggregate = aggregate;
    this.value = new Frame();
  }

  /**
   * Opens the iterator, configuring it to iterate over the telemetry in the
   * channels with the given keys within the provided time range.
   *
   * @param tr - The time range to iterate over.
   * @param keys - The keys of the channels to iterate over.
   */
  async open(tr: TimeRange, keys: ChannelKeys): Promise<void> {
    this.stream = await this.client.stream(
      Iterator.ENDPOINT,
      request,
      // eslint-disable-next-line @typescript-eslint/ban-ts-comment
      // @ts-expect-error
      response
    );
    await this.execute({ command: Command.Open, keys, range: tr });
    this.value = new Frame();
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
    if (this.stream == null) throw NOT_OPEN;
    this.stream.closeSend();
    const [, exc] = await this.stream.receive();
    if (exc == null)
      throw new UnexpectedError("received null response on iterator closure");
    if (!(exc instanceof EOF)) throw exc;
  }

  private resetValue(): void {
    if (this.value == null || !this.aggregate) this.value = new Frame();
  }

  private async execute(request: Request): Promise<boolean> {
    this.resetValue();
    if (this.stream == null) throw NOT_OPEN;
    const err = this.stream.send(request);
    if (err != null) throw err;
    while (true) {
      const [res, err] = await this.stream.receive();
      if (err != null) throw err;
      if (res == null)
        throw new UnexpectedError("received null response from iterator");
      if (res.variant === ResponseVariant.Ack) return res.ack;
      if (res.frame != null) this.value = this.value.concatF(new Frame(res.frame));
    }
  }
}
