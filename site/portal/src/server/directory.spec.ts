// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { ADMIN_ROLE, clerk, memory } from "@/server/directory";

interface FakeUser {
  id: string;
  firstName: string | null;
  lastName: string | null;
  username: string | null;
  primaryEmailAddressId: string | null;
  emailAddresses: { id: string; emailAddress: string }[];
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
  members?: { role: string; publicUserData?: { identifier: string } | null }[];
  organizations?: { id: string; name: string }[];
  listed?: { id: string; fullName: string | null; primaryEmailAddress: unknown }[];
}

const createClient = ({
  users = [user()],
  memberships = [],
  members = [],
  organizations = [],
  listed = [],
}: FakeArgs = {}) => {
  const calls: Record<string, unknown[]> = {};
  const record = (name: string, args: unknown): void => {
    (calls[name] ??= []).push(args);
  };
  const client = {
    users: {
      getUser: async (id: string) => {
        const found = users.find((u) => u.id === id);
        if (found == null) throw new Error("Not Found");
        return found;
      },
      getOrganizationMembershipList: async (args: Paged & { userId: string }) => {
        record("memberships", args);
        return page(memberships, args);
      },
      getUserList: async (args: { userId: string[]; limit: number }) => {
        record("userList", args);
        return {
          data: listed.filter((u) => args.userId.includes(u.id)).slice(0, args.limit),
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
        const found = organizations.find((o) => o.id === organizationId);
        if (found == null) throw Object.assign(new Error("Not Found"), { status: 404 });
        if (found.name === "") throw Object.assign(new Error("Down"), { status: 500 });
        return found;
      },
    },
  };
  return { directory: clerk(client as unknown as Parameters<typeof clerk>[0]), calls };
};

describe("directory", () => {
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

    it("should list the addresses of a team's admins", async () => {
      expect(await directory.admins("org_acme")).toEqual(["ada@example.com"]);
      expect(await directory.admins("org_other")).toEqual([]);
    });

    it("should list and find organizations", async () => {
      expect(await directory.teams()).toEqual([
        { clerkOrgID: "org_acme", name: "Acme" },
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

      it("should throw when Clerk does not know the user", async () => {
        const { directory } = createClient();
        await expect(directory.person("user_x")).rejects.toThrow("Not Found");
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

    describe("admins", () => {
      it("should list the identifiers of admins across every page", async () => {
        const { directory, calls } = createClient({
          members: [
            ...range(120, () => ({
              role: "org:member",
              publicUserData: { identifier: "member@example.com" },
            })),
            { role: ADMIN_ROLE, publicUserData: { identifier: "ada@example.com" } },
            { role: ADMIN_ROLE, publicUserData: { identifier: "" } },
            { role: ADMIN_ROLE, publicUserData: null },
          ],
        });
        expect(await directory.admins("org_acme")).toEqual(["ada@example.com"]);
        expect(calls.members).toEqual([
          { organizationId: "org_acme", limit: 100, offset: 0 },
          { organizationId: "org_acme", limit: 100, offset: 100 },
        ]);
      });
    });

    describe("teams", () => {
      it("should list every organization across every page", async () => {
        const { directory, calls } = createClient({
          organizations: range(101, (i) => ({ id: `org_${i}`, name: `Org ${i}` })),
        });
        const teams = await directory.teams();
        expect(teams).toHaveLength(101);
        expect(teams[0]).toEqual({ clerkOrgID: "org_0", name: "Org 0" });
        expect(calls.organizations).toHaveLength(2);
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
        const { directory } = createClient({
          organizations: [{ id: "org_down", name: "" }],
        });
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
  });
});
