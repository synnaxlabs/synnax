// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import Registry from "../channel/registry";
import { TimeRange, TypedArray, UnparsedTimeStamp } from "../telem";
import Transport from "../transport";

import { AUTO_SPAN, TypedIterator } from "./iterator";
import { ArrayPayload } from "./payload";
import { RecordWriter } from "./writer";

export default class FrameClient {
  private readonly transport: Transport;
  private readonly channels: Registry;

  constructor(transport: Transport, channels: Registry) {
    this.transport = transport;
    this.channels = channels;
  }

  /**
   * Opens a new iterator over the given channels within the provided time range.
   *
   * @param tr - A time range to iterate over.
   * @param keys - A list of channel keys to iterate over.
   * @param aggregate - Whether to accumulate iteration results or reset them
   * on every iterator method call.
   * @returns a new {@link TypedIterator}.
   */
  async newIterator(
    tr: TimeRange,
    keys: string[],
    aggregate: boolean
  ): Promise<TypedIterator> {
    const iter = new TypedIterator(
      this.transport.streamClient,
      this.channels,
      aggregate
    );
    await iter.open(tr, keys);
    return iter;
  }

  /**
   * Opens a new writer on the given channels.
   *
   * @param keys - The keys of the channels to write to. A writer cannot write to
   * a channel that is not in this list. See the {@link RecordWriter} documentation
   * for more information.
   * @returns a new {@link RecordWriter}.
   */
  async newWriter(start: UnparsedTimeStamp, keys: string[]): Promise<RecordWriter> {
    const writer = new RecordWriter(this.transport.streamClient, this.channels);
    await writer.open(start, keys);
    return writer;
  }

  /**
   * Writes telemetry to the given channel starting at the given timestamp.
   *
   * @param to - The key of the channel to write to.
   * @param start - The starting timestamp of the first sample in data.
   * @param data  - The telemetry to write. This telemetry must have the same
   * data type as the channel.
   * @throws if the channel does not exist.
   */
  async write(to: string, start: UnparsedTimeStamp, data: TypedArray): Promise<void> {
    const writer = await this.newWriter(start, [to]);
    await writer.write({ [to]: data });
    if (!(await writer.commit())) throw new Error("Failed to commit.");
    await writer.close();
  }

  /**
   * Reads telemetry from the channel between the two timestamps.
   *
   * @param from - The key of the channel to read from.
   * @param start - The starting timestamp of the range to read from.
   * @param end - The ending timestamp of the range to read from.
   * @returns a typed array containing the retrieved telemetry.
   * @throws if the channel does not exist.
   * @throws if the telemetry between start and end is not contiguous.
   */
  async read(
    from: string,
    start: UnparsedTimeStamp,
    end: UnparsedTimeStamp
  ): Promise<TypedArray | undefined> {
    const arr = await this.readArray(from, start, end);
    if (arr == null || arr.dataType == null)
      throw new Error(`Channel ${from} does not exist.`);
    return new arr.dataType.Array(arr.data.buffer);
  }

  /**
   * Reads a segment from the channel between the two timestamps.
   *
   * @param from - The key of the channel to read from.
   * @param start - The starting timestamp of the range to read from.
   * @param end - The ending timestamp of the range to read from.
   * @returns a segment containing the retrieved telemetry.
   * @throws if the channel does not exist.
   * @throws if the telemetry between start and end is not contiguous.
   */
  async readArray(
    from: string,
    start: UnparsedTimeStamp,
    end: UnparsedTimeStamp
  ): Promise<ArrayPayload | undefined> {
    const tr = new TimeRange(start, end);
    const iter = await this.newIterator(tr, [from], /* accumulate */ true);
    let arr: ArrayPayload | undefined;
    try {
      if (await iter.seekFirst()) {
        // eslint-disable-next-line no-empty
        while (await iter.next(AUTO_SPAN)) {}
        arr = (await iter.value())[from];
      }
    } finally {
      await iter.close();
    }
    return arr;
  }
}
