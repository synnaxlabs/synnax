// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { uuid } from "@synnaxlabs/x";
import { z } from "zod";

import { License } from "@/platform/license";

/** The URL scheme the hub hands a login back through. */
const SCHEME = "synnax-desktop";

const LINK_HOST = "activate";

const INCORRECT_FORMAT_ERROR_MESSAGE = `Login links must be of the form ${SCHEME}://${LINK_HOST}?...`;

/** Mints the state a login carries out to the hub and back. */
export const mintState = (): string => uuid.create();

export interface LoginParams {
  state: string;
  fingerprint: string[];
  /** The name this machine shows in the account. */
  name: string;
  version?: string;
}

/** The hub page that links this machine, with what it needs in the query. */
export const loginURL = ({
  state,
  fingerprint,
  name,
  version,
}: LoginParams): string => {
  const url = new URL(License.LOGIN_URL);
  url.searchParams.set("state", state);
  url.searchParams.set("fp", License.joinFingerprint(fingerprint));
  url.searchParams.set("name", name);
  if (version != null) url.searchParams.set("v", version);
  return url.toString();
};

/** What the hub hands back once the machine is linked. */
export interface Linked {
  state: string;
  key: string;
  secret: string;
  activation: string;
  email: string;
  user: string;
  /** True only when the hub turned usage data on for this login. */
  analyticsAllowed: boolean;
}

/**
 * Reads the link the hub opens the app with.
 * @throws {Error} if the URL is not a login link or a field is missing.
 */
export const parseLink = (url: string): Linked => {
  let parsed: URL;
  try {
    parsed = new URL(url);
  } catch (e) {
    throw new Error(INCORRECT_FORMAT_ERROR_MESSAGE, { cause: e });
  }
  if (parsed.protocol !== `${SCHEME}:` || parsed.host !== LINK_HOST)
    throw new Error(INCORRECT_FORMAT_ERROR_MESSAGE);
  const read = (field: Exclude<keyof Linked, "analyticsAllowed">): string => {
    const value = parsed.searchParams.get(field);
    if (value == null || value === "")
      throw new Error(`The login link is missing its ${field}`);
    return value;
  };
  return {
    state: read("state"),
    key: read("key"),
    secret: read("secret"),
    activation: read("activation"),
    email: read("email"),
    user: read("user"),
    analyticsAllowed: parsed.searchParams.get("analytics") === "on",
  };
};

export type RenewResult =
  { variant: "renewed"; key: string } | { variant: "unlinked"; message: string };

/** The code the hub puts on a refusal for a machine it no longer knows. */
const UNLINKED_CODE = "unlinked";

const refusalZ = z.object({ error: z.string(), code: z.string().optional() });
const renewedZ = z.object({ key: z.string() });

interface Refusal {
  message: string;
  code?: string;
}

const readRefusal = async (res: Response): Promise<Refusal> => {
  try {
    const body = refusalZ.safeParse(await res.json());
    if (body.success) return { message: body.data.error, code: body.data.code };
  } catch {
    // A body that is not JSON carries no message.
  }
  return { message: res.statusText };
};

/**
 * Asks the hub for a fresh license key for the machine the secret belongs to. Unlinked
 * means the hub no longer knows the machine, so the secret is spent. Only the hub's own
 * unlinked answer counts: a proxy can refuse with the same status.
 * @throws {Error} if the hub cannot be reached or refuses for another reason.
 */
export const renew = async (secret: string): Promise<RenewResult> => {
  const res = await fetch(License.RENEW_URL, {
    method: "POST",
    headers: { authorization: `Bearer ${secret}` },
  });
  if (res.ok) {
    const { key } = renewedZ.parse(await res.json());
    return { variant: "renewed", key };
  }
  const { message, code } = await readRefusal(res);
  if ((res.status === 401 || res.status === 403) && code === UNLINKED_CODE)
    return { variant: "unlinked", message };
  throw new Error(`Could not renew the license: ${message}`);
};
