import { Payload } from './transport';

export interface StreamReceiver<RS extends Payload> {
  receive(): Promise<[RS | undefined, Error | undefined]>;
}

export interface StreamSender<RQ extends Payload> {
  send(req: RQ): Error | undefined;
}

export interface StreamSenderCloser<RQ extends Payload>
  extends StreamSender<RQ> {
  closeSend(): void;
}

export interface ClientStream<RQ extends Payload, RS extends Payload>
  extends StreamSenderCloser<RQ>,
    StreamReceiver<RS> {}

export interface StreamClient {
  stream<RQ extends Payload, RS extends Payload>(
    target: string
  ): Promise<ClientStream<RQ, RS>>;
}
