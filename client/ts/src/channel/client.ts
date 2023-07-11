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
  Rate,
  NativeTypedArray,
  UnparsedDensity,
  UnparsedTimeStamp,
  Series,
  TimeRange,
  AsyncTermSearcher,
  toArray,
} from "@synnaxlabs/x";

import { ChannelCreator } from "@/channel/creator";
import {
  ChannelKey,
  ChannelKeyOrName,
  ChannelParams,
  ChannelPayload,
  channelPayload,
  NewChannelPayload,
} from "@/channel/payload";
import { analyzeChannelParams, ChannelRetriever } from "@/channel/retriever";
import { QueryError } from "@/errors";
import { FrameClient } from "@/framer";

/**
 * Represents a Channel in a Synnax database. It should not be instantiated
 * directly, but rather created or retrieved from a {@link ChannelClient}.
 */
export class Channel {
  private readonly _frameClient: FrameClient | null;
  key: ChannelKey;
  name: string;
  rate: Rate;
  dataType: DataType;
  leaseholder: number;
  index: ChannelKey;
  isIndex: boolean;

  constructor({
    dataType,
    rate,
    name,
    leaseholder = 0,
    key = 0,
    density = 0,
    isIndex = false,
    index = 0,
    frameClient,
  }: NewChannelPayload & {
    frameClient?: FrameClient;
    density?: UnparsedDensity;
  }) {
    this.key = key;
    this.name = name;
    this.rate = new Rate(rate ?? 0);
    this.dataType = new DataType(dataType);
    this.leaseholder = leaseholder;
    this.index = index;
    this.isIndex = isIndex;
    this._frameClient = frameClient ?? null;
  }

  private get framer(): FrameClient {
    if (this._frameClient == null)
      throw new Error("cannot read from a channel that has not been created");
    return this._frameClient;
  }

  get payload(): ChannelPayload {
    return channelPayload.parse({
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
  async write(start: UnparsedTimeStamp, data: NativeTypedArray): Promise<void> {
    return await this.framer.write(this.key, start, data);
  }
}

/**
 * The core client class for executing channel operations against a Synnax
 * cluster.
 */
export class ChannelClient implements AsyncTermSearcher<string, ChannelKey, Channel> {
  private readonly frameClient: FrameClient;
  private readonly retriever: ChannelRetriever;
  private readonly creator: ChannelCreator;

  constructor(
    segmentClient: FrameClient,
    retriever: ChannelRetriever,
    creator: ChannelCreator
  ) {
    this.frameClient = segmentClient;
    this.retriever = retriever;
    this.creator = creator;
  }

  async create(channel: NewChannelPayload): Promise<Channel>;

  async create(channels: NewChannelPayload[]): Promise<Channel[]>;

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
    channels: NewChannelPayload | NewChannelPayload[]
  ): Promise<Channel | Channel[]> {
    const single = !Array.isArray(channels);
    const res = this.sugar(await this.creator.create(toArray(channels)));
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
    const { single, actual, normalized } = analyzeChannelParams(channels);
    if (normalized.length === 0) return [];
    const res = this.sugar(await this.retriever.retrieve(channels));
    if (!single) return res;
    if (res.length === 0) throw new QueryError(`channel matching ${actual} not found`);
    if (res.length > 1)
      throw new QueryError(`multiple channels matching ${actual} found`);
    return res[0];
  }

  async search(term: string): Promise<Channel[]> {
    return this.sugar(await this.retriever.search(term));
  }

  private sugar(payloads: ChannelPayload[]): Channel[] {
    const { frameClient } = this;
    return payloads.map((p) => new Channel({ ...p, frameClient }));
  }
}
