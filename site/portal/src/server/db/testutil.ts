// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import path from "node:path";

import { PGlite } from "@electric-sql/pglite";
import { sql } from "drizzle-orm";
import { drizzle } from "drizzle-orm/pglite";
import { migrate } from "drizzle-orm/pglite/migrator";

import { type Store } from "@/server/db/db";
import * as schema from "@/server/db/schema";

const MIGRATIONS = path.resolve(import.meta.dirname, "../../../drizzle");

/** Memory is a store on an in-process Postgres. */
export interface Memory extends Store {
  /** clear deletes every row, keeping the schema. */
  clear: () => Promise<void>;
}

/** openMemory opens an in-process Postgres with the portal's migrations applied. */
export const openMemory = async (): Promise<Memory> => {
  const db = drizzle(new PGlite(), { schema });
  await migrate(db, { migrationsFolder: MIGRATIONS });
  return {
    query: db,
    transact: async (fn) => await db.transaction(fn),
    clear: async () => {
      await db.execute(
        sql`TRUNCATE event, activation, license, organization RESTART IDENTITY CASCADE`,
      );
    },
  };
};
