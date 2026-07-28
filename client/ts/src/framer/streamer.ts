// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { EOF, type Stream, type WebSocketClient } from "@synnaxlabs/freighter";
import {
  breaker,
  errors,
  observe,
  Rate,
  sync,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
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
  excludeGroups: z.uint32().array().optional(),
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
  /** excludeGroups sets writer group IDs whose frames should be filtered out by the
   Core. Used for telemetry bypass deduplication. */
  excludeGroups: z.uint32().array().default([]),
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
 * For detailed documentation, see https://docs.synnaxlabs.com/reference/client/working-with-data/streaming-data
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
    client = client.withCodec(new WSStreamerCodec(adapter.codec));
    const stream = await client.stream("/frame/stream", reqZ, resZ);
    const streamer = new BaseStreamer(
      stream,
      adapter,
      cfg.downsampleFactor,
      cfg.throttleRate,
      cfg.excludeGroups,
    );
    stream.send({
      keys: Array.from(adapter.keys),
      downsampleFactor: cfg.downsampleFactor,
      throttleRate: cfg.throttleRate,
      excludeGroups: cfg.excludeGroups,
    });
    await stream.receive();
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

class BaseStreamer implements Streamer {
  private readonly stream: StreamProxy<typeof reqZ, typeof resZ>;
  private readonly adapter: ReadAdapter;
  private readonly downsampleFactor: number;
  private readonly throttleRate: Rate;
  private readonly excludeGroups: number[];

  constructor(
    stream: Stream<typeof reqZ, typeof resZ>,
    adapter: ReadAdapter,
    downsampleFactor: number = 1,
    throttleRate: Rate = new Rate(0),
    excludeGroups: number[] = [],
  ) {
    this.stream = new StreamProxy("Streamer", stream);
    this.adapter = adapter;
    this.downsampleFactor = downsampleFactor;
    this.throttleRate = throttleRate;
    this.excludeGroups = excludeGroups;
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
      throw errors.fromUnknown(err);
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
      excludeGroups: this.excludeGroups,
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
  private current: Streamer | null = null;
  private readonly breaker: breaker.Breaker;
  private readonly opener: StreamOpener;
  private readonly config: ParsedStreamerConfig;
  private readonly onReopen?: () => void;
  private readonly onDrop?: (error: Error) => void;
  private readonly closeNotifier = new sync.Notifier();
  // A stream that outlives this span proves the incident over; a shorter
  // life keeps the reconnect backoff escalating.
  private readonly stableAfter: TimeSpan;
  private openedAt = TimeStamp.now();
  private closed = false;

  private constructor(
    opener: StreamOpener,
    config: StreamerConfig,
    breakerConfig: breaker.Config = {},
    onReopen?: () => void,
    onDrop?: (error: Error) => void,
  ) {
    this.opener = opener;
    this.config = streamerConfigZ.parse(config);
    const {
      maxRetries = Infinity,
      baseInterval = TimeSpan.seconds(1),
      // reconnect attempts are cheap: a low cap bounds how long a returned
      // cluster goes unnoticed while backoff is escalated
      maxInterval = TimeSpan.seconds(5),
      scale = 2,
      jitter = 0.25,
    } = breakerConfig ?? {};
    this.breaker = new breaker.Breaker({
      maxRetries,
      baseInterval,
      maxInterval,
      scale,
      jitter,
      // close() interrupts a pending backoff so a retired streamer's
      // reconnect loop ends promptly instead of sleeping through it
      sleepFn: async (duration) => {
        await this.closeNotifier.wait(duration);
      },
    });
    this.stableAfter = new TimeSpan(maxInterval);
    this.onReopen = onReopen;
    this.onDrop = onDrop;
  }

  /**
   * Opens a new hardened streamer with the given configuration.
   * @param opener - The function to use for opening streamers
   * @param config - The configuration for the streamer
   * @param breakerConfig - Retry behavior for reconnect attempts. Defaults to
   * retrying forever on capped, jittered exponential backoff.
   * @param onReopen - Called after every successful reconnect (not the initial
   * open). Frames may have been dropped between the failure and the reopen.
   * @param onDrop - Called when the stream fails and reconnection begins.
   * @returns A promise that resolves to a new hardened streamer
   */
  static async open(
    opener: StreamOpener,
    config: StreamerConfig,
    breakerConfig?: breaker.Config,
    onReopen?: () => void,
    onDrop?: (error: Error) => void,
  ): Promise<HardenedStreamer> {
    const h = new HardenedStreamer(opener, config, breakerConfig, onReopen, onDrop);
    await h.runStreamer(false);
    return h;
  }

  private async runStreamer(notifyReopen: boolean = true): Promise<void> {
    // A long-lived stream that dropped is a fresh incident: reconnect
    // immediately. A short-lived one is a failing cycle (e.g. a server that
    // accepts streams and instantly ends them): keep backing off, or the
    // loop dials hot.
    if (notifyReopen)
      if (TimeStamp.since(this.openedAt).greaterThan(this.stableAfter))
        this.breaker.reset();
      else if (!(await this.breaker.wait())) throw new EOF();
    while (!this.closed)
      try {
        if (this.current != null) this.current.close();
        this.current = null;
        const next = await this.opener(this.config);
        // closed while the open was in flight: the fresh stream is already
        // retired and must not be handed to callers
        if (this.closed) {
          next.close();
          break;
        }
        this.current = next;
        this.openedAt = TimeStamp.now();
        if (notifyReopen) this.onReopen?.();
        return;
      } catch (e) {
        this.current = null;
        if (this.closed) break;
        if (!(await this.breaker.wait())) throw errors.fromUnknown(e);
        console.error("failed to open streamer", e);
      }
    throw new EOF();
  }

  private get wrapped(): Streamer {
    if (this.current == null) throw new Error("stream closed");
    return this.current;
  }

  async update(channels: channel.Params): Promise<void> {
    if (this.closed) throw new EOF();
    this.config.channels = paramsZ.parse(channels);
    try {
      await this.wrapped.update(channels);
    } catch (e) {
      if (this.closed || EOF.matches(e)) throw errors.fromUnknown(e);
      this.onDrop?.(errors.fromUnknown(e));
      await this.runStreamer();
      return await this.update(channels);
    }
  }

  async next(): Promise<IteratorResult<Frame>> {
    try {
      return { done: false, value: await this.read() };
    } catch (e) {
      if (EOF.matches(e)) return { done: true, value: undefined };
      throw errors.fromUnknown(e);
    }
  }

  async read(): Promise<Frame> {
    if (this.closed) throw new EOF();
    try {
      return await this.wrapped.read();
    } catch (e) {
      if (this.closed) throw new EOF();
      // an EOF the client did not ask for is a drop: the server ended the
      // stream, so reconnect like any other failure
      this.onDrop?.(errors.fromUnknown(e));
      await this.runStreamer();
      return await this.read();
    }
  }

  /** Stops reconnecting and closes the stream. Idempotent; never throws. */
  close(): void {
    if (this.closed) return;
    this.closed = true;
    this.closeNotifier.notify();
    this.current?.close();
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
