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
import { drizzle } from "drizzle-orm/pglite";
import { migrate } from "drizzle-orm/pglite/migrator";
import { type TestProject } from "vitest/node";

declare module "vitest" {
  export interface ProvidedContext {
    /** dbTemplate is a gzipped, base64 dump of a fully migrated Postgres. */
    dbTemplate: string;
  }
}

const MIGRATIONS = path.resolve(import.meta.dirname, "../../../drizzle");

/** setup provides a fully migrated Postgres dump to the spec files. */
export const setup = async (project: TestProject): Promise<void> => {
  const pg = new PGlite();
  try {
    await migrate(drizzle(pg), { migrationsFolder: MIGRATIONS });
    const dump = await pg.dumpDataDir("gzip");
    project.provide(
      "dbTemplate",
      Buffer.from(await dump.arrayBuffer()).toString("base64"),
    );
  } finally {
    await pg.close();
  }
};
