// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { HTTPClientFactory } from "@synnaxlabs/freighter";
import type { Middleware, UnaryClient } from "@synnaxlabs/freighter";
import { z } from "zod";

import { AuthError } from "./errors";
import { UserPayload, UserPayloadSchema } from "./user/payload";

export const tokenMiddleware = (token: () => Promise<string>): Middleware => {
  return async (md, next) => {
    try {
      md.params.Authorization = `Bearer ${await token()}`;
    } catch (err) {
      return [md, err as Error];
    }
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
  private static readonly ENDPOINT = "/auth/login";
  private token: string | undefined;
  private readonly client: UnaryClient;
  private readonly credentials: InsecureCredentials;
  authenticating: Promise<void> | undefined;
  authenticated: boolean;
  user: UserPayload | undefined;

  constructor(factory: HTTPClientFactory, creds: InsecureCredentials) {
    this.client = factory.newPOST();
    this.credentials = creds;
    this.authenticated = false;
    this.authenticate();
  }

  authenticate(): void {
    this.authenticating = new Promise((resolve, reject) => {
      this.client
        .send<InsecureCredentials, TokenResponse>(
          AuthenticationClient.ENDPOINT,
          this.credentials,
          TokenResponseSchema
        )
        .then(([res, err]) => {
          if (err != null) {
            reject(err);
            return;
          }
          this.token = res?.token;
          this.user = res?.user;
          this.authenticated = true;
          resolve();
        })
        .catch(reject);
    });
  }

  private async maybeWaitAuthenticated(): Promise<void> {
    if (this.authenticating != null) await this.authenticating;
    this.authenticating = undefined;
  }

  middleware(): Middleware {
    return tokenMiddleware(async () => {
      await this.maybeWaitAuthenticated();
      if (this.token == null) {
        throw new AuthError("[auth] - attempting to authenticate without a token");
      }
      return this.token;
    });
  }
}
