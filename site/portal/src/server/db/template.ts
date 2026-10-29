// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { mkdtemp, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";

import { PGlite } from "@electric-sql/pglite";
import { drizzle } from "drizzle-orm/pglite";
import { migrate } from "drizzle-orm/pglite/migrator";
import { type TestProject } from "vitest/node";

declare module "vitest" {
  export interface ProvidedContext {
    /** dbTemplate is the path of a Postgres data directory dump, fully migrated. */
    dbTemplate: string;
  }
}

const MIGRATIONS = path.resolve(import.meta.dirname, "../../../drizzle");

/**
 * setup is a Vitest global setup. It migrates one Postgres and dumps it, so each spec
 * file loads the dump instead of running initdb and the migrations again.
 */
export const setup = async (project: TestProject): Promise<() => Promise<void>> => {
  const dir = await mkdtemp(path.join(tmpdir(), "portal-db-"));
  const file = path.join(dir, "template.tar");
  const pg = new PGlite();
  await migrate(drizzle(pg), { migrationsFolder: MIGRATIONS });
  const dump = await pg.dumpDataDir("none");
  await writeFile(file, new Uint8Array(await dump.arrayBuffer()));
  await pg.close();
  project.provide("dbTemplate", file);
  return async () => await rm(dir, { recursive: true, force: true });
};
