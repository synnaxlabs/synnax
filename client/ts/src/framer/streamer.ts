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

import { type channel } from "@/channel";
import { paramsZ } from "@/channel/payload";
import { type ChannelRetriever, ReadAdapter } from "@/framer/adapter";
import { WSStreamerCodec } from "@/framer/codec";
import { Frame, frameZ } from "@/framer/frame";
import { StreamProxy } from "@/framer/streamProxy";

const reqZ = z.object({
  keys: z.number().array(),
  downsampleFactor: z.int(),
  throttleRate: Rate.z.optional(),
  excludeGroups: z.uint32().array().optional(),
  keepalive: TimeSpan.z.optional(),
});

/**
 * Request interface for streaming frames from a Synnax cluster.
 * Contains the keys of channels to stream from and a downsample factor.
 */
export interface StreamerRequest extends z.infer<typeof reqZ> {}

const resZ = z.object({
  frame: frameZ,
  /** Marks an empty response the Core emits so a dead connection is detectable. */
  keepalive: z.boolean().optional(),
});

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
  /** Interval at which the Core emits keepalive responses so a silently dead
   connection fails reads instead of hanging forever. TimeSpan.ZERO disables
   detection. Defaults to 5 seconds. */
  keepalive: TimeSpan.z.default(TimeSpan.seconds(5)),
});

/** Zod schema for {@link StreamerConfig}. A bare channel list parses as a config. */
export const streamerConfigZ = intermediateStreamerConfigZ.or(
  paramsZ.transform((channels) => intermediateStreamerConfigZ.parse({ channels })),
);

/** Config for a streamer. Pass it to `client.telem.openStreamer`. */
export type StreamerConfig = z.input<typeof streamerConfigZ>;

/**
 * Streams frames of telemetry from a Synnax cluster in real time. Open one with the
 * client's openStreamer method, never directly. Read frames with `read` or by
 * iterating the streamer. Close it in a `finally` block to free its resources.
 *
 * @see https://docs.synnaxlabs.com/reference/client/working-with-data/streaming-data
 */
export interface Streamer extends AsyncIterator<Frame>, AsyncIterable<Frame> {
  /** The keys of the channels currently being streamed from. */
  keys: channel.Key[];
  /** Replaces the list of channels being streamed from. */
  update: (channels: channel.Params) => Promise<void>;
  /** Close the streamer and free all associated resources. */
  close: () => void;
  /**
   * Read the next frame of telemetry.
   * @throws {Unreachable} if keepalives were flowing and the stream then stays silent
   * past the keepalive deadline: the connection is presumed dead.
   */
  read: () => Promise<Frame>;
}

/** A function that opens a streamer. */
export interface StreamOpener {
  (config: StreamerConfig): Promise<Streamer>;
}

// Deadline for the Core to acknowledge a streamer once the socket is open. The
// handshake already proved the connection, so only a death in the window between the
// two reaches this.
const OPEN_ACK_TIMEOUT = TimeSpan.seconds(30);

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
      cfg.keepalive,
    );
    stream.send({
      keys: Array.from(adapter.keys),
      downsampleFactor: cfg.downsampleFactor,
      throttleRate: cfg.throttleRate,
      excludeGroups: cfg.excludeGroups,
      keepalive: cfg.keepalive,
    });
    // A keepalive can beat the open ack onto the wire, so the ack is the first
    // non-keepalive response.
    const ack = (async () => {
      let res = await stream.receive();
      while (res.keepalive === true) res = await stream.receive();
    })();
    const span = OPEN_ACK_TIMEOUT.toString();
    try {
      await sync.withTimeout(
        ack,
        OPEN_ACK_TIMEOUT,
        () =>
          new Unreachable({
            message: `streamer was not acknowledged within ${span}`,
          }),
      );
    } catch (err) {
      streamer.close();
      throw errors.fromUnknown(err);
    }
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

// Missing this many keepalive intervals in a row fails the pending read: one is normal
// jitter, three is a dead connection.
const KEEPALIVE_DEADLINE_FACTOR = 3;

class BaseStreamer implements Streamer {
  private readonly stream: StreamProxy<typeof reqZ, typeof resZ>;
  private readonly adapter: ReadAdapter;
  private readonly downsampleFactor: number;
  private readonly throttleRate: Rate;
  private readonly excludeGroups: number[];
  private readonly deadline: TimeSpan;
  // Set once the Core proves keepalive support by sending one, so the deadline never
  // arms against a Core that will not send them.
  private armed = false;

  constructor(
    stream: Stream<typeof reqZ, typeof resZ>,
    adapter: ReadAdapter,
    downsampleFactor: number = 1,
    throttleRate: Rate = new Rate(0),
    excludeGroups: number[] = [],
    keepalive: TimeSpan = TimeSpan.ZERO,
  ) {
    this.stream = new StreamProxy("Streamer", stream);
    this.adapter = adapter;
    this.downsampleFactor = downsampleFactor;
    this.throttleRate = throttleRate;
    this.excludeGroups = excludeGroups;
    this.deadline = TimeSpan.milliseconds(
      keepalive.milliseconds * KEEPALIVE_DEADLINE_FACTOR,
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
      const res = await this.receiveWithDeadline();
      if (res.keepalive === true) {
        if (!this.deadline.isZero) this.armed = true;
        continue;
      }
      return this.adapter.adapt(new Frame(res.frame));
    }
  }

  private async receiveWithDeadline(): Promise<z.infer<typeof resZ>> {
    const received = this.stream.receive();
    if (!this.armed) return await received;
    let timer: ReturnType<typeof setTimeout> | undefined;
    const deadline = new Promise<never>((_, reject) => {
      timer = setTimeout(() => {
        const silence = this.deadline.toString();
        const message = `streamer received no response for ${silence}`;
        reject(new Unreachable({ message }));
      }, this.deadline.milliseconds);
    });
    try {
      return await Promise.race([received, deadline]);
    } catch (err) {
      // The read already failed for its caller; a late settle of the losing receive
      // must not surface as an unhandled rejection.
      received.catch(() => {});
      throw errors.fromUnknown(err);
    } finally {
      clearTimeout(timer);
    }
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
