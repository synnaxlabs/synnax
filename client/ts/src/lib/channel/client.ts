import SegmentClient from '../segment/client';
import {
	DataType,
	Density,
	Rate,
	TypedArray,
	UnparsedDensity,
	UnparsedTimeStamp,
} from '../telem';

import ChannelCreator from './creator';
import { CreateChannelProps } from './creator';
import { ChannelPayload, channelPayloadSchema } from './payload';
import ChannelRetriever from './retriever';

/**
 * Represents a Channel in a Synnax database. It should not be instantiated
 * directly, but rather created or retrieved from a {@link ChannelClient}.
 */
export class Channel {
	private readonly _segmentClient: SegmentClient | null;
	payload: ChannelPayload;

	constructor({
		dataType,
		rate,
		name = '',
		nodeId = 0,
		key = '',
		density = 0,
		isIndex = false,
		index = '',
		segmentClient,
	}: CreateChannelProps & {
		segmentClient?: SegmentClient;
		key?: string;
		density?: UnparsedDensity;
	}) {
		this.payload = channelPayloadSchema.parse({
			dataType: new DataType(dataType).valueOf(),
			rate: new Rate(rate).valueOf(),
			name,
			nodeId,
			key,
			density: new Density(density).valueOf(),
			isIndex,
			index,
		});
		this._segmentClient = segmentClient ?? null;
	}

	private get segmentClient(): SegmentClient {
		if (!this._segmentClient) {
			throw new Error('Cannot read from a channel that has not been created');
		}
		return this._segmentClient;
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
	): Promise<TypedArray | undefined> {
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
		return (await this.createMany([new Channel(props)]))[0];
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
	async createMany(channels: Channel[]): Promise<Channel[]> {
		return this.sugar(
			...(await this.creator.createMany(channels.map((c) => c.payload)))
		);
	}

	/**
	 * Retrieves a channel from the database using the given parameters.
	 * @param props.key - The key of the channel to retrieve.
	 * @param props.name - The name of the channel to retrieve. If props.key is set,
	 * this will be ignored.
	 * @returns The retrieved channel.
	 * @raises {QueryError} If the channel does not exist or if multiple results are returned.
	 */
	async retrieve({ key, name }: { key?: string; name?: string }): Promise<Channel> {
		return this.sugar(await this.retriever.retrieve({ key, name }))[0];
	}

	/**
	 * Retrieves all channels from the database. Warning: this can be an expensive operation
	 * if there are many channels in the database.
	 * @returns All channels in the database.
	 */
	async retrieveAll(): Promise<Channel[]> {
		return this.sugar(...(await this.retriever.retrieveAll()));
	}

	/**
	 * Filters channels from the database using the given parameters.
	 * @param props.names - The names of the channels to retrieve.
	 * @param props.keys - The keys of the channels to retrieve.
	 * @param props.nodeId - The ID of the node that holds the lease on the channels to retrieve.
	 * @returns The retrieved channels.
	 */
	async filter({
		names,
		keys,
		nodeId,
	}: {
		names?: string[];
		keys?: string[];
		nodeId?: number;
	}): Promise<Channel[]> {
		return this.sugar(...(await this.retriever.filter({ names, keys, nodeId })));
	}

	private sugar(...payloads: ChannelPayload[]): Channel[] {
		const { segmentClient } = this;
		return payloads.map((p) => new Channel({ ...p, segmentClient }));
	}
}
