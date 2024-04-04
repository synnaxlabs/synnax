// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type StreamClient } from "@synnaxlabs/freighter";
import {
  type TypedArray,
  type Series,
  type TimeRange,
  type CrudeTimeStamp,
  TimeStamp,
} from "@synnaxlabs/x";

import { type KeyOrName, type Params } from "@/channel/payload";
import { type Retriever, analyzeParams } from "@/channel/retriever";
import { Authority } from "@/control/authority";
import { Frame } from "@/framer/frame";
import { Iterator } from "@/framer/iterator";
import { Streamer } from "@/framer/streamer";
import { Writer, WriterMode, type WriterConfig } from "@/framer/writer";

export class Client {
  private readonly stream: StreamClient;
  private readonly retriever: Retriever;

  constructor(stream: StreamClient, retriever: Retriever) {
    this.stream = stream;
    this.retriever = retriever;
  }

  /**
   * Opens a new iterator over the given channels within the provided time range.
   *
   * @param tr - A time range to iterate over.
   * @param keys - A list of channel keys to iterate over.
   * @returns a new {@link TypedIterator}.
   */
  async openIterator(tr: TimeRange, channels: Params): Promise<Iterator> {
    return await Iterator._open(tr, channels, this.retriever, this.stream);
  }

  /**
   * Opens a new writer on the given channels.
   *
   * @param keys - The keys of the channels to write to. A writer cannot write to
   * a channel that is not in this list. See the {@link RecordWriter} documentation
   * for more information.
   * @returns a new {@link RecordWriter}.
   */
  async openWriter({
    start,
    channels,
    controlSubject,
    authorities = Authority.Absolute,
    mode = WriterMode.PersistStream,
  }: WriterConfig): Promise<Writer> {
    return await Writer._open(this.retriever, this.stream, {
      start: start ?? TimeStamp.now(),
      controlSubject,
      channels,
      authorities,
      mode,
    });
  }

  async openStreamer(
    params: Params,
    from: TimeStamp = TimeStamp.now(),
  ): Promise<Streamer> {
    return await Streamer._open(from, params, this.retriever, this.stream);
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
  async write(to: KeyOrName, start: CrudeTimeStamp, data: TypedArray): Promise<void> {
    const w = await this.openWriter({
      start,
      channels: to,
      mode: WriterMode.PersistOnly,
    });
    try {
      await w.write(to, data);
      await w.commit();
    } finally {
      await w.close();
    }
  }

  async read(tr: TimeRange, channel: KeyOrName): Promise<Series>;

  async read(tr: TimeRange, channels: Params): Promise<Frame>;

  async read(tr: TimeRange, channels: Params): Promise<Series | Frame> {
    const { single } = analyzeParams(channels);
    const fr = await this.readFrame(tr, channels);
    if (single) return fr.series[0];
    return fr;
  }

  private async readFrame(tr: TimeRange, params: Params): Promise<Frame> {
    const i = await this.openIterator(tr, params);
    const frame = new Frame();
    try {
      for await (const f of i) frame.push(f);
    } finally {
      await i.close();
    }
    return frame;
  }
}
