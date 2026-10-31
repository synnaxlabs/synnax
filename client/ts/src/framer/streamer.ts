// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  EOF,
  type Stream,
  Unreachable,
  type WebSocketClient,
} from "@synnaxlabs/freighter";
import { errors, Rate, sync, TimeSpan, zod } from "@synnaxlabs/x";
import { z } from "zod";

import { channel } from "@/channel";
import { UnexpectedError } from "@/errors";
import { type ChannelRetriever, ReadAdapter } from "@/framer/adapter";
import { WSStreamerCodec } from "@/framer/codec";
import { Frame, frameZ } from "@/framer/frame";
import { StreamProxy } from "@/framer/streamProxy";

const reqZ = z.object({
  keys: z.number().array(),
  downsampleFactor: z.uint32(),
  throttleRate: Rate.z.optional(),
  excludeGroups: z.uint32().array().optional(),
  keepAlive: TimeSpan.z.optional(),
  ackRequested: z.boolean().optional(),
});

/**
 * Request interface for streaming frames from a Synnax cluster.
 * Contains the keys of channels to stream from and a downsample factor.
 */
export interface StreamerRequest extends z.infer<typeof reqZ> {}

const resZ = z.object({
  frame: frameZ,
  /** Marks an empty response the Core emits so a dead connection is detectable. */
  keepAlive: z.boolean().optional(),
  /** Marks the empty response that answers a request with ackRequested set. */
  ack: z.boolean().optional(),
});

/**
 * Response interface for streaming frames from a Synnax cluster.
 * Contains a frame of telemetry data.
 */
export interface StreamerResponse extends z.infer<typeof resZ> {}

const intermediateStreamerConfigZ = z.object({
  /** The channels to stream data from. Can be channel keys, names, or payloads. */
  channels: channel.paramsZ,
  /** Optional factor to downsample the data by. Defaults to 1 (no downsampling). */
  downsampleFactor: z.uint32().default(1),
  /** Optional throttle rate in Hz to limit the rate of frames sent to the client. Defaults to 0 (no throttling). */
  throttleRate: Rate.z.default(new Rate(0)),
  /** excludeGroups sets writer group IDs whose frames should be filtered out by the
   Core. Used for telemetry bypass deduplication. */
  excludeGroups: z.uint32().array().default([]),
  /** Interval at which the Core emits keep-alive responses so a silently dead
   connection fails reads instead of hanging forever. TimeSpan.ZERO disables
   detection. Defaults to 5 seconds. */
  keepAlive: TimeSpan.z.default(TimeSpan.seconds(5)),
});

/** Zod schema for {@link StreamerConfig}. A bare channel list parses as a config. */
export const streamerConfigZ = intermediateStreamerConfigZ.or(
  channel.paramsZ.transform((channels) =>
    intermediateStreamerConfigZ.parse({ channels }),
  ),
);

/** Config for a streamer. Pass it to `client.telem.openStreamer`. */
export type StreamerConfig = z.input<typeof streamerConfigZ>;

/** Options for {@link Streamer.update}. */
export interface StreamerUpdateOptions {
  /**
   * Resolves the update only once the Core streams the new channels, so a write issued
   * after it is delivered. Frames must be read concurrently, or the update never
   * resolves. Cores that predate the ack never send one, so only set this against a
   * Core that supports it.
   */
  waitForAck?: boolean;
}

/**
 * Streams frames of telemetry from a Synnax cluster in real time. Open one with the
 * client's openStreamer method, never directly. Read frames with `read` or by
 * iterating the streamer. Close it in a `finally` block to free its resources.
 *
 * @see https://docs.synnaxlabs.com/reference/client/stream-data
 */
export interface Streamer extends AsyncIterator<Frame>, AsyncIterable<Frame> {
  /** The keys of the channels currently being streamed from. */
  keys: channel.Key[];
  /**
   * Replaces the list of channels being streamed from.
   * @param options - Set waitForAck to resolve only once the Core streams the channels.
   * @throws the error of the concurrent read when the stream fails before the ack, or
   * EOF when the streamer closes first.
   */
  update: (channels: channel.Params, options?: StreamerUpdateOptions) => Promise<void>;
  /** Close the streamer and free all associated resources. */
  close: () => void;
  /**
   * Read the next frame of telemetry.
   * @throws {Unreachable} if keep-alives were flowing and the stream then stays silent
   * past the keep-alive deadline: the connection is presumed dead.
   * @throws {UnexpectedError} if the Core sends an ack that answers no update.
   */
  read: () => Promise<Frame>;
}

/** A function that opens a streamer. */
export interface StreamOpener {
  (config: StreamerConfig): Promise<Streamer>;
}

/**
 * Creates a function that opens streamers with the given channel resolver and client.
 * @param retrieveChannels - Resolves channel params to payloads for the codec
 * @param client - The WebSocket client to use for streaming
 * @returns A function that opens streamers with the given configuration
 */
export const createStreamOpener =
  (retrieveChannels: ChannelRetriever, client: WebSocketClient): StreamOpener =>
  async (config) => {
    const cfg = zod.parse(streamerConfigZ, config, { label: "streamer config" });
    const adapter = await ReadAdapter.open(retrieveChannels, cfg.channels);
    client = client.withCodec(new WSStreamerCodec(adapter.codec));
    const stream = await client.stream("/frame/stream", reqZ, resZ);
    const streamer = new BaseStreamer(
      stream,
      adapter,
      cfg.downsampleFactor,
      cfg.throttleRate,
      cfg.excludeGroups,
      cfg.keepAlive,
    );
    stream.send({
      keys: Array.from(adapter.keys),
      downsampleFactor: cfg.downsampleFactor,
      throttleRate: cfg.throttleRate,
      excludeGroups: cfg.excludeGroups,
      keepAlive: cfg.keepAlive,
    });
    // A keep-alive can beat the open ack onto the wire, so the ack is the first
    // non-keep-alive response.
    let res = await stream.receive();
    while (res.keepAlive === true) res = await stream.receive();
    return streamer;
  };

/**
 * Opens a new streamer with the given configuration.
 * @param retrieveChannels - Resolves channel params to payloads for the codec
 * @param client - The WebSocket client to use for streaming
 * @returns A promise that resolves to a new streamer
 */
export const openStreamer = async (
  retrieveChannels: ChannelRetriever,
  client: WebSocketClient,
  config: StreamerConfig,
): Promise<Streamer> => await createStreamOpener(retrieveChannels, client)(config);

// Missing this many keep-alive intervals in a row fails the pending read: one is normal
// jitter, three is a dead connection.
const KEEP_ALIVE_DEADLINE_FACTOR = 3;

class BaseStreamer implements Streamer {
  private readonly stream: StreamProxy<typeof reqZ, typeof resZ>;
  private readonly adapter: ReadAdapter;
  private readonly downsampleFactor: number;
  private readonly throttleRate: Rate;
  private readonly excludeGroups: number[];
  private readonly deadline: TimeSpan;
  // Set once the Core proves keep-alive support by sending one, so the deadline never
  // arms against a Core that will not send them.
  private armed = false;
  // Updates waiting on an ack, oldest first. The Core acks in request order.
  private readonly pendingAcks: PromiseWithResolvers<void>[] = [];
  private failure?: Error;

  constructor(
    stream: Stream<typeof reqZ, typeof resZ>,
    adapter: ReadAdapter,
    downsampleFactor: number = 1,
    throttleRate: Rate = new Rate(0),
    excludeGroups: number[] = [],
    keepAlive: TimeSpan = TimeSpan.ZERO,
  ) {
    this.stream = new StreamProxy("Streamer", stream);
    this.adapter = adapter;
    this.downsampleFactor = downsampleFactor;
    this.throttleRate = throttleRate;
    this.excludeGroups = excludeGroups;
    this.deadline = TimeSpan.milliseconds(
      keepAlive.milliseconds * KEEP_ALIVE_DEADLINE_FACTOR,
    );
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
    while (true) {
      let res: z.infer<typeof resZ>;
      try {
        res = await this.receiveWithDeadline();
      } catch (e) {
        const err = errors.fromUnknown(e);
        this.failure = err;
        this.pendingAcks.splice(0).forEach(({ reject }) => reject(err));
        throw err;
      }
      if (res.keepAlive === true) {
        if (!this.deadline.isZero) this.armed = true;
        continue;
      }
      if (res.ack === true) {
        const ack = this.pendingAcks.shift();
        if (ack == null)
          throw new UnexpectedError("received an ack with no pending update");
        ack.resolve();
        continue;
      }
      return this.adapter.adapt(new Frame(res.frame));
    }
  }

  private async receiveWithDeadline(): Promise<z.infer<typeof resZ>> {
    const received = this.stream.receive();
    if (!this.armed) return await received;
    return await sync.withTimeout(
      received,
      this.deadline,
      () =>
        new Unreachable({
          message: `streamer received no response for ${this.deadline.toString()}`,
        }),
    );
  }

  async update(
    channels: channel.Params,
    { waitForAck = false }: StreamerUpdateOptions = {},
  ): Promise<void> {
    const hasChanged = await this.adapter.update(channels);
    // A read that failed while the channels resolved left no reader for the ack.
    if (this.failure != null) throw this.failure;
    if (!hasChanged && !waitForAck) return;
    this.adapter.advance();
    this.stream.send({
      keys: Array.from(this.adapter.keys),
      downsampleFactor: this.downsampleFactor,
      throttleRate: this.throttleRate,
      excludeGroups: this.excludeGroups,
      ackRequested: waitForAck,
    });
    if (!waitForAck) return;
    const ack = Promise.withResolvers<void>();
    this.pendingAcks.push(ack);
    await ack.promise;
  }

  close(): void {
    // Rejected updates stay queued, so the acks the Core already owes still find them.
    this.pendingAcks.forEach(({ reject }) => reject(new EOF()));
    this.stream.closeSend();
  }

  [Symbol.asyncIterator](): AsyncIterator<Frame, any, undefined> {
    return this;
  }
}
