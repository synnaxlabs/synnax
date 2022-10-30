"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.Channel = void 0;
/**
 * Represents a Channel in a Synnax database. It should not be instantiated
 * directly, but rather created or retrieved from a {@link ChannelClient}.
 */
class Channel {
    constructor(payload, segmentClient) {
        this.payload = payload;
        this.segmentClient = segmentClient;
    }
    get key() {
        if (!this.payload.key) {
            throw new Error('Channel key is not set');
        }
        return this.payload.key;
    }
    get name() {
        if (!this.payload.name) {
            throw new Error('Channel name is not set');
        }
        return this.payload.name;
    }
    get nodeId() {
        if (this.payload.nodeId === undefined) {
            throw new Error('Channel nodeId is not set');
        }
        return this.payload.nodeId;
    }
    get rate() {
        return this.payload.rate;
    }
    get dataType() {
        return this.payload.dataType;
    }
    get density() {
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
    async read(start, end) {
        return await this.segmentClient.read(this.key, start, end);
    }
    /**
     * Writes telemetry to the channel starting at the given timestamp.
     *
     * @param start - The starting timestamp of the first sample in data.
     * @param data - THe telemetry to write to the channel.
     */
    async write(start, data) {
        return await this.segmentClient.write(this.key, start, data);
    }
}
exports.Channel = Channel;
/**
 * The core client class for executing channel operations against a Synnax
 * cluster.
 */
class ChannelClient {
    constructor(segmentClient, retriever, creator) {
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
    async create(props) {
        return (await this.createMany(Object.assign(Object.assign({}, props), { count: 1 })))[0];
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
    async createMany(props) {
        return this.sugar(...(await this.creator.createMany(props)));
    }
    /**
     * Retrieves channels with the given keys.
     *
     * @param keys - The keys of the channels to retrieve.
     * @returns The retrieved channels.
     * @throws QueryError if any of the channels can't be found.
     */
    async retrieveByKeys(...keys) {
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
    async retrieveByNames(...names) {
        return this.sugar(...(await this.retriever.retrieveByNames(...names)));
    }
    /**
     * Retrieves channels whose lease node is the given ID.
     *
     * @param nodeId - The ID of the node to retrieve channels for.
     * @returns A list of retrieved channels matching the given node ID.
     */
    async retrieveByNodeId(nodeId) {
        return this.sugar(...(await this.retriever.retrieveByNodeID(nodeId)));
    }
    async retrieveAll() {
        return this.sugar(...(await this.retriever.retrieveAll()));
    }
    sugar(...payloads) {
        return payloads.map((p) => new Channel(p, this.segmentClient));
    }
}
exports.default = ChannelClient;
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiY2xpZW50LmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiLi4vLi4vLi4vLi4vc3JjL2xpYi9jaGFubmVsL2NsaWVudC50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiOzs7QUFjQTs7O0dBR0c7QUFDSCxNQUFhLE9BQU87SUFJbEIsWUFBWSxPQUF1QixFQUFFLGFBQTRCO1FBQy9ELElBQUksQ0FBQyxPQUFPLEdBQUcsT0FBTyxDQUFDO1FBQ3ZCLElBQUksQ0FBQyxhQUFhLEdBQUcsYUFBYSxDQUFDO0lBQ3JDLENBQUM7SUFFRCxJQUFJLEdBQUc7UUFDTCxJQUFJLENBQUMsSUFBSSxDQUFDLE9BQU8sQ0FBQyxHQUFHLEVBQUU7WUFDckIsTUFBTSxJQUFJLEtBQUssQ0FBQyx3QkFBd0IsQ0FBQyxDQUFDO1NBQzNDO1FBQ0QsT0FBTyxJQUFJLENBQUMsT0FBTyxDQUFDLEdBQUcsQ0FBQztJQUMxQixDQUFDO0lBRUQsSUFBSSxJQUFJO1FBQ04sSUFBSSxDQUFDLElBQUksQ0FBQyxPQUFPLENBQUMsSUFBSSxFQUFFO1lBQ3RCLE1BQU0sSUFBSSxLQUFLLENBQUMseUJBQXlCLENBQUMsQ0FBQztTQUM1QztRQUNELE9BQU8sSUFBSSxDQUFDLE9BQU8sQ0FBQyxJQUFJLENBQUM7SUFDM0IsQ0FBQztJQUVELElBQUksTUFBTTtRQUNSLElBQUksSUFBSSxDQUFDLE9BQU8sQ0FBQyxNQUFNLEtBQUssU0FBUyxFQUFFO1lBQ3JDLE1BQU0sSUFBSSxLQUFLLENBQUMsMkJBQTJCLENBQUMsQ0FBQztTQUM5QztRQUNELE9BQU8sSUFBSSxDQUFDLE9BQU8sQ0FBQyxNQUFNLENBQUM7SUFDN0IsQ0FBQztJQUVELElBQUksSUFBSTtRQUNOLE9BQU8sSUFBSSxDQUFDLE9BQU8sQ0FBQyxJQUFJLENBQUM7SUFDM0IsQ0FBQztJQUVELElBQUksUUFBUTtRQUNWLE9BQU8sSUFBSSxDQUFDLE9BQU8sQ0FBQyxRQUFRLENBQUM7SUFDL0IsQ0FBQztJQUVELElBQUksT0FBTztRQUNU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