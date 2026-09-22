// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { uuid } from "@synnaxlabs/x";

import { License } from "@/platform/license";

/** The URL scheme the portal hands a sign-in back through. */
export const SCHEME = "synnax-desktop";

const LINK_PREFIX = `${SCHEME}://activate?`;

const INCORRECT_FORMAT_ERROR_MESSAGE = `Sign-in links must be of the form ${LINK_PREFIX}...`;

/** Mints the state a sign-in carries out to the portal and back. */
export const mintState = (): string => uuid.create();

export interface SignInParams {
  state: string;
  fingerprint: string[];
  /** The name this machine shows in the portal. */
  name: string;
  version?: string;
}

/** The portal page that links this machine, with what it needs in the query. */
export const signInURL = ({
  state,
  fingerprint,
  name,
  version,
}: SignInParams): string => {
  const url = new URL(License.PORTAL_SIGN_IN_URL);
  url.searchParams.set("state", state);
  url.searchParams.set("fp", License.joinFingerprint(fingerprint));
  url.searchParams.set("name", name);
  if (version != null) url.searchParams.set("v", version);
  return url.toString();
};

/** What the portal hands back once the machine is linked. */
export interface Linked {
  state: string;
  token: string;
  secret: string;
  activation: string;
  email: string;
}

const LINK_FIELDS: (keyof Linked)[] = [
  "state",
  "token",
  "secret",
  "activation",
  "email",
];

/**
 * Reads the link the portal opens the app with.
 * @throws {Error} if the URL is not a sign-in link or a field is missing.
 */
export const parseLink = (url: string): Linked => {
  if (!url.startsWith(LINK_PREFIX)) throw new Error(INCORRECT_FORMAT_ERROR_MESSAGE);
  const params = new URL(url).searchParams;
  const linked: Partial<Linked> = {};
  for (const field of LINK_FIELDS) {
    const value = params.get(field);
    if (value == null || value === "")
      throw new Error(`The sign-in link is missing its ${field}`);
    linked[field] = value;
  }
  return linked as Linked;
};

export type RenewResult =
  { variant: "renewed"; token: string } | { variant: "unlinked"; message: string };

const readMessage = async (res: Response): Promise<string> => {
  try {
    const body: unknown = await res.json();
    if (body != null && typeof body === "object" && "error" in body)
      return String(body.error);
  } catch {
    // A body that is not JSON carries no message.
  }
  return res.statusText;
};

/**
 * Asks the portal for a fresh token for the machine the secret belongs to. Unlinked
 * means the portal no longer knows the machine, so the secret is spent.
 * @throws {Error} if the portal cannot be reached or refuses for another reason.
 */
export const renew = async (secret: string): Promise<RenewResult> => {
  const res = await fetch(License.PORTAL_RENEW_URL, {
    method: "POST",
    headers: { authorization: `Bearer ${secret}` },
  });
  if (res.status === 401 || res.status === 403)
    return { variant: "unlinked", message: await readMessage(res) };
  if (!res.ok)
    throw new Error(`The portal refused the renewal: ${await readMessage(res)}`);
  const body: unknown = await res.json();
  if (body == null || typeof body !== "object" || !("token" in body))
    throw new Error("The portal answered without a token");
  return { variant: "renewed", token: String(body.token) };
};
