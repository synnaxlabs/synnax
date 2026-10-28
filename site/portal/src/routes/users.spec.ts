// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIRoute } from "astro";
import { eq } from "drizzle-orm";
import { beforeAll, beforeEach, describe, expect, it } from "vitest";

import { POST as deleteRoute } from "@/pages/api/users/[id]/delete";
import {
  activation,
  event,
  invitation,
  license,
  type Organization,
  organization,
  referral,
} from "@/server/db/schema";
import { type Memory, openMemory } from "@/server/db/testutil";
import { ADMIN_ROLE, type Team } from "@/server/directory";
import { HASH_A, HASH_B, NOW } from "@/server/license/testutil";
import {
  body,
  type ContextArgs,
  createActivation,
  createAPIContext,
  createHarness,
  createInvitation,
  createLicense,
  createOrganization,
  createReferral,
  type Harness,
  STAFF_ORG_ID,
} from "@/testutil";

const ACME = "org_acme";
const ADA = "user_ada";
const BOB = "user_bob";
const STAFF = "user_staff";

const ACME_ADMIN: Team = { clerkOrgID: ACME, name: "Acme", role: ADMIN_ROLE };
const STAFF_TEAM: Team = {
  clerkOrgID: STAFF_ORG_ID,
  name: "Synnax Labs",
  role: ADMIN_ROLE,
};

describe("user routes", () => {
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
    addPerson(ADA, "Ada", [ACME_ADMIN]);
    addPerson(BOB, "Bob", [ACME_ADMIN]);
    addPerson(STAFF, "Sam", [STAFF_TEAM]);
    acme = await createOrganization(store, { kind: "team", clerkOrgID: ACME });
  });

  const addPerson = (userID: string, name: string, teams: Team[]): void => {
    h.directory.people[userID] = { email: `${userID}@example.com`, name };
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

  /** createDesktop gives a user a personal organization linked to one machine. */
  const createDesktop = async (userID: string): Promise<Organization> => {
    const personal = await createOrganization(store, {
      kind: "personal",
      ownerUserID: userID,
    });
    const lic = await createLicense(store, {
      organization: personal.key,
      edition: "desktop",
    });
    const act = await createActivation(store, {
      license: lic.key,
      fingerprint: [HASH_A],
    });
    await store.query.insert(event).values({
      at: NOW,
      kind: "link",
      actor: userID,
      organization: personal.key,
      license: lic.key,
      activation: act.key,
    });
    return personal;
  };

  describe("POST /api/users/[id]/delete", () => {
    const remove = async (id = ADA) =>
      await call(deleteRoute, { params: { id }, path: `/api/users/${id}/delete` });

    const owned = async (org: Organization) => ({
      organizations: await store.query
        .select()
        .from(organization)
        .where(eq(organization.key, org.key)),
      licenses: await store.query
        .select()
        .from(license)
        .where(eq(license.organization, org.key)),
      events: await store.query
        .select()
        .from(event)
        .where(eq(event.organization, org.key)),
    });

    it("should reject a visitor who is not logged in", async () => {
      await expectError(await remove(), 401, "Log in first");
    });

    it("should reject a member who is not staff", async () => {
      h.signIn(BOB);
      await expectError(await remove(), 403, "Staff only");
      expect(await h.directory.account(ADA)).not.toBeNull();
    });

    it("should answer 404 for a user Clerk does not know", async () => {
      h.signIn(STAFF);
      await expectError(await remove("user_gone"), 404, "User not found");
    });

    it("should delete the account and take it off its teams", async () => {
      h.signIn(STAFF);
      expect((await remove()).status).toBe(204);
      expect(await h.directory.account(ADA)).toBeNull();
      expect((await h.directory.roster(ACME)).map((m) => m.userID)).toEqual([BOB]);
    });

    it("should delete the personal organization with its machines", async () => {
      const personal = await createDesktop(ADA);
      h.signIn(STAFF);
      await remove();
      expect(await owned(personal)).toEqual({
        organizations: [],
        licenses: [],
        events: [],
      });
      expect(await store.query.select().from(activation)).toEqual([]);
    });

    it("should delete the referral the user claimed", async () => {
      await createReferral(store, { key: "ada-link-001", userID: ADA, claimedAt: NOW });
      await createReferral(store, { key: "bob-link-001", userID: BOB, claimedAt: NOW });
      h.signIn(STAFF);
      await remove();
      expect((await store.query.select().from(referral)).map((r) => r.userID)).toEqual([
        BOB,
      ]);
    });

    it("should delete the invitations sent to the user's address", async () => {
      await createInvitation(store, {
        organization: acme.key,
        email: `${ADA}@example.com`,
      });
      await createInvitation(store, {
        organization: acme.key,
        email: "cy@example.com",
      });
      h.signIn(STAFF);
      await remove();
      expect((await store.query.select().from(invitation)).map((i) => i.email)).toEqual(
        ["cy@example.com"],
      );
    });

    it("should keep the user's events on a team and the records of others", async () => {
      await createDesktop(ADA);
      const bob = await createDesktop(BOB);
      const lic = await createLicense(store, { organization: acme.key });
      await createActivation(store, { license: lic.key, fingerprint: [HASH_B] });
      await store.query.insert(event).values({
        at: NOW,
        kind: "activate",
        actor: ADA,
        organization: acme.key,
        license: lic.key,
      });
      h.signIn(STAFF);
      await remove();
      expect(await owned(acme)).toMatchObject({
        organizations: [expect.objectContaining({ key: acme.key })],
        licenses: [expect.objectContaining({ key: lic.key })],
        events: [expect.objectContaining({ actor: ADA })],
      });
      expect((await owned(bob)).licenses).toHaveLength(1);
      expect(await store.query.select().from(activation)).toHaveLength(2);
    });

    it("should delete a user who never visited the portal", async () => {
      h.signIn(STAFF);
      expect((await remove()).status).toBe(204);
      expect(await store.query.select().from(organization)).toEqual([acme]);
    });

    it("should refuse a member of the staff team", async () => {
      addPerson(ADA, "Ada", [STAFF_TEAM]);
      h.signIn(STAFF);
      await expectError(
        await remove(),
        400,
        "Ada is on the staff team and cannot be deleted",
      );
      expect(await h.directory.account(ADA)).not.toBeNull();
    });

    it("should delete nothing when Clerk fails", async () => {
      const personal = await createDesktop(ADA);
      await createInvitation(store, {
        organization: acme.key,
        email: `${ADA}@example.com`,
      });
      h.directory.deleteAccount = async () => {
        throw new Error("clerk: down");
      };
      h.signIn(STAFF);
      await expectError(await remove(), 500, "Internal error");
      expect((await owned(personal)).licenses).toHaveLength(1);
      expect(await store.query.select().from(invitation)).toHaveLength(1);
      expect(await h.directory.account(ADA)).not.toBeNull();
    });
  });
});
