// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient, type WebSocketClient } from "@synnaxlabs/freighter";
import {
  type CrudeSeries,
  type CrudeTimeRange,
  type CrudeTimeStamp,
  type MultiSeries,
  type TimeRange,
  TimeSpan,
} from "@synnaxlabs/x";

import { channel } from "@/channel";
import { Deleter } from "@/framer/deleter";
import { Frame, ONTOLOGY_TYPE } from "@/framer/frame";
import { Iterator, type IteratorConfig } from "@/framer/iterator";
import { openStreamer, type Streamer, type StreamerConfig } from "@/framer/streamer";
import { Writer, type WriterConfig, WriterMode } from "@/framer/writer";
import { ontology } from "@/ontology";

export const ontologyID = (key: channel.Key): ontology.ID =>
  new ontology.ID({ type: ONTOLOGY_TYPE, key: key.toString() });

const normalizeConfig = <T extends { channels: channel.Params }>(
  config: T | channel.Params,
): T => {
  if (
    Array.isArray(config) ||
    typeof config !== "object" ||
    (typeof config === "object" && "key" in config)
  )
    return { channels: config } as T;
  return config;
};

export class Client {
  private readonly streamClient: WebSocketClient;
  private readonly retriever: channel.Retriever;
  private readonly deleter: Deleter;

  constructor(
    stream: WebSocketClient,
    unary: UnaryClient,
    retriever: channel.Retriever,
  ) {
    this.streamClient = stream;
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
    channels: channel.Params,
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
  async openWriter(config: WriterConfig | channel.Params): Promise<Writer> {
    return await Writer._open(
      this.retriever,
      this.streamClient,
      normalizeConfig<WriterConfig>(config),
    );
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
  async openStreamer(channels: channel.Params): Promise<Streamer>;

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
  async openStreamer(config: StreamerConfig): Promise<Streamer>;

  /** Overload to provide interface compatibility with @see StreamOpener */
  async openStreamer(config: StreamerConfig | channel.Params): Promise<Streamer>;

  async openStreamer(config: StreamerConfig | channel.Params): Promise<Streamer> {
    return await openStreamer(
      this.retriever,
      this.streamClient,
      normalizeConfig<StreamerConfig>(config),
    );
  }

  async write(
    start: CrudeTimeStamp,
    channel: channel.KeyOrName,
    data: CrudeSeries,
  ): Promise<void>;

  async write(
    start: CrudeTimeStamp,
    channels: channel.KeysOrNames,
    data: CrudeSeries[],
  ): Promise<void>;

  async write(
    start: CrudeTimeStamp,
    data: Record<channel.KeyOrName, CrudeSeries>,
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
    channels: channel.Params | Record<channel.KeyOrName, CrudeSeries>,
    data?: CrudeSeries | CrudeSeries[],
  ): Promise<void> {
    if (data == null) {
      const data_ = channels as Record<channel.KeyOrName, CrudeSeries>;
      const w = await this.openWriter({
        start,
        channels: Object.keys(data_),
        mode: WriterMode.Persist,
        errOnUnauthorized: true,
        enableAutoCommit: true,
        autoIndexPersistInterval: TimeSpan.MAX,
      });
      await w.write(data_);
      return await w.close();
    }
    const w = await this.openWriter({
      start,
      channels: channels as channel.Params,
      mode: WriterMode.Persist,
      errOnUnauthorized: true,
      enableAutoCommit: true,
      autoIndexPersistInterval: TimeSpan.MAX,
    });
    await w.write(channels as channel.Params, data);
    await w.close();
  }

  async read(tr: CrudeTimeRange, channel: channel.KeyOrName): Promise<MultiSeries>;

  async read(tr: CrudeTimeRange, channels: channel.Params): Promise<Frame>;

  async read(
    tr: CrudeTimeRange,
    channels: channel.Params,
  ): Promise<MultiSeries | Frame> {
    const { single } = channel.analyzeParams(channels);
    const fr = await this.readFrame(tr, channels);
    if (single) return fr.get(channels as channel.KeyOrName);
    return fr;
  }

  private async readFrame(
    tr: CrudeTimeRange,
    channels: channel.Params,
  ): Promise<Frame> {
    const i = await this.openIterator(tr, channels);
    const frame = new Frame();
    try {
      for await (const f of i) frame.push(f);
    } finally {
      await i.close();
    }
    return frame;
  }

  async delete(channels: channel.Params, timeRange: TimeRange): Promise<void> {
    const { normalized, variant } = channel.analyzeParams(channels);
    if (variant === "keys")
      return await this.deleter.delete({
        keys: normalized as channel.Key[],
        bounds: timeRange,
      });
    return await this.deleter.delete({
      names: normalized as string[],
      bounds: timeRange,
    });
  }
}
