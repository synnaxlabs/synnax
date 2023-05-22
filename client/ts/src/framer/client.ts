// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  NativeTypedArray,
  LazyArray,
  TimeRange,
  UnparsedTimeStamp,
  TimeStamp,
} from "@synnaxlabs/x";

import { Frame } from "./frame";
import { AUTO_SPAN, Iterator } from "./iterator";
import { Streamer as StreamReader } from "./streamer";
import { Writer } from "./writer";

import {
  ChannelKey,
  ChannelKeyOrName,
  ChannelKeys,
  ChannelParams,
} from "@/channel/payload";
import { ChannelRetriever } from "@/channel/retriever";
import { Transport } from "@/transport";

export class FrameClient {
  private readonly transport: Transport;
  private readonly retriever: ChannelRetriever;

  constructor(transport: Transport, retriever: ChannelRetriever) {
    this.transport = transport;
    this.retriever = retriever;
  }

  /**
   * Opens a new iterator over the given channels within the provided time range.
   *
   * @param tr - A time range to iterate over.
   * @param keys - A list of channel keys to iterate over.
   * @returns a new {@link TypedIterator}.
   */
  async newIterator(tr: TimeRange, keys: ChannelKeys): Promise<Iterator> {
    const i = new Iterator(this.transport.streamClient);
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
  async newWriter(start: UnparsedTimeStamp, ...keys: ChannelKeys): Promise<Writer> {
    const w = new Writer(this.transport.streamClient);
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
    to: ChannelKey,
    start: UnparsedTimeStamp,
    data: NativeTypedArray
  ): Promise<void> {
    const f = new Frame();
    f.pushA(to, new LazyArray(data));
    const w = await this.newWriter(start, to);
    try {
      await w.write(f);
      if (!(await w.commit())) throw new Error("failed to commit");
    } catch {
      await w.close();
    }
  }

  async read(tr: TimeRange, channel: ChannelKeyOrName): Promise<LazyArray>;

  async read(tr: TimeRange, ...channels: ChannelParams[]): Promise<Frame>;

  async read(tr: TimeRange, ...channels: ChannelParams[]): Promise<LazyArray | Frame> {
    const fr = await this.readFrame(tr, ...channels);
    return fr;
  }

  private async readFrame(tr: TimeRange, ...params: ChannelParams[]): Promise<Frame> {
    const channels = await this.retriever.retrieve(...params);
    const i = await this.newIterator(
      tr,
      channels.map((c) => c.key)
    );
    let frame = new Frame();
    try {
      if (await i.seekFirst())
        while (await i.next(AUTO_SPAN)) frame = frame.concatF(i.value);
    } finally {
      await i.close();
    }
    return frame;
  }

  async newStreamer(...params: ChannelParams[]): Promise<StreamReader>;

  async newStreamer(from: TimeStamp, ...params: ChannelParams[]): Promise<StreamReader>;

  async newStreamer(
    from: TimeStamp | ChannelParams,
    ...params: ChannelParams[]
  ): Promise<StreamReader> {
    const channels = await this.retriever.retrieve(...params);
    const start = from instanceof TimeStamp ? from : TimeStamp.now();
    const i = new StreamReader(this.transport.streamClient);
    await i.open(
      start,
      channels.map((c) => c.key)
    );
    return i;
  }
}
