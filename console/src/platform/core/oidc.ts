// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { auth, AuthError, InvalidCredentialsError } from "@synnaxlabs/client";
import { errors } from "@synnaxlabs/x";
import { invoke } from "@tauri-apps/api/core";
import { z } from "zod";

import { writeToken } from "@/platform/core/token";
import { Runtime } from "@/platform/runtime";
import { Session } from "@/session";

/** The scopes a login asks the provider for: the ID token, with the `name` claim. */
const SCOPES = "openid profile";

const PENDING_KEY = "synnax-oidc-login";

const INVITE_REFUSED =
  "This invite code is not valid. Ask your administrator for a new one.";

const base64url = (bytes: Uint8Array): string =>
  btoa(String.fromCharCode(...bytes))
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/, "");

const randomString = (): string =>
  base64url(crypto.getRandomValues(new Uint8Array(32)));

// the verifier stays here, and the challenge goes to the provider
interface PKCE {
  verifier: string;
  challenge: string;
}

const createPKCE = async (): Promise<PKCE> => {
  const verifier = randomString();
  const digest = await crypto.subtle.digest(
    "SHA-256",
    new TextEncoder().encode(verifier),
  );
  return { verifier, challenge: base64url(new Uint8Array(digest)) };
};

interface AuthorizeURLArgs {
  login: auth.OIDCLogin;
  redirectUri: string;
  challenge: string;
  state: string;
}

// keeps the query values the provider's address already holds
// throws for a scheme other than HTTPS or HTTP, which could run script in the Console
const authorizeURL = ({
  login,
  redirectUri,
  challenge,
  state,
}: AuthorizeURLArgs): string => {
  const url = new URL(login.authorizationEndpoint);
  if (url.protocol !== "https:" && url.protocol !== "http:")
    throw new Error(
      `Login address of ${login.provider} must use HTTPS or HTTP, not ${url.protocol}`,
    );
  Object.entries({
    response_type: "code",
    client_id: login.clientId,
    redirect_uri: redirectUri,
    scope: SCOPES,
    code_challenge: challenge,
    code_challenge_method: "S256",
    state,
  }).forEach(([key, value]) => url.searchParams.set(key, value));
  return url.toString();
};

/**
 * Raised when the provider returns without a code because the person cancelled the
 * login or the provider refused it. Its message says why.
 */
export class UnfinishedLoginError extends errors.createTyped(
  "console.oidc.unfinished",
) {}

const UNFINISHED = "You cancelled the login, or the provider refused it.";

// throws when the provider refused, or when the redirect belongs to another login
const parseCallback = (query: string, state: string): string => {
  const params = new URLSearchParams(query);
  const error = params.get("error");
  if (error === "access_denied")
    throw new UnfinishedLoginError(
      params.get("error_description")?.trim() || UNFINISHED,
    );
  if (error != null)
    throw new Error(
      params.get("error_description") ?? `The provider refused: ${error}`,
    );
  const code = params.get("code");
  if (params.get("state") !== state || code == null)
    throw new Error("The login redirect does not belong to this login");
  return code;
};

/**
 * Proves a provider account from the desktop Console: opens the provider's login page
 * in the browser and waits for it to redirect to a loopback port.
 * @returns The credentials, which the Core accepts once.
 * @throws {UnfinishedLoginError} when the person cancels or the provider denies access.
 * @throws {Error} when the person does not finish within five minutes, or the provider
 * returns another error.
 */
export const authorizeDesktop = async (
  login: auth.OIDCLogin,
): Promise<auth.OIDCCredentials> => {
  const { verifier, challenge } = await createPKCE();
  const state = randomString();
  const port = await invoke<number>("oidc_listen");
  const redirectUri = `http://127.0.0.1:${port}/callback`;
  await Runtime.openExternal(authorizeURL({ login, redirectUri, challenge, state }));
  const query = await invoke<string>("oidc_wait", { port, state });
  const code = parseCallback(query, state);
  return { provider: login.provider, code, verifier, redirectUri };
};

/**
 * Links the provider account that the credentials prove to the invited user and opens
 * a session, without a logged-in client.
 * @param invite - The invite code.
 * @param cookie - Whether the Core keeps the session's token in a cookie.
 * @returns The session's token, which is empty when the cookie holds it.
 * @throws {Error} with a message for the person, caused by the Core's error, when the
 * Core refuses the invite code or the proof, which it reports as one error.
 */
export const redeemInvite = async (
  core: auth.TokenlessParams,
  invite: string,
  credentials: auth.OIDCCredentials,
  cookie = false,
): Promise<string> => {
  try {
    const { token } = await auth.redeemInvite(auth.createTokenlessClient(core), {
      token: invite,
      credentials,
      cookie,
    });
    return token;
  } catch (e) {
    if (!AuthError.matches(e)) throw errors.fromUnknown(e);
    throw new Error(INVITE_REFUSED, { cause: e });
  }
};

/**
 * Gives the invited user its first username and password and opens a session, without
 * a logged-in client. The desktop Console saves the session's token in the keychain.
 * @param invite - The invite code.
 * @param cookie - Whether the Core keeps the session's token in a cookie.
 * @returns The session's token, which is empty when the cookie holds it.
 * @throws {Error} with a message for the person, caused by the Core's error, when the
 * Core refuses the invite code.
 * @throws the Core's error when it refuses the username or password.
 */
export const redeemInviteWithPassword = async (
  core: Pick<Session.Core.Core, "key" | "host" | "port" | "secure">,
  invite: string,
  credentials: auth.Credentials,
  cookie = false,
): Promise<string> => {
  let token: string;
  try {
    ({ token } = await auth.redeemInviteWithPassword(auth.createTokenlessClient(core), {
      token: invite,
      credentials,
      cookie,
    }));
  } catch (e) {
    if (!InvalidCredentialsError.matches(e)) throw errors.fromUnknown(e);
    throw new Error(INVITE_REFUSED, { cause: e });
  }
  if (Session.Runtime.ENGINE === "tauri")
    await writeToken({ ...core, username: "" }, token);
  return token;
};

/**
 * Logs in through a provider from the desktop Console: exchanges the proof from
 * {@link authorizeDesktop} for a session and saves the session's token in the keychain.
 * The code never reaches the Core's shared record, so other windows resume the saved
 * session instead of spending the same code.
 * @param invite - An invite code to redeem, which links the account before the session
 * opens.
 * @param signal - Ends the login once aborted, with its abort reason: it then sends
 * nothing more to the Core and saves no token.
 * @throws {UnfinishedLoginError} when the person cancels or the provider denies access.
 * @throws {Error} when the person does not finish within five minutes, the provider
 * returns another error, or the Core rejects the code.
 */
export const loginDesktop = async (
  core: Pick<Session.Core.Core, "key" | "host" | "port" | "secure">,
  login: auth.OIDCLogin,
  invite?: string,
  signal?: AbortSignal,
): Promise<void> => {
  const credentials = await authorizeDesktop(login);
  signal?.throwIfAborted();
  const token =
    invite == null
      ? (
          await auth.createSession(auth.createTokenlessClient(core), {
            method: "oidc",
            ...credentials,
          })
        ).token
      : await redeemInvite(core, invite, credentials);
  signal?.throwIfAborted();
  // recordLogin clears the username, so the session is saved under the account the
  // record has after it
  await writeToken({ ...core, username: "" }, token);
};

const purposeZ = z.enum(["login", "link", "invite"]);

/**
 * What a provider redirect finishes: a login replaces the session, a link adds the
 * account to the logged-in user, and an invite links the account to the invited user
 * and opens a session.
 */
export type Purpose = z.infer<typeof purposeZ>;

const pendingZ = z.object({
  purpose: purposeZ,
  /** The key of the user who started a link. */
  subject: z.string().optional(),
  /** The code that redeems an invite. */
  invite: z.string().optional(),
  core: z.string(),
  provider: z.string(),
  verifier: z.string(),
  state: z.string(),
  redirectUri: z.string(),
});

type PendingStart = Pick<z.infer<typeof pendingZ>, "purpose" | "subject" | "invite">;

const leaveForProvider = async (
  core: string,
  login: auth.OIDCLogin,
  start: PendingStart,
): Promise<void> => {
  const { verifier, challenge } = await createPKCE();
  const state = randomString();
  const redirectUri = `${window.location.origin}/`;
  const url = authorizeURL({ login, redirectUri, challenge, state });
  const pending: z.infer<typeof pendingZ> = {
    ...start,
    core,
    provider: login.provider,
    verifier,
    state,
    redirectUri,
  };
  sessionStorage.setItem(PENDING_KEY, JSON.stringify(pending));
  window.location.assign(url);
};

/**
 * Starts a login through a provider from the browser Console: saves the verifier for
 * the page load that follows, then leaves for the provider's login page. The provider
 * redirects back to the Console's own origin.
 * @param subject - The key of the logged-in user, for a redirect that links an account
 * to them. A redirect with no subject is a login.
 * @throws {Error} when the browser cannot create the PKCE pair or save the login.
 */
export const beginBrowserLogin = async (
  core: string,
  login: auth.OIDCLogin,
  subject?: string,
): Promise<void> =>
  await leaveForProvider(core, login, {
    purpose: subject == null ? "login" : "link",
    subject,
  });

/**
 * Starts redeeming an invite from the browser Console, as
 * {@link beginBrowserLogin} starts a login. The invite code stays in session storage
 * and never reaches a URL.
 * @throws {Error} when the browser cannot create the PKCE pair or save the login.
 */
export const beginBrowserInvite = async (
  core: string,
  login: auth.OIDCLogin,
  invite: string,
): Promise<void> => await leaveForProvider(core, login, { purpose: "invite", invite });

/** A browser login that returned from its provider. */
export interface CompletedLogin {
  purpose: Purpose;
  /** The key of the user who started a link. */
  subject?: string;
  /** The code of the invite to redeem. */
  invite?: string;
  /** The key of the Core the login is for. */
  core: string;
  credentials: auth.OIDCCredentials;
}

// A saved login that is not valid, such as one from an older Console, reads as none.
const parsePending = (raw: string): z.infer<typeof pendingZ> | undefined => {
  try {
    return pendingZ.parse(JSON.parse(raw));
  } catch {
    return undefined;
  }
};

const readPending = (): z.infer<typeof pendingZ> | undefined => {
  const raw = sessionStorage.getItem(PENDING_KEY);
  return raw == null ? undefined : parsePending(raw);
};

/**
 * @returns the purpose of the browser login in progress, if any. A saved login that is
 * not valid has no purpose.
 */
export const pendingPurpose = (): Purpose | undefined => readPending()?.purpose;

/**
 * Finishes a browser login on the page load the provider redirected to. Removes the
 * code from the address bar either way. Resolves undefined when the page did not come
 * back from a login, and then clears a saved login that is not valid.
 * @throws {UnfinishedLoginError} when the person cancelled or the provider refused.
 * @throws {Error} when the provider returned another error, the redirect belongs to
 * another login, or the saved login is not valid.
 */
export const completeBrowserLogin = (): CompletedLogin | undefined => {
  const raw = sessionStorage.getItem(PENDING_KEY);
  if (raw == null) return undefined;
  const pending = parsePending(raw);
  const { search } = window.location;
  const params = new URLSearchParams(search);
  if (!params.has("code") && !params.has("error")) {
    if (pending == null) sessionStorage.removeItem(PENDING_KEY);
    return undefined;
  }
  sessionStorage.removeItem(PENDING_KEY);
  window.history.replaceState(null, "", window.location.pathname);
  if (pending == null)
    throw new Error("The saved provider login is not valid. Please log in again.");
  const code = parseCallback(search, pending.state);
  return {
    purpose: pending.purpose,
    subject: pending.subject,
    invite: pending.invite,
    core: pending.core,
    credentials: {
      provider: pending.provider,
      code,
      verifier: pending.verifier,
      redirectUri: pending.redirectUri,
    },
  };
};
