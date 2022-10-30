/**
 * Represents a Channel in a Synnax database. It should not be instantiated
 * directly, but rather created or retrieved from a {@link ChannelClient}.
 */
export class Channel {
    segmentClient;
    payload;
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
/**
 * The core client class for executing channel operations against a Synnax
 * cluster.
 */
export default class ChannelClient {
    segmentClient;
    retriever;
    creator;
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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiY2xpZW50LmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiLi4vLi4vLi4vLi4vc3JjL2xpYi9jaGFubmVsL2NsaWVudC50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFjQTs7O0dBR0c7QUFDSCxNQUFNLE9BQU8sT0FBTztJQUNELGFBQWEsQ0FBZ0I7SUFDOUMsT0FBTyxDQUFpQjtJQUV4QixZQUFZLE9BQXVCLEVBQUUsYUFBNEI7UUFDL0QsSUFBSSxDQUFDLE9BQU8sR0FBRyxPQUFPLENBQUM7UUFDdkIsSUFBSSxDQUFDLGFBQWEsR0FBRyxhQUFhLENBQUM7SUFDckMsQ0FBQztJQUVELElBQUksR0FBRztRQUNMLElBQUksQ0FBQyxJQUFJLENBQUMsT0FBTyxDQUFDLEdBQUcsRUFBRTtZQUNyQixNQUFNLElBQUksS0FBSyxDQUFDLHdCQUF3QixDQUFDLENBQUM7U0FDM0M7UUFDRCxPQUFPLElBQUksQ0FBQyxPQUFPLENBQUMsR0FBRyxDQUFDO0lBQzFCLENBQUM7SUFFRCxJQUFJLElBQUk7UUFDTixJQUFJLENBQUMsSUFBSSxDQUFDLE9BQU8sQ0FBQyxJQUFJLEVBQUU7WUFDdEIsTUFBTSxJQUFJLEtBQUssQ0FBQyx5QkFBeUIsQ0FBQyxDQUFDO1NBQzVDO1FBQ0QsT0FBTyxJQUFJLENBQUMsT0FBTyxDQUFDLElBQUksQ0FBQztJQUMzQixDQUFDO0lBRUQsSUFBSSxNQUFNO1FBQ1IsSUFBSSxJQUFJLENBQUMsT0FBTyxDQUFDLE1BQU0sS0FBSyxTQUFTLEVBQUU7WUFDckMsTUFBTSxJQUFJLEtBQUssQ0FBQywyQkFBMkIsQ0FBQyxDQUFDO1NBQzlDO1FBQ0QsT0FBTyxJQUFJLENBQUMsT0FBTyxDQUFDLE1BQU0sQ0FBQztJQUM3QixDQUFDO0lBRUQsSUFBSSxJQUFJO1FBQ04sT0FBTyxJQUFJLENBQUMsT0FBTyxDQUFDLElBQUksQ0FBQztJQUMzQixDQUFDO0lBRUQsSUFBSSxRQUFRO1FBQ1YsT0FBTyxJQUFJLENBQUMsT0FBTyxDQUFDLFFBQVEs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