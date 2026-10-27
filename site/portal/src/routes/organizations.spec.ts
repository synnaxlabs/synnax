// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIRoute } from "astro";
import { beforeAll, beforeEach, describe, expect, it } from "vitest";

import { POST as revokeInvitationRoute } from "@/pages/api/organizations/[key]/invitations/[id]/revoke";
import { POST as inviteRoute } from "@/pages/api/organizations/[key]/invite";
import { POST as removeRoute } from "@/pages/api/organizations/[key]/members/[user]/remove";
import { POST as roleRoute } from "@/pages/api/organizations/[key]/members/[user]/role";
import { event, type Organization } from "@/server/db/schema";
import { type Memory, openMemory } from "@/server/db/testutil";
import { ADMIN_ROLE, MEMBER_ROLE, type Team } from "@/server/directory";
import { NOW } from "@/server/license/testutil";
import {
  body,
  type ContextArgs,
  createAPIContext,
  createHarness,
  createOrganization,
  type Harness,
  STAFF_ORG_ID,
} from "@/testutil";

const ACME = "org_acme";
const ADMIN = "user_admin";
const MEMBER = "user_member";
const STAFF = "user_staff";
const OUTSIDER = "user_outsider";

const ACME_ADMIN: Team = { clerkOrgID: ACME, name: "Acme", role: ADMIN_ROLE };
const ACME_MEMBER: Team = { ...ACME_ADMIN, role: MEMBER_ROLE };
const STAFF_TEAM: Team = {
  clerkOrgID: STAFF_ORG_ID,
  name: "Synnax Labs",
  role: ADMIN_ROLE,
};

describe("organization routes", () => {
  let store: Memory;
  let h: Harness;
  let acme: Organization;

  beforeAll(async () => {
    store = await openMemory();
  });

  beforeEach(async () => {
    await store.clear();
    h = createHarness(store);
    h.directory.organizations.push({ clerkOrgID: ACME, name: "Acme" });
    addPerson(ADMIN, [ACME_ADMIN]);
    addPerson(MEMBER, [ACME_MEMBER]);
    addPerson(STAFF, [STAFF_TEAM]);
    addPerson(OUTSIDER, [{ clerkOrgID: "org_other", name: "Other", role: ADMIN_ROLE }]);
    acme = await createOrganization(store, { kind: "team", clerkOrgID: ACME });
  });

  const addPerson = (userID: string, teams: Team[] = []): void => {
    h.directory.people[userID] = { email: `${userID}@acme.com`, name: userID };
    h.directory.members[userID] = teams;
  };

  const call = async (route: APIRoute, args: ContextArgs = {}): Promise<Response> =>
    await route(createAPIContext(h.portal, args));

  const expectError = async (
    res: Response,
    status: number,
    error: string,
  ): Promise<void> => {
    expect(res.status).toBe(status);
    expect(await body(res)).toEqual({ error });
  };

  const events = async () => await store.query.select().from(event);

  const roleOf = (userID: string): string | undefined =>
    h.directory.members[userID]?.find((t) => t.clerkOrgID === ACME)?.role;

  describe("POST /api/organizations/[key]/invite", () => {
    const invite = async (fields: Record<string, string> = {}, key = acme.key) =>
      await call(inviteRoute, {
        params: { key },
        path: `/api/organizations/${key}/invite`,
        body: { email: "cy@acme.com", role: MEMBER_ROLE, ...fields },
      });

    it("should send an invitation whose link lands on the sign-up page", async () => {
      h.signIn(ADMIN);
      const res = await invite({ email: " cy@acme.com " });
      expect(res.status).toBe(204);
      expect(h.directory.invitations).toEqual([
        {
          clerkOrgID: ACME,
          email: "cy@acme.com",
          role: MEMBER_ROLE,
          inviterUserID: ADMIN,
          redirectURL: "https://portal.synnaxlabs.com/sign-up",
        },
      ]);
    });

    it("should record the invitation", async () => {
      h.signIn(ADMIN);
      await invite();
      expect(await events()).toEqual([
        expect.objectContaining({
          kind: "invite",
          at: NOW,
          actor: ADMIN,
          organization: acme.key,
          detail: { email: "cy@acme.com", role: MEMBER_ROLE },
        }),
      ]);
    });

    it("should let staff invite to a team they are not on", async () => {
      h.signIn(STAFF);
      expect((await invite()).status).toBe(204);
      expect(h.directory.invitations[0]).toMatchObject({ inviterUserID: STAFF });
    });

    it("should refuse a visitor who is not logged in", async () => {
      await expectError(await invite(), 401, "Log in first");
      expect(h.directory.invitations).toHaveLength(0);
    });

    it("should refuse a member who is not an admin", async () => {
      h.signIn(MEMBER);
      await expectError(await invite(), 403, "Only a team admin can change members");
      expect(h.directory.invitations).toHaveLength(0);
    });

    it("should answer an admin of another team as if the team did not exist", async () => {
      h.signIn(OUTSIDER);
      await expectError(await invite(), 404, "Organization not found");
      expect(h.directory.invitations).toHaveLength(0);
    });

    it("should answer 404 for a personal organization", async () => {
      const personal = await createOrganization(store, {
        kind: "personal",
        ownerUserID: ADMIN,
      });
      h.signIn(STAFF);
      await expectError(await invite({}, personal.key), 404, "Organization not found");
    });

    it("should refuse an empty email", async () => {
      h.signIn(ADMIN);
      await expectError(await invite({ email: "  " }), 400, "Enter an email address");
      expect(await events()).toHaveLength(0);
    });

    it("should refuse a role Clerk does not define", async () => {
      h.signIn(ADMIN);
      await expectError(await invite({ role: "org:owner" }), 400, "Choose a role");
    });
  });

  describe("POST /api/organizations/[key]/members/[user]/role", () => {
    const setRole = async (user: string, role: string) =>
      await call(roleRoute, { params: { key: acme.key, user }, body: { role } });

    it("should change the member's role and record it", async () => {
      h.signIn(ADMIN);
      expect((await setRole(MEMBER, ADMIN_ROLE)).status).toBe(204);
      expect(roleOf(MEMBER)).toBe(ADMIN_ROLE);
      expect(await events()).toEqual([
        expect.objectContaining({
          kind: "change_role",
          actor: ADMIN,
          organization: acme.key,
          detail: {
            member: MEMBER,
            email: "user_member@acme.com",
            from: MEMBER_ROLE,
            to: ADMIN_ROLE,
          },
        }),
      ]);
    });

    it("should let staff change a role", async () => {
      h.signIn(STAFF);
      expect((await setRole(MEMBER, ADMIN_ROLE)).status).toBe(204);
      expect(roleOf(MEMBER)).toBe(ADMIN_ROLE);
    });

    it("should let an admin step down once another admin exists", async () => {
      addPerson("user_second", [ACME_ADMIN]);
      h.signIn(ADMIN);
      expect((await setRole(ADMIN, MEMBER_ROLE)).status).toBe(204);
      expect(roleOf(ADMIN)).toBe(MEMBER_ROLE);
    });

    it("should refuse to demote the last admin", async () => {
      h.signIn(STAFF);
      await expectError(
        await setRole(ADMIN, MEMBER_ROLE),
        400,
        "A team needs an admin. Make another member an admin first.",
      );
      expect(roleOf(ADMIN)).toBe(ADMIN_ROLE);
      expect(await events()).toHaveLength(0);
    });

    it("should record nothing when the member already has the role", async () => {
      h.signIn(ADMIN);
      expect((await setRole(MEMBER, MEMBER_ROLE)).status).toBe(204);
      expect(await events()).toHaveLength(0);
    });

    it("should answer 404 for a user who is not on the team", async () => {
      h.signIn(ADMIN);
      await expectError(await setRole(OUTSIDER, ADMIN_ROLE), 404, "Member not found");
    });

    it("should refuse a member who is not an admin", async () => {
      h.signIn(MEMBER);
      await expectError(
        await setRole(MEMBER, ADMIN_ROLE),
        403,
        "Only a team admin can change members",
      );
      expect(roleOf(MEMBER)).toBe(MEMBER_ROLE);
    });

    it("should refuse a role Clerk does not define", async () => {
      h.signIn(ADMIN);
      await expectError(await setRole(MEMBER, "org:owner"), 400, "Choose a role");
    });
  });

  describe("POST /api/organizations/[key]/members/[user]/remove", () => {
    const remove = async (user: string) =>
      await call(removeRoute, { params: { key: acme.key, user } });

    it("should take the member off the team and record it", async () => {
      h.signIn(ADMIN);
      expect((await remove(MEMBER)).status).toBe(204);
      expect(roleOf(MEMBER)).toBeUndefined();
      expect(await events()).toEqual([
        expect.objectContaining({
          kind: "remove_member",
          actor: ADMIN,
          organization: acme.key,
          detail: { member: MEMBER, email: "user_member@acme.com" },
        }),
      ]);
    });

    it("should let staff remove a member", async () => {
      h.signIn(STAFF);
      expect((await remove(MEMBER)).status).toBe(204);
      expect(roleOf(MEMBER)).toBeUndefined();
    });

    it("should refuse to remove the last admin", async () => {
      h.signIn(STAFF);
      await expectError(
        await remove(ADMIN),
        400,
        "A team needs an admin. Make another member an admin first.",
      );
      expect(roleOf(ADMIN)).toBe(ADMIN_ROLE);
    });

    it("should refuse a member who is not an admin", async () => {
      h.signIn(MEMBER);
      await expectError(
        await remove(ADMIN),
        403,
        "Only a team admin can change members",
      );
    });

    it("should answer 404 for a user who is not on the team", async () => {
      h.signIn(ADMIN);
      await expectError(await remove(OUTSIDER), 404, "Member not found");
    });
  });

  describe("POST /api/organizations/[key]/invitations/[id]/revoke", () => {
    beforeEach(() => {
      h.directory.invitations.push({
        clerkOrgID: ACME,
        email: "cy@acme.com",
        role: MEMBER_ROLE,
        inviterUserID: ADMIN,
        redirectURL: "https://portal.synnaxlabs.com/sign-up",
      });
    });

    const revoke = async (id: string) =>
      await call(revokeInvitationRoute, { params: { key: acme.key, id } });

    it("should cancel the invitation and record it", async () => {
      h.signIn(ADMIN);
      expect((await revoke("inv_0")).status).toBe(204);
      expect(await h.directory.outstanding(ACME)).toEqual([]);
      expect(await events()).toEqual([
        expect.objectContaining({
          kind: "revoke_invitation",
          actor: ADMIN,
          organization: acme.key,
          detail: { email: "cy@acme.com", role: MEMBER_ROLE },
        }),
      ]);
    });

    it("should let staff cancel an invitation", async () => {
      h.signIn(STAFF);
      expect((await revoke("inv_0")).status).toBe(204);
    });

    it("should answer 404 for an invitation the team does not have", async () => {
      h.signIn(ADMIN);
      await expectError(await revoke("inv_9"), 404, "Invitation not found");
      expect(await events()).toHaveLength(0);
    });

    it("should refuse a member who is not an admin", async () => {
      h.signIn(MEMBER);
      await expectError(
        await revoke("inv_0"),
        403,
        "Only a team admin can change members",
      );
    });
  });
});
