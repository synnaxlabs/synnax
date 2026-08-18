// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { EOF, type Stream, type WebSocketClient } from "@synnaxlabs/freighter";
import { errors, Rate, zod } from "@synnaxlabs/x";
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
 * Creates a function that opens streamers with the given channel resolver and
 * client.
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
 * @param retrieveChannels - Resolves channel params to payloads for the codec
 * @param client - The WebSocket client to use for streaming
 * @param config - The configuration for the streamer
 * @returns A promise that resolves to a new streamer
 */
export const openStreamer = async (
  retrieveChannels: ChannelRetriever,
  client: WebSocketClient,
  config: StreamerConfig,
): Promise<Streamer> => await createStreamOpener(retrieveChannels, client)(config);

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
