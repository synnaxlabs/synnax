// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { QueryError } from "..";
import Registry from "../channel/registry";
import { TimeRange, NativeTypedArray, UnparsedTimeStamp, TelemArray } from "../telem";
import Transport from "../transport";

import { Frame } from "./frame";
import { AUTO_SPAN, FrameIterator } from "./iterator";
import { FrameWriter } from "./writer";

export class FrameClient {
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
  ): Promise<FrameIterator> {
    const i = new FrameIterator(this.transport.streamClient, aggregate);
    await i.open(tr, keys);
    return i;
  }

  /**
   * Opens a new writer on the given channels.
   *
   * @param keys - The keys of the channels to write to. A writer cannot write to
   * a channel that is not in this list. See the {@link RecordWriter} documentation
   * for more information.
   * @returns a new {@link RecordWriter}.
   */
  async newWriter(start: UnparsedTimeStamp, keys: string[]): Promise<FrameWriter> {
    const w = new FrameWriter(this.transport.streamClient);
    await w.open(start, keys);
    return w;
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
  async write(
    to: string,
    start: UnparsedTimeStamp,
    data: NativeTypedArray
  ): Promise<void> {
    const f = new Frame();
    f.pushA(to, TelemArray.fromNative(data));
    const w = await this.newWriter(start, [to]);
    try {
      await w.write(f);
      if (!(await w.commit())) throw new Error("failed to commit");
    } catch {
      await w.close();
    }
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
    end: UnparsedTimeStamp,
    throwOnEmpty = true
  ): Promise<NativeTypedArray> {
    return (await this.readArray(from, start, end, throwOnEmpty)).data;
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
    end: UnparsedTimeStamp,
    throwOnEmpty = true
  ): Promise<TelemArray> {
    const tr = new TimeRange(start, end);
    const frame = await this.readFrame(tr, [from]);
    const arrs = frame.getA(from);
    if (arrs.length === 0 && throwOnEmpty)
      throw new QueryError(
        `no telemetry found for channel ${from} between ${tr.toString()}`
      );
    return arrs[0];
  }

  async readFrame(tr: TimeRange, keys: string[]): Promise<Frame> {
    const i = await this.newIterator(tr, keys, /* accumulate */ true);
    try {
      if (await i.seekFirst()) while (await i.next(AUTO_SPAN));
    } finally {
      await i.close();
    }
    return i.value;
  }
}
