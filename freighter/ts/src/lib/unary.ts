import { Payload } from './transport';

export interface UnaryClient<RQ extends Payload, RS extends Payload> {
  send(target: string, req: RQ): Promise<[RS | undefined, Error | undefined]>;
}

export interface UnaryServer<RQ extends Payload, RS extends Payload> {
  bind_handle(
    handle: (req: RQ) => Promise<[RS | undefined, Error | undefined]>
  ): void;
}

export interface Unary<I extends Payload, O extends Payload>
  extends UnaryClient<I, O>,
    UnaryServer<I, O> {}
