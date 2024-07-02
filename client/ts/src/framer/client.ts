// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type StreamClient, UnaryClient } from "@synnaxlabs/freighter";
import {
  type CrudeSeries,
  type CrudeTimeRange,
  type CrudeTimeStamp,
  type MultiSeries,
  TimeRange,
  TimeSpan,
} from "@synnaxlabs/x";

import { type Key, type KeyOrName, KeysOrNames, type Params } from "@/channel/payload";
import { analyzeChannelParams, type Retriever } from "@/channel/retriever";
import { Deleter } from "@/framer/deleter";
import { Frame } from "@/framer/frame";
import { Iterator, IteratorConfig } from "@/framer/iterator";
import { Streamer, type StreamerConfig } from "@/framer/streamer";
import { Writer, type WriterConfig, WriterMode } from "@/framer/writer";

export class Client {
  private readonly streamClient: StreamClient;
  // private readonly unaryClient: UnaryClient;
  private readonly retriever: Retriever;
  private readonly deleter: Deleter;

  constructor(stream: StreamClient, unary: UnaryClient, retriever: Retriever) {
    this.streamClient = stream;
    // this.unaryClient = unary;
    this.retriever = retriever;
    this.deleter = new Deleter(unary);
  }

  /**
   * Opens a new iterator over the given channels within the provided time range.
   *
   * @param tr - A time range to iterate over.
   * @param channels - A list of channels (by name or key) to iterate over.
   * @param opts - see {@link IteratorConfig}
   * @returns a new {@link Iterator}.
   */
  async openIterator(
    tr: CrudeTimeRange,
    channels: Params,
    opts?: IteratorConfig,
  ): Promise<Iterator> {
    return await Iterator._open(tr, channels, this.retriever, this.streamClient, opts);
  }

  /**
   * Opens a new writer on the given channels.
   *
   * @param config - The configuration for the created writer, see documentation for
   * writerConfig for more detail.
   * @returns a new {@link Writer}.
   */
  async openWriter(config: WriterConfig | Params): Promise<Writer> {
    if (Array.isArray(config) || typeof config !== "object")
      config = { channels: config as Params };
    return await Writer._open(this.retriever, this.streamClient, config);
  }

  /***
   * Opens a new streamer on the given channels.
   *
   * @param channels - A key, name, list of keys, or list of names of the channels to
   * stream values from.
   * @throws a QueryError if any of the given channels do not exist.
   * @returns a new {@link Streamer} that must be closed when done streaming, otherwise
   * a network socket will remain open.
   */
  async openStreamer(channels: Params): Promise<Streamer>;

  /**
   * Opens a new streamer with the provided configuration.
   *
   * @param config - Configuration parameters for the streamer.
   * @param config.channels - The channels to stream values from. Can be a key, name,
   * list of keys, or list of names.
   * @param config.from - If this parameter is set and is before the current time,
   * the streamer will first read and receive historical data from before this point
   * and then will start reading new values.
   *
   */
  async openStreamer(config: StreamerConfig | Params): Promise<Streamer>;

  async openStreamer(config: StreamerConfig | Params): Promise<Streamer> {
    if (Array.isArray(config) || typeof config !== "object")
      config = { channels: config as Params };
    return await Streamer._open(this.retriever, this.streamClient, config);
  }

  async write(
    start: CrudeTimeStamp,
    channel: KeyOrName,
    data: CrudeSeries,
  ): Promise<void>;

  async write(
    start: CrudeTimeStamp,
    channels: KeysOrNames,
    data: CrudeSeries[],
  ): Promise<void>;

  async write(
    start: CrudeTimeStamp,
    data: Record<KeyOrName, CrudeSeries>,
  ): Promise<void>;

  /**
   * Writes telemetry to the given channel starting at the given timestamp.
   *
   * @param channels - The key of the channel to write to.
   * @param start - The starting timestamp of the first sample in data.
   * @param data  - The telemetry to write. This telemetry must have the same
   * data type as the channel.
   * @throws if the channel does not exist.
   */
  async write(
    start: CrudeTimeStamp,
    channels: Params | Record<KeyOrName, CrudeSeries>,
    data?: CrudeSeries | CrudeSeries[],
  ): Promise<void> {
    if (data == null) {
      const data_ = channels as Record<KeyOrName, CrudeSeries>;
      const w = await this.openWriter({
        start,
        channels: Object.keys(data_),
        mode: WriterMode.PersistOnly,
      });
      try {
        await w.write(data_);
        await w.commit();
      } finally {
        await w.close();
      }
      return;
    }
    const w = await this.openWriter({
      start,
      channels: channels as Params,
      mode: WriterMode.PersistOnly,
      errOnUnauthorized: true,
      enableAutoCommit: true,
      autoIndexPersistInterval: TimeSpan.MAX,
    });
    try {
      await w.write(channels as Params, data);
    } finally {
      await w.close();
    }
  }

  async read(tr: CrudeTimeRange, channel: KeyOrName): Promise<MultiSeries>;

  async read(tr: CrudeTimeRange, channels: Params): Promise<Frame>;

  async read(tr: CrudeTimeRange, channels: Params): Promise<MultiSeries | Frame> {
    const { single } = analyzeChannelParams(channels);
    const fr = await this.readFrame(tr, channels);
    if (single) return fr.get(channels as KeyOrName);
    return fr;
  }

  private async readFrame(tr: CrudeTimeRange, params: Params): Promise<Frame> {
    const i = await this.openIterator(tr, params);
    const frame = new Frame();
    try {
      for await (const f of i) frame.push(f);
    } finally {
      await i.close();
    }
    return frame;
  }

  async delete(channels: Params, timeRange: TimeRange): Promise<void> {
    const { normalized, variant } = analyzeChannelParams(channels);
    if (variant === "keys")
      return await this.deleter.delete({
        keys: normalized as Key[],
        bounds: timeRange,
      });
    return await this.deleter.delete({
      names: normalized as string[],
      bounds: timeRange,
    });
  }
}
