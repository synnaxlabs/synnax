import { ZodSchema } from 'zod';

/**
 * An interface for an entity that implements a simple request-response
 * transport between two entities.
 */
export interface UnaryClient {
  /**
   * Sends a request to the target server and waits until a response is received.
   * @param target - The target server to send the request to.
   * @param req - The request to send.
   * @param resSchema - The schema to validate the response against.
   */
  send<RQ, RS>(
    target: string,
    req: RQ,
    resSchema: ZodSchema<RS>
  ): Promise<[RS | undefined, Error | undefined]>;
}
