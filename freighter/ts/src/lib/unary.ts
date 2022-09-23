import { ZodSchema } from 'zod';

export interface UnaryClient {
  send<RQ, RS>(
    target: string,
    req: RQ,
    resSchema: ZodSchema<RS>
  ): Promise<[RS | undefined, Error | undefined]>;
}
