// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type alamos } from "@synnaxlabs/alamos";
import {
  type channel,
  framer,
  NotFoundError,
  QueryError,
  type Synnax,
} from "@synnaxlabs/client";
import { type destructor, MultiSeries, type TimeRange } from "@synnaxlabs/x";

import { cache } from "@/telem/client/cache";
import { Reader } from "@/telem/client/reader";
import { Streamer, type StreamHandler } from "@/telem/client/streamer";

/**
 * A client that can be used to retrieve a channel from the Synnax cluster
 * by its key.
 */
export interface ChannelClient {
  /**
   * Retrieves a channel from the Synnax cluster by its key.
   *
   * @param key - The key of the channel to retrieve.
   * @returns the channel with the given key.
   * @throws QueryError if the channel does not exist.
   */
  retrieveChannel: (key: channel.Key | channel.Name) => Promise<channel.Payload>;
}

/** A client that can be used to read telemetry from the Synnax cluster. */
export interface ReadClient {
  /**
   * Reads telemetry from the given channel for the given time range at the
   * given fidelity. Fidelity is an upper bound on the alignmentMultiple of
   * the returned series: a fidelity of 1n (the default) returns raw data,
   * and higher values allow the client to receive decimated data whose
   * alignmentMultiple is at most the requested fidelity.
   *
   * @param tr  - The time range to read from.
   * @param key - The key of the channel to read from.
   * @param fidelity - The maximum acceptable alignmentMultiple. Defaults to
   *   1n (raw data).
   * @returns a MultiSeries of data for the channel. NOTE: Series in the
   * response may have different lengths and different alignmentMultiple
   * values when data at multiple fidelity tiers satisfies the range.
   */
  read: (
    tr: TimeRange,
    key: channel.Key,
    fidelity?: bigint,
  ) => Promise<MultiSeries>;
}

/** A client that can be used to stream telemetry from the Synnax cluster. */
export interface StreamClient {
  stream: (
    handler: StreamHandler,
    keys: channel.Key[],
  ) => Promise<destructor.Destructor>;
}

/**
 * Client provides an interface for performing basic telemetry operations
 * against a Synnax cluster. This interface is a simplification of the Synnax
 * client to make it easy to stub out for testing.
 */
export interface Client extends ChannelClient, ReadClient, StreamClient {
  /** Close closes the client, releasing all resources from the cache. */
  close: () => Promise<void>;
}

/**
 * NoopClient is a Client implementation that does nothing. This client is swapped
 * in when disconnected from the Synnax cluster.
 */
export class NoopClient implements Client {
  /** Implements ChannelClient. */
  async retrieveChannel(): Promise<channel.Payload> {
    throw new NotFoundError("NoopClient does not support retrieving channels");
  }

  /** Implements ReadClient. */
  async read(): Promise<MultiSeries> {
    return new MultiSeries([]);
  }

  /** Stream implements StreamClient. */
  async stream(): Promise<destructor.Async> {
    return async () => {};
  }

  /** Close implements CLient. */
  async close(): Promise<void> {}
}

interface CoreProps {
  core: Synnax;
  instrumentation: alamos.Instrumentation;
}

/**
 * Core wraps a Synnax client to implement the pluto telemetry Client interface,
 * adding a transparent caching layer.
 */
export class Core implements Client {
  private readonly ins: alamos.Instrumentation;
  private readonly cache: cache.Cache;
  private readonly reader: Reader;
  private readonly streamer: Streamer;
  private readonly channelRetriever: channel.Retriever;

  constructor({ instrumentation, core }: CoreProps) {
    this.ins = instrumentation;
    this.channelRetriever = core.channels.createDebouncedBatchRetriever(10);
    this.cache = new cache.Cache({
      channelRetriever: this.channelRetriever,
      instrumentation: this.ins.child("cache"),
    });
    this.reader = new Reader({
      cache: this.cache,
      readRemote: async (tr, keys, fidelity) => {
        // Translate bigint fidelity (alignmentMultiple) to the iterator's
        // integer downsampleFactor. A fidelity of 1n means raw, no decimation.
        const downsampleFactor = fidelity > 1n ? Number(fidelity) : 1;
        return await core.read(tr, keys, { downsampleFactor });
      },
      instrumentation: this.ins.child("reader"),
    });
    this.streamer = new Streamer({
      cache: this.cache,
      openStreamer: async (keys) =>
        framer.HardenedStreamer.open(core.openStreamer.bind(core), keys),
      instrumentation: this.ins.child("streamer"),
    });
  }

  /** Implements ChannelClient. */
  async retrieveChannel(key: channel.Key | channel.Name): Promise<channel.Payload> {
    const res = await this.channelRetriever.retrieve([key] as channel.Params);
    if (res.length === 0) throw new QueryError(`channel ${key} not found`);
    return res[0];
  }

  /** Implements ReadClient. */
  async read(
    tr: TimeRange,
    key: channel.Key,
    fidelity?: bigint,
  ): Promise<MultiSeries> {
    return await this.reader.read(tr, key, fidelity);
  }

  async stream(
    handler: StreamHandler,
    keys: channel.Key[],
  ): Promise<destructor.Destructor> {
    return await this.streamer.stream(handler, keys);
  }

  /** Implements Client. */
  async close(): Promise<void> {
    this.ins.L.info("closing client");
    await this.streamer.close();
    await this.reader.close();
    this.cache.close();
  }
}
