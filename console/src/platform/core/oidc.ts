// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { auth, AuthError } from "@synnaxlabs/client";
import { errors } from "@synnaxlabs/x";
import { invoke } from "@tauri-apps/api/core";
import { z } from "zod";

import { writeToken } from "@/platform/core/token";
import { Runtime } from "@/platform/runtime";
import { type Session } from "@/session";

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
const authorizeURL = ({
  login,
  redirectUri,
  challenge,
  state,
}: AuthorizeURLArgs): string => {
  const url = new URL(login.authorizationEndpoint);
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

// throws when the provider refused, or when the redirect belongs to another login
const parseCallback = (query: string, state: string): string => {
  const params = new URLSearchParams(query);
  const error = params.get("error");
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
 * @throws {Error} when the person does not finish within five minutes, or the provider
 * refuses.
 */
export const authorizeDesktop = async (
  login: auth.OIDCLogin,
): Promise<auth.OIDCCredentials> => {
  const { verifier, challenge } = await createPKCE();
  const state = randomString();
  const port = await invoke<number>("oidc_listen");
  const redirectUri = `http://127.0.0.1:${port}/callback`;
  await Runtime.openExternal(authorizeURL({ login, redirectUri, challenge, state }));
  const query = await invoke<string>("oidc_wait", { state });
  const code = parseCallback(query, state);
  return { provider: login.provider, code, verifier, redirectUri };
};

/**
 * Links the provider account that the credentials prove to the invited user and opens
 * a session, without a logged-in client.
 * @param invite - The invite code.
 * @param cookie - Whether the Core keeps the session's token in a cookie.
 * @returns The session's token, which is empty when the cookie holds it.
 * @throws {Error} with a message for the person when the Core refuses the invite code
 * or the proof, which it reports as one error.
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
    const err = errors.fromUnknown(e);
    // a status appends a cause, so a wrapping error could not show this text alone
    if (AuthError.matches(err)) err.message = INVITE_REFUSED;
    throw err;
  }
};

/**
 * Logs in through a provider from the desktop Console: exchanges the proof from
 * {@link authorizeDesktop} for a session and saves the session's token in the keychain.
 * The code never reaches the Core's shared record, so other windows resume the saved
 * session instead of spending the same code.
 * @param invite - An invite code to redeem, which links the account before the session
 * opens.
 * @throws {Error} when the person does not finish within five minutes, the provider
 * refuses, or the Core rejects the code.
 */
export const loginDesktop = async (
  core: Pick<Session.Core.Core, "key" | "host" | "port" | "secure">,
  login: auth.OIDCLogin,
  invite?: string,
): Promise<void> => {
  const credentials = await authorizeDesktop(login);
  const token =
    invite == null
      ? (
          await auth.createSession(auth.createTokenlessClient(core), {
            method: "oidc",
            ...credentials,
          })
        ).token
      : await redeemInvite(core, invite, credentials);
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
  const pending: z.infer<typeof pendingZ> = {
    ...start,
    core,
    provider: login.provider,
    verifier,
    state,
    redirectUri,
  };
  sessionStorage.setItem(PENDING_KEY, JSON.stringify(pending));
  window.location.assign(authorizeURL({ login, redirectUri, challenge, state }));
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

/** The purpose of the browser login in progress, if any. */
export const pendingPurpose = (): Purpose | undefined => {
  const raw = sessionStorage.getItem(PENDING_KEY);
  return raw == null ? undefined : pendingZ.parse(JSON.parse(raw)).purpose;
};

/**
 * Finishes a browser login on the page load the provider redirected to. Removes the
 * code from the address bar either way. Resolves undefined when the page did not come
 * back from a login.
 * @throws {Error} when the provider refused or the redirect belongs to another login.
 */
export const completeBrowserLogin = (): CompletedLogin | undefined => {
  const raw = sessionStorage.getItem(PENDING_KEY);
  const { search } = window.location;
  const params = new URLSearchParams(search);
  if (raw == null || (!params.has("code") && !params.has("error"))) return undefined;
  sessionStorage.removeItem(PENDING_KEY);
  window.history.replaceState(null, "", window.location.pathname);
  const pending = pendingZ.parse(JSON.parse(raw));
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
