// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeAll, beforeEach, describe, expect, it } from "vitest";

import { event, type License } from "@/server/db/schema";
import { type Memory, openMemory } from "@/server/db/testutil";
import { listForLicense } from "@/server/event";
import { NOW } from "@/server/license/testutil";
import { createLicense, createOrganization } from "@/testutil";

const MINUTE = 60 * 1000;

describe("event.listForLicense", () => {
  let store: Memory;
  let lic: License;
  let other: License;
  beforeAll(async () => {
    store = await openMemory();
  });
  beforeEach(async () => {
    await store.clear();
    const org = await createOrganization(store, { kind: "team", clerkOrgID: "org_a" });
    lic = await createLicense(store, { organization: org.key });
    other = await createLicense(store, { organization: org.key });
  });

  const record = async (license: string, minutesAgo: number, actor = "user_a") => {
    const [row] = await store.query
      .insert(event)
      .values({
        kind: "download",
        actor,
        license,
        at: new Date(NOW.getTime() - minutesAgo * MINUTE),
      })
      .returning();
    return row;
  };

  it("should list a license's events newest first", async () => {
    const oldest = await record(lic.key, 30);
    const newest = await record(lic.key, 1);
    const middle = await record(lic.key, 10);
    await record(other.key, 0);
    expect(await listForLicense(store, lic.key)).toEqual([newest, middle, oldest]);
  });

  it("should keep the most recent events up to the limit", async () => {
    for (let i = 60; i > 0; i--) await record(lic.key, i, `user_${i}`);
    const page = await listForLicense(store, lic.key);
    expect(page).toHaveLength(50);
    expect(page[0].actor).toBe("user_1");
    expect(page[49].actor).toBe("user_50");
    expect((await listForLicense(store, lic.key, 3)).map((e) => e.actor)).toEqual([
      "user_1",
      "user_2",
      "user_3",
    ]);
  });

  it("should list nothing for a license without events", async () => {
    expect(await listForLicense(store, lic.key)).toEqual([]);
  });
});
