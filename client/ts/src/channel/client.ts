// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  DataType,
  Density,
  Rate,
  NativeTypedArray,
  UnparsedDensity,
  UnparsedTimeStamp,
  toArray,
  LazyArray,
  TimeRange,
} from "@synnaxlabs/x";

import { QueryError } from "..";

import { ChannelCreator } from "./creator";
import {
  ChannelKey,
  ChannelKeyOrName,
  ChannelParams,
  ChannelPayload,
  channelPayload,
  UnparsedChannel,
} from "./payload";
import { analyzeChannelParams, ChannelRetriever } from "./retriever";

import { FrameClient } from "@/framer";

/**
 * Represents a Channel in a Synnax database. It should not be instantiated
 * directly, but rather created or retrieved from a {@link ChannelClient}.
 */
export class Channel {
  private readonly _frameClient: FrameClient | null;
  payload: ChannelPayload;

  constructor({
    dataType,
    rate,
    name = "",
    leaseholder = 0,
    key = 0,
    density = 0,
    isIndex = false,
    index = 0,
    segmentClient,
  }: UnparsedChannel & {
    segmentClient?: FrameClient;
    density?: UnparsedDensity;
  }) {
    this.payload = channelPayload.parse({
      dataType: new DataType(dataType).valueOf(),
      rate: new Rate(rate ?? 0).valueOf(),
      name,
      leaseholder,
      key,
      density: new Density(density).valueOf(),
      isIndex,
      index,
    });
    this._frameClient = segmentClient ?? null;
  }

  private get framer(): FrameClient {
    if (this._frameClient == null)
      throw new Error("cannot read from a channel that has not been created");
    return this._frameClient;
  }

  get key(): number {
    if (this.payload.key == null) throw new Error("channel key is not set");
    return this.payload.key;
  }

  get name(): string {
    if (this.payload.name == null) throw new Error("channel name is not set");
    return this.payload.name;
  }

  get leaseholder(): number {
    if (this.payload.leaseholder == null) throw new Error("chanel nodeKey is not set");
    return this.payload.leaseholder;
  }

  get rate(): Rate {
    return this.payload.rate;
  }

  get isIndexed(): boolean {
    return this.index !== 0;
  }

  get index(): ChannelKey {
    return this.payload.index ?? 0;
  }

  get isIndex(): boolean {
    return this.payload.isIndex ?? false;
  }

  get dataType(): DataType {
    return this.payload.dataType;
  }

  /**
   * Reads telemetry from the channel between the two timestamps.
   *
   * @param start - The starting timestamp of the range to read from.
   * @param end - The ending timestamp of the range to read from.
   * @returns A typed array containing the retrieved
   */
  async read(tr: TimeRange): Promise<LazyArray | undefined> {
    return await this.framer.read(tr, this.key);
  }

  /**
   * Writes telemetry to the channel starting at the given timestamp.
   *
   * @param start - The starting timestamp of the first sample in data.
   * @param data - THe telemetry to write to the channel.
   */
  async write(start: UnparsedTimeStamp, data: NativeTypedArray): Promise<void> {
    return await this.framer.write(this.key, start, data);
  }
}

/**
 * The core client class for executing channel operations against a Synnax
 * cluster.
 */
export class ChannelClient {
  private readonly segmentClient: FrameClient;
  private readonly retriever: ChannelRetriever;
  private readonly creator: ChannelCreator;

  constructor(
    segmentClient: FrameClient,
    retriever: ChannelRetriever,
    creator: ChannelCreator
  ) {
    this.segmentClient = segmentClient;
    this.retriever = retriever;
    this.creator = creator;
  }

  async create(channel: UnparsedChannel): Promise<Channel>;

  async create(channels: UnparsedChannel[]): Promise<Channel[]>;

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
  async create(
    channels: UnparsedChannel | UnparsedChannel[]
  ): Promise<Channel | Channel[]> {
    const single = !Array.isArray(channels);
    const res = this.sugar(await this.creator.create(channels));
    return single ? res[0] : res;
  }

  async retrieve(channel: ChannelKeyOrName): Promise<Channel>;

  async retrieve(channels: ChannelParams): Promise<Channel[]>;

  /**
   * Retrieves a channel from the database using the given parameters.
   * @param props.key - The key of the channel to retrieve.
   * @param props.name - The name of the channel to retrieve. If props.key is set,
   * this will be ignored.
   * @returns The retrieved channel.
   * @raises {QueryError} If the channel does not exist or if multiple results are returned.
   */
  async retrieve(channels: ChannelParams): Promise<Channel | Channel[]> {
    const { single, actual } = analyzeChannelParams(channels);
    const res = this.sugar(await this.retriever.retrieve(channels));
    if (single) {
      if (res.length === 0)
        throw new QueryError(`channel matching ${actual} not found`);
      if (res.length > 1)
        throw new QueryError(`multiple channels matching ${actual} found`);
    }
    return single ? res[0] : res;
  }

  async retrieveAll(): Promise<Channel[]> {
    return this.sugar(await this.retriever.retrieveAll());
  }

  private sugar(payloads: ChannelPayload | ChannelPayload[]): Channel[] {
    const { segmentClient } = this;
    return toArray(payloads).map((p) => new Channel({ ...p, segmentClient }));
  }
}
