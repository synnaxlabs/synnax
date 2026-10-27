// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { auth } from "@synnaxlabs/client";
import { invoke } from "@tauri-apps/api/core";
import { z } from "zod";

import { writeToken } from "@/platform/core/token";
import { Runtime } from "@/platform/runtime";
import { type Session } from "@/session";

/** The scopes a login asks the provider for: the ID token, with the `name` claim. */
const SCOPES = "openid profile";

const PENDING_KEY = "synnax-oidc-login";

const base64url = (bytes: Uint8Array): string =>
  btoa(String.fromCharCode(...bytes))
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/, "");

const randomString = (): string =>
  base64url(crypto.getRandomValues(new Uint8Array(32)));

/** A PKCE pair: the verifier stays here, and the challenge goes to the provider. */
export interface PKCE {
  verifier: string;
  challenge: string;
}

/** Creates a PKCE pair with an S256 challenge. */
export const createPKCE = async (): Promise<PKCE> => {
  const verifier = randomString();
  const digest = await crypto.subtle.digest(
    "SHA-256",
    new TextEncoder().encode(verifier),
  );
  return { verifier, challenge: base64url(new Uint8Array(digest)) };
};

export interface AuthorizeURLArgs {
  login: auth.OIDCLogin;
  redirectUri: string;
  challenge: string;
  state: string;
}

/**
 * Returns the provider's login address for an authorization code request. Query values
 * the address already holds are kept.
 */
export const authorizeURL = ({
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

/**
 * Returns the authorization code of a redirect's query string.
 * @throws {Error} when the provider returned an error, or when the state does not
 * match the login, which means the redirect came from another login.
 */
export const parseCallback = (query: string, state: string): string => {
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
 * Logs in through a provider from the desktop Console: opens the provider's login page
 * in the browser, waits for it to redirect to a loopback port, exchanges the code for
 * a session, and saves the session's token in the keychain. The code never reaches the
 * Core's shared record, so other windows resume the saved session instead of spending
 * the same code.
 * @throws {Error} when the person does not finish within five minutes, the provider
 * refuses, or the Core rejects the code.
 */
export const loginDesktop = async (
  core: Pick<Session.Core.Core, "key" | "host" | "port" | "secure">,
  login: auth.OIDCLogin,
): Promise<void> => {
  const { verifier, challenge } = await createPKCE();
  const state = randomString();
  const port = await invoke<number>("oidc_listen");
  const redirectUri = `http://127.0.0.1:${port}/callback`;
  await Runtime.openExternal(authorizeURL({ login, redirectUri, challenge, state }));
  const query = await invoke<string>("oidc_wait");
  const code = parseCallback(query, state);
  const { token } = await auth.createSession(core, {
    method: "oidc",
    provider: login.provider,
    code,
    verifier,
    redirectUri,
  });
  // recordLogin clears the username, so the session is saved under the account the
  // record has after it
  await writeToken({ ...core, username: "" }, token);
};

const pendingZ = z.object({
  core: z.string(),
  provider: z.string(),
  verifier: z.string(),
  state: z.string(),
  redirectUri: z.string(),
});

/**
 * Starts a login through a provider from the browser Console: saves the verifier for
 * the page load that follows, then leaves for the provider's login page. The provider
 * redirects back to the Console's own origin.
 * @throws {Error} when the browser cannot create the PKCE pair or save the login.
 */
export const beginBrowserLogin = async (
  core: string,
  login: auth.OIDCLogin,
): Promise<void> => {
  const { verifier, challenge } = await createPKCE();
  const state = randomString();
  const redirectUri = `${window.location.origin}/`;
  const pending: z.infer<typeof pendingZ> = {
    core,
    provider: login.provider,
    verifier,
    state,
    redirectUri,
  };
  sessionStorage.setItem(PENDING_KEY, JSON.stringify(pending));
  window.location.assign(authorizeURL({ login, redirectUri, challenge, state }));
};

/** A browser login that returned from its provider. */
export interface CompletedLogin {
  /** The key of the Core the login is for. */
  core: string;
  credentials: auth.OIDCCredentials;
}

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
    core: pending.core,
    credentials: {
      provider: pending.provider,
      code,
      verifier: pending.verifier,
      redirectUri: pending.redirectUri,
    },
  };
};
