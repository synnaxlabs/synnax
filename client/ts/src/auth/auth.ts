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

const createReqZ = z.object({ proof: proofZ, cookie: z.boolean().optional() });
const createResZ = z.object({
  token: z.string(),
  session: sessionZ,
  user: user.userZ.optional(),
});
const renewResZ = z.object({ session: sessionZ, user: user.userZ.optional() });
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

// routes the Core serves without a token, so a request to one never logs in
const TOKENLESS_ENDPOINTS = [SESSION_CREATE_ENDPOINT, LEGACY_LOGIN_ENDPOINT];

const MAX_RETRIES = 3;

// marks a request that carries its own token and never logs in: a logout or a resume.
// The middleware strips it before the request leaves.
const OWN_TOKEN_PARAM = "synnax-own-token";

const changePasswordReqZ = z.object({
  username: z.string(),
  password: z.string(),
  newPassword: z.string().min(1),
});
const changePasswordResZ = z.object({});

const RETRY_ON = [InvalidTokenError, ExpiredTokenError, SessionEndedError] as const;

type AuthState =
  { authenticated: false } | { authenticated: true; token: string; session: Session };

/** How a client proves who it is on its first request. One field is enough. */
export interface Authority {
  /** A username and password to log in with. */
  credentials?: Credentials;
  /** The token of an earlier login to resume. */
  token?: string;
  /** Whether the Core keeps the token in a cookie the client never sees. */
  cookie?: boolean;
  /** Whether the session outlives the client, so closing the client leaves it open. */
  persistent?: boolean;
}

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
  private credentials?: Credentials;
  private resumeToken?: string;
  private readonly cookie: boolean;
  private readonly persistent: boolean;
  private readonly observer: Observer;
  private authState: AuthState = { authenticated: false };
  authenticating: Promise<Error | null> | undefined;
  private generation = 0;
  // whether this client opened the session with credentials; a resumed session
  // belongs to whoever saved its token or cookie
  private ownsSession = false;

  constructor(client: UnaryClient, authority: Authority, observer: Observer = {}) {
    this.client = client;
    this.credentials =
      authority.credentials == null ? undefined : { ...authority.credentials };
    this.resumeToken = authority.token;
    this.cookie = authority.cookie ?? false;
    this.persistent = authority.persistent ?? false;
    this.observer = observer;
  }

  /** Replaces the stored credentials and clears auth state for a fresh login. */
  setCredentials({ username, password }: Credentials): void {
    this.credentials = { username, password };
    this.resumeToken = undefined;
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

  /**
   * Changes the logged-in user's password.
   * @throws {Error} if the client is not logged in, or holds no current password
   * because it resumed a session from a token or cookie.
   */
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
   * @throws {AuthError} when the login before the request fails.
   * @throws {SessionEndedError} when the session ends and the retries run out.
   */
  async renew(): Promise<Session> {
    const generation = this.generation;
    const res = await this.client.send(
      SESSION_RENEW_ENDPOINT,
      undefined,
      z.void(),
      renewResZ,
    );
    // a session another login replaced in the meantime keeps its own record
    if (
      generation === this.generation &&
      this.authState.authenticated &&
      this.authState.session.key === res.session.key
    )
      this.authState = { ...this.authState, session: res.session };
    return res.session;
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
   * Ends the client's own session and forgets it and the credentials, so later requests
   * fail until new credentials are set. Waits for a login in flight, then ends its
   * session too. A session the Core already ended counts as logged out.
   * @throws {Error} when the Core cannot be reached or rejects the delete for another
   * reason.
   */
  async logout(): Promise<void> {
    // a login still in flight would open a session this logout never ends
    if (!this.authState.authenticated) await this.authenticating;
    const { authState } = this;
    if (!authState.authenticated) {
      this.resumeToken = undefined;
      this.credentials = undefined;
      return;
    }
    const { key } = authState.session;
    try {
      await this.sendAs(
        authState.token,
        SESSION_DELETE_ENDPOINT,
        { keys: [key] },
        deleteReqZ,
        emptyResZ,
      );
    } catch (err) {
      if (!RETRY_ON.some((e) => e.matches(err))) throw errors.fromUnknown(err);
    } finally {
      this.resumeToken = undefined;
      // a request that logged in again during the delete holds a session this logout
      // did not end
      if (this.authState.authenticated && this.authState.session.key === key) {
        this.authState = { authenticated: false };
        this.authenticating = undefined;
        this.credentials = undefined;
      }
    }
  }

  /**
   * Lets go of the session when the client closes. A session this client opened with
   * credentials ends unless it is persistent; one resumed from a saved token or cookie
   * stays open for whoever saved it.
   * @throws {Error} if the Core cannot be reached or refuses to end the session.
   */
  async release(): Promise<void> {
    // a login still in flight decides whether this client owns the session
    if (!this.authState.authenticated) await this.authenticating;
    if (this.ownsSession && !this.persistent) return await this.logout();
    this.authState = { authenticated: false };
    this.authenticating = undefined;
  }

  /** Sends a request that carries the given token and never logs in or retries. */
  private async sendAs<RQ extends z.ZodType, RS extends z.ZodType>(
    token: string,
    target: string,
    req: z.input<RQ>,
    reqZ: RQ,
    resZ: RS,
  ): Promise<z.output<RS>> {
    return await this.client.send(target, req, reqZ, resZ, {
      [OWN_TOKEN_PARAM]: token,
    });
  }

  /**
   * Logs in with the credentials, or resumes the session a token or cookie names. A
   * token the Core rejects is forgotten, so the next attempt needs credentials; a Core
   * that cannot be reached has not rejected it.
   */
  private async login(): Promise<Error | null> {
    // a login issued before a setCredentials is stale: its result must not touch the
    // state or reach the observer
    const generation = this.generation;
    try {
      let state: AuthState;
      let owned: boolean;
      let u: user.User | undefined;
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
        state = { authenticated: true, token: res.token, session: res.session };
        u = res.user;
        owned = true;
      } else if (this.resumeToken != null || this.cookie) {
        // a cookie session has no token; the browser attaches the cookie
        const token = this.resumeToken ?? "";
        const res = await this.sendAs(
          token,
          SESSION_RENEW_ENDPOINT,
          undefined,
          z.void(),
          renewResZ,
        );
        state = { authenticated: true, token, session: res.session };
        u = res.user;
        owned = false;
      } else throw new AuthError("no credentials, token, or cookie to log in with");
      if (generation !== this.generation) return null;
      this.authState = state;
      this.ownsSession = owned;
      this.observer.onUser?.(u);
      this.observer.onSuccess?.();
      return null;
    } catch (err) {
      const parsed = errors.fromUnknown(err);
      if (generation !== this.generation) return null;
      this.authState = { authenticated: false };
      if (AuthError.matches(parsed)) {
        this.resumeToken = undefined;
        this.observer.onFailure?.(parsed);
      }
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
      const own = reqCtx.params[OWN_TOKEN_PARAM];
      if (own != null) {
        delete reqCtx.params[OWN_TOKEN_PARAM];
        if (own !== "") reqCtx.params.Authorization = `Bearer ${own}`;
        return await next(reqCtx);
      }
      const tokenless = TOKENLESS_ENDPOINTS.some((t) => reqCtx.target.endsWith(t));
      if (!this.authenticated && !tokenless) {
        this.authenticating ??= this.login();
        const err = await this.authenticating;
        if (err != null) {
          // clear the memoized failure so the next request retries the login
          this.authenticating = undefined;
          throw err;
        }
      }
      // a cookie session has no token to send; the browser attaches the cookie
      const { authState } = this;
      if (authState.authenticated && authState.token !== "")
        reqCtx.params.Authorization = `Bearer ${authState.token}`;
      try {
        return await next(reqCtx);
      } catch (err) {
        if (RETRY_ON.some((e) => e.matches(err)) && retries < MAX_RETRIES) {
          // only the session this request carried is dead; a login that finished in
          // the meantime holds a different one
          if (this.authState === authState) {
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
