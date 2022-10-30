import { ChannelPayload } from './payload';
import Retriever from './retriever';
export default class Registry {
    private retriever;
    private channels;
    constructor(retriever: Retriever);
    get(key: string): Promise<ChannelPayload>;
    getN(...keys: string[]): Promise<ChannelPayload[]>;
}
