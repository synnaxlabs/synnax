// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { and, count, eq, gt, inArray } from "drizzle-orm";

import { type Store } from "@/server/db/db";
import { event } from "@/server/db/schema";
import { tooMany } from "@/server/errors";

export const WINDOW_MS = 60 * 60 * 1000;
export const PER_ACTOR = 30;
export const PER_ORGANIZATION = 200;

const COUNTED = ["activate", "activate_denied", "token"] as const;

export interface CheckArgs {
  actor: string;
  organization: string;
  now: Date;
}

/**
 * check throws a 429 when the caller or the organization has issued more tokens in
 * the last hour than the limits allow. The audit log is the counter, so no other
 * state exists.
 */
export const check = async (
  store: Store,
  { actor, organization, now }: CheckArgs,
): Promise<void> => {
  const since = new Date(now.getTime() - WINDOW_MS);
  const [byActor] = await store.query
    .select({ n: count() })
    .from(event)
    .where(
      and(eq(event.actor, actor), inArray(event.kind, COUNTED), gt(event.at, since)),
    );
  if (byActor.n >= PER_ACTOR) throw tooMany();
  const [byOrg] = await store.query
    .select({ n: count() })
    .from(event)
    .where(
      and(
        eq(event.organization, organization),
        inArray(event.kind, COUNTED),
        gt(event.at, since),
      ),
    );
  if (byOrg.n >= PER_ORGANIZATION) throw tooMany();
};
