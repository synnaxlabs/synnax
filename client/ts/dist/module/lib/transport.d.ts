import { HTTPClientFactory, Middleware, StreamClient, UnaryClient, URL } from '@synnaxlabs/freighter';
export default class Transport {
    url: URL;
    httpFactory: HTTPClientFactory;
    streamClient: StreamClient;
    constructor(url: URL);
    getClient(): UnaryClient;
    postClient(): UnaryClient;
    use(...middleware: Middleware[]): void;
}
