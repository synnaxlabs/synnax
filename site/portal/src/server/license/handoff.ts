// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { and, eq, gt, lte } from "drizzle-orm";

import { schema, type Store } from "@/server/db";
import { badRequest } from "@/server/errors";
import { hashSecret } from "@/server/license/desktop";

/** TTL_MS is how long the portal holds a license key for the Console to collect. */
const TTL_MS = 60 * 60 * 1000;

const CODE = /^[\w-]{32,128}$/;

/** readCode checks a handoff code from a link. Throws a 400 for a malformed one. */
export const readCode = (code: string): string => {
  if (!CODE.test(code))
    throw badRequest("The link from the Console is damaged. Open it again.");
  return code;
};

export interface OfferArgs {
  code: string;
  key: string;
  now: Date;
}

/**
 * offer holds a license key for the Console that made `code`, replacing any key held for
 * it, and drops the keys that nobody collected in time.
 */
export const offer = async (
  store: Store,
  { code, key, now }: OfferArgs,
): Promise<void> => {
  const expiresAt = new Date(now.getTime() + TTL_MS);
  await store.query.delete(schema.handoff).where(lte(schema.handoff.expiresAt, now));
  await store.query
    .insert(schema.handoff)
    .values({ codeHash: hashSecret(code), key, expiresAt })
    .onConflictDoUpdate({ target: schema.handoff.codeHash, set: { key, expiresAt } });
};

export interface TakeArgs {
  code: string;
  now: Date;
}

/**
 * take returns the license key held for `code` and stops holding it, so only one
 * request gets it. Returns null while no key is held.
 */
export const take = async (
  store: Store,
  { code, now }: TakeArgs,
): Promise<string | null> => {
  const [row] = await store.query
    .delete(schema.handoff)
    .where(
      and(
        eq(schema.handoff.codeHash, hashSecret(code)),
        gt(schema.handoff.expiresAt, now),
      ),
    )
    .returning({ key: schema.handoff.key });
  return row?.key ?? null;
};
