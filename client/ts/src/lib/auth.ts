import {
  HTTPClientFactory,
  Middleware,
  UnaryClient,
} from '@synnaxlabs/freighter';
import { z } from 'zod';

import { AuthError } from './errors';
import { UserPayload, UserPayloadSchema } from './user/payload';

export const tokenMiddleware = (token: () => Promise<string>): Middleware => {
  return async (md, next) => {
    md.params['Authorization'] = `Bearer ${await token()}`;
    return await next(md);
  };
};

export const InsecureCredentialsSchema = z.object({
  username: z.string(),
  password: z.string(),
});

export type InsecureCredentials = z.infer<typeof InsecureCredentialsSchema>;

export const TokenResponseSchema = z.object({
  token: z.string(),
  user: UserPayloadSchema,
});

export type TokenResponse = z.infer<typeof TokenResponseSchema>;

export default class AuthenticationClient {
  private static ENDPOINT = '/auth/login';
  private token: string | undefined;
  private client: UnaryClient;
  private credentials: InsecureCredentials;
  private authenticated: boolean;
  user: UserPayload | undefined;

  constructor(factory: HTTPClientFactory, creds: InsecureCredentials) {
    this.client = factory.postClient();
    this.credentials = creds;
    this.authenticated = false;
  }

  async authenticate() {
    const [res, err] = await this.client.send<
      InsecureCredentials,
      TokenResponse
    >(AuthenticationClient.ENDPOINT, this.credentials, TokenResponseSchema);
    if (err) {
      throw err;
    }
    this.token = res?.token;
    this.user = res?.user;
  }

  middleware(): Middleware {
    return tokenMiddleware(async () => {
      if (!this.authenticated) await this.authenticate();
      if (!this.token) {
        throw new AuthError(
          '[auth] - attempting to authenticate without a token'
        );
      }
      return this.token as string;
    });
  }
}
