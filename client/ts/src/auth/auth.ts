// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Middleware, sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { z } from "zod";

import { ExpiredTokenError, InvalidTokenError } from "@/errors";
import { user } from "@/user";

const insecureCredentialsZ = z.object({ username: z.string(), password: z.string() });
interface InsecureCredentials extends z.infer<typeof insecureCredentialsZ> {}

const tokenResponseZ = z.object({ token: z.string(), user: user.userZ });

const LOGIN_ENDPOINT = "/auth/login";

const MAX_RETRIES = 3;

const changePasswordReqZ = z.object({
  username: z.string(),
  password: z.string(),
  newPassword: z.string().min(1),
});
const changePasswordResZ = z.object({});

const RETRY_ON = [InvalidTokenError, ExpiredTokenError] as const;

type AuthState =
  | { authenticated: false }
  | { authenticated: true; user: user.User; token: string };

export class Client {
  private readonly client: UnaryClient;
  private readonly credentials: InsecureCredentials;
  private authState: AuthState = { authenticated: false };
  authenticating: Promise<Error | null> | undefined;
  private retryCount: number;

  constructor(client: UnaryClient, credentials: InsecureCredentials) {
    this.client = client;
    this.credentials = credentials;
    this.retryCount = 0;
  }

  get authenticated(): boolean {
    return this.authState.authenticated;
  }

  get user(): user.User | undefined {
    return this.authState.authenticated ? this.authState.user : undefined;
  }

  get token(): string | undefined {
    return this.authState.authenticated ? this.authState.token : undefined;
  }

  async retrieveUser(): Promise<user.User> {
    if (!this.authState.authenticated) await this.authenticating;
    const { authState } = this;
    if (!authState.authenticated)
      throw new Error("Authentication failed: user not available");
    return authState.user;
  }

  async changePassword(newPassword: string): Promise<void> {
    if (!this.authenticated) throw new Error("Not authenticated");
    await sendRequired<typeof changePasswordReqZ, typeof changePasswordResZ>(
      this.client,
      "/auth/change-password",
      {
        username: this.credentials.username,
        password: this.credentials.password,
        newPassword,
      },
      changePasswordReqZ,
      changePasswordResZ,
    );
    this.credentials.password = newPassword;
  }

  middleware(): Middleware {
    const mw: Middleware = async (reqCtx, next) => {
      if (!this.authenticated && !reqCtx.target.endsWith(LOGIN_ENDPOINT)) {
        this.authenticating ??= new Promise((resolve, reject) => {
          this.client
            .send(
              LOGIN_ENDPOINT,
              this.credentials,
              insecureCredentialsZ,
              tokenResponseZ,
            )
            .then(([res, err]) => {
              if (err != null) return resolve(err);
              if (res == null) return resolve(new Error("No response from login"));
              this.authState = {
                authenticated: true,
                user: res.user,
                token: res.token,
              };
              resolve(null);
            })
            .catch(reject);
        });
        const err = await this.authenticating;
        if (err != null) return [reqCtx, err];
      }
      reqCtx.params.Authorization = `Bearer ${this.token}`;
      const [resCtx, err] = await next(reqCtx);
      if (RETRY_ON.some((e) => e.matches(err)) && this.retryCount < MAX_RETRIES) {
        this.authState = { authenticated: false };
        this.authenticating = undefined;
        this.retryCount += 1;
        return mw(reqCtx, next);
      }
      this.retryCount = 0;
      return [resCtx, err];
    };
    return mw;
  }
}
