// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { EOF, type Stream, type WebSocketClient } from "@synnaxlabs/freighter";
import { breaker, observe, Rate, TimeSpan } from "@synnaxlabs/x";
import { z } from "zod";

import { type channel } from "@/channel";
import { paramsZ } from "@/channel/payload";
import { ReadAdapter } from "@/framer/adapter";
import { WSStreamerCodec } from "@/framer/codec";
import { Frame, frameZ } from "@/framer/frame";
import { StreamProxy } from "@/framer/streamProxy";

const reqZ = z.object({
  keys: z.number().array(),
  downsampleFactor: z.int(),
  throttleRate: Rate.z.optional(),
});

/**
 * Request interface for streaming frames from a Synnax cluster.
 * Contains the keys of channels to stream from and a downsample factor.
 */
export interface StreamerRequest extends z.infer<typeof reqZ> {}

const resZ = z.object({ frame: frameZ });

/**
 * Response interface for streaming frames from a Synnax cluster.
 * Contains a frame of telemetry data.
 */
export interface StreamerResponse extends z.infer<typeof resZ> {}

const intermediateStreamerConfigZ = z.object({
  /** The channels to stream data from. Can be channel keys, names, or payloads. */
  channels: paramsZ,
  /** Optional factor to downsample the data by. Defaults to 1 (no downsampling). */
  downsampleFactor: z.int().default(1),
  /** Optional throttle rate in Hz to limit the rate of frames sent to the client. Defaults to 0 (no throttling). */
  throttleRate: Rate.z.default(new Rate(0)),
  /** useHighPerformanceCodec sets whether the writer will use the Synnax frame encoder
   as opposed to the standard JSON encoding mechanisms for frames. */
  useHighPerformanceCodec: z.boolean().default(true),
});

export const streamerConfigZ = intermediateStreamerConfigZ.or(
  paramsZ.transform((channels) => intermediateStreamerConfigZ.parse({ channels })),
);

export type StreamerConfig = z.input<typeof streamerConfigZ>;
type ParsedStreamerConfig = z.output<typeof streamerConfigZ>;

/**
 * A streamer is used to stream frames of telemetry in real-time from a Synnax cluster.
 * It should not be constructed directly, and should instead be created using the
 * client's openStreamer method.
 *
 * To open a streamer, use the openStreamer method on the client and pass it in the list
 * of channels you'd like to receive data from. Once the streamer has been opened, call
 * the `read` method to read the next frame of telemetry, or use the streamer as an
 * async iterator to iterate over the frames of telemetry as they are received.
 *
 * The list of channels being streamed can be updated at any time by using the `update`
 * method.
 *
 * Once done, call the `close` method to close the streamer and free all associated
 * resources. We recommend using the streamer within a try-finally block to ensure
 * that it is closed properly in the event of an error.
 *
 * For details documentation, see https://docs.synnaxlabs.com/reference/typescript-client/stream-data
 */
export interface Streamer extends AsyncIterator<Frame>, AsyncIterable<Frame> {
  /** The keys of the channels currently being streamed from. */
  keys: channel.Key[];
  /**
   * Update the list of channels being streamed from. This replaces the list of channels
   * being streamed from with the new list of channels.
   */
  update: (channels: channel.Params) => Promise<void>;
  /** Close the streamer and free all associated resources. */
  close: () => void;
  /** Read the next frame of telemetry. */
  read: () => Promise<Frame>;
}

/**
 * A function that opens a streamer.
 */
export interface StreamOpener {
  (config: StreamerConfig): Promise<Streamer>;
}

/**
 * Creates a function that opens streamers with the given retriever and client.
 * @param retriever - The channel retriever to use for resolving channel information
 * @param client - The WebSocket client to use for streaming
 * @returns A function that opens streamers with the given configuration
 */
export const createStreamOpener =
  (retriever: channel.Retriever, client: WebSocketClient): StreamOpener =>
  async (config) => {
    const cfg = streamerConfigZ.parse(config);
    const adapter = await ReadAdapter.open(retriever, cfg.channels);
    if (cfg.useHighPerformanceCodec)
      client = client.withCodec(new WSStreamerCodec(adapter.codec));
    const stream = await client.stream("/frame/stream", reqZ, resZ);
    const streamer = new CoreStreamer(
      stream,
      adapter,
      cfg.downsampleFactor,
      cfg.throttleRate,
    );
    stream.send({
      keys: Array.from(adapter.keys),
      downsampleFactor: cfg.downsampleFactor,
      throttleRate: cfg.throttleRate,
    });
    const [, err] = await stream.receive();
    if (err != null) throw err;
    return streamer;
  };

/**
 * Opens a new streamer with the given configuration.
 * @param retriever - The channel retriever to use for resolving channel information
 * @param client - The WebSocket client to use for streaming
 * @param config - The configuration for the streamer
 * @returns A promise that resolves to a new streamer
 */
export const openStreamer = async (
  retriever: channel.Retriever,
  client: WebSocketClient,
  config: StreamerConfig,
): Promise<Streamer> => await createStreamOpener(retriever, client)(config);

class CoreStreamer implements Streamer {
  private readonly stream: StreamProxy<typeof reqZ, typeof resZ>;
  private readonly adapter: ReadAdapter;
  private readonly downsampleFactor: number;
  private readonly throttleRate: Rate;

  constructor(
    stream: Stream<typeof reqZ, typeof resZ>,
    adapter: ReadAdapter,
    downsampleFactor: number = 1,
    throttleRate: Rate = new Rate(0),
  ) {
    this.stream = new StreamProxy("Streamer", stream);
    this.adapter = adapter;
    this.downsampleFactor = downsampleFactor;
    this.throttleRate = throttleRate;
  }

  get keys(): channel.Key[] {
    return Array.from(this.adapter.keys);
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
    const hasChanged = await this.adapter.update(channels);
    if (!hasChanged) return;
    this.stream.send({
      keys: Array.from(this.adapter.keys),
      downsampleFactor: this.downsampleFactor,
      throttleRate: this.throttleRate,
    });
  }

  close(): void {
    this.stream.closeSend();
  }

  [Symbol.asyncIterator](): AsyncIterator<Frame, any, undefined> {
    return this;
  }
}

/**
 * A hardened streamer that automatically reconnects on failure.
 * This streamer wraps a regular streamer and adds automatic reconnection
 * logic when the connection is lost or errors occur.
 */
export class HardenedStreamer implements Streamer {
  private wrapped_: Streamer | null = null;
  private readonly breaker: breaker.Breaker;
  private readonly opener: StreamOpener;
  private readonly config: ParsedStreamerConfig;

  private constructor(
    opener: StreamOpener,
    config: StreamerConfig,
    breakerConfig: breaker.Config = {},
  ) {
    this.opener = opener;
    this.config = streamerConfigZ.parse(config);
    const {
      maxRetries = 5000,
      baseInterval = TimeSpan.seconds(1),
      scale = 1,
    } = breakerConfig ?? {};
    this.breaker = new breaker.Breaker({ maxRetries, baseInterval, scale });
  }

  /**
   * Opens a new hardened streamer with the given configuration.
   * @param opener - The function to use for opening streamers
   * @param config - The configuration for the streamer
   * @returns A promise that resolves to a new hardened streamer
   */
  static async open(
    opener: StreamOpener,
    config: StreamerConfig,
    breakerConfig?: breaker.Config,
  ): Promise<HardenedStreamer> {
    const h = new HardenedStreamer(opener, config, breakerConfig);
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
        console.error("failed to open streamer", e);
        continue;
      }
  }

  private get wrapped(): Streamer {
    if (this.wrapped_ == null) throw new Error("stream closed");
    return this.wrapped_;
  }

  async update(channels: channel.Params): Promise<void> {
    this.config.channels = paramsZ.parse(channels);
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

/**
 * Wraps a standard streamer to implement an observable interface for handling changes
 * to channel values through an onChange handler.
 */
export class ObservableStreamer<V = Frame>
  extends observe.Observer<Frame, V>
  implements observe.ObservableAsyncCloseable<V>
{
  private readonly streamer: Streamer;
  private readonly closePromise: Promise<void>;

  /**
   * Creates a new observable streamer.
   * @param streamer - The streamer to wrap
   * @param transform - An optional transform function to apply to each frame
   * @template V - The type of the transformed value. Only relevant if transform is
   * provided. Defaults to Frame.
   */
  constructor(streamer: Streamer, transform?: observe.Transform<Frame, V>) {
    super(transform);
    this.streamer = streamer;
    this.closePromise = this.stream();
  }

  async update(channels: channel.Params): Promise<void> {
    await this.streamer.update(channels);
  }

  async close(): Promise<void> {
    this.streamer.close();
    await this.closePromise;
  }

  private async stream(): Promise<void> {
    for await (const frame of this.streamer) this.notify(frame);
  }
}
