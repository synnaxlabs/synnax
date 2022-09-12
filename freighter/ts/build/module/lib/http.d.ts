import { AxiosRequestConfig } from 'axios';
import { EncoderDecoder } from './encoder';
import Endpoint from './endpoint';
import { Payload } from './transport';
export default class HTTPClient {
    endpoint: Endpoint;
    encoder: EncoderDecoder;
    constructor(endpoint: Endpoint, encoder: EncoderDecoder);
    get(): GETClient;
    post(): POSTClient;
}
declare class Core {
    endpoint: Endpoint;
    encoder: EncoderDecoder;
    constructor(endpoint: Endpoint, encoder: EncoderDecoder);
    get headers(): {
        'Content-Type': string;
    };
    requestConfig(): AxiosRequestConfig;
}
export declare class GETClient extends Core {
    send<RQ extends Payload, RS extends Payload>(target: string, req: RQ): Promise<[RS | undefined, Error | undefined]>;
}
export declare class POSTClient extends Core {
    send<RQ extends Payload, RS extends Payload>(target: string, req: RQ): Promise<[RS | undefined, Error | undefined]>;
}
export {};
