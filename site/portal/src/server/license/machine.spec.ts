// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { eq } from "drizzle-orm";
import { beforeAll, beforeEach, describe, expect, it } from "vitest";

import { activation, event } from "@/server/db/schema";
import { type Memory, openMemory } from "@/server/db/testutil";
import { MAX_NAME_LENGTH } from "@/server/license/limits";
import { readName, rename } from "@/server/license/machine";
import { HASH_A } from "@/server/license/testutil";
import { createActivation, createLicense, createOrganization } from "@/testutil";

describe("machine.readName", () => {
  it("should trim the name it is given", () => {
    expect(readName("  Test stand  ")).toBe("Test stand");
  });

  it("should cut a name longer than the bound", () => {
    expect(readName("a".repeat(MAX_NAME_LENGTH + 10))).toHaveLength(MAX_NAME_LENGTH);
  });

  it("should refuse a name that is blank or missing", () => {
    expect(() => readName("   ")).toThrow("The machine needs a name");
    expect(() => readName(undefined)).toThrow("The machine needs a name");
  });
});

describe("machine.rename", () => {
  let store: Memory;
  beforeAll(async () => {
    store = await openMemory();
  });
  beforeEach(async () => await store.clear());

  const createMachine = async (name: string | null) => {
    const org = await createOrganization(store, { kind: "team", clerkOrgID: "org_a" });
    const lic = await createLicense(store, { organization: org.key });
    return await createActivation(store, {
      license: lic.key,
      fingerprint: [HASH_A],
      name,
    });
  };

  it("should rename the machine and record the change", async () => {
    const act = await createMachine("Old");
    await rename(store, { activationKey: act.key, name: "New", actor: "user_a" });
    const [row] = await store.query
      .select()
      .from(activation)
      .where(eq(activation.key, act.key));
    expect(row.name).toBe("New");
    const events = await store.query.select().from(event);
    expect(events).toHaveLength(1);
    expect(events[0]).toMatchObject({
      kind: "rename",
      actor: "user_a",
      organization: expect.any(String),
      license: act.license,
      activation: act.key,
      detail: { from: "Old", to: "New" },
    });
  });

  it("should record nothing when the name is unchanged", async () => {
    const act = await createMachine("Same");
    await rename(store, { activationKey: act.key, name: "Same", actor: "user_a" });
    expect(await store.query.select().from(event)).toHaveLength(0);
  });

  it("should throw a 404 for an unknown machine", async () => {
    await expect(
      rename(store, {
        activationKey: crypto.randomUUID(),
        name: "New",
        actor: "user_a",
      }),
    ).rejects.toMatchObject({ status: 404, message: "Activation not found" });
  });
});
