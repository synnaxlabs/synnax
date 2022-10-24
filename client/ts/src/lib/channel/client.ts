import SegmentClient from '../segment/client';
import {
  DataType,
  Density,
  Rate,
  TypedArray,
  UnparsedTimeStamp,
} from '../telem';

import ChannelCreator from './creator';
import { CreateChannelProps } from './creator';
import { ChannelPayload } from './payload';
import ChannelRetriever from './retriever';

/**
 * Represents a Channel in a Synnax database. It should not be instantiated
 * directly, but rather created or retrieved from a {@link ChannelClient}.
 */
export class Channel {
  private readonly segmentClient: SegmentClient;
  private payload: ChannelPayload;

  constructor(payload: ChannelPayload, segmentClient: SegmentClient) {
    this.payload = payload;
    this.segmentClient = segmentClient;
  }

  get key(): string {
    if (!this.payload.key) {
      throw new Error('Channel key is not set');
    }
    return this.payload.key;
  }

  get name(): string {
    if (!this.payload.name) {
      throw new Error('Channel name is not set');
    }
    return this.payload.name;
  }

  get nodeId(): number {
    if (this.payload.nodeId === undefined) {
      throw new Error('Channel nodeId is not set');
    }
    return this.payload.nodeId;
  }

  get rate(): Rate {
    return this.payload.rate;
  }

  get dataType(): DataType {
    return this.payload.dataType;
  }

  get density(): Density {
    if (!this.payload.density) {
      throw new Error('Channel density is not set');
    }
    return this.payload.density;
  }

  /**
   * Reads telemetry from the channel between the two timestamps.
   *
   * @param start - The starting timestamp of the range to read from.
   * @param end - The ending timestamp of the range to read from.
   * @returns A typed array containing the retrieved
   */
  async read(
    start: UnparsedTimeStamp,
    end: UnparsedTimeStamp
  ): Promise<TypedArray> {
    return await this.segmentClient.read(this.key, start, end);
  }

  /**
   * Writes telemetry to the channel starting at the given timestamp.
   *
   * @param start - The starting timestamp of the first sample in data.
   * @param data - THe telemetry to write to the channel.
   */
  async write(start: UnparsedTimeStamp, data: TypedArray) {
    return await this.segmentClient.write(this.key, start, data);
  }
}

/**
 * The core client class for executing channel operations against a Synnax
 * cluster.
 */
export default class ChannelClient {
  private readonly segmentClient: SegmentClient;
  private readonly retriever: ChannelRetriever;
  private readonly creator: ChannelCreator;

  constructor(
    segmentClient: SegmentClient,
    retriever: ChannelRetriever,
    creator: ChannelCreator
  ) {
    this.segmentClient = segmentClient;
    this.retriever = retriever;
    this.creator = creator;
  }

  /**
   * Creates a new channel with the given properties.
   *
   * @param props.rate - The rate of the channel.
   * @param props.dataType - The data type of the channel.
   * @param props.name - The name of the channel. Optional.
   * @param props.nodeId - The ID of the node that holds the lease on the
   *   channel. If you don't know what this is, don't worry about it.
   * @returns The created channel.
   */
  async create(props: CreateChannelProps): Promise<Channel> {
    return (await this.createMany({ ...props, count: 1 }))[0];
  }

  /**
   * Creates N channels using the given parameters as a template.
   *
   * @param props.rate - The rate of the channel.
   * @param props.dataType - The data type of the channel.
   * @param props.name - The name of the channel. Optional.
   * @param props.nodeId - The ID of the node that holds the lease on the
   *   channel. If you don't know what this is, don't worry about it.
   * @param props.count - The number of channels to create.
   * @returns The created channels.
   */
  async createMany(
    props: CreateChannelProps & { count: number }
  ): Promise<Channel[]> {
    return this.sugar(...(await this.creator.createMany(props)));
  }

  /**
   * Retrieves channels with the given keys.
   *
   * @param keys - The keys of the channels to retrieve.
   * @returns The retrieved channels.
   * @throws QueryError if any of the channels can't be found.
   */
  async retrieveByKeys(...keys: string[]): Promise<Channel[]> {
    return this.sugar(...(await this.retriever.retrieveByKeys(...keys)));
  }

  /**
   * Retrieves channels with the given names.
   *
   * @param names - The list of names to retrieve channels for.
   * @returns A list of retrieved channels matching the given names. If a
   *   channel with a given name can't be found, it will be omitted from the
   *   list.
   */
  async retrieveByNames(...names: string[]): Promise<Channel[]> {
    return this.sugar(...(await this.retriever.retrieveByNames(...names)));
  }

  /**
   * Retrieves channels whose lease node is the given ID.
   *
   * @param nodeId - The ID of the node to retrieve channels for.
   * @returns A list of retrieved channels matching the given node ID.
   */
  async retrieveByNodeId(nodeId: number): Promise<Channel[]> {
    return this.sugar(...(await this.retriever.retrieveByNodeID(nodeId)));
  }

  private sugar(...payloads: ChannelPayload[]): Channel[] {
    return payloads.map((p) => new Channel(p, this.segmentClient));
  }
}
