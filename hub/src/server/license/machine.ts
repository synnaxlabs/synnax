// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { eq } from "drizzle-orm";

import { type Store } from "@/server/db/db";
import { activation, event, license } from "@/server/db/schema";
import { badRequest, notFound } from "@/server/errors";

/** MAX_NAME_LENGTH bounds the name a machine carries in the ledger. */
export const MAX_NAME_LENGTH = 64;

/**
 * readName trims a posted machine name to its bound.
 * @throws {HTTPError} 400 when nothing is left of it.
 */
export const readName = (raw: string | undefined): string => {
  const name = (raw ?? "").trim().slice(0, MAX_NAME_LENGTH);
  if (name === "") throw badRequest("The machine needs a name");
  return name;
};

export interface RenameArgs {
  activationKey: string;
  name: string;
  actor: string;
}

/** rename changes what a machine is called wherever the portal names it. */
export const rename = async (
  store: Store,
  { activationKey, name, actor }: RenameArgs,
): Promise<void> => {
  const [row] = await store.query
    .select({ activation, license })
    .from(activation)
    .innerJoin(license, eq(activation.license, license.key))
    .where(eq(activation.key, activationKey));
  if (row == null) throw notFound("Activation");
  if (row.activation.name === name) return;
  await store.query
    .update(activation)
    .set({ name })
    .where(eq(activation.key, activationKey));
  await store.query.insert(event).values({
    kind: "rename",
    actor,
    organization: row.license.organization,
    license: row.license.key,
    activation: activationKey,
    detail: { from: row.activation.name, to: name },
  });
};
