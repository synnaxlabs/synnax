// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type Context,
  type Middleware,
  type Next,
  type UnaryClient,
} from "@synnaxlabs/freighter";
import { errors } from "@synnaxlabs/x";
import { z } from "zod";

import {
  type Credentials,
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

const createReqZ = z.object({ proof: proofZ });
const createResZ = z.object({
  token: z.string(),
  session: sessionZ,
  user: user.userZ.optional(),
});
const retrieveReqZ = z.object({ subjects: ontology.idZ.array().optional() });
const retrieveResZ = z.object({ sessions: sessionZ.array() });
const deleteReqZ = z.object({ keys: sessionKeyZ.array() });
const emptyResZ = z.object({});

// the login route a Core without sessions serves; kept for one release
const LEGACY_LOGIN_ENDPOINT = "/auth/login";
const SESSION_CREATE_ENDPOINT = "/auth/session/create";
const SESSION_RENEW_ENDPOINT = "/auth/session/renew";
const SESSION_RETRIEVE_ENDPOINT = "/auth/session/retrieve";
const SESSION_DELETE_ENDPOINT = "/auth/session/delete";

/** The routes the client sends to as it opens, keeps, and ends a session. */
export const SESSION_ENDPOINTS = [
  LEGACY_LOGIN_ENDPOINT,
  SESSION_CREATE_ENDPOINT,
  SESSION_RENEW_ENDPOINT,
  SESSION_RETRIEVE_ENDPOINT,
  SESSION_DELETE_ENDPOINT,
] as const;

const MAX_RETRIES = 3;

const changePasswordReqZ = z.object({
  username: z.string(),
  password: z.string(),
  newPassword: z.string().min(1),
});
const changePasswordResZ = z.object({});

const RETRY_ON = [InvalidTokenError, ExpiredTokenError, SessionEndedError] as const;

type AuthState =
  { authenticated: false } | { authenticated: true; session: Session; token: string };

/** Login outcome hooks. */
export interface Observer {
  /**
   * Fires with the user after every successful login, before onSuccess, and with
   * undefined when setCredentials drops the user.
   */
  onUser?: (user: user.User | undefined) => void;
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
  private generation = 0;
  private logoutToken: string | undefined;

  constructor(client: UnaryClient, credentials: Credentials, observer: Observer = {}) {
    this.client = client;
    this.credentials = credentials;
    this.observer = observer;
  }

  /** Replaces the stored credentials and clears auth state for a fresh login. */
  setCredentials({ username, password }: Credentials): void {
    this.credentials.username = username;
    this.credentials.password = password;
    this.authState = { authenticated: false };
    this.authenticating = undefined;
    this.generation += 1;
    this.observer.onUser?.(undefined);
  }

  get authenticated(): boolean {
    return this.authState.authenticated;
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

  /**
   * Reports activity on the session so it does not idle out.
   * @throws {AuthError} when the login before the request fails.
   * @throws {SessionEndedError} when the session ends and the retries run out.
   */
  async renew(): Promise<void> {
    await this.client.send(SESSION_RENEW_ENDPOINT, undefined, z.void(), emptyResZ);
  }

  /**
   * Lists sessions: the caller's own by default, or those of the given subjects when
   * the caller may retrieve them.
   * @throws {AccessDeniedError} when the caller may not retrieve another subject's
   * sessions.
   * @throws {AuthError} when the login before the request fails.
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

  /**
   * Ends the given sessions. Unknown keys are ignored.
   * @throws {AccessDeniedError} when the caller may not end another subject's sessions.
   * @throws {AuthError} when the login before the request fails.
   */
  async deleteSessions(...keys: SessionKey[]): Promise<void> {
    await this.client.send(SESSION_DELETE_ENDPOINT, { keys }, deleteReqZ, emptyResZ);
  }

  /**
   * Ends the client's own session and forgets it. A later request logs in again. A
   * session the Core already ended counts as logged out.
   * @throws {Error} when the Core cannot be reached or rejects the delete for another
   * reason.
   */
  async logout(): Promise<void> {
    const { authState } = this;
    if (!authState.authenticated) return;
    const { key } = authState.session;
    this.logoutToken = authState.token;
    try {
      await this.deleteSessions(key);
    } catch (err) {
      if (!RETRY_ON.some((e) => e.matches(err))) throw errors.fromUnknown(err);
    } finally {
      this.logoutToken = undefined;
      // a request that logged in again during the delete holds a session this logout
      // did not end
      if (this.authState.authenticated && this.authState.session.key === key) {
        this.authState = { authenticated: false };
        this.authenticating = undefined;
      }
    }
  }

  middleware(): Middleware {
    // retries count per request: the login a retry issues passes through this same
    // middleware, so a shared counter would reset on its success
    const attempt = async (
      reqCtx: Context,
      next: Next,
      retries: number,
    ): Promise<Context> => {
      // a logout ends its own token's session; a dead one must not log in again
      if (this.logoutToken != null && reqCtx.target.endsWith(SESSION_DELETE_ENDPOINT)) {
        reqCtx.params.Authorization = `Bearer ${this.logoutToken}`;
        return await next(reqCtx);
      }
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
              token: res.token,
            };
            this.observer.onUser?.(res.user);
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
      const sent = this.token;
      reqCtx.params.Authorization = `Bearer ${sent}`;
      try {
        return await next(reqCtx);
      } catch (err) {
        if (RETRY_ON.some((e) => e.matches(err)) && retries < MAX_RETRIES) {
          // a failure on an older session must not erase one logged in since
          if (this.token === sent) {
            this.authState = { authenticated: false };
            this.authenticating = undefined;
          }
          return await attempt(reqCtx, next, retries + 1);
        }
        throw errors.fromUnknown(err);
      }
    };
    return async (reqCtx, next) => await attempt(reqCtx, next, 0);
  }
}
