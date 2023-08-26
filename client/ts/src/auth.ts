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
      console.log("GET TOKEN");
      const tk = token();
      console.log("TOKEN PROMISE", tk);
      const tk_ = await tk;
      console.log("TOKEN RECEIVED", tk_);
      md.params.Authorization = `Bearer ${tk_}`;
    } catch (err) {
      console.log("ERR", err);
      return [md, err as Error];
    }
    console.log("TOKEN", md);
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
      console.log("S", this.authenticating, this.authenticated);
      try {
        if (!this.authenticated) await this.authenticating;
      } catch (err) {
        console.log("A", err);
        throw err;
      }
      console.log("E");
      if (this.token == null) {
        throw new AuthError("[auth] - attempting to authenticate without a token");
      }
      console.log("F", this.token);
      return this.token;
    });
  }
}
