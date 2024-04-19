// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import {
  DataType,
  Rate,
  type TypedArray,
  type CrudeDensity,
  type Series,
  type TimeRange,
  type AsyncTermSearcher,
  toArray,
  type CrudeTimeStamp,
} from "@synnaxlabs/x";

import {
  type Key,
  type KeyOrName,
  type Params,
  type Payload,
  payload,
  type NewPayload,
} from "@/channel/payload";
import {
  analyzeParams,
  CacheRetriever,
  ClusterRetriever,
  DebouncedBatchRetriever,
  type Retriever,
} from "@/channel/retriever";
import { type Writer } from "@/channel/writer";
import { MultipleResultsError, NoResultsError, ValidationError } from "@/errors";
import { type framer } from "@/framer";

interface CreateOptions {
  retrieveIfNameExists?: boolean;
}

/**
 * Represents a Channel in a Synnax database. Typically, channels should not be
 * instantiated directly, but instead created via the `.channels.create` or retrieved
 * via the `.channels.retrieve` method on a Synnax client.
 *
 * Please refer to the [Synnax documentation](https://docs.synnaxlabs.com) for detailed
 * information on what channels are and how to use them.
 */
export class Channel {
  private readonly _frameClient: framer.Client | null;
  /**
   * A unique key identifying the channel in the Synnax database. This key is
   * automatically assigned by Synnax.
   */
  readonly key: Key;
  /**
   * A human-readable name for the channel. This name is not guaranteed to be
   * unique.
   */
  readonly name: string;
  /**
   * The rate at which the channel samples telemetry. This only applies to fixed rate
   * channels, and will be 0 if the channel is indexed.
   */
  readonly rate: Rate;
  /**
   * The data type of the channel.
   */
  readonly dataType: DataType;
  /**
   * The key of the node in the Synnax cluster that holds the 'lease' over the channel
   * i.e. it's the only node in the cluster allowed to accept writes to the channel. This
   * property is mostly for internal use.
   */
  readonly leaseholder: number;
  /**
   * The key of the index channel that this channel is associated with i.e. the channel
   * that stores its timestamps.
   */
  readonly index: Key;
  /**
   * This is set to true if the channel is an index channel, and false otherwise.
   */
  readonly isIndex: boolean;
  /**
   * An alias for the channel under a specific range. This parameter is unstable and
   * should not be relied upon in the current version of Synnax.
   */
  readonly alias: string | undefined;

  constructor({
    dataType,
    rate,
    name,
    leaseholder = 0,
    key = 0,
    isIndex = false,
    index = 0,
    frameClient,
    alias,
  }: NewPayload & {
    frameClient?: framer.Client;
    density?: CrudeDensity;
  }) {
    this.key = key;
    this.name = name;
    this.rate = new Rate(rate ?? 0);
    this.dataType = new DataType(dataType);
    this.leaseholder = leaseholder;
    this.index = index;
    this.isIndex = isIndex;
    this.alias = alias;
    this._frameClient = frameClient ?? null;
  }

  private get framer(): framer.Client {
    if (this._frameClient == null)
      throw new ValidationError("cannot read from a channel that has not been created");
    return this._frameClient;
  }

  /**
   * Returns the payload representation of this channel i.e. a pure JS object with
   * all of the channel fields but without any methods. This is used internally for
   * network transportation, but also provided to you as a convenience.
   */
  get payload(): Payload {
    return payload.parse({
      key: this.key,
      name: this.name,
      rate: this.rate.valueOf(),
      dataType: this.dataType.valueOf(),
      leaseholder: this.leaseholder,
      index: this.index,
      isIndex: this.isIndex,
    });
  }

  /**
   * Reads telemetry from the channel between the two timestamps.
   *
   * @param start - The starting timestamp of the range to read from.
   * @param end - The ending timestamp of the range to read from.
   * @returns A typed array containing the retrieved
   */
  async read(tr: TimeRange): Promise<Series | undefined> {
    return await this.framer.read(tr, this.key);
  }

  /**
   * Writes telemetry to the channel starting at the given timestamp.
   *
   * @param start - The starting timestamp of the first sample in data.
   * @param data - THe telemetry to write to the channel.
   */
  async write(start: CrudeTimeStamp, data: TypedArray): Promise<void> {
    return await this.framer.write(this.key, start, data);
  }
}

/**
 * The core client class for executing channel operations against a Synnax
 * cluster. This class should not be instantiated directly, and instead should be used
 * through the `channels` property of an {@link Synnax} client.
 */
export class Client implements AsyncTermSearcher<string, Key, Channel> {
  private readonly frameClient: framer.Client;
  private readonly client: UnaryClient;
  readonly retriever: Retriever;
  readonly writer: Writer;

  constructor(
    frameClient: framer.Client,
    retriever: Retriever,
    client: UnaryClient,
    writer: Writer,
  ) {
    this.frameClient = frameClient;
    this.retriever = retriever;
    this.client = client;
    this.writer = writer;
  }

  /**
   * Creates a single channel with the given properties.
   *
   * @param name - A human-readable name for the channel.
   * @param rate - The rate of the channel. This only applies to fixed rate channels.
   * @param dataType - The data type for the samples stored in the channel.
   * @param index - The key of the index channel that this channel should be associated
   * with. An 'index' channel is a channel that stores timestamps for other channels. Refer
   * to the Synnax documentation (https://docs.synnaxlabs.com) for more information. The
   * index channel must have already been created. This field does not need to be specified
   * if the channel is an index channel, or the channel is a fixed rate channel. If this
   * value is specified, the 'rate' parameter will be ignored.
   * @param isIndex - Set to true if the channel is an index channel, and false otherwise.
   * Index channels must have a data type of `DataType.TIMESTAMP`.
   * @returns the created channel. {@see Channel}
   * @throws {ValidationError} if any of the parameters for creating the channel are
   * invalid.
   *
   * @example
   * ```typescript
   * const indexChannel = await client.channels.create({
   *    name: "time",
   *    dataType: DataType.TIMESTAMP,
   *    isIndex: true,
   * })
   *
   *
   * const dataChannel = await client.channels.create({
   *    name: "temperature",
   *    dataType: DataType.FLOAT,
   *    index: indexChannel.key,
   * });
   * ```
   */
  async create(channel: NewPayload, options?: CreateOptions): Promise<Channel>;

  /**
   * Creates multiple channels with the given properties. The order of the channels
   * returned is guaranteed to match the order of the channels passed in.
   *
   * @param channels - An array of channel properties to create.
   * For each channel, the following properties should be considered:
   *
   * @param name - A human-readable name for the channel.
   * @param rate - The rate of the channel. This only applies to fixed rate channels. If
   * the 'index' parameter is specified or 'isIndex' is set to true, this parameter will
   * be ignored.
   * @param dataType - The data type for the samples stored in the channel.
   * @param index - The key of the index channel that this channel should be associated
   * with. An 'index' channel is a channel that stores timestamps for other channels. Refer
   * to the Synnax documentation (https://docs.synnaxlabs.com) for more information. The
   * index channel must have already been created. This field does not need to be specified
   * if the channel is an index channel, or the channel is a fixed rate channel. If this
   * value is specified, the 'rate' parameter will be ignored.
   * @param isIndex - Set to true if the channel is an index channel, and false otherwise.
   * Index channels must have a data type of `DataType.TIMESTAMP`.
   *
   * @param channels
   */
  async create(channels: NewPayload[], options?: CreateOptions): Promise<Channel[]>;

  async create(
    channels: NewPayload | NewPayload[],
    options: CreateOptions = {},
  ): Promise<Channel | Channel[]> {
    const { retrieveIfNameExists = false } = options;
    const single = !Array.isArray(channels);
    let toCreate = toArray(channels);
    let created: Channel[] = [];
    if (retrieveIfNameExists) {
      const res = await this.retriever.retrieve(toCreate.map((c) => c.name));
      const existingNames = new Set(res.map((c) => c.name));
      toCreate = toCreate.filter((c) => !existingNames.has(c.name));
      created = this.sugar(res);
    }
    created = created.concat(this.sugar(await this.writer.create(toCreate)));
    return single ? created[0] : created;
  }

  /**
   * Retrieves a channel from the database using the given key or name.
   *
   * @param channel - The key or name of the channel to retrieve.
   * @param options - Optional parameters to control the retrieval process.
   * @param options.dataTypes - Limits the query to only channels with the specified data
   * type.
   * @param options.notDataTypes - Limits the query to only channels without the specified
   * data type.
   *
   * @returns The retrieved channel.
   * @throws {NotFoundError} if the channel does not exist in the cluster.
   * @throws {MultipleResultsError} is only thrown if the channel is retrieved by name,
   * and multiple channels with the same name exist in the cluster.
   *
   * @example
   *
   * ```typescript
   * const channel = await client.channels.retrieve("temperature");
   * const channel = await client.channels.retrieve(1);
   * ```
   */
  async retrieve(channel: KeyOrName, rangeKey?: string): Promise<Channel>;

  /**
   * Retrieves multiple channels from the database using the provided keys or the
   * provided names.
   *
   * @param channels - The keys or the names of the channels to retrieve. Note that
   * this method does not support mixing keys and names in the same call.
   * @param options - Optional parameters to control the retrieval process.
   * @param options.dataTypes - Limits the query to only channels with the specified data
   * type.
   * @param options.notDataTypes - Limits the query to only channels without the specified
   *
   */
  async retrieve(channels: Params, rangeKey?: string): Promise<Channel[]>;

  /**
   * Retrieves a channel from the database using the given parameters.
   *
   * this will be ignored.
   * @returns The retrieved channel.
   * @raises {QueryError} If the channel does not exist or if multiple results are returned.
   */
  async retrieve(channels: Params, rangeKey?: string): Promise<Channel | Channel[]> {
    const { single, actual, normalized } = analyzeParams(channels);
    if (normalized.length === 0) return [];
    const res = this.sugar(await this.retriever.retrieve(channels, rangeKey));
    if (!single) return res;
    if (res.length === 0)
      throw new NoResultsError(`channel matching ${actual} not found`);
    if (res.length > 1)
      throw new MultipleResultsError(`multiple channels matching ${actual} found`);
    return res[0];
  }

  async delete(channels: Params): Promise<void> {
    const { normalized, variant } = analyzeParams(channels);
    if (variant === "keys") return await this.writer.delete({ keys: normalized });
    return await this.writer.delete({ names: normalized });
  }

  async search(term: string, rangeKey?: string): Promise<Channel[]> {
    return this.sugar(await this.retriever.search(term, rangeKey));
  }

  newSearcherUnderRange(rangeKey?: string): AsyncTermSearcher<string, Key, Channel> {
    return new SearcherUnderRange(this, rangeKey);
  }

  async page(offset: number, limit: number, rangeKey?: string): Promise<Channel[]> {
    return this.sugar(await this.retriever.page(offset, limit, rangeKey));
  }

  createDebouncedBatchRetriever(deb: number = 10): Retriever {
    return new CacheRetriever(
      new DebouncedBatchRetriever(new ClusterRetriever(this.client), deb),
    );
  }

  private sugar(payloads: Payload[]): Channel[] {
    const { frameClient } = this;
    return payloads.map((p) => new Channel({ ...p, frameClient }));
  }
}

class SearcherUnderRange implements AsyncTermSearcher<string, Key, Channel> {
  private readonly client: Client;
  private readonly rangeKey?: string;

  constructor(client: Client, rangeKey?: string) {
    this.client = client;
    this.rangeKey = rangeKey;
  }

  async search(term: string): Promise<Channel[]> {
    return await this.client.search(term, this.rangeKey);
  }

  async page(offset: number, limit: number): Promise<Channel[]> {
    return await this.client.page(offset, limit, this.rangeKey);
  }

  async retrieve(channels: Key[]): Promise<Channel[]> {
    return await this.client.retrieve(channels, this.rangeKey);
  }
}
