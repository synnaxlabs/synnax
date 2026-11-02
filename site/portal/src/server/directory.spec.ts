// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { ADMIN_ROLE, clerk, isInternal, memory } from "@/server/directory";

interface FakeUser {
  id: string;
  firstName: string | null;
  lastName: string | null;
  username: string | null;
  primaryEmailAddressId: string | null;
  emailAddresses: {
    id: string;
    emailAddress: string;
    verification?: { status: string } | null;
  }[];
  hasImage: boolean;
  imageUrl: string;
  createdAt: number;
}

const user = (overrides: Partial<FakeUser> = {}): FakeUser => ({
  id: "user_a",
  firstName: "Ada",
  lastName: "Lovelace",
  username: "ada",
  primaryEmailAddressId: "email_2",
  emailAddresses: [
    { id: "email_1", emailAddress: "old@example.com" },
    { id: "email_2", emailAddress: "ada@example.com" },
  ],
  hasImage: false,
  imageUrl: "https://img.clerk.com/default",
  createdAt: Date.parse("2026-09-01T00:00:00Z"),
  ...overrides,
});

interface Paged {
  limit: number;
  offset: number;
}

const page = <T>(items: T[], { limit, offset }: Paged): { data: T[] } => ({
  data: items.slice(offset, offset + limit),
});

const range = <T>(n: number, f: (i: number) => T): T[] =>
  Array.from({ length: n }, (_, i) => f(i));

interface FakeArgs {
  users?: FakeUser[];
  memberships?: { organization: { id: string; name: string }; role: string }[];
  members?: {
    role: string;
    publicUserData?: {
      userId: string;
      identifier: string;
      firstName: string | null;
      lastName: string | null;
    } | null;
  }[];
  organizations?: { id: string; name: string; membersCount?: number }[];
  listed?: { id: string; fullName: string | null; primaryEmailAddress: unknown }[];
  /** unavailable are organization and user ids Clerk fails to read with a 500. */
  unavailable?: string[];
  /** refusal is what Clerk throws on every write. */
  refusal?: Error;
}

const createClient = ({
  users = [user()],
  memberships = [],
  members = [],
  organizations = [],
  listed = [],
  unavailable = [],
  refusal,
}: FakeArgs = {}) => {
  const calls: Record<string, unknown[]> = {};
  const record = (name: string, args: unknown): void => {
    (calls[name] ??= []).push(args);
  };
  const write = async (name: string, args: unknown): Promise<void> => {
    if (refusal != null) throw refusal;
    record(name, args);
  };
  const client = {
    users: {
      getOrganizationMembershipList: async (args: Paged & { userId: string }) => {
        record("memberships", args);
        return page(memberships, args);
      },
      getUser: async (id: string) => {
        if (unavailable.includes(id))
          throw Object.assign(new Error("Down"), { status: 500 });
        const found = users.find((u) => u.id === id);
        if (found == null) throw Object.assign(new Error("Not Found"), { status: 404 });
        return found;
      },
      deleteUser: async (id: string) => await write("deletedUsers", id),
      getUserList: async (args: Paged & { userId?: string[] }) => {
        record("userList", args);
        if (args.userId == null) return page(users, args);
        const { userId } = args;
        return {
          data: listed.filter((u) => userId.includes(u.id)).slice(0, args.limit),
        };
      },
    },
    organizations: {
      getOrganizationMembershipList: async (
        args: Paged & { organizationId: string },
      ) => {
        record("members", args);
        return page(members, args);
      },
      getOrganizationList: async (args: Paged) => {
        record("organizations", args);
        return page(organizations, args);
      },
      getOrganization: async ({ organizationId }: { organizationId: string }) => {
        if (unavailable.includes(organizationId))
          throw Object.assign(new Error("Down"), { status: 500 });
        const found = organizations.find((o) => o.id === organizationId);
        if (found == null) throw Object.assign(new Error("Not Found"), { status: 404 });
        return found;
      },
      createOrganizationMembership: async (args: unknown) => await write("added", args),
      updateOrganizationMembership: async (args: unknown) => await write("roles", args),
      deleteOrganizationMembership: async (args: unknown) =>
        await write("removed", args),
      createOrganization: async (args: { name: string }) => {
        await write("created", args);
        return { id: "org_new", name: args.name };
      },
      updateOrganization: async (id: string, args: unknown) =>
        await write("renamed", [id, args]),
      deleteOrganization: async (id: string) => await write("deleted", id),
    },
  };
  return { directory: clerk(client as unknown as Parameters<typeof clerk>[0]), calls };
};

describe("directory", () => {
  describe("isInternal", () => {
    it("should accept a Synnax Labs address in any case", () => {
      expect(isInternal("em@synnaxlabs.com")).toBe(true);
      expect(isInternal("Em@SynnaxLabs.com")).toBe(true);
    });

    it("should refuse an address on another domain", () => {
      expect(isInternal("ada@example.com")).toBe(false);
      expect(isInternal("ada@notsynnaxlabs.com")).toBe(false);
      expect(isInternal("ada@synnaxlabs.com.example.com")).toBe(false);
      expect(isInternal("")).toBe(false);
    });
  });

  describe("memory", () => {
    const directory = memory({
      people: {
        user_a: { email: "ada@example.com", name: "Ada" },
        user_b: { email: "bob@example.com", name: "Bob" },
        user_c: { email: "", name: "Cy" },
      },
      members: {
        user_a: [{ clerkOrgID: "org_acme", name: "Acme", role: ADMIN_ROLE }],
        user_b: [{ clerkOrgID: "org_acme", name: "Acme", role: "org:member" }],
        user_c: [{ clerkOrgID: "org_acme", name: "Acme", role: ADMIN_ROLE }],
      },
      organizations: [{ clerkOrgID: "org_acme", name: "Acme" }],
    });

    it("should return a known person", async () => {
      expect(await directory.person("user_a")).toEqual({
        email: "ada@example.com",
        name: "Ada",
      });
    });

    it("should throw for an unknown person", async () => {
      await expect(directory.person("user_x")).rejects.toThrow("no Clerk user user_x");
    });

    it("should return a user's memberships, or none", async () => {
      expect(await directory.memberships("user_b")).toEqual([
        { clerkOrgID: "org_acme", name: "Acme", role: "org:member" },
      ]);
      expect(await directory.memberships("user_x")).toEqual([]);
    });

    it("should list a team's members, or none", async () => {
      expect(await directory.roster("org_acme")).toEqual([
        { userID: "user_a", name: "Ada", email: "ada@example.com", role: ADMIN_ROLE },
        { userID: "user_b", name: "Bob", email: "bob@example.com", role: "org:member" },
        { userID: "user_c", name: "Cy", email: "", role: ADMIN_ROLE },
      ]);
      expect(await directory.roster("org_other")).toEqual([]);
    });

    it("should list and find users, newest first", async () => {
      const dir = memory({
        people: {
          user_a: { email: "ada@example.com", name: "Ada", joinedAt: new Date(1) },
          user_b: { email: "bob@example.com", name: "Bob", joinedAt: new Date(2) },
        },
      });
      expect(await dir.users()).toEqual([
        {
          userID: "user_b",
          name: "Bob",
          email: "bob@example.com",
          joinedAt: new Date(2),
        },
        {
          userID: "user_a",
          name: "Ada",
          email: "ada@example.com",
          joinedAt: new Date(1),
        },
      ]);
      expect(await directory.account("user_c")).toEqual({
        userID: "user_c",
        name: "Cy",
        email: "",
        joinedAt: new Date(0),
      });
      expect(await directory.account("user_x")).toBeNull();
    });

    it("should list and find organizations", async () => {
      expect(await directory.teams()).toEqual([
        { clerkOrgID: "org_acme", name: "Acme", members: 3 },
      ]);
      expect(await directory.team("org_acme")).toEqual({
        clerkOrgID: "org_acme",
        name: "Acme",
      });
      expect(await directory.team("org_x")).toBeNull();
    });

    it("should name known users and leave out unknown ones", async () => {
      expect(await directory.names(["user_a", "user_x"])).toEqual({ user_a: "Ada" });
    });

    it("should read a person's address as their one verified address", async () => {
      expect(await directory.addresses("user_a")).toEqual(["ada@example.com"]);
      expect(await directory.addresses("user_c")).toEqual([]);
      expect(await directory.addresses("user_x")).toEqual([]);
    });

    it("should add a user to a team it knows", async () => {
      const dir = memory({ organizations: [{ clerkOrgID: "org_acme", name: "Acme" }] });
      await dir.add("org_acme", "user_z", "org:member");
      expect(await dir.memberships("user_z")).toEqual([
        { clerkOrgID: "org_acme", name: "Acme", role: "org:member" },
      ]);
      await expect(dir.add("org_x", "user_z", "org:member")).rejects.toThrow(
        "no Clerk organization org_x",
      );
    });

    it("should change roles, remove members, and create and rename teams", async () => {
      const dir = memory({
        members: {
          user_a: [{ clerkOrgID: "org_acme", name: "Acme", role: "org:member" }],
        },
        organizations: [{ clerkOrgID: "org_acme", name: "Acme" }],
      });
      await dir.setRole("org_acme", "user_a", ADMIN_ROLE);
      expect(await dir.memberships("user_a")).toEqual([
        { clerkOrgID: "org_acme", name: "Acme", role: ADMIN_ROLE },
      ]);
      await dir.rename("org_acme", "Acme Rockets");
      expect(await dir.team("org_acme")).toEqual({
        clerkOrgID: "org_acme",
        name: "Acme Rockets",
      });
      expect((await dir.memberships("user_a"))[0].name).toBe("Acme Rockets");
      await dir.remove("org_acme", "user_a");
      expect(await dir.memberships("user_a")).toEqual([]);
      const created = await dir.create("Globex");
      expect((await dir.teams())[0]).toEqual({ ...created, members: 0 });
      await dir.delete(created.clerkOrgID);
      expect((await dir.teams()).map((t) => t.clerkOrgID)).toEqual(["org_acme"]);
    });

    it("should serve records added after it was created", async () => {
      const empty = memory();
      empty.people.user_z = { email: "z@example.com", name: "Zed" };
      expect(await empty.person("user_z")).toEqual({
        email: "z@example.com",
        name: "Zed",
      });
    });
  });

  describe("clerk", () => {
    describe("person", () => {
      it("should read the primary address and the full name", async () => {
        const { directory } = createClient();
        expect(await directory.person("user_a")).toEqual({
          email: "ada@example.com",
          name: "Ada Lovelace",
        });
      });

      it("should fall back to the first address without a primary one", async () => {
        const { directory } = createClient({
          users: [user({ primaryEmailAddressId: null })],
        });
        expect((await directory.person("user_a")).email).toBe("old@example.com");
      });

      it("should read an empty address for a user with none", async () => {
        const { directory } = createClient({
          users: [user({ emailAddresses: [], firstName: null, lastName: null })],
        });
        expect(await directory.person("user_a")).toEqual({ email: "", name: "ada" });
      });

      it("should name a user by the first name alone", async () => {
        const { directory } = createClient({ users: [user({ lastName: "" })] });
        expect((await directory.person("user_a")).name).toBe("Ada");
      });

      it("should name a user by the username without a full name", async () => {
        const { directory } = createClient({
          users: [user({ firstName: null, lastName: null })],
        });
        expect((await directory.person("user_a")).name).toBe("ada");
      });

      it("should name a user by the address without a name or username", async () => {
        const { directory } = createClient({
          users: [user({ firstName: null, lastName: null, username: null })],
        });
        expect((await directory.person("user_a")).name).toBe("ada@example.com");
      });

      it("should read the picture of a user who set one", async () => {
        const { directory } = createClient({
          users: [user({ hasImage: true, imageUrl: "https://img.clerk.com/ada" })],
        });
        expect((await directory.person("user_a")).image).toBe(
          "https://img.clerk.com/ada",
        );
      });

      it("should leave out the picture Clerk generates for a user", async () => {
        const { directory } = createClient();
        expect((await directory.person("user_a")).image).toBeUndefined();
      });

      it("should throw when Clerk does not know the user", async () => {
        const { directory } = createClient();
        await expect(directory.person("user_x")).rejects.toThrow("Not Found");
      });
    });

    describe("users", () => {
      it("should list every user across every page, newest first", async () => {
        const { directory, calls } = createClient({
          users: range(101, (i) => user({ id: `user_${i}`, createdAt: i })),
        });
        const users = await directory.users();
        expect(users).toHaveLength(101);
        expect(users[3]).toEqual({
          userID: "user_3",
          name: "Ada Lovelace",
          email: "ada@example.com",
          joinedAt: new Date(3),
        });
        expect(calls.userList).toEqual([
          { limit: 100, offset: 0, orderBy: "-created_at" },
          { limit: 100, offset: 100, orderBy: "-created_at" },
        ]);
      });

      it("should name a user without a name by username, then address", async () => {
        const { directory } = createClient({
          users: [
            user({ id: "user_a", firstName: null, lastName: null }),
            user({ id: "user_b", firstName: null, lastName: null, username: null }),
          ],
        });
        expect((await directory.users()).map((u) => u.name)).toEqual([
          "ada",
          "ada@example.com",
        ]);
      });
    });

    describe("account", () => {
      it("should read one user", async () => {
        const { directory } = createClient();
        expect(await directory.account("user_a")).toEqual({
          userID: "user_a",
          name: "Ada Lovelace",
          email: "ada@example.com",
          joinedAt: new Date("2026-09-01T00:00:00Z"),
        });
      });

      it("should answer null for a user Clerk does not know", async () => {
        const { directory } = createClient();
        expect(await directory.account("user_x")).toBeNull();
      });

      it("should throw any other Clerk failure", async () => {
        const { directory } = createClient({ unavailable: ["user_down"] });
        await expect(directory.account("user_down")).rejects.toThrow("Down");
      });
    });

    describe("addresses", () => {
      it("should return only the verified addresses, in lowercase", async () => {
        const { directory } = createClient({
          users: [
            user({
              emailAddresses: [
                {
                  id: "email_1",
                  emailAddress: "Ada@Example.com",
                  verification: { status: "verified" },
                },
                {
                  id: "email_2",
                  emailAddress: "old@example.com",
                  verification: { status: "unverified" },
                },
                { id: "email_3", emailAddress: "new@example.com", verification: null },
              ],
            }),
          ],
        });
        expect(await directory.addresses("user_a")).toEqual(["ada@example.com"]);
      });
    });

    describe("memberships", () => {
      it("should read every page of memberships", async () => {
        const { directory, calls } = createClient({
          memberships: range(150, (i) => ({
            organization: { id: `org_${i}`, name: `Org ${i}` },
            role: "org:member",
          })),
        });
        const teams = await directory.memberships("user_a");
        expect(teams).toHaveLength(150);
        expect(teams[149]).toEqual({
          clerkOrgID: "org_149",
          name: "Org 149",
          role: "org:member",
        });
        expect(calls.memberships).toEqual([
          { userId: "user_a", limit: 100, offset: 0 },
          { userId: "user_a", limit: 100, offset: 100 },
        ]);
      });

      it("should ask for another page after an exactly full one", async () => {
        const { directory, calls } = createClient({
          memberships: range(100, (i) => ({
            organization: { id: `org_${i}`, name: `Org ${i}` },
            role: "org:member",
          })),
        });
        expect(await directory.memberships("user_a")).toHaveLength(100);
        expect(calls.memberships).toHaveLength(2);
      });
    });

    describe("roster", () => {
      const data = (
        userId: string,
        identifier: string,
        firstName: string | null = null,
        lastName: string | null = null,
      ) => ({ userId, identifier, firstName, lastName });

      it("should list the members of a team across every page", async () => {
        const { directory, calls } = createClient({
          members: range(120, (i) => ({
            role: "org:member",
            publicUserData: data(`user_${i}`, `m${i}@example.com`),
          })),
        });
        const roster = await directory.roster("org_acme");
        expect(roster).toHaveLength(120);
        expect(roster[119]).toEqual({
          userID: "user_119",
          name: "m119@example.com",
          email: "m119@example.com",
          role: "org:member",
        });
        expect(calls.members).toEqual([
          { organizationId: "org_acme", limit: 100, offset: 0 },
          { organizationId: "org_acme", limit: 100, offset: 100 },
        ]);
      });

      it("should name a member by their full name", async () => {
        const { directory } = createClient({
          members: [
            {
              role: ADMIN_ROLE,
              publicUserData: data("user_a", "ada@example.com", "Ada", "Lovelace"),
            },
            {
              role: ADMIN_ROLE,
              publicUserData: data("user_b", "bob@example.com", "Bob"),
            },
          ],
        });
        expect((await directory.roster("org_acme")).map((m) => m.name)).toEqual([
          "Ada Lovelace",
          "Bob",
        ]);
      });

      it("should leave out a membership without a user", async () => {
        const { directory } = createClient({
          members: [
            { role: ADMIN_ROLE, publicUserData: data("user_a", "ada@example.com") },
            { role: ADMIN_ROLE, publicUserData: null },
          ],
        });
        expect((await directory.roster("org_acme")).map((m) => m.userID)).toEqual([
          "user_a",
        ]);
      });
    });

    describe("teams", () => {
      it("should list every organization across every page", async () => {
        const { directory, calls } = createClient({
          organizations: range(101, (i) => ({
            id: `org_${i}`,
            name: `Org ${i}`,
            membersCount: i,
          })),
        });
        const teams = await directory.teams();
        expect(teams).toHaveLength(101);
        expect(teams[3]).toEqual({ clerkOrgID: "org_3", name: "Org 3", members: 3 });
        expect(calls.organizations).toEqual([
          { limit: 100, offset: 0, includeMembersCount: true },
          { limit: 100, offset: 100, includeMembersCount: true },
        ]);
      });

      it("should throw when Clerk leaves out a member count", async () => {
        const { directory } = createClient({
          organizations: [{ id: "org_acme", name: "Acme" }],
        });
        await expect(directory.teams()).rejects.toThrow(
          "Clerk listed organization org_acme without its member count",
        );
      });
    });

    describe("team", () => {
      it("should read one organization", async () => {
        const { directory } = createClient({
          organizations: [{ id: "org_acme", name: "Acme" }],
        });
        expect(await directory.team("org_acme")).toEqual({
          clerkOrgID: "org_acme",
          name: "Acme",
        });
      });

      it("should answer null for an organization Clerk does not know", async () => {
        const { directory } = createClient({ organizations: [] });
        expect(await directory.team("org_x")).toBeNull();
      });

      it("should throw any other Clerk failure", async () => {
        const { directory } = createClient({ unavailable: ["org_down"] });
        await expect(directory.team("org_down")).rejects.toThrow("Down");
      });
    });

    describe("names", () => {
      it("should name users by full name, then address, then id", async () => {
        const { directory } = createClient({
          listed: [
            { id: "user_a", fullName: "Ada Lovelace", primaryEmailAddress: null },
            {
              id: "user_b",
              fullName: null,
              primaryEmailAddress: { emailAddress: "bob@example.com" },
            },
            { id: "user_c", fullName: null, primaryEmailAddress: null },
          ],
        });
        expect(await directory.names(["user_a", "user_b", "user_c", "user_x"])).toEqual(
          {
            user_a: "Ada Lovelace",
            user_b: "bob@example.com",
            user_c: "user_c",
          },
        );
      });

      it("should name more users than Clerk lists in one call", async () => {
        const ids = range(101, (i) => `user_${i}`);
        const { directory, calls } = createClient({
          listed: ids.map((id) => ({ id, fullName: id, primaryEmailAddress: null })),
        });
        expect(Object.keys(await directory.names(ids))).toHaveLength(101);
        expect(
          calls.userList.map((c) => (c as { userId: string[] }).userId.length),
        ).toEqual([100, 1]);
      });

      it("should not call Clerk for no ids", async () => {
        const { directory, calls } = createClient();
        expect(await directory.names([])).toEqual({});
        expect(calls.userList).toBeUndefined();
      });
    });

    describe("member and team changes", () => {
      it("should add a member without an email", async () => {
        const { directory, calls } = createClient();
        await directory.add("org_acme", "user_b", "org:member");
        expect(calls.added).toEqual([
          { organizationId: "org_acme", userId: "user_b", role: "org:member" },
        ]);
      });

      it("should change a member's role", async () => {
        const { directory, calls } = createClient();
        await directory.setRole("org_acme", "user_b", ADMIN_ROLE);
        expect(calls.roles).toEqual([
          { organizationId: "org_acme", userId: "user_b", role: ADMIN_ROLE },
        ]);
      });

      it("should remove a member", async () => {
        const { directory, calls } = createClient();
        await directory.remove("org_acme", "user_b");
        expect(calls.removed).toEqual([
          { organizationId: "org_acme", userId: "user_b" },
        ]);
      });

      it("should create a team with no members", async () => {
        const { directory, calls } = createClient();
        expect(await directory.create("Acme")).toEqual({
          clerkOrgID: "org_new",
          name: "Acme",
        });
        expect(calls.created).toEqual([{ name: "Acme" }]);
      });

      it("should delete a team", async () => {
        const { directory, calls } = createClient();
        await directory.delete("org_acme");
        expect(calls.deleted).toEqual(["org_acme"]);
      });

      it("should delete an account", async () => {
        const { directory, calls } = createClient();
        await directory.deleteAccount("user_a");
        expect(calls.deletedUsers).toEqual(["user_a"]);
      });

      it("should treat an account Clerk does not know as deleted", async () => {
        const refusal = Object.assign(new Error("Not Found"), { status: 404 });
        const { directory } = createClient({ refusal });
        await expect(directory.deleteAccount("user_a")).resolves.toBeUndefined();
      });

      it("should treat a team Clerk does not know as deleted", async () => {
        const refusal = Object.assign(new Error("Not Found"), { status: 404 });
        const { directory } = createClient({ refusal });
        await expect(directory.delete("org_acme")).resolves.toBeUndefined();
      });

      it("should rename a team", async () => {
        const { directory, calls } = createClient();
        await directory.rename("org_acme", "Acme Rockets");
        expect(calls.renamed).toEqual([["org_acme", { name: "Acme Rockets" }]]);
      });

      it("should answer a refused change as a 400 with its reason", async () => {
        const refusal = Object.assign(new Error("Unprocessable"), {
          status: 422,
          errors: [{ message: "last admin", longMessage: "Keep one admin." }],
        });
        const { directory } = createClient({ refusal });
        for (const change of [
          directory.add("org_acme", "user_b", "org:member"),
          directory.setRole("org_acme", "user_b", ADMIN_ROLE),
          directory.remove("org_acme", "user_b"),
          directory.rename("org_acme", "Acme Rockets"),
        ])
          await expect(change).rejects.toMatchObject({
            status: 400,
            message: "Keep one admin.",
          });
      });
    });
  });
});
