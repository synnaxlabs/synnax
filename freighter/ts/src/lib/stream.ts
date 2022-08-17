import {Payload, Transport} from "./transport";

export interface StreamReceiver<I extends Payload> {
  receive(): Promise<[I, Error]>
}

export interface StreamSender<O extends Payload> {
  send(req: O): Promise<Error>
}

export interface StreamSenderCloser<O extends Payload> extends StreamSender<O> {
  close(): Promise<Error>
}

export interface ClientStream<I extends Payload, O extends Payload>
  extends Transport, StreamReceiver<I>, StreamSenderCloser<O> {
}

export interface StreamClient<I extends Payload, O extends Payload>
  extends Transport {
  stream(target: string): ClientStream<I, O>
}
