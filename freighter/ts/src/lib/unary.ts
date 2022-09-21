export interface UnaryClient<RQ, RS> {
  send(target: string, req: RQ): Promise<[RS | undefined, Error | undefined]>;
}

export interface UnaryServer<RQ, RS> {
  bind_handle(
    handle: (req: RQ) => Promise<[RS | undefined, Error | undefined]>
  ): void;
}
