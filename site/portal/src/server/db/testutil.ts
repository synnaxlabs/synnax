// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { PGlite } from "@electric-sql/pglite";
import { sql } from "drizzle-orm";
import { drizzle } from "drizzle-orm/pglite";
import { inject } from "vitest";

import { schema } from "@/server/db";
import { type Store } from "@/server/db/db";

/** Memory is a store on an in-process Postgres. */
export interface Memory extends Store {
  /** clear deletes every row, keeping the schema. */
  clear: () => Promise<void>;
}

/** openMemory opens an in-process Postgres with the portal's migrations applied. */
export const openMemory = async (): Promise<Memory> => {
  const dump = new Blob([Buffer.from(inject("dbTemplate"), "base64")]);
  const db = drizzle(new PGlite({ loadDataDir: dump }), { schema });
  return {
    query: db,
    transact: async (fn) => await db.transaction(fn),
    clear: async () => {
      await db.execute(
        sql`TRUNCATE event, handoff, invitation, referral, activation, license,
          organization RESTART IDENTITY CASCADE`,
      );
    },
  };
};

/** failingEvents runs `body` while recording an event of `kind` fails. */
export const failingEvents = async <T>(
  store: Memory,
  kind: string,
  body: () => Promise<T>,
): Promise<T> => {
  await store.query.execute(
    sql.raw(`alter table event add constraint failing check (kind <> '${kind}')`),
  );
  try {
    return await body();
  } finally {
    await store.query.execute(sql`alter table event drop constraint failing`);
  }
};
