// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeAll, beforeEach, describe, expect, it } from "vitest";

import { event, type Organization } from "@/server/db/schema";
import { type Memory, openMemory } from "@/server/db/testutil";
import { ADMIN_ROLE, memory } from "@/server/directory";
import { NOW } from "@/server/license/testutil";
import { list, view } from "@/server/referral";
import { resolve } from "@/server/session";
import { NONE } from "@/server/usage";
import { createOrganization, createReferral, STAFF_ORG_ID } from "@/testutil";

const EARLIER = new Date(NOW.getTime() - 60 * 60 * 1000);

describe("referral", () => {
  let store: Memory;
  let ada: Organization;
  const directory = memory({
    people: {
      user_a: { name: "Ada Lovelace", email: "ada@example.com" },
      user_b: { name: "Bob", email: "bob@example.com" },
      user_staff: { name: "Emiliano", email: "em@synnaxlabs.com" },
    },
    members: {
      user_staff: [{ clerkOrgID: STAFF_ORG_ID, name: "Synnax Labs", role: ADMIN_ROLE }],
    },
  });

  beforeAll(async () => {
    store = await openMemory();
  });

  beforeEach(async () => {
    await store.clear();
    ada = await createOrganization(store, { kind: "personal", ownerUserID: "user_a" });
  });

  const login = async (org: Organization, at = NOW): Promise<void> => {
    await store.query
      .insert(event)
      .values({ kind: "link", actor: "user_a", organization: org.key, at });
  };

  describe("list", () => {
    it("should list every referral, the newest first, with its stage", async () => {
      await createReferral(store, { key: "sent-link-01", createdAt: EARLIER });
      await createReferral(store, { key: "open-link-01", openedAt: NOW });
      await createReferral(store, {
        key: "signed-up-01",
        openedAt: NOW,
        userID: "user_b",
        claimedAt: NOW,
      });
      await createReferral(store, {
        key: "installed-01",
        openedAt: NOW,
        userID: "user_a",
        claimedAt: NOW,
      });
      await login(ada);
      const entries = await list(store, directory, NOW);
      expect(entries.map((e) => [e.referral.key, e.stage])).toEqual([
        ["open-link-01", "opened"],
        ["signed-up-01", "signed_up"],
        ["installed-01", "installed"],
        ["sent-link-01", "sent"],
      ]);
    });

    it("should name the referrer and the user who claimed the link", async () => {
      await createReferral(store, { userID: "user_a", claimedAt: NOW });
      await login(ada);
      const [entry] = await list(store, directory, NOW);
      expect(entry.referrer).toBe("Emiliano");
      expect(entry.account).toMatchObject({
        userID: "user_a",
        email: "ada@example.com",
      });
      expect(entry.usage.lastActive).toEqual(NOW);
    });

    it("should name Synnax Labs for a referrer that Clerk deleted", async () => {
      await createReferral(store, { createdBy: "user_gone" });
      const [entry] = await list(store, directory, NOW);
      expect(entry.referrer).toBe("Synnax Labs");
      expect(entry.account).toBeNull();
      expect(entry.usage).toEqual(NONE);
    });
  });

  describe("view", () => {
    const viewAs = async (userID: string | null, key = "Ada-link-001") =>
      await view(store, directory, {
        key,
        session: userID == null ? null : await resolve(directory, userID, STAFF_ORG_ID),
        now: NOW,
      });

    it("should return null for an unknown link", async () => {
      expect(await viewAs(null, "nope")).toBeNull();
    });

    it("should show who sent the link to a visitor who is not logged in", async () => {
      await createReferral(store);
      expect(await viewAs(null)).toEqual({
        key: "Ada-link-001",
        referrer: "Emiliano",
        viewer: null,
      });
    });

    it("should say whether the visitor logged in to Synnax Desktop", async () => {
      await createReferral(store);
      expect((await viewAs("user_a"))?.viewer).toEqual({
        email: "ada@example.com",
        staff: false,
        installed: false,
      });
      await login(ada);
      expect((await viewAs("user_a"))?.viewer?.installed).toBe(true);
    });

    it("should say when the visitor is staff", async () => {
      await createReferral(store);
      expect((await viewAs("user_staff"))?.viewer?.staff).toBe(true);
    });
  });
});
