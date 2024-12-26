// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { type AsyncTermSearcher } from "@synnaxlabs/x/search";
import {
  type CrudeDensity,
  type CrudeTimeStamp,
  DataType,
  type MultiSeries,
  Rate,
  type TimeRange,
  type TypedArray,
} from "@synnaxlabs/x/telem";
import { toArray } from "@synnaxlabs/x/toArray";
import { z } from "zod";

import {
  type Key,
  type KeyOrName,
  type NewPayload,
  type Params,
  type Payload,
  payload,
} from "@/channel/payload";
import {
  analyzeChannelParams,
  CacheRetriever,
  ClusterRetriever,
  DebouncedBatchRetriever,
  type RetrieveOptions,
  type Retriever,
} from "@/channel/retriever";
import { type Writer } from "@/channel/writer";
import { ValidationError } from "@/errors";
import { type framer } from "@/framer";
import { ontology } from "@/ontology";
import { group } from "@/ontology/group";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

interface CreateOptions {
  retrieveIfNameExists?: boolean;
}

/**
 * Represents a Channel in a Synnax database. Typically, channels should not be
 * instantiated directly, but instead created via the `.channels.create` or retrieved
 * via the `.channels.retrieve` method on a Synnax client.
 *
 * Please refer to the [Synnax
 * documentation](https://docs.synnaxlabs.com/reference/concepts/channels) for detailed
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
   * This is set to true if the channel is an internal channel, and false otherwise.
   */
  readonly internal: boolean;
  /**
   * An alias for the channel under a specific range. This parameter is unstable and
   * should not be relied upon in the current version of Synnax.
   */
  readonly alias: string | undefined;
  /**
   * Whether the channel is virtual. Virtual channels do not store any data in the
   * database, but can still be used for streaming purposes.
   */
  readonly virtual: boolean;

  constructor({
    dataType,
    rate,
    name,
    leaseholder = 0,
    key = 0,
    isIndex = false,
    index = 0,
    internal = false,
    virtual = false,
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
    this.internal = internal;
    this.alias = alias;
    this.virtual = virtual;
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
      internal: this.internal,
    });
  }

  /***
   * @returns the ontology ID of the channel
   */
  get ontologyID(): ontology.ID {
    return new ontology.ID({ type: "channel", key: this.key.toString() });
  }

  /**
   * Reads telemetry from the channel between the two timestamps.
   *
   * @param start - The starting timestamp of the range to read from.
   * @param end - The ending timestamp of the range to read from.
   * @returns A typed array containing the retrieved
   */
  async read(tr: TimeRange): Promise<MultiSeries> {
    return await this.framer.read(tr, this.key);
  }

  /**
   * Writes telemetry to the channel starting at the given timestamp.
   *
   * @param start - The starting timestamp of the first sample in data.
   * @param data - THe telemetry to write to the channel.
   */
  async write(start: CrudeTimeStamp, data: TypedArray): Promise<void> {
    return await this.framer.write(start, this.key, data);
  }
}

const RETRIEVE_GROUP_ENDPOINT = "/channel/retrieve-group";

const retrieveGroupReqZ = z.object({});

const retrieveGroupResZ = z.object({
  group: group.groupZ,
});

/**
 * The core client class for executing channel operations against a Synnax
 * cluster. This class should not be instantiated directly, and instead should be used
 * through the `channels` property of an {@link Synnax} client.
 */
export class Client implements AsyncTermSearcher<string, Key, Channel> {
  readonly type = "channel";
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
   * with. An 'index' channel is a channel that stores timestamps for other channels.
   * Refer to the Synnax documentation
   * (https://docs.synnaxlabs.com/reference/concepts/channels) for more information. The
   * index channel must have already been created. This field does not need to be
   * specified if the channel is an index channel, or the channel is a fixed rate
   * channel. If this value is specified, the 'rate' parameter will be ignored.
   * @param isIndex - Set to true if the channel is an index channel, and false
   * otherwise. Index channels must have a data type of `DataType.TIMESTAMP`.
   * @returns the created channel. {@link Channel}
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
   * @throws {MultipleFoundError} is only thrown if the channel is retrieved by name,
   * and multiple channels with the same name exist in the cluster.
   *
   * @example
   *
   * ```typescript
   * const channel = await client.channels.retrieve("temperature");
   * const channel = await client.channels.retrieve(1);
   * ```
   */
  async retrieve(channel: KeyOrName, options?: RetrieveOptions): Promise<Channel>;

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
  async retrieve(channels: Params, options?: RetrieveOptions): Promise<Channel[]>;

  /**
   * Retrieves a channel from the database using the given parameters.
   *
   * this will be ignored.
   * @returns The retrieved channel.
   * @raises {QueryError} If the channel does not exist or if multiple results are returned.
   */
  async retrieve(
    channels: Params,
    options?: RetrieveOptions,
  ): Promise<Channel | Channel[]> {
    const isSingle = !Array.isArray(channels);
    const res = this.sugar(await this.retriever.retrieve(channels, options));
    checkForMultipleOrNoResults("channel", channels, res, isSingle);
    return isSingle ? res[0] : res;
  }

  async search(term: string, options?: RetrieveOptions): Promise<Channel[]> {
    return this.sugar(await this.retriever.search(term, options));
  }

  /***
   * Deletes channels from the database using the given keys or names.
   * @param channels - The keys or names of the channels to delete.
   */
  async delete(channels: Params): Promise<void> {
    const { normalized, variant } = analyzeChannelParams(channels);
    if (variant === "keys")
      return await this.writer.delete({ keys: normalized as Key[] });
    return await this.writer.delete({ names: normalized as string[] });
  }

  async rename(key: Key, name: string): Promise<void>;
  async rename(keys: Key[], names: string[]): Promise<void>;

  async rename(keys: Key | Key[], names: string | string[]): Promise<void> {
    return await this.writer.rename(toArray(keys), toArray(names));
  }

  newSearcherWithOptions(
    options: RetrieveOptions,
  ): AsyncTermSearcher<string, Key, Channel> {
    return {
      type: this.type,
      search: async (term: string) => await this.search(term, options),
      retrieve: async (keys: Key[]) => await this.retrieve(keys, options),
      page: async (offset: number, limit: number) =>
        await this.page(offset, limit, options),
    };
  }

  async page(
    offset: number,
    limit: number,
    options?: Omit<RetrieveOptions, "limit" | "offset">,
  ): Promise<Channel[]> {
    return this.sugar(await this.retriever.page(offset, limit, options));
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

  async retrieveGroup(): Promise<group.Group> {
    const res = await sendRequired(
      this.client,
      RETRIEVE_GROUP_ENDPOINT,
      {},
      retrieveGroupReqZ,
      retrieveGroupResZ,
    );
    return new group.Group(res.group.name, res.group.key);
  }
}
