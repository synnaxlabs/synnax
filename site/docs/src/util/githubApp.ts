// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/x";
import { sign } from "crypto";
import { z } from "zod";

const API_URL = "https://api.github.com";
const INSTALLATION_URL = `${API_URL}/repos/synnaxlabs/synnax/installation`;
// GitHub rejects an `iat` ahead of its clock, so the JWT starts in the past.
const CLOCK_DRIFT = TimeSpan.seconds(60);
const JWT_LIFETIME = TimeSpan.minutes(9);
const REFRESH_MARGIN = TimeSpan.minutes(5);

const installationZ = z.object({ id: z.number() });
const accessTokenZ = z.object({ token: z.string(), expires_at: z.string() });

export interface TokenSourceOptions {
  /** The ID of the GitHub App. */
  appID?: string;
  /** The private key of the GitHub App, in PEM. */
  privateKey?: string;
  /** A static token, such as a GitHub Actions job token, for when no App is set. */
  token?: string;
  fetch?: typeof fetch;
  now?: () => number;
}

interface Token {
  value: string;
  /** When to mint the next token, in milliseconds since the epoch. */
  refresh: number;
}

const encode = (value: object): string =>
  Buffer.from(JSON.stringify(value)).toString("base64url");

const jwt = (appID: string, privateKey: string, now: number): string => {
  const seconds = Math.floor(now / 1000);
  const header = encode({ alg: "RS256", typ: "JWT" });
  const payload = encode({
    iat: seconds - CLOCK_DRIFT.seconds,
    exp: seconds + JWT_LIFETIME.seconds,
    iss: appID,
  });
  const signature = sign("sha256", Buffer.from(`${header}.${payload}`), privateKey);
  return `${header}.${payload}.${signature.toString("base64url")}`;
};

/**
 * Builds a source of installation tokens for the GitHub App installed on the Synnax
 * repository. A token serves until five minutes before it expires.
 * @returns A function that resolves to a token. Without an App, it resolves to the
 * static token, or is undefined if there is none.
 * @throws {Error} if only one of the ID and the key is set. The function it returns
 * rejects if GitHub answers with an error.
 */
export const tokenSource = ({
  appID,
  privateKey,
  token,
  fetch: fetcher = globalThis.fetch,
  now = Date.now,
}: TokenSourceOptions): (() => Promise<string>) | undefined => {
  if (appID == null && privateKey == null)
    return token == null ? undefined : () => Promise.resolve(token);
  if (appID == null || privateKey == null)
    throw new Error("A GitHub App needs both an ID and a private key");

  const request = async (url: string, method: string): Promise<unknown> => {
    const response = await fetcher(url, {
      method,
      headers: {
        Accept: "application/vnd.github+json",
        Authorization: `Bearer ${jwt(appID, privateKey, now())}`,
      },
    });
    if (!response.ok) throw new Error(`GitHub App API returned ${response.status}`);
    return await response.json();
  };

  const mint = async (): Promise<Token> => {
    const { id } = installationZ.parse(await request(INSTALLATION_URL, "GET"));
    const url = `${API_URL}/app/installations/${id}/access_tokens`;
    const { token, expires_at } = accessTokenZ.parse(await request(url, "POST"));
    const refresh = new Date(expires_at).getTime() - REFRESH_MARGIN.milliseconds;
    return { value: token, refresh };
  };

  let cached: Promise<Token> | null = null;
  const start = (): Promise<Token> => {
    const token = mint();
    cached = token;
    // A failed mint is not cached, so the next call retries.
    void token.catch(() => {
      if (cached === token) cached = null;
    });
    return token;
  };

  return async () => {
    const current = cached ?? start();
    const { value, refresh } = await current;
    if (refresh > now()) return value;
    const next = cached != null && cached !== current ? cached : start();
    return (await next).value;
  };
};
