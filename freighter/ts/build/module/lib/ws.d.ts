import { EncoderDecoder } from './encoder';
import Endpoint from './endpoint';
import { ClientStream, StreamClient } from './stream';
import { Payload } from './transport';
export declare class WebSocketClientStream<RQ extends Payload, RS extends Payload> implements ClientStream<RQ, RS> {
    private encoder;
    private ws;
    private server_closed?;
    private send_closed;
    private receiveDataQueue;
    private receiveCallbacksQueue;
    constructor(encoder: EncoderDecoder, ws: WebSocket);
    send(req: RQ): Error | undefined;
    receive(): Promise<[RS | undefined, Error | undefined]>;
    closeSend(): void;
    private receiveMsg;
    private listenForMessages;
}
export declare class WebSocketClient implements StreamClient {
    endpoint: Endpoint;
    encoder: EncoderDecoder;
    constructor(encoder: EncoderDecoder, endpoint: Endpoint);
    stream<RQ extends Payload, RS extends Payload>(target: string): Promise<ClientStream<RQ, RS>>;
}
