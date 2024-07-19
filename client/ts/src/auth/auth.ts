// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Middleware, UnaryClient } from "@synnaxlabs/freighter";
import { z } from "zod";

import { InvalidTokenError } from "@/errors";
import { user } from "@/user";

export const insecureCredentialsZ = z.object({
  username: z.string(),
  password: z.string(),
});
export type InsecureCredentials = z.infer<typeof insecureCredentialsZ>;

export const tokenResponseZ = z.object({
  token: z.string(),
  user: user.payloadZ,
});

export type TokenResponse = z.infer<typeof tokenResponseZ>;

const LOGIN_ENDPOINT = "/auth/login";
const REGISTER_ENDPOINT = "/auth/register";

const MAX_RETRIES = 3;

export class Client {
  token: string | undefined;
  private readonly client: UnaryClient;
  private readonly credentials: InsecureCredentials;
  private authenticating: Promise<Error | null> | undefined;
  authenticated: boolean;
  user: user.Payload | undefined;
  private retryCount: number;

  constructor(client: UnaryClient, credentials: InsecureCredentials) {
    this.client = client;
    this.authenticated = false;
    this.credentials = credentials;
    this.retryCount = 0;
  }

  middleware(): Middleware {
    const mw: Middleware = async (reqCtx, next) => {
      if (!this.authenticated && !reqCtx.target.endsWith(LOGIN_ENDPOINT)) {
        if (this.authenticating == null)
          this.authenticating = new Promise((resolve, reject) => {
            this.client
              .send(
                LOGIN_ENDPOINT,

                this.credentials,
                insecureCredentialsZ,
                tokenResponseZ,
              )
              .then(([res, err]) => {
                if (err != null) return resolve(err);
                this.token = res?.token;
                this.user = res?.user;
                this.authenticated = true;
                resolve(null);
              })
              .catch(reject);
          });
        const err = await this.authenticating;
        if (err != null) return [reqCtx, err];
      }
      reqCtx.params.Authorization = `Bearer ${this.token}`;
      const [resCtx, err] = await next(reqCtx);
      if (InvalidTokenError.matches(err) && this.retryCount < MAX_RETRIES) {
        this.authenticated = false;
        this.authenticating = undefined;
        this.retryCount += 1;
        return mw(reqCtx, next);
      }
      this.retryCount = 0;
      return [resCtx, err];
    };
    return mw;
  }

  async register(username: string, password: string): Promise<user.Payload> {
    const [res, err] = await this.client.send(
      REGISTER_ENDPOINT,
      { username: username, password: password },
      insecureCredentialsZ,
      tokenResponseZ,
    );

    if (err != null) {
      throw err;
    }
    return res.user;
  }
}
