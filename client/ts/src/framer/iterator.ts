// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Stream, type StreamClient } from "@synnaxlabs/freighter";
import {
  type CrudeTimeRange,
  type CrudeTimeSpan,
  type CrudeTimeStamp,
  errors,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { z } from "zod";

import { channel } from "@/channel";
import { ReadAdapter } from "@/framer/adapter";
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
  command: z.enum(Command),
  span: TimeSpan.z.optional(),
  bounds: TimeRange.z.optional(),
  stamp: TimeStamp.z.optional(),
  keys: channel.keyZ.array().optional(),
  chunkSize: z.number().optional(),
});
interface Request extends z.infer<typeof reqZ> {}

const resZ = z.object({
  variant: z.enum(ResponseVariant),
  ack: z.boolean(),
  command: z.enum(Command),
  error: errors.payloadZ.optional().nullable(),
  frame: frameZ.optional(),
});

export interface IteratorConfig {
  /** chunkSize is the maximum number of samples contained per channel in the frame
   * resulting from a call to next with {@link AUTO_SPAN}.
   */
  chunkSize?: number;
}

/**
 * Used to iterate over a clusters telemetry in time-order. It should not be
 * instantiated directly, and should instead be instantiated via the SegmentClient.
 *
 * Using an iterator is ideal when querying/processing large ranges of data, but
 * is relatively complex and difficult to use. If you're looking to retrieve
 *  telemetry between two timestamps, see the SegmentClient.read method.
 */
export class Iterator {
  private readonly stream: StreamProxy<typeof reqZ, typeof resZ>;
  private readonly adapter: ReadAdapter;
  value: Frame;

  private constructor(stream: Stream<typeof reqZ, typeof resZ>, adapter: ReadAdapter) {
    this.stream = new StreamProxy("Iterator", stream);
    this.value = new Frame();
    this.adapter = adapter;
  }

  /**
   * Opens the iterator, configuring it to iterate over the telemetry in the
   * channels with the given keys within the provided time range.
   *
   * @param tr - The time range to iterate over.
   * @param channels - The channels for the iterator to iterate over (can be provided
   * in keys or names).
   * @param retriever - Retriever used to retrieve channel keys from names.
   * @param client - The stream client allowing streaming of iterated data.
   * @param opts - See {@link IteratorConfig}.
   */
  static async _open(
    tr: CrudeTimeRange,
    channels: channel.Params,
    retriever: channel.Retriever,
    client: StreamClient,
    opts: IteratorConfig = {},
  ): Promise<Iterator> {
    const adapter = await ReadAdapter.open(retriever, channels);
    const stream = await client.stream("/frame/iterate", reqZ, resZ);
    const iter = new Iterator(stream, adapter);
    await iter.execute({
      command: Command.Open,
      keys: Array.from(adapter.keys),
      bounds: new TimeRange(tr),
      chunkSize: opts.chunkSize ?? 1e5,
    });
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
  async next(span: CrudeTimeSpan = AUTO_SPAN): Promise<boolean> {
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
  async prev(span: CrudeTimeSpan = AUTO_SPAN): Promise<boolean> {
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
  async seekLE(stamp: CrudeTimeStamp): Promise<boolean> {
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
  async seekGE(stamp: CrudeTimeStamp): Promise<boolean> {
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

  [Symbol.asyncIterator](): AsyncIterator<Frame, any, undefined> {
    return new IteratorIterator(this);
  }

  private async execute(request: Request): Promise<boolean> {
    this.stream.send(request);
    this.value = new Frame();
    while (true) {
      const res = await this.stream.receive();
      if (res.variant === ResponseVariant.Ack) return res.ack;
      this.value.push(this.adapter.adapt(new Frame(res.frame)));
    }
  }
}

class IteratorIterator implements AsyncIterator<Frame> {
  private readonly iter: Iterator;
  private open: boolean = false;

  constructor(iter: Iterator) {
    this.iter = iter;
  }

  async next(): Promise<IteratorResult<Frame, any>> {
    try {
      let ok = true;
      if (!this.open) {
        if (!(await this.iter.seekFirst())) ok = false;
        this.open = true;
      }
      if (!(await this.iter.next())) ok = false;
      if (!ok) await this.iter.close();
      return { done: !ok, value: this.iter.value };
    } catch (e) {
      await this.iter.close();
      throw e;
    }
  }
}
