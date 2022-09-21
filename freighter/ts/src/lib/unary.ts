export interface UnaryClient<RQ, RS> {
  send(target: string, req: RQ): Promise<[RS | undefined, Error | undefined]>;
}
