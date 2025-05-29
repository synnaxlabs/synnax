// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { EOF, type Stream, type WebSocketClient } from "@synnaxlabs/freighter";
import { breaker, observe, TimeSpan } from "@synnaxlabs/x";
import { z } from "zod";

import { type channel } from "@/channel";
import { ReadAdapter } from "@/framer/adapter";
import { WSStreamerCodec } from "@/framer/codec";
import { Frame, frameZ } from "@/framer/frame";
import { StreamProxy } from "@/framer/streamProxy";

const reqZ = z.object({ keys: z.number().array(), downsampleFactor: z.number() });

export interface StreamerRequest extends z.infer<typeof reqZ> {}

const resZ = z.object({ frame: frameZ });

export interface StreamerResponse extends z.infer<typeof resZ> {}

const ENDPOINT = "/frame/stream";

export interface StreamerConfig {
  channels: channel.Params;
  downsampleFactor?: number;
  useExperimentalCodec?: boolean;
}

export interface Streamer extends AsyncIterator<Frame>, AsyncIterable<Frame> {
  keys: channel.Key[];
  update: (channels: channel.Params) => Promise<void>;
  close: () => void;
  read: () => Promise<Frame>;
}

export interface StreamOpener {
  (config: StreamerConfig | channel.Params): Promise<Streamer>;
}

export const createStreamOpener =
  (retriever: channel.Retriever, client: WebSocketClient): StreamOpener =>
  async (config) => {
    let cfg: StreamerConfig;
    if (Array.isArray(config) || typeof config !== "object")
      cfg = { channels: config as channel.Params, downsampleFactor: 1 };
    else cfg = config as StreamerConfig;
    const adapter = await ReadAdapter.open(retriever, cfg.channels);
    if (cfg.useExperimentalCodec)
      client = client.withCodec(new WSStreamerCodec(adapter.codec));
    const stream = await client.stream(ENDPOINT, reqZ, resZ);
    const streamer = new CoreStreamer(stream, adapter);
    stream.send({ keys: adapter.keys, downsampleFactor: cfg.downsampleFactor ?? 1 });
    const [, err] = await stream.receive();
    if (err != null) throw err;
    return streamer;
  };

export const openStreamer = async (
  retriever: channel.Retriever,
  client: WebSocketClient,
  config: StreamerConfig,
): Promise<Streamer> => await createStreamOpener(retriever, client)(config);

class CoreStreamer implements Streamer {
  private readonly stream: StreamProxy<typeof reqZ, typeof resZ>;
  private readonly adapter: ReadAdapter;
  private readonly downsampleFactor: number;

  constructor(stream: Stream<typeof reqZ, typeof resZ>, adapter: ReadAdapter) {
    this.stream = new StreamProxy("Streamer", stream);
    this.adapter = adapter;
    this.downsampleFactor = 1;
  }

  get keys(): channel.Key[] {
    return this.adapter.keys;
  }

  async next(): Promise<IteratorResult<Frame, any>> {
    try {
      const frame = await this.read();
      return { done: false, value: frame };
    } catch (err) {
      if (EOF.matches(err)) return { done: true, value: undefined };
      throw err;
    }
  }

  async read(): Promise<Frame> {
    return this.adapter.adapt(new Frame((await this.stream.receive()).frame));
  }

  async update(channels: channel.Params): Promise<void> {
    await this.adapter.update(channels);
    this.stream.send({
      keys: this.adapter.keys,
      downSampleFactor: this.downsampleFactor,
    });
  }

  close(): void {
    this.stream.closeSend();
  }

  [Symbol.asyncIterator](): AsyncIterator<Frame, any, undefined> {
    return this;
  }
}

export class HardenedStreamer implements Streamer {
  private wrapped_: Streamer | null = null;
  private readonly breaker: breaker.Breaker;
  private readonly opener: StreamOpener;
  private readonly config: StreamerConfig;

  private constructor(opener: StreamOpener, config: StreamerConfig | channel.Params) {
    this.opener = opener;
    if (Array.isArray(config) || typeof config !== "object")
      this.config = { channels: config as channel.Params, downsampleFactor: 1 };
    else this.config = config as StreamerConfig;
    this.breaker = new breaker.Breaker({
      maxRetries: 5000,
      baseInterval: TimeSpan.seconds(1),
      scale: 1,
    });
  }

  static async open(
    opener: StreamOpener,
    config: StreamerConfig | channel.Params,
  ): Promise<HardenedStreamer> {
    const h = new HardenedStreamer(opener, config);
    await h.runStreamer();
    return h;
  }

  private async runStreamer(): Promise<void> {
    while (true)
      try {
        if (this.wrapped_ != null) this.wrapped_.close();
        this.wrapped_ = await this.opener(this.config);
        this.breaker.reset();
        return;
      } catch (e) {
        this.wrapped_ = null;
        if (!(await this.breaker.wait())) throw e;
        continue;
      }
  }

  private get wrapped(): Streamer {
    if (this.wrapped_ == null) throw new Error("stream closed");
    return this.wrapped_;
  }

  async update(channels: channel.Params): Promise<void> {
    this.config.channels = channels;
    try {
      await this.wrapped.update(channels);
    } catch {
      await this.runStreamer();
      return await this.update(channels);
    }
  }

  async next(): Promise<IteratorResult<Frame>> {
    try {
      return { done: false, value: await this.read() };
    } catch (e) {
      if (EOF.matches(e)) return { done: true, value: undefined };
      throw e;
    }
  }

  async read(): Promise<Frame> {
    try {
      const fr = await this.wrapped.read();
      this.breaker.reset();
      return fr;
    } catch (e) {
      if (EOF.matches(e)) throw e;
      await this.runStreamer();
      return await this.read();
    }
  }
  close(): void {
    this.wrapped.close();
  }

  get keys(): channel.Key[] {
    return this.wrapped.keys;
  }

  [Symbol.asyncIterator](): AsyncIterator<Frame> {
    return this;
  }
}

export class ObservableStreamer<V = Frame>
  extends observe.Observer<Frame, V>
  implements observe.ObservableAsyncCloseable<V>
{
  private readonly streamer: Streamer;
  private readonly closePromise: Promise<void>;

  constructor(streamer: Streamer, transform?: observe.Transform<Frame, V>) {
    super(transform);
    this.streamer = streamer;
    this.closePromise = this.stream();
  }

  async close(): Promise<void> {
    this.streamer.close();
    await this.closePromise;
  }

  private async stream(): Promise<void> {
    for await (const frame of this.streamer) this.notify(frame);
  }
}
