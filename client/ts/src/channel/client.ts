// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { type AsyncTermSearcher } from "@synnaxlabs/x/search";
import {
  DataType,
  Rate,
  type NativeTypedArray,
  type CrudeDensity,
  type Series,
  type TimeRange,
  type CrudeTimeSpan,
} from "@synnaxlabs/x/telem";
import { toArray } from "@synnaxlabs/x/toArray";

import { type Creator } from "@/channel/creator";
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
  type PageOptions,
  type RetrieveOptions,
  type Retriever,
} from "@/channel/retriever";
import { MultipleResultsError, NotFoundError } from "@/errors";
import { type framer } from "@/framer";

/**
 * Represents a Channel in a Synnax database. It should not be instantiated
 * directly, but rather created or retrieved from a {@link Client}.
 */
export class Channel {
  private readonly _frameClient: framer.Client | null;
  key: Key;
  name: string;
  rate: Rate;
  dataType: DataType;
  leaseholder: number;
  index: Key;
  isIndex: boolean;
  alias: string | undefined;

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
      throw new Error("cannot read from a channel that has not been created");
    return this._frameClient;
  }

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
  async write(start: CrudeTimeSpan, data: NativeTypedArray): Promise<void> {
    return await this.framer.write(this.key, start, data);
  }
}

/**
 * The core client class for executing channel operations against a Synnax
 * cluster.
 */
export class Client implements AsyncTermSearcher<string, Key, Channel> {
  private readonly frameClient: framer.Client;
  private readonly retriever: Retriever;
  private readonly creator: Creator;
  private readonly client: UnaryClient;

  constructor(
    segmentClient: framer.Client,
    retriever: Retriever,
    client: UnaryClient,
    creator: Creator,
  ) {
    this.frameClient = segmentClient;
    this.retriever = retriever;
    this.client = client;
    this.creator = creator;
  }

  async create(channel: NewPayload): Promise<Channel>;

  async create(channels: NewPayload[]): Promise<Channel[]>;

  /**
   * Creates a new channel with the given properties.
   *
   * @param props.rate - The rate of the channel.
   * @param props.dataType - The data type of the channel.
   * @param props.name - The name of the channel. Optional.
   * @param props.nodeKey - The ID of the node that holds the lease on the
   *   channel. If you don't know what this is, don't worry about it.
   * @returns The created channel.
   */
  async create(channels: NewPayload | NewPayload[]): Promise<Channel | Channel[]> {
    const single = !Array.isArray(channels);
    const res = this.sugar(await this.creator.create(toArray(channels)));
    return single ? res[0] : res;
  }

  async retrieve(channel: KeyOrName, options?: RetrieveOptions): Promise<Channel>;

  async retrieve(channels: Params, options?: RetrieveOptions): Promise<Channel[]>;

  /**
   * Retrieves a channel from the database using the given parameters.
   * @param props.key - The key of the channel to retrieve.
   * @param props.name - The name of the channel to retrieve. If props.key is set,
   * this will be ignored.
   * @returns The retrieved channel.
   * @raises {QueryError} If the channel does not exist or if multiple results are returned.
   */
  async retrieve(
    channels: Params,
    options?: RetrieveOptions,
  ): Promise<Channel | Channel[]> {
    const { single, actual, normalized } = analyzeParams(channels);
    if (normalized.length === 0) return [];
    const res = this.sugar(await this.retriever.retrieve(channels, options));
    if (!single) return res;
    if (res.length === 0)
      throw new NotFoundError(`channel matching ${actual} not found`);
    if (res.length > 1)
      throw new MultipleResultsError(`multiple channels matching ${actual} found`);
    return res[0];
  }

  async search(term: string, options?: RetrieveOptions): Promise<Channel[]> {
    return this.sugar(await this.retriever.search(term, options));
  }

  newSearcherWithOptions(
    options: RetrieveOptions,
  ): AsyncTermSearcher<string, Key, Channel> {
    return new SearcherWithOptions(this, options);
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
}

class SearcherWithOptions implements AsyncTermSearcher<string, Key, Channel> {
  private readonly client: Client;
  private readonly options: RetrieveOptions;

  constructor(client: Client, options: RetrieveOptions) {
    this.client = client;
    this.options = options;
  }

  async search(term: string, options?: RetrieveOptions): Promise<Channel[]> {
    return await this.client.search(term, { ...this.options, ...options });
  }

  async page(offset: number, limit: number, options?: PageOptions): Promise<Channel[]> {
    return await this.client.page(offset, limit, { ...this.options, ...options });
  }

  async retrieve(channels: Key[], options?: RetrieveOptions): Promise<Channel[]> {
    return await this.client.retrieve(channels, { ...this.options, ...options });
  }
}
