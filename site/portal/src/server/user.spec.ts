// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeAll, beforeEach, describe, expect, it } from "vitest";

import { event, type EventKind, type Organization } from "@/server/db/schema";
import { type Memory, openMemory } from "@/server/db/testutil";
import { ADMIN_ROLE, memory, type Records } from "@/server/directory";
import { HASH_A, NOW } from "@/server/license/testutil";
import { NONE } from "@/server/usage";
import { arrival, list, retrieve, summarize } from "@/server/user";
import { createActivation, createLicense, createOrganization } from "@/testutil";

const at = (iso: string): Date => new Date(iso);

describe("user", () => {
  let store: Memory;
  let ada: Organization;
  let bob: Organization;

  beforeAll(async () => {
    store = await openMemory();
  });

  beforeEach(async () => {
    await store.clear();
    ada = await createOrganization(store, { kind: "personal", ownerUserID: "user_a" });
    bob = await createOrganization(store, {
      kind: "personal",
      name: "Bob",
      ownerUserID: "user_b",
    });
  });

  const record = async (
    org: Organization,
    kind: EventKind,
    iso: string,
  ): Promise<void> => {
    await store.query
      .insert(event)
      .values({ kind, actor: "user_a", organization: org.key, at: at(iso) });
  };

  const people: Records["people"] = {
    user_a: {
      name: "Ada Lovelace",
      email: "ada@example.com",
      joinedAt: at("2026-09-16T05:00:00Z"),
    },
    user_b: {
      name: "Bob",
      email: "bob@example.com",
      joinedAt: at("2026-09-15T00:00:00Z"),
    },
    user_c: {
      name: "Cy",
      email: "cy@example.com",
      joinedAt: at("2026-09-16T23:00:00Z"),
    },
  };

  describe("list", () => {
    it("should list every account with its usage", async () => {
      await record(ada, "renew", "2026-09-17T06:00:00Z");
      const entries = await list(store, memory({ people }), NOW);
      expect(entries.map((e) => [e.account.userID, e.usage.lastActive])).toEqual([
        ["user_c", null],
        ["user_a", at("2026-09-17T06:00:00Z")],
        ["user_b", null],
      ]);
      expect(entries[0]).toEqual({
        account: { userID: "user_c", ...people.user_c },
        usage: NONE,
      });
    });

    const usageOfAda = async () =>
      (await list(store, memory({ people }), NOW)).find(
        (e) => e.account.userID === "user_a",
      )?.usage;

    it("should read the last login, the last activity, and the active days", async () => {
      await record(ada, "link", "2026-09-01T10:00:00Z");
      await record(ada, "renew", "2026-09-01T16:00:00Z");
      await record(ada, "renew", "2026-09-16T08:00:00Z");
      await record(ada, "renew", "2026-09-17T06:00:00Z");
      expect(await usageOfAda()).toEqual({
        lastLogin: at("2026-09-01T10:00:00Z"),
        lastActive: at("2026-09-17T06:00:00Z"),
        days: 3,
      });
    });

    it("should count only the active days among the last 30", async () => {
      await record(ada, "link", "2026-08-18T23:59:59Z");
      await record(ada, "renew", "2026-08-19T00:00:00Z");
      expect((await usageOfAda())?.days).toBe(1);
    });

    it("should read a user who renews without a login in the log", async () => {
      await record(ada, "renew", "2026-09-17T06:00:00Z");
      expect(await usageOfAda()).toEqual({
        lastLogin: null,
        lastActive: at("2026-09-17T06:00:00Z"),
        days: 1,
      });
    });

    it("should leave out events that are not Desktop use", async () => {
      await record(ada, "unlink", "2026-09-17T06:00:00Z");
      await record(bob, "rename", "2026-09-17T06:00:00Z");
      const team = await createOrganization(store, {
        kind: "team",
        clerkOrgID: "org_acme",
      });
      await record(team, "renew", "2026-09-17T06:00:00Z");
      const entries = await list(store, memory({ people }), NOW);
      expect(entries.map((e) => e.usage)).toEqual([NONE, NONE, NONE]);
    });
  });

  describe("retrieve", () => {
    it("should read the account, its usage, its machines, and its teams", async () => {
      await record(ada, "renew", "2026-09-17T06:00:00Z");
      const lic = await createLicense(store, {
        organization: ada.key,
        edition: "desktop",
      });
      const machine = await createActivation(store, {
        license: lic.key,
        fingerprint: [HASH_A],
      });
      const directory = memory({
        people,
        organizations: [{ clerkOrgID: "org_acme", name: "Acme" }],
        members: {
          user_a: [{ clerkOrgID: "org_acme", name: "Acme", role: ADMIN_ROLE }],
        },
      });
      const detail = await retrieve(store, directory, "user_a", NOW);
      expect(detail.account).toEqual({ userID: "user_a", ...people.user_a });
      expect(detail.usage.lastActive).toEqual(at("2026-09-17T06:00:00Z"));
      expect(detail.personal?.key).toBe(ada.key);
      expect(detail.machines.map((m) => m.activation.key)).toEqual([machine.key]);
      expect(detail.teams).toEqual([
        {
          organization: expect.objectContaining({ kind: "team", name: "Acme" }),
          role: ADMIN_ROLE,
        },
      ]);
    });

    it("should read only the user's own usage", async () => {
      await record(ada, "renew", "2026-09-17T06:00:00Z");
      await record(bob, "renew", "2026-09-16T06:00:00Z");
      const detail = await retrieve(store, memory({ people }), "user_b", NOW);
      expect(detail.usage.lastActive).toEqual(at("2026-09-16T06:00:00Z"));
    });

    it("should read a user with no personal organization", async () => {
      const detail = await retrieve(store, memory({ people }), "user_c", NOW);
      expect(detail.personal).toBeNull();
      expect(detail.machines).toEqual([]);
      expect(detail.usage).toEqual(NONE);
    });

    it("should throw a 404 for a user Clerk does not know", async () => {
      await expect(retrieve(store, memory({ people }), "user_x", NOW)).rejects.toThrow(
        "User not found",
      );
    });
  });

  describe("arrival", () => {
    const arrive = async (userID: string, previous: Date | null) =>
      await arrival(store, memory({ people }), {
        userID,
        machine: "stand-1",
        previous,
        now: NOW,
      });

    it("should read the first use of a day with the active days", async () => {
      await record(ada, "renew", "2026-09-16T08:00:00Z");
      await record(ada, "renew", "2026-09-17T12:00:00Z");
      expect(await arrive("user_a", at("2026-09-16T08:00:00Z"))).toEqual({
        account: { userID: "user_a", ...people.user_a },
        machine: "stand-1",
        previous: at("2026-09-16T08:00:00Z"),
        at: NOW,
        days: 2,
      });
    });

    it("should skip a later use on the same day", async () => {
      expect(await arrive("user_a", at("2026-09-17T06:00:00Z"))).toBeNull();
    });

    it("should skip a user Clerk does not know", async () => {
      expect(await arrive("user_x", null)).toBeNull();
    });
  });

  describe("summarize", () => {
    it("should count the use of one UTC day", async () => {
      const cy = await createOrganization(store, {
        kind: "personal",
        name: "Cy",
        ownerUserID: "user_c",
      });
      const dee = await createOrganization(store, {
        kind: "personal",
        name: "Dee",
        ownerUserID: "user_d",
      });
      await record(ada, "link", "2026-09-01T10:00:00Z");
      await record(ada, "renew", "2026-09-16T08:00:00Z");
      await record(bob, "renew", "2026-09-10T08:00:00Z");
      await record(cy, "link", "2026-08-17T23:00:00Z");
      await record(dee, "link", "2026-09-16T23:59:59Z");
      await record(dee, "renew", "2026-09-17T01:00:00Z");
      expect(
        await summarize(store, memory({ people }), at("2026-09-16T12:00:00Z")),
      ).toEqual({
        day: at("2026-09-16T00:00:00Z"),
        active: { day: 2, week: 3, month: 3 },
        accounts: 2,
        firstLogins: 1,
      });
    });

    it("should leave out Synnax Labs users", async () => {
      const internal = {
        ...people,
        user_a: { ...people.user_a, email: "ada@synnaxlabs.com" },
      };
      await record(ada, "link", "2026-09-16T08:00:00Z");
      await record(bob, "link", "2026-09-16T09:00:00Z");
      expect(
        await summarize(
          store,
          memory({ people: internal }),
          at("2026-09-16T12:00:00Z"),
        ),
      ).toEqual({
        day: at("2026-09-16T00:00:00Z"),
        active: { day: 1, week: 1, month: 1 },
        accounts: 1,
        firstLogins: 1,
      });
    });
  });
});
