// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Creates the Synnax Labs organization the internal licenses belong to. Its key is
// fixed because tokens already issued carry it. Run once per database:
//
//   DATABASE_URL=... STAFF_ORG_ID=org_... pnpm --filter @synnaxlabs/hub seed

import { open } from "../src/server/db/db.ts";
import { organization } from "../src/server/db/schema.ts";

const INTERNAL_ORGANIZATION_KEY = "64156293-6534-416c-99d1-8db73a22ca6a";

const required = (name: string): string => {
  const value = process.env[name];
  if (value == null || value === "") throw new Error(`${name} is not set`);
  return value;
};

const store = open(required("DATABASE_URL"));
const [row] = await store.query
  .insert(organization)
  .values({
    key: INTERNAL_ORGANIZATION_KEY,
    kind: "team",
    name: "Synnax Labs",
    clerkOrgID: required("STAFF_ORG_ID"),
  })
  .onConflictDoUpdate({
    target: organization.key,
    set: { clerkOrgID: required("STAFF_ORG_ID") },
  })
  .returning();
console.log(`organization ${row.name} ready as ${row.key}`);
