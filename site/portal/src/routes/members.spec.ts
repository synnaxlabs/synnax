// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeAll, beforeEach, describe, expect, it } from "vitest";

import { POST as inviteRoute } from "@/pages/api/members/invite";
import { type Memory, openMemory } from "@/server/db/testutil";
import { ADMIN_ROLE, MEMBER_ROLE } from "@/server/directory";
import { body, createAPIContext, createHarness, type Harness } from "@/testutil";

const ACME = "org_acme";

describe("member routes", () => {
  let store: Memory;
  let harness: Harness;
  beforeAll(async () => {
    store = await openMemory();
  });
  beforeEach(async () => {
    await store.clear();
    harness = createHarness(store);
    harness.directory.people.user_admin = { email: "ada@acme.com", name: "Ada" };
    harness.directory.people.user_member = { email: "bob@acme.com", name: "Bob" };
    harness.directory.members.user_admin = [
      { clerkOrgID: ACME, name: "Acme", role: ADMIN_ROLE },
    ];
    harness.directory.members.user_member = [
      { clerkOrgID: ACME, name: "Acme", role: MEMBER_ROLE },
    ];
  });

  const invite = async (fields: Record<string, string>) =>
    await inviteRoute(
      createAPIContext(harness.portal, {
        path: "/api/members/invite",
        body: {
          organization: ACME,
          email: "cy@acme.com",
          role: MEMBER_ROLE,
          ...fields,
        },
      }),
    );

  describe("POST /api/members/invite", () => {
    it("should send an invitation whose link lands on the sign-up page", async () => {
      harness.signIn("user_admin");
      const res = await invite({ email: " cy@acme.com " });
      expect(res.status).toBe(204);
      expect(harness.directory.invitations).toEqual([
        {
          clerkOrgID: ACME,
          email: "cy@acme.com",
          role: MEMBER_ROLE,
          inviterUserID: "user_admin",
          redirectURL: "https://portal.synnaxlabs.com/sign-up",
        },
      ]);
    });

    it("should refuse a visitor who is not logged in", async () => {
      const res = await invite({});
      expect(res.status).toBe(401);
      expect(await body(res)).toEqual({ error: "Log in first" });
      expect(harness.directory.invitations).toHaveLength(0);
    });

    it("should refuse a member who is not an admin", async () => {
      harness.signIn("user_member");
      const res = await invite({});
      expect(res.status).toBe(403);
      expect(await body(res)).toEqual({ error: "Only a team admin can invite" });
      expect(harness.directory.invitations).toHaveLength(0);
    });

    it("should refuse an admin of another team", async () => {
      harness.signIn("user_admin");
      const res = await invite({ organization: "org_other" });
      expect(res.status).toBe(403);
      expect(await body(res)).toEqual({ error: "Only a team admin can invite" });
      expect(harness.directory.invitations).toHaveLength(0);
    });

    it("should refuse an empty email", async () => {
      harness.signIn("user_admin");
      const res = await invite({ email: "  " });
      expect(res.status).toBe(400);
      expect(await body(res)).toEqual({ error: "Enter an email address" });
    });

    it("should refuse a role Clerk does not define", async () => {
      harness.signIn("user_admin");
      const res = await invite({ role: "org:owner" });
      expect(res.status).toBe(400);
      expect(await body(res)).toEqual({ error: "Choose a role" });
    });
  });
});
