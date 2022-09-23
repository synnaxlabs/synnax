import { ZodSchema } from 'zod';

export interface StreamReceiver<RS> {
  receive(): Promise<[RS | undefined, Error | undefined]>;
}

export interface StreamSender<RQ> {
  send(req: RQ): Error | undefined;
}

export interface StreamSenderCloser<RQ> extends StreamSender<RQ> {
  closeSend(): void;
}

export interface Stream<RQ, RS>
  extends StreamSenderCloser<RQ>,
    StreamReceiver<RS> {}

export interface StreamClient {
  stream<RQ, RS>(
    target: string,
    reqSchema: ZodSchema<RQ>,
    resSchema: ZodSchema<RS>
  ): Promise<Stream<RQ, RS>>;
}
