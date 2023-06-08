// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { errorZ, Stream, StreamClient } from "@synnaxlabs/freighter";
import {
  TimeRange,
  TimeSpan,
  TimeStamp,
  UnparsedTimeSpan,
  UnparsedTimeStamp,
} from "@synnaxlabs/x";
import { z } from "zod";

import { ChannelParams } from "@/channel/payload";
import { ChannelRetriever } from "@/channel/retriever";
import { BackwardFrameAdapter } from "@/framer/adapter";
import { Frame, frameZ } from "@/framer/frame";
import { StreamProxy } from "@/framer/streamProxy";

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

const reqZ = z.object({
  command: z.nativeEnum(Command),
  span: TimeSpan.z.optional(),
  bounds: TimeRange.z.optional(),
  stamp: TimeStamp.z.optional(),
  keys: z.number().array().optional(),
});

type Request = z.infer<typeof reqZ>;

const resZ = z.object({
  variant: z.nativeEnum(ResponseVariant),
  ack: z.boolean(),
  command: z.nativeEnum(Command),
  error: errorZ.optional().nullable(),
  frame: frameZ.optional(),
});

type Response = z.infer<typeof resZ>;

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
  private readonly stream: StreamProxy<Request, Response>;
  private readonly adapter: BackwardFrameAdapter;
  value: Frame;

  private constructor(
    stream: Stream<Request, Response>,
    adapter: BackwardFrameAdapter
  ) {
    this.stream = new StreamProxy("Iterator", stream);
    this.value = new Frame();
    this.adapter = adapter;
  }

  /**
   * Opens the iterator, configuring it to iterate over the telemetry in the
   * channels with the given keys within the provided time range.
   *
   * @param tr - The time range to iterate over.
   * @param keys - The keys of the channels to iterate over.
   */
  static async _open(
    tr: TimeRange,
    channels: ChannelParams[],
    retriever: ChannelRetriever,
    client: StreamClient
  ): Promise<Iterator> {
    const adapter = await BackwardFrameAdapter.open(retriever, channels);
    // @ts-expect-error
    const stream = await client.stream(Iterator.ENDPOINT, reqZ, resZ);
    const iter = new Iterator(stream, adapter);
    await iter.execute({ command: Command.Open, keys: adapter.keys, bounds: tr });
    return iter;
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
  async next(span: UnparsedTimeSpan = AUTO_SPAN): Promise<boolean> {
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
  async prev(span: UnparsedTimeSpan = AUTO_SPAN): Promise<boolean> {
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
    await this.stream.closeAndAck();
  }

  private async execute(request: Request): Promise<boolean> {
    this.stream.send(request);
    while (true) {
      const res = await this.stream.receive();
      if (res.variant === ResponseVariant.Ack) return res.ack;
      this.value = this.adapter.adapt(new Frame(res.frame));
    }
  }
}
