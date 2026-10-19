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
import { errors, TimeStamp } from "@synnaxlabs/x";
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

const clusterInfoZ = z.object({
  clusterKey: z.string(),
  nodeVersion: z.string().optional(),
  nodeKey: z.number().optional(),
  nodeTime: TimeStamp.z,
});

const createReqZ = z.object({ proof: proofZ, cookie: z.boolean().optional() });
const createResZ = z.object({
  token: z.string(),
  session: sessionZ,
  user: user.userZ.optional(),
  clusterInfo: clusterInfoZ.optional(),
});
const renewResZ = z.object({ session: sessionZ, user: user.userZ.optional() });
const retrieveReqZ = z.object({ subjects: ontology.idZ.array().optional() });
const retrieveResZ = z.object({ sessions: sessionZ.array() });
const deleteReqZ = z.object({ keys: sessionKeyZ.array() });
const emptyResZ = z.object({});

/** Opens a session from a proof. Works without a token. */
/** The login route a Core without sessions serves. Kept exempt for one release. */
export const LEGACY_LOGIN_ENDPOINT = "/auth/login";
export const SESSION_CREATE_ENDPOINT = "/auth/session/create";
/** Reports activity on the caller's session. */
export const SESSION_RENEW_ENDPOINT = "/auth/session/renew";
/** Lists sessions. */
export const SESSION_RETRIEVE_ENDPOINT = "/auth/session/retrieve";
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
  | { authenticated: true; token: string; session?: Session; user?: user.User };

/** How a client proves who it is on its first request. One field is enough. */
export interface Authority {
  /** A username and password to log in with. */
  credentials?: Credentials;
  /** The token of an earlier login to resume. */
  token?: string;
  /** Whether the Core keeps the token in a cookie the client never sees. */
  cookie?: boolean;
}

/** Login outcome hooks. The connection machine is the sole consumer. */
export interface Observer {
  /** Fires after every successful login. */
  onSuccess?: () => void;
  /** Fires on a definitive credential rejection, never on token self-heal. */
  onFailure?: (error: Error) => void;
}

export class Client {
  private readonly client: UnaryClient;
  private credentials?: Credentials;
  private resumeToken?: string;
  private readonly cookie: boolean;
  private readonly observer: Observer;
  private authState: AuthState = { authenticated: false };
  authenticating: Promise<Error | null> | undefined;
  private validating = false;
  private generation = 0;
  private logoutToken: string | undefined;

  constructor(client: UnaryClient, authority: Authority, observer: Observer = {}) {
    this.client = client;
    this.credentials =
      authority.credentials == null ? undefined : { ...authority.credentials };
    this.resumeToken = authority.token;
    this.cookie = authority.cookie ?? false;
    this.observer = observer;
  }

  /** Replaces the stored credentials and clears auth state for a fresh login. */
  setCredentials({ username, password }: Credentials): void {
    this.credentials = { username, password };
    this.resumeToken = undefined;
    this.authState = { authenticated: false };
    this.authenticating = undefined;
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
    return this.authState.authenticated ? this.authState.session?.subject : undefined;
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
    if (this.credentials == null)
      throw new Error("Cannot change a password without the current one");
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
   * Reports activity on the session so it does not idle out, and returns the session
   * the Core sees.
   */
  async renew(): Promise<Session> {
    const res = await this.client.send(
      SESSION_RENEW_ENDPOINT,
      undefined,
      z.void(),
      renewResZ,
    );
    if (this.authState.authenticated)
      this.authState = {
        ...this.authState,
        session: res.session,
        user: res.user ?? this.authState.user,
      };
    return res.session;
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

  /**
   * Ends the client's own session and forgets it. A later request logs in again. A
   * session the Core already ended counts as logged out.
   */
  async logout(): Promise<void> {
    const { authState } = this;
    if (!authState.authenticated) return;
    const { key } = authState.session ?? (await this.renew());
    this.logoutToken = authState.token;
    try {
      await this.deleteSessions(key);
    } catch (err) {
      if (!RETRY_ON.some((e) => e.matches(err))) throw errors.fromUnknown(err);
    } finally {
      this.logoutToken = undefined;
      this.resumeToken = undefined;
      // a request that logged in again during the delete holds a session this logout
      // did not end
      if (this.authState.authenticated && this.authState.session?.key === key) {
        this.authState = { authenticated: false };
        this.authenticating = undefined;
      }
    }
  }

  /**
   * Logs in with the credentials, or resumes the session a token or cookie names. A
   * token the Core rejects is forgotten, so the next attempt needs credentials.
   */
  private async login(): Promise<Error | null> {
    // logins issued before a setCredentials are stale; their result must not reach
    // the observer
    const generation = this.generation;
    try {
      if (this.credentials != null) {
        const res = await this.client.send(
          SESSION_CREATE_ENDPOINT,
          {
            proof: { method: "password", ...this.credentials },
            cookie: this.cookie || undefined,
          },
          createReqZ,
          createResZ,
        );
        if (generation !== this.generation) return null;
        this.authState = {
          authenticated: true,
          token: res.token,
          session: res.session,
          user: res.user,
        };
      } else if (this.resumeToken != null || this.cookie) {
        // the renew request below needs the token on it, so the state is
        // authenticated before the Core has confirmed it
        this.authState = { authenticated: true, token: this.resumeToken ?? "" };
        this.validating = true;
        try {
          await this.renew();
        } finally {
          this.validating = false;
        }
        if (generation !== this.generation) return null;
      } else throw new AuthError("no credentials, token, or cookie to log in with");
      this.observer.onSuccess?.();
      return null;
    } catch (err) {
      const parsed = errors.fromUnknown(err);
      this.authState = { authenticated: false };
      this.resumeToken = undefined;
      if (generation === this.generation && AuthError.matches(parsed))
        this.observer.onFailure?.(parsed);
      return parsed;
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
      // a logout ends the session its token belongs to; a dead one must not log in
      // again. A cookie session has no token, and the cookie goes with the request.
      if (this.logoutToken != null && reqCtx.target.endsWith(SESSION_DELETE_ENDPOINT)) {
        if (this.logoutToken !== "")
          reqCtx.params.Authorization = `Bearer ${this.logoutToken}`;
        return await next(reqCtx);
      }
      if (!this.authenticated && !reqCtx.target.endsWith(SESSION_CREATE_ENDPOINT)) {
        this.authenticating ??= this.login();
        const err = await this.authenticating;
        if (err != null) {
          // clear the memoized failure so the next request retries the login
          this.authenticating = undefined;
          throw err;
        }
      }
      // a cookie session has no token to send; the browser attaches the cookie
      if (this.token != null && this.token !== "")
        reqCtx.params.Authorization = `Bearer ${this.token}`;
      try {
        return await next(reqCtx);
      } catch (err) {
        if (
          !this.validating &&
          RETRY_ON.some((e) => e.matches(err)) &&
          retries < MAX_RETRIES
        ) {
          this.authState = { authenticated: false };
          this.authenticating = undefined;
          return await attempt(reqCtx, next, retries + 1);
        }
        throw errors.fromUnknown(err);
      }
    };
    return async (reqCtx, next) => await attempt(reqCtx, next, 0);
  }
}
