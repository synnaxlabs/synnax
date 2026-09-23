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
//   DATABASE_URL=... STAFF_ORG_ID=org_... pnpm --filter @synnaxlabs/hub create-internal-organization

import { neon } from "@neondatabase/serverless";
import { eq } from "drizzle-orm";
import { drizzle } from "drizzle-orm/neon-http";

import { organization } from "../src/server/db/schema.ts";

const INTERNAL_ORGANIZATION_KEY = "64156293-6534-416c-99d1-8db73a22ca6a";

const required = (name: string): string => {
  const value = process.env[name];
  if (value == null || value === "") throw new Error(`${name} is not set`);
  return value;
};

const db = drizzle(neon(required("DATABASE_URL")));
const clerkOrgID = required("STAFF_ORG_ID");

// The session mirror or the webhook may have recorded the Clerk organization under a
// random key already. Its key moves to the fixed one; a license referencing the old
// key makes that update fail loudly.
const [byKey] = await db
  .select()
  .from(organization)
  .where(eq(organization.key, INTERNAL_ORGANIZATION_KEY));
const [byClerk] = await db
  .select()
  .from(organization)
  .where(eq(organization.clerkOrgID, clerkOrgID));
let row: typeof organization.$inferSelect;
if (byKey != null)
  [row] = await db
    .update(organization)
    .set({ clerkOrgID })
    .where(eq(organization.key, INTERNAL_ORGANIZATION_KEY))
    .returning();
else if (byClerk != null)
  [row] = await db
    .update(organization)
    .set({ key: INTERNAL_ORGANIZATION_KEY, name: "Synnax Labs" })
    .where(eq(organization.key, byClerk.key))
    .returning();
else
  [row] = await db
    .insert(organization)
    .values({
      key: INTERNAL_ORGANIZATION_KEY,
      kind: "team",
      name: "Synnax Labs",
      clerkOrgID,
    })
    .returning();
console.log(`organization ${row.name} ready as ${row.key}`);
