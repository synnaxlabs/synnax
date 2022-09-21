export interface StreamReceiver<RS> {
  receive(): Promise<[RS | undefined, Error | undefined]>;
}

export interface StreamSender<RQ> {
  send(req: RQ): Error | undefined;
}

export interface StreamSenderCloser<RQ> extends StreamSender<RQ> {
  closeSend(): void;
}

export interface ClientStream<RQ, RS>
  extends StreamSenderCloser<RQ>,
    StreamReceiver<RS> {}

export interface StreamClient<RQ, RS> {
  stream(target: string): Promise<ClientStream<RQ, RS>>;
}
