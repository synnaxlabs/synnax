// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/x";
import { z } from "zod";

import { HANDOFF_URL, RENEW_URL, UNLINK_URL } from "@/platform/license/urls";

export type RenewResult =
  { variant: "renewed"; key: string } | { variant: "unlinked"; message: string };

/** The code the hub puts on a refusal for a machine it no longer knows. */
const UNLINKED_CODE = "unlinked";

const UNLINK_TIMEOUT = TimeSpan.seconds(5);

const refusalZ = z.object({ error: z.string(), code: z.string().optional() });
const keyZ = z.object({ key: z.string() });

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
  const res = await fetch(RENEW_URL, {
    method: "POST",
    headers: { authorization: `Bearer ${secret}` },
  });
  if (res.ok) {
    const { key } = keyZ.parse(await res.json());
    return { variant: "renewed", key };
  }
  const { message, code } = await readRefusal(res);
  if ((res.status === 401 || res.status === 403) && code === UNLINKED_CODE)
    return { variant: "unlinked", message };
  throw new Error(`Could not renew the license: ${message}`);
};

/**
 * Asks the hub to log out the machine the renewal secret belongs to. The machine leaves
 * its account, and the secret stops working.
 * @throws {Error} if the hub refuses, or does not answer within five seconds.
 */
export const unlink = async (secret: string): Promise<void> => {
  const res = await fetch(UNLINK_URL, {
    method: "POST",
    headers: { authorization: `Bearer ${secret}` },
    signal: AbortSignal.timeout(UNLINK_TIMEOUT.milliseconds),
  });
  if (res.ok) return;
  const { message } = await readRefusal(res);
  throw new Error(`Could not log out: ${message}`);
};

/**
 * Asks the hub for the license key it holds for a handoff code. The hub gives the key
 * out once.
 * @returns the key, or null while the hub holds none for the code.
 * @throws {Error} if the hub refuses.
 * @throws {TypeError} if the hub cannot be reached.
 */
export const receive = async (
  code: string,
  signal?: AbortSignal,
): Promise<string | null> => {
  const res = await fetch(HANDOFF_URL, {
    method: "POST",
    headers: { authorization: `Bearer ${code}` },
    signal,
  });
  if (res.status === 204) return null;
  if (res.ok) return keyZ.parse(await res.json()).key;
  const { message } = await readRefusal(res);
  throw new Error(`Could not get the license key: ${message}`);
};
