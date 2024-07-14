// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos } from "@synnaxlabs/alamos";
import { type channel, QueryError, type Synnax } from "@synnaxlabs/client";
import { type AsyncDestructor, id, type TimeRange } from "@synnaxlabs/x";

import { cache } from "@/telem/client/cache";
import { Reader } from "@/telem/client/reader";
import { Streamer, type StreamHandler } from "@/telem/client/streamer";
import { type ReadResponse } from "@/telem/client/types";

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
  retrieveChannel: (key: channel.KeyOrName) => Promise<channel.Payload>;
}

/** A client that can be used to read telemetry from the Synnax cluster. */
export interface ReadClient {
  /**
   * Reads telemetry from the given channels for the given time range.
   *
   * @param tr  - The time range to read from.
   * @param keys - The keys of the channels to read from.
   * @returns a record with a read response for each channel in keys, regardless of
   * whether or not data was found for the given channel. NOTE: Responses are not
   * guaranteed to have the same topology i.e each response may contain a different
   * number of Series with different lengths. It's up to the caller to use the
   * 'alignment' field of the Series to normalize the data shape if necessary.
   */
  read: (
    tr: TimeRange,
    keys: channel.Keys,
  ) => Promise<Record<channel.Key, ReadResponse>>;
}

/** A client that can be used to stream telemetry from the Synnax cluster. */
export interface StreamClient {
  stream: (handler: StreamHandler, keys: channel.Keys) => Promise<AsyncDestructor>;
}

/**
 * Client provides an interface for performing basic telemetry operations
 * against a Synnax cluster. This interface is a simplification of the Synnax
 * client to make it easy to stub out for testing.
 */
export interface Client extends ChannelClient, ReadClient, StreamClient {
  key: string;
  /** Close closes the client, releasing all resources from the cache. */
  close: () => Promise<void>;
}

/**
 * Proxy is a Client implementation that proxies all operations to another Client,
 * allowing the underlying Client to be swapped out at runtime. If no Client is
 * set, all operations will throw an error.
 */
export class Proxy implements Client {
  key: string = id.id();
  _client: Client | null = null;

  async swap(client: Client | null): Promise<void> {
    this.key = id.id();
    await this._client?.close();
    this._client = client;
  }

  /** Implements ChannelClient. */
  async retrieveChannel(key: channel.KeyOrName): Promise<channel.Payload> {
    return await this.client.retrieveChannel(key);
  }

  /** Implements ReadClient. */
  async read(
    tr: TimeRange,
    channels: channel.Keys,
  ): Promise<Record<channel.Key, ReadResponse>> {
    return await this.client.read(tr, channels);
  }

  /** Stream implements StreamClient. */
  async stream(handler: StreamHandler, keys: channel.Keys): Promise<AsyncDestructor> {
    return await this.client.stream(handler, keys);
  }

  /** Close implements CLient. */
  async close(): Promise<void> {
    await this.client.close();
  }

  private get client(): Client {
    if (this._client == null) throw new QueryError("No cluster has been connected");
    return this._client;
  }
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
  readonly key: string = id.id();
  private readonly ins: alamos.Instrumentation;

  private readonly cache: cache.Cache;
  private readonly reader: Reader;
  private readonly streamer: Streamer;
  private readonly channelRetriever: channel.Retriever;

  constructor({ instrumentation, core }: CoreProps) {
    this.ins = instrumentation ?? alamos.NOOP;
    this.channelRetriever = core.channels.createDebouncedBatchRetriever(10);
    this.cache = new cache.Cache({
      channelRetriever: this.channelRetriever,
      instrumentation: this.ins.child("cache"),
    });
    this.reader = new Reader({
      cache: this.cache,
      readRemote: async (tr, keys) => await core.read(tr, keys),
      instrumentation: this.ins.child("reader"),
    });
    this.streamer = new Streamer({
      cache: this.cache,
      core,
      instrumentation: this.ins.child("streamer"),
    });
  }

  /** Implements ChannelClient. */
  async retrieveChannel(key: channel.KeyOrName): Promise<channel.Payload> {
    const res = await this.channelRetriever.retrieve([key] as channel.Params);
    if (res.length === 0) throw new QueryError(`channel ${key} not found`);
    return res[0];
  }

  /** Implements ChannelClient */
  async read(
    tr: TimeRange,
    keys: channel.Keys,
  ): Promise<Record<channel.Key, ReadResponse>> {
    return await this.reader.read(tr, keys);
  }

  async stream(handler: StreamHandler, keys: channel.Keys): Promise<AsyncDestructor> {
    return await this.streamer.stream(handler, keys);
  }

  /** Implements Client. */
  async close(): Promise<void> {
    this.ins.L.info("closing client", { key: this.key });
    await this.streamer.close();
    this.cache.close();
  }
}
