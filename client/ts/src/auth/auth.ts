// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Middleware, type UnaryClient } from "@synnaxlabs/freighter";
import { errors, TimeStamp } from "@synnaxlabs/x";
import { z } from "zod";

import {
  type Credentials,
  credentialsZ,
  proofZ,
  type Session,
  type SessionKey,
  sessionKeyZ,
  sessionZ,
} from "@/auth/types.gen";
import {
  AuthError,
  ExpiredTokenError,
  InvalidTokenError,
  SessionEndedError,
} from "@/errors";
import { ontology } from "@/ontology";
import { user } from "@/user";

const clusterInfoZ = z.object({
  clusterKey: z.string(),
  nodeVersion: z.string().optional(),
  nodeKey: z.number().optional(),
  nodeTime: TimeStamp.z,
});

const createReqZ = z.object({ proof: proofZ });
const createResZ = z.object({
  token: z.string(),
  session: sessionZ,
  user: user.userZ.optional(),
  clusterInfo: clusterInfoZ.optional(),
});
const retrieveReqZ = z.object({ subjects: ontology.idZ.array().optional() });
const retrieveResZ = z.object({ sessions: sessionZ.array() });
const deleteReqZ = z.object({ keys: sessionKeyZ.array() });
const emptyResZ = z.object({});

/** Opens a session from a proof. Works without a token. */
export const SESSION_CREATE_ENDPOINT = "/auth/session/create";
/** Reports activity on the caller's session. */
const SESSION_RENEW_ENDPOINT = "/auth/session/renew";
/** Lists sessions. */
const SESSION_RETRIEVE_ENDPOINT = "/auth/session/retrieve";
/** Ends sessions. */
export const SESSION_DELETE_ENDPOINT = "/auth/session/delete";

const MAX_RETRIES = 3;

const changePasswordReqZ = z.object({
  username: z.string(),
  password: z.string(),
  newPassword: z.string().min(1),
});
const changePasswordResZ = z.object({});

const RETRY_ON = [InvalidTokenError, ExpiredTokenError, SessionEndedError] as const;

type AuthState =
  | { authenticated: false }
  | { authenticated: true; session: Session; user?: user.User; token: string };

/** Login outcome hooks. The connection machine is the sole consumer. */
export interface Observer {
  /** Fires after every successful login. */
  onSuccess?: () => void;
  /** Fires on a definitive credential rejection, never on token self-heal. */
  onFailure?: (error: Error) => void;
}

export class Client {
  private readonly client: UnaryClient;
  private readonly credentials: Credentials;
  private readonly observer: Observer;
  private authState: AuthState = { authenticated: false };
  authenticating: Promise<Error | null> | undefined;
  private retryCount: number;
  private generation = 0;

  constructor(client: UnaryClient, credentials: Credentials, observer: Observer = {}) {
    this.client = client;
    this.credentials = credentials;
    this.observer = observer;
    this.retryCount = 0;
  }

  /** Replaces the stored credentials and clears auth state for a fresh login. */
  setCredentials({ username, password }: Credentials): void {
    this.credentials.username = username;
    this.credentials.password = password;
    this.authState = { authenticated: false };
    this.authenticating = undefined;
    this.retryCount = 0;
    this.generation += 1;
  }

  get authenticated(): boolean {
    return this.authState.authenticated;
  }

  /** The user record of the logged-in subject, when it is a user. */
  get user(): user.User | undefined {
    return this.authState.authenticated ? this.authState.user : undefined;
  }

  /** The session the client holds, once logged in. */
  get session(): Session | undefined {
    return this.authState.authenticated ? this.authState.session : undefined;
  }

  /** The subject the client logged in as, once logged in. */
  get subject(): ontology.ID | undefined {
    return this.authState.authenticated ? this.authState.session.subject : undefined;
  }

  get token(): string | undefined {
    return this.authState.authenticated ? this.authState.token : undefined;
  }

  async retrieveUser(): Promise<user.User> {
    if (!this.authState.authenticated) await this.authenticating;
    const { authState } = this;
    if (!authState.authenticated || authState.user == null)
      throw new Error("Authentication failed: user not available");
    return authState.user;
  }

  async changePassword(newPassword: string): Promise<void> {
    if (!this.authenticated) throw new Error("Not authenticated");
    await this.client.send(
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

  /** Reports activity on the session so it does not idle out. */
  async renew(): Promise<void> {
    await this.client.send(SESSION_RENEW_ENDPOINT, undefined, z.void(), emptyResZ);
  }

  /**
   * Lists sessions: the caller's own by default, or those of the given subjects when
   * the caller may retrieve them.
   */
  async retrieveSessions(subjects?: ontology.ID[]): Promise<Session[]> {
    const res = await this.client.send(
      SESSION_RETRIEVE_ENDPOINT,
      { subjects },
      retrieveReqZ,
      retrieveResZ,
    );
    return res.sessions;
  }

  /** Ends the given sessions. Unknown keys are ignored. */
  async deleteSessions(...keys: SessionKey[]): Promise<void> {
    await this.client.send(SESSION_DELETE_ENDPOINT, { keys }, deleteReqZ, emptyResZ);
  }

  /** Ends the client's own session and forgets it. A later request logs in again. */
  async logout(): Promise<void> {
    const { authState } = this;
    if (!authState.authenticated) return;
    await this.deleteSessions(authState.session.key);
    this.authState = { authenticated: false };
    this.authenticating = undefined;
  }

  middleware(): Middleware {
    const mw: Middleware = async (reqCtx, next) => {
      if (!this.authenticated && !reqCtx.target.endsWith(SESSION_CREATE_ENDPOINT)) {
        this.authenticating ??= (async (): Promise<Error | null> => {
          // logins issued before a setCredentials are stale; their result must
          // not reach the observer
          const generation = this.generation;
          try {
            const res = await this.client.send(
              SESSION_CREATE_ENDPOINT,
              { proof: { method: "password", ...this.credentials } },
              createReqZ,
              createResZ,
            );
            if (generation !== this.generation) return null;
            this.authState = {
              authenticated: true,
              session: res.session,
              user: res.user,
              token: res.token,
            };
            this.observer.onSuccess?.();
            return null;
          } catch (err) {
            const parsed = errors.fromUnknown(err);
            if (generation === this.generation && AuthError.matches(parsed))
              this.observer.onFailure?.(parsed);
            return parsed;
          }
        })();
        const err = await this.authenticating;
        if (err != null) {
          // clear the memoized failure so the next request retries the login
          this.authenticating = undefined;
          throw err;
        }
      }
      reqCtx.params.Authorization = `Bearer ${this.token}`;
      try {
        const resCtx = await next(reqCtx);
        this.retryCount = 0;
        return resCtx;
      } catch (err) {
        if (RETRY_ON.some((e) => e.matches(err)) && this.retryCount < MAX_RETRIES) {
          this.authState = { authenticated: false };
          this.authenticating = undefined;
          this.retryCount += 1;
          return await mw(reqCtx, next);
        }
        this.retryCount = 0;
        throw errors.fromUnknown(err);
      }
    };
    return mw;
  }
}
