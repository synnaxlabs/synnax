import SegmentClient from '../segment/client';
import { DataType, Density, Rate, TypedArray, UnparsedTimeStamp } from '../telem';
import ChannelCreator from './creator';
import { CreateChannelProps } from './creator';
import { ChannelPayload } from './payload';
import ChannelRetriever from './retriever';
/**
 * Represents a Channel in a Synnax database. It should not be instantiated
 * directly, but rather created or retrieved from a {@link ChannelClient}.
 */
export declare class Channel {
    private readonly segmentClient;
    payload: ChannelPayload;
    constructor(payload: ChannelPayload, segmentClient: SegmentClient);
    get key(): string;
    get name(): string;
    get nodeId(): number;
    get rate(): Rate;
    get dataType(): DataType;
    get density(): Density;
    /**
     * Reads telemetry from the channel between the two timestamps.
     *
     * @param start - The starting timestamp of the range to read from.
     * @param end - The ending timestamp of the range to read from.
     * @returns A typed array containing the retrieved
     */
    read(start: UnparsedTimeStamp, end: UnparsedTimeStamp): Promise<TypedArray>;
    /**
     * Writes telemetry to the channel starting at the given timestamp.
     *
     * @param start - The starting timestamp of the first sample in data.
     * @param data - THe telemetry to write to the channel.
     */
    write(start: UnparsedTimeStamp, data: TypedArray): Promise<boolean>;
}
/**
 * The core client class for executing channel operations against a Synnax
 * cluster.
 */
export default class ChannelClient {
    private readonly segmentClient;
    private readonly retriever;
    private readonly creator;
    constructor(segmentClient: SegmentClient, retriever: ChannelRetriever, creator: ChannelCreator);
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
    create(props: CreateChannelProps): Promise<Channel>;
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
    createMany(props: CreateChannelProps & {
        count: number;
    }): Promise<Channel[]>;
    /**
     * Retrieves channels with the given keys.
     *
     * @param keys - The keys of the channels to retrieve.
     * @returns The retrieved channels.
     * @throws QueryError if any of the channels can't be found.
     */
    retrieveByKeys(...keys: string[]): Promise<Channel[]>;
    /**
     * Retrieves channels with the given names.
     *
     * @param names - The list of names to retrieve channels for.
     * @returns A list of retrieved channels matching the given names. If a
     *   channel with a given name can't be found, it will be omitted from the
     *   list.
     */
    retrieveByNames(...names: string[]): Promise<Channel[]>;
    /**
     * Retrieves channels whose lease node is the given ID.
     *
     * @param nodeId - The ID of the node to retrieve channels for.
     * @returns A list of retrieved channels matching the given node ID.
     */
    retrieveByNodeId(nodeId: number): Promise<Channel[]>;
    retrieveAll(): Promise<Channel[]>;
    private sugar;
}
