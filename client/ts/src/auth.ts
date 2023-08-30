// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Middleware, UnaryClient } from "@synnaxlabs/freighter";
import { z } from "zod";

import { AuthError } from "@/errors";
import { UserPayload, userPayloadSchema } from "@/user";

export const tokenMiddleware = (token: () => Promise<string>): Middleware => {
  return async (md, next) => {
    try {
      const tk = await token();
      md.params.Authorization = `Bearer ${tk}`;
    } catch (err) {
      return [md, err as Error];
    }
    return await next(md);
  };
};

export const insecureCredentialsZ = z.object({
  username: z.string(),
  password: z.string(),
});
export type InsecureCredentials = z.infer<typeof insecureCredentialsZ>;

export const tokenResponseZ = z.object({
  token: z.string(),
  user: userPayloadSchema,
});

export type TokenResponse = z.infer<typeof tokenResponseZ>;

export class AuthenticationClient {
  private static readonly ENDPOINT = "/auth/login";
  private token: string | undefined;
  private readonly client: UnaryClient;
  private readonly credentials: InsecureCredentials;
  authenticating: Promise<void> | undefined;
  authenticated: boolean;
  user: UserPayload | undefined;

  constructor(client: UnaryClient, creds: InsecureCredentials) {
    this.client = client;
    this.credentials = creds;
    this.authenticated = false;
    this.authenticate();
  }

  authenticate(): void {
    this.authenticating = new Promise((resolve, reject) => {
      this.client
        .send<typeof insecureCredentialsZ, typeof tokenResponseZ>(
          AuthenticationClient.ENDPOINT,
          this.credentials,
          tokenResponseZ
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
        .catch((r) => reject(r));
    });
  }

  middleware(): Middleware {
    return tokenMiddleware(async () => {
      if (!this.authenticated) await this.authenticating;
      if (this.token == null) {
        throw new AuthError("[auth] - attempting to authenticate without a token");
      }
      return this.token;
    });
  }
}
