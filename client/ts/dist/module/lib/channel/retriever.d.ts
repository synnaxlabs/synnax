import Transport from '../transport';
import { ChannelPayload } from './payload';
export default class Retriever {
    private static ENDPOINT;
    private client;
    constructor(transport: Transport);
    private execute;
    retrieveByKeys(...keys: string[]): Promise<ChannelPayload[]>;
    retrieveByNames(...names: string[]): Promise<ChannelPayload[]>;
    retrieveByNodeID(nodeId: number): Promise<ChannelPayload[]>;
    retrieveAll(): Promise<ChannelPayload[]>;
}
