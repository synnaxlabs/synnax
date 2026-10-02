// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeAll, beforeEach, describe, expect, it } from "vitest";

import { type Organization, organization } from "@/server/db/schema";
import { type Memory, openMemory } from "@/server/db/testutil";
import { memory } from "@/server/directory";
import {
  adoptTeam,
  ensurePersonal,
  isMember,
  listAll,
  listForMember,
  mirrorTeam,
  organizationsFor,
  pick,
  retrieve,
} from "@/server/organization";
import { type Session } from "@/server/session";
import { createOrganization } from "@/testutil";

const SESSION: Session = {
  userID: "user_a",
  email: "ada@example.com",
  name: "Ada Lovelace",
  teams: [
    { clerkOrgID: "org_acme", name: "Acme", role: "org:member" },
    { clerkOrgID: "org_globex", name: "Globex", role: "org:admin" },
  ],
  clerkOrgIDs: ["org_acme", "org_globex"],
  staff: false,
};

const organizationOf = (key: string, kind: Organization["kind"]): Organization => ({
  key,
  kind,
  name: key,
  clerkOrgID: kind === "team" ? `org_${key}` : null,
  ownerUserID: kind === "personal" ? "user_a" : null,
  createdAt: new Date(0),
});

describe("organization", () => {
  let store: Memory;
  beforeAll(async () => {
    store = await openMemory();
  });
  beforeEach(async () => await store.clear());

  describe("ensurePersonal", () => {
    it("should create the user's personal organization on first sight", async () => {
      const org = await ensurePersonal(store, { userID: "user_a", name: "Ada" });
      expect(org).toMatchObject({
        kind: "personal",
        name: "Ada",
        ownerUserID: "user_a",
        clerkOrgID: null,
      });
      expect(await store.query.select().from(organization)).toHaveLength(1);
    });

    it("should return the existing organization without renaming it", async () => {
      const first = await ensurePersonal(store, { userID: "user_a", name: "Ada" });
      const second = await ensurePersonal(store, { userID: "user_a", name: "Other" });
      expect(second).toEqual(first);
      expect(await store.query.select().from(organization)).toHaveLength(1);
    });

    it("should return the one organization when two requests race", async () => {
      const [a, b] = await Promise.all([
        ensurePersonal(store, { userID: "user_a", name: "Ada" }),
        ensurePersonal(store, { userID: "user_a", name: "Ada" }),
      ]);
      expect(a.key).toBe(b.key);
      expect(await store.query.select().from(organization)).toHaveLength(1);
    });
  });

  describe("mirrorTeam", () => {
    it("should record a Clerk organization as a team", async () => {
      const org = await mirrorTeam(store, { clerkOrgID: "org_acme", name: "Acme" });
      expect(org).toMatchObject({
        kind: "team",
        name: "Acme",
        clerkOrgID: "org_acme",
        ownerUserID: null,
      });
    });

    it("should rename the team when Clerk renames it, keeping its key", async () => {
      const first = await mirrorTeam(store, { clerkOrgID: "org_acme", name: "Acme" });
      const second = await mirrorTeam(store, {
        clerkOrgID: "org_acme",
        name: "Acme Corp",
      });
      expect(second.key).toBe(first.key);
      expect(second.name).toBe("Acme Corp");
      expect(await store.query.select().from(organization)).toHaveLength(1);
    });
  });

  describe("adoptTeam", () => {
    it("should mirror the Clerk organization the directory names", async () => {
      const directory = memory({
        organizations: [{ clerkOrgID: "org_acme", name: "Acme" }],
      });
      const org = await adoptTeam(store, directory, "org_acme");
      expect(org).toMatchObject({ kind: "team", name: "Acme", clerkOrgID: "org_acme" });
      expect(await retrieve(store, org.key)).toEqual(org);
    });

    it("should throw a 400 when Clerk does not know the organization", async () => {
      await expect(adoptTeam(store, memory(), "org_missing")).rejects.toMatchObject({
        status: 400,
        message: "Clerk has no organization org_missing",
      });
      expect(await store.query.select().from(organization)).toHaveLength(0);
    });
  });

  describe("listForMember", () => {
    it("should return the personal organization before the teams", async () => {
      const acme = await createOrganization(store, {
        kind: "team",
        name: "Acme",
        clerkOrgID: "org_acme",
      });
      const personal = await createOrganization(store, {
        kind: "personal",
        ownerUserID: "user_a",
      });
      const orgs = await listForMember(store, {
        userID: "user_a",
        clerkOrgIDs: ["org_acme"],
      });
      expect(orgs.map((o) => o.key)).toEqual([personal.key, acme.key]);
    });

    it("should leave out organizations the user is not in", async () => {
      await createOrganization(store, { kind: "team", clerkOrgID: "org_other" });
      await createOrganization(store, { kind: "personal", ownerUserID: "user_b" });
      const personal = await createOrganization(store, {
        kind: "personal",
        ownerUserID: "user_a",
      });
      const orgs = await listForMember(store, { userID: "user_a", clerkOrgIDs: [] });
      expect(orgs.map((o) => o.key)).toEqual([personal.key]);
    });

    it("should return nothing for a user with no organizations", async () => {
      expect(
        await listForMember(store, { userID: "user_a", clerkOrgIDs: ["org_acme"] }),
      ).toEqual([]);
    });
  });

  describe("organizationsFor", () => {
    it("should create the personal organization and mirror every team", async () => {
      const orgs = await organizationsFor(store, SESSION);
      expect(orgs.map((o) => [o.kind, o.name])).toEqual([
        ["personal", "Ada Lovelace"],
        ["team", "Acme"],
        ["team", "Globex"],
      ]);
      expect(await store.query.select().from(organization)).toHaveLength(3);
    });

    it("should not duplicate organizations on a second call", async () => {
      const first = await organizationsFor(store, SESSION);
      const second = await organizationsFor(store, SESSION);
      expect(second.map((o) => o.key).sort()).toEqual(first.map((o) => o.key).sort());
      expect(await store.query.select().from(organization)).toHaveLength(3);
    });
  });

  describe("pick", () => {
    const personal = organizationOf("personal", "personal");
    const acme = organizationOf("acme", "team");
    const globex = organizationOf("globex", "team");
    const all = [personal, acme, globex];

    it("should select the requested organization", () => {
      expect(pick(all, "globex", "acme")).toBe(globex);
    });

    it("should return null for a requested organization the user is not in", () => {
      expect(pick(all, "other", "acme")).toBeNull();
    });

    it("should fall back to the remembered organization", () => {
      expect(pick(all, null, "personal")).toBe(personal);
    });

    it("should ignore a remembered organization the user left", () => {
      expect(pick(all, null, "gone")).toBe(acme);
    });

    it("should default a team member to their first team", () => {
      expect(pick(all, null, null)).toBe(acme);
    });

    it("should default anyone else to their personal organization", () => {
      expect(pick([personal], null, null)).toBe(personal);
    });

    it("should return null when there is nothing to pick", () => {
      expect(pick([], null, null)).toBeNull();
    });
  });

  describe("isMember", () => {
    const membership = { userID: "user_a", clerkOrgIDs: ["org_acme"] };

    it("should admit the owner of a personal organization", () => {
      expect(isMember(organizationOf("p", "personal"), membership)).toBe(true);
    });

    it("should refuse anyone else on a personal organization", () => {
      expect(
        isMember(organizationOf("p", "personal"), { ...membership, userID: "user_b" }),
      ).toBe(false);
    });

    it("should admit a member of a team", () => {
      expect(isMember(organizationOf("acme", "team"), membership)).toBe(true);
    });

    it("should refuse a non-member of a team", () => {
      expect(isMember(organizationOf("globex", "team"), membership)).toBe(false);
    });

    it("should refuse everyone on a team with no Clerk id", () => {
      expect(
        isMember({ ...organizationOf("acme", "team"), clerkOrgID: null }, membership),
      ).toBe(false);
    });
  });

  describe("listAll", () => {
    it("should return every organization by name", async () => {
      await createOrganization(store, { kind: "team", name: "Zeta", clerkOrgID: "z" });
      await createOrganization(store, { kind: "team", name: "Alpha", clerkOrgID: "a" });
      await createOrganization(store, {
        kind: "personal",
        name: "Mid",
        ownerUserID: "user_a",
      });
      expect((await listAll(store)).map((o) => o.name)).toEqual([
        "Alpha",
        "Mid",
        "Zeta",
      ]);
    });
  });

  describe("retrieve", () => {
    it("should return the organization with the key", async () => {
      const org = await createOrganization(store, { kind: "team", clerkOrgID: "a" });
      expect(await retrieve(store, org.key)).toEqual(org);
    });

    it("should return undefined for an unknown key", async () => {
      expect(await retrieve(store, crypto.randomUUID())).toBeUndefined();
    });
  });
});
