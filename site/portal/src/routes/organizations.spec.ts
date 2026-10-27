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
import { POST as renameRoute } from "@/pages/api/organizations/[key]/name";
import { POST as registerRoute } from "@/pages/api/organizations/index";
import { event, invitation, type Organization, organization } from "@/server/db/schema";
import { failingEvents, type Memory, openMemory } from "@/server/db/testutil";
import { ADMIN_ROLE, MEMBER_ROLE, type Team } from "@/server/directory";
import { OPEN_MS } from "@/server/invitation";
import { NOW } from "@/server/license/testutil";
import {
  body,
  type ContextArgs,
  createAPIContext,
  createHarness,
  createInvitation,
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

  const invitations = async () => await store.query.select().from(invitation);

  const failMail = (): void => {
    h.mail.send = async () => {
      throw new Error("mail: down");
    };
  };

  const roleOf = (userID: string): string | undefined =>
    h.directory.members[userID]?.find((t) => t.clerkOrgID === ACME)?.role;

  describe("POST /api/organizations/[key]/invite", () => {
    const invite = async (fields: Record<string, string> = {}, key = acme.key) =>
      await call(inviteRoute, {
        params: { key },
        path: `/api/organizations/${key}/invite`,
        body: { email: "cy@acme.com", role: MEMBER_ROLE, ...fields },
      });

    it("should save and email an invitation open for 30 days", async () => {
      h.signIn(ADMIN);
      expect((await invite({ email: " Cy@Acme.com " })).status).toBe(204);
      const [row] = await invitations();
      expect(row).toMatchObject({
        organization: acme.key,
        email: "cy@acme.com",
        role: MEMBER_ROLE,
        sentAt: NOW,
        expiresAt: new Date(NOW.getTime() + OPEN_MS),
      });
      expect(h.mail.sent).toEqual([
        {
          to: ["cy@acme.com"],
          subject: "Join Acme on Synnax",
          heading: "Join Acme on Synnax",
          paragraphs: [
            "user_admin invited you to join Acme in the Synnax portal.",
            "Log in or create an account with cy@acme.com to accept.",
          ],
          facts: [
            { label: "Team", value: "Acme" },
            { label: "Role", value: "Member" },
            { label: "Expires", value: "2026-10-17" },
          ],
          action: { label: "Accept invitation", href: `/invitations/${row.key}` },
          reason: "If you did not expect this invitation, you can ignore it.",
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

    it("should save and send nothing when the event cannot be recorded", async () => {
      h.signIn(ADMIN);
      const res = await failingEvents(store, "invite", invite);
      expect(res.status).toBe(500);
      expect(await invitations()).toEqual([]);
      expect(h.mail.sent).toEqual([]);
    });

    it("should save and record nothing when the email fails", async () => {
      h.signIn(ADMIN);
      failMail();
      expect((await invite()).status).toBe(500);
      expect(await invitations()).toEqual([]);
      expect(await events()).toEqual([]);
    });

    it("should send an open invitation again with a new role and expiry", async () => {
      const sent = await createInvitation(store, {
        organization: acme.key,
        email: "cy@acme.com",
        sentAt: new Date("2026-10-01T00:00:00Z"),
        expiresAt: new Date("2026-10-02T00:00:00Z"),
      });
      h.signIn(ADMIN);
      expect((await invite({ role: ADMIN_ROLE })).status).toBe(204);
      expect(await invitations()).toEqual([
        {
          ...sent,
          role: ADMIN_ROLE,
          sentAt: NOW,
          expiresAt: new Date(NOW.getTime() + OPEN_MS),
        },
      ]);
      expect(h.mail.sent[0].action.href).toBe(`/invitations/${sent.key}`);
    });

    it("should let staff invite to a team they are not on", async () => {
      h.signIn(STAFF);
      expect((await invite()).status).toBe(204);
      expect(h.mail.sent[0].paragraphs[0]).toBe(
        "Synnax Labs invited you to join Acme in the Synnax portal.",
      );
    });

    it("should refuse a visitor who is not logged in", async () => {
      await expectError(await invite(), 401, "Log in first");
      expect(await invitations()).toEqual([]);
    });

    it("should refuse a member who is not an admin", async () => {
      h.signIn(MEMBER);
      await expectError(await invite(), 403, "Only a team admin can change members");
      expect(await invitations()).toEqual([]);
    });

    it("should answer an admin of another team as if it did not exist", async () => {
      h.signIn(OUTSIDER);
      await expectError(await invite(), 404, "Organization not found");
      expect(await invitations()).toEqual([]);
    });

    it("should answer 404 for a personal organization", async () => {
      const personal = await createOrganization(store, {
        kind: "personal",
        ownerUserID: ADMIN,
      });
      h.signIn(STAFF);
      await expectError(await invite({}, personal.key), 404, "Organization not found");
    });

    it("should refuse an empty or invalid email", async () => {
      h.signIn(ADMIN);
      for (const email of ["  ", "cy"])
        await expectError(await invite({ email }), 400, "Enter an email address");
      expect(await events()).toEqual([]);
    });

    it("should refuse someone already on the team, in any case", async () => {
      h.signIn(ADMIN);
      await expectError(
        await invite({ email: "USER_MEMBER@acme.com" }),
        400,
        "user_member@acme.com is already on Acme",
      );
      expect(await invitations()).toEqual([]);
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

    it("should keep the role when the change cannot be recorded", async () => {
      h.signIn(ADMIN);
      const res = await failingEvents(
        store,
        "change_role",
        async () => await setRole(MEMBER, ADMIN_ROLE),
      );
      expect(res.status).toBe(500);
      expect(roleOf(MEMBER)).toBe(MEMBER_ROLE);
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

    it("should keep the member when the removal cannot be recorded", async () => {
      h.signIn(ADMIN);
      const res = await failingEvents(
        store,
        "remove_member",
        async () => await remove(MEMBER),
      );
      expect(res.status).toBe(500);
      expect(roleOf(MEMBER)).toBe(MEMBER_ROLE);
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
    let sent: Awaited<ReturnType<typeof createInvitation>>;

    beforeEach(async () => {
      sent = await createInvitation(store, {
        organization: acme.key,
        email: "cy@acme.com",
      });
    });

    const revoke = async (id: string) =>
      await call(revokeInvitationRoute, { params: { key: acme.key, id } });

    it("should delete the invitation and record it", async () => {
      h.signIn(ADMIN);
      expect((await revoke(sent.key)).status).toBe(204);
      expect(await invitations()).toEqual([]);
      expect(await events()).toEqual([
        expect.objectContaining({
          kind: "revoke_invitation",
          actor: ADMIN,
          organization: acme.key,
          detail: { email: "cy@acme.com", role: MEMBER_ROLE },
        }),
      ]);
    });

    it("should keep an invitation whose revocation cannot be recorded", async () => {
      h.signIn(ADMIN);
      const res = await failingEvents(
        store,
        "revoke_invitation",
        async () => await revoke(sent.key),
      );
      expect(res.status).toBe(500);
      expect(await invitations()).toEqual([sent]);
    });

    it("should let staff revoke an invitation", async () => {
      h.signIn(STAFF);
      expect((await revoke(sent.key)).status).toBe(204);
    });

    it("should answer 404 for an invitation the team does not have", async () => {
      const other = await createOrganization(store, {
        kind: "team",
        name: "Other",
        clerkOrgID: "org_other",
      });
      const elsewhere = await createInvitation(store, {
        organization: other.key,
        email: "cy@other.com",
      });
      h.signIn(ADMIN);
      for (const id of [elsewhere.key, crypto.randomUUID(), "inv_9"])
        await expectError(await revoke(id), 404, "Invitation not found");
      expect(await invitations()).toHaveLength(2);
      expect(await events()).toEqual([]);
    });

    it("should refuse a member who is not an admin", async () => {
      h.signIn(MEMBER);
      await expectError(
        await revoke(sent.key),
        403,
        "Only a team admin can change members",
      );
    });
  });

  describe("POST /api/organizations", () => {
    const register = async (fields: Record<string, string> = {}) =>
      await call(registerRoute, {
        path: "/api/organizations",
        body: { name: "Globex", email: "hank@globex.com", ...fields },
      });

    const globex = async () =>
      (await store.query.select().from(organization)).find((o) => o.name === "Globex");

    it("should register the team in Clerk and the portal", async () => {
      h.signIn(STAFF);
      const res = await register({ name: " Globex " });
      expect(res.status).toBe(200);
      const row = await globex();
      expect(await body(res)).toEqual({ key: row?.key });
      expect(row).toMatchObject({ kind: "team", clerkOrgID: "org_1" });
      expect(h.directory.organizations[0]).toEqual({
        clerkOrgID: "org_1",
        name: "Globex",
      });
    });

    it("should leave staff off the team", async () => {
      h.signIn(STAFF);
      await register();
      expect(await h.directory.roster("org_1")).toEqual([]);
    });

    it("should invite the first admin from Synnax Labs", async () => {
      h.signIn(STAFF);
      await register({ email: " hank@globex.com " });
      expect(await invitations()).toEqual([
        expect.objectContaining({
          organization: (await globex())?.key,
          email: "hank@globex.com",
          role: ADMIN_ROLE,
        }),
      ]);
      expect(h.mail.sent).toEqual([
        expect.objectContaining({
          to: ["hank@globex.com"],
          paragraphs: expect.arrayContaining([
            "Synnax Labs invited you to join Globex in the Synnax portal.",
          ]),
        }),
      ]);
    });

    it("should record the registration and the invitation", async () => {
      h.signIn(STAFF);
      await register();
      const org = (await globex())?.key;
      expect(await events()).toEqual([
        expect.objectContaining({
          kind: "register",
          actor: STAFF,
          organization: org,
          detail: { name: "Globex" },
        }),
        expect.objectContaining({
          kind: "invite",
          actor: STAFF,
          organization: org,
          detail: { email: "hank@globex.com", role: ADMIN_ROLE },
        }),
      ]);
    });

    it("should keep the team when the invitation fails", async () => {
      h.signIn(STAFF);
      failMail();
      await expectError(
        await register(),
        502,
        "Globex is registered, but its invitation failed. Invite the admin from its " +
          "Members tab.",
      );
      expect(await globex()).toBeDefined();
      expect((await events()).map((e) => e.kind)).toEqual(["register"]);
      expect(await invitations()).toEqual([]);
    });

    it("should refuse a name another team has, in any case", async () => {
      h.signIn(STAFF);
      await expectError(
        await register({ name: "ACME" }),
        400,
        "An organization named ACME exists",
      );
      expect(h.directory.organizations).toHaveLength(1);
    });

    it("should refuse an empty name", async () => {
      h.signIn(STAFF);
      await expectError(await register({ name: " " }), 400, "Name the organization");
    });

    it("should refuse a name that is too long", async () => {
      h.signIn(STAFF);
      await expectError(
        await register({ name: "a".repeat(65) }),
        400,
        "The name must be at most 64 characters",
      );
    });

    it("should refuse an invalid email before creating the team", async () => {
      h.signIn(STAFF);
      await expectError(
        await register({ email: "hank" }),
        400,
        "Enter the admin's email address",
      );
      expect(h.directory.organizations).toHaveLength(1);
      expect(await events()).toHaveLength(0);
    });

    it("should refuse a team admin who is not staff", async () => {
      h.signIn(ADMIN);
      expect((await register()).status).toBe(403);
      expect(h.directory.organizations).toHaveLength(1);
    });

    it("should refuse a visitor who is not logged in", async () => {
      await expectError(await register(), 401, "Log in first");
    });
  });

  describe("POST /api/organizations/[key]/name", () => {
    const rename = async (name: string, key = acme.key) =>
      await call(renameRoute, {
        params: { key },
        path: `/api/organizations/${key}/name`,
        body: { name },
      });

    it("should rename the team in Clerk and the portal", async () => {
      h.signIn(STAFF);
      expect((await rename("Acme Rockets")).status).toBe(204);
      expect(await h.directory.team(ACME)).toEqual({
        clerkOrgID: ACME,
        name: "Acme Rockets",
      });
      const [row] = await store.query.select().from(organization);
      expect(row).toMatchObject({ key: acme.key, name: "Acme Rockets" });
    });

    it("should record the old and new names", async () => {
      h.signIn(STAFF);
      await rename("Acme Rockets");
      expect(await events()).toEqual([
        expect.objectContaining({
          kind: "rename_organization",
          actor: STAFF,
          organization: acme.key,
          detail: { name: "Acme Rockets", from: "Acme" },
        }),
      ]);
    });

    it("should keep the name when the rename cannot be recorded", async () => {
      h.signIn(STAFF);
      const res = await failingEvents(
        store,
        "rename_organization",
        async () => await rename("Acme Rockets"),
      );
      expect(res.status).toBe(500);
      expect(await h.directory.team(ACME)).toEqual({ clerkOrgID: ACME, name: "Acme" });
      const [row] = await store.query.select().from(organization);
      expect(row.name).toBe("Acme");
    });

    it("should change nothing for the same name", async () => {
      h.signIn(STAFF);
      expect((await rename(" Acme ")).status).toBe(204);
      expect(await events()).toHaveLength(0);
    });

    it("should let only the case of the name change", async () => {
      h.signIn(STAFF);
      expect((await rename("ACME")).status).toBe(204);
      expect((await h.directory.team(ACME))?.name).toBe("ACME");
    });

    it("should refuse a name another team has", async () => {
      h.directory.organizations.push({ clerkOrgID: "org_globex", name: "Globex" });
      h.signIn(STAFF);
      await expectError(
        await rename("globex"),
        400,
        "An organization named globex exists",
      );
    });

    it("should refuse a team admin who is not staff", async () => {
      h.signIn(ADMIN);
      expect((await rename("Acme Rockets")).status).toBe(403);
      expect((await h.directory.team(ACME))?.name).toBe("Acme");
    });
  });
});
