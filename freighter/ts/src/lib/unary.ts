import {Transport, Payload} from "./transport"


export interface UnaryClient<I extends Payload, O extends Payload> extends Transport {
  send(target: string, req: I): Promise<[O, Error]>
}

export interface UnaryServer<I extends Payload, O extends Payload> extends Transport {
  bind_handle(handle: (req: I) => Promise<[O, Error]>): void
}

export interface Unary<I extends Payload, O extends Payload> extends UnaryClient<I, O>, UnaryServer<I, O> {
}
