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

  get density(): Density | undefined {
    return this.payload.density;
  }

  async read(
    start: UnparsedTimeStamp,
    end: UnparsedTimeStamp
  ): Promise<TypedArray> {
    return await this.segmentClient.read(this.key, start, end);
  }
}

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

  async create(props: CreateChannelProps): Promise<Channel> {
    return this.sugar(await this.creator.create(props))[0];
  }

  async createMany(
    props: CreateChannelProps & { count: number }
  ): Promise<Channel[]> {
    return this.sugar(...(await this.creator.createMany(props)));
  }

  async retrieveByKeys(...keys: string[]): Promise<Channel[]> {
    return this.sugar(...(await this.retriever.retrieveByKeys(...keys)));
  }

  async retrieveByNames(...names: string[]): Promise<Channel[]> {
    return this.sugar(...(await this.retriever.retrieveByNames(...names)));
  }

  async retrieveByNodeID(nodeId: number): Promise<Channel[]> {
    return this.sugar(...(await this.retriever.retrieveByNodeID(nodeId)));
  }

  private sugar(...payloads: ChannelPayload[]): Channel[] {
    return payloads.map((p) => new Channel(p, this.segmentClient));
  }
}
