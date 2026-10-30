// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type clerkClient } from "@clerk/astro/server";
import { errors } from "@synnaxlabs/x";

import { badRequest } from "@/server/errors";

/** Person is a Clerk user as the portal names and mails them. */
export interface Person {
  email: string;
  name: string;
  /** image is the URL of the user's picture, absent when they have not set one. */
  image?: string;
}

/** Account is one Clerk user as the staff console lists them. */
export interface Account {
  userID: string;
  name: string;
  /** email is empty when Clerk holds no address for the user. */
  email: string;
  joinedAt: Date;
}

/** Listed is one organization in the Clerk dashboard, keyed by its Clerk id. */
export interface Listed {
  clerkOrgID: string;
  name: string;
}

/** Listing is one organization in the Clerk dashboard with its number of members. */
export interface Listing extends Listed {
  members: number;
}

/** Team is one Clerk organization the user belongs to. */
export interface Team extends Listed {
  role: string;
}

/** Member is one person on a team. */
export interface Member {
  userID: string;
  name: string;
  /** email is empty when Clerk holds no address for the member. */
  email: string;
  role: string;
}

/** ADMIN_ROLE is Clerk's role for members who manage a team. */
export const ADMIN_ROLE = "org:admin";
export const MEMBER_ROLE = "org:member";

export const ROLES: { key: string; name: string }[] = [
  { key: ADMIN_ROLE, name: "Admin" },
  { key: MEMBER_ROLE, name: "Member" },
];

export const roleName = (role: string): string =>
  ROLES.find((r) => r.key === role)?.name ?? role;

const INTERNAL_DOMAIN = "@synnaxlabs.com";

/** isInternal is true for a Synnax Labs email address. */
export const isInternal = (email: string): boolean =>
  email.toLowerCase().endsWith(INTERNAL_DOMAIN);

/** Directory reads the people and organizations Clerk holds. */
export interface Directory {
  /** person throws when Clerk does not know the user. */
  person: (userID: string) => Promise<Person>;
  memberships: (userID: string) => Promise<Team[]>;
  /** roster returns every member of a team. */
  roster: (clerkOrgID: string) => Promise<Member[]>;
  /** users returns every user in Clerk, newest first. */
  users: () => Promise<Account[]>;
  /** account returns null when Clerk does not know the user. */
  account: (userID: string) => Promise<Account | null>;
  /**
   * deleteAccount removes a user and their memberships. It does nothing for an unknown
   * user.
   */
  deleteAccount: (userID: string) => Promise<void>;
  /** teams returns every organization in Clerk, newest first. */
  teams: () => Promise<Listing[]>;
  /** team returns null when Clerk does not know the organization. */
  team: (clerkOrgID: string) => Promise<Listed | null>;
  /** names resolves user ids to the name each reads as, leaving out unknown ids. */
  names: (userIDs: string[]) => Promise<Record<string, string>>;
  /** addresses returns the user's verified email addresses in lowercase. */
  addresses: (userID: string) => Promise<string[]>;
  /**
   * add puts a user on a team with a role. Clerk sends no email.
   * @throws {HTTPError} 400 with Clerk's reason when Clerk refuses.
   */
  add: (clerkOrgID: string, userID: string, role: string) => Promise<void>;
  /**
   * setRole changes a member's role on a team.
   * @throws {HTTPError} 400 with Clerk's reason when Clerk refuses.
   */
  setRole: (clerkOrgID: string, userID: string, role: string) => Promise<void>;
  /**
   * remove takes a member off a team.
   * @throws {HTTPError} 400 with Clerk's reason when Clerk refuses.
   */
  remove: (clerkOrgID: string, userID: string) => Promise<void>;
  /** create makes a team with no members. */
  create: (name: string) => Promise<Listed>;
  /** delete removes a team and its memberships. It does nothing for an unknown team. */
  delete: (clerkOrgID: string) => Promise<void>;
  /**
   * rename changes a team's name.
   * @throws {HTTPError} 400 with Clerk's reason when Clerk refuses.
   */
  rename: (clerkOrgID: string, name: string) => Promise<void>;
}

type Client = ReturnType<typeof clerkClient>;

const PAGE = 100;

const paged = async <T, R>(
  fetch: (offset: number) => Promise<{ data: T[] }>,
  map: (item: T) => R,
): Promise<R[]> => {
  const all: R[] = [];
  for (let offset = 0; ; offset += PAGE) {
    const { data } = await fetch(offset);
    all.push(...data.map(map));
    if (data.length < PAGE) return all;
  }
};

interface Named {
  firstName: string | null;
  lastName: string | null;
  username: string | null;
  primaryEmailAddressId: string | null;
  emailAddresses: { id: string; emailAddress: string }[];
}

/** primaryEmail reads a user's primary address, then any address, then none. */
const primaryEmail = (user: Named): string =>
  user.emailAddresses.find((e) => e.id === user.primaryEmailAddressId)?.emailAddress ??
  user.emailAddresses[0]?.emailAddress ??
  "";

const displayName = (user: Named, email: string): string => {
  const full = [user.firstName, user.lastName].filter((p) => p != null && p !== "");
  if (full.length > 0) return full.join(" ");
  return user.username ?? email;
};

const toAccount = (user: Named & { id: string; createdAt: number }): Account => {
  const email = primaryEmail(user);
  return {
    userID: user.id,
    name: displayName(user, email),
    email,
    joinedAt: new Date(user.createdAt),
  };
};

const isNotFound = (e: unknown): boolean =>
  e instanceof Error && "status" in e && e.status === 404;

interface APIError {
  status?: number;
  errors?: { message?: string; longMessage?: string }[];
}

/** refused turns Clerk's refusal of a request into a 400 that carries its reason. */
const refused = (e: unknown): Error => {
  const { status, errors: reasons } = (e ?? {}) as APIError;
  if (status !== 400 && status !== 422) return errors.fromUnknown(e);
  const first = reasons?.[0];
  const reason = first?.longMessage ?? first?.message ?? "Clerk refused the request";
  return badRequest(reason, { cause: e });
};

export const clerk = (client: Client): Directory => ({
  person: async (userID) => {
    const user = await client.users.getUser(userID);
    const email = primaryEmail(user);
    return {
      email,
      name: displayName(user, email),
      image: user.hasImage ? user.imageUrl : undefined,
    };
  },
  memberships: async (userID) =>
    await paged(
      async (offset) =>
        await client.users.getOrganizationMembershipList({
          userId: userID,
          limit: PAGE,
          offset,
        }),
      (m) => ({
        clerkOrgID: m.organization.id,
        name: m.organization.name,
        role: m.role,
      }),
    ),
  roster: async (clerkOrgID) => {
    const members = await paged(
      async (offset) =>
        await client.organizations.getOrganizationMembershipList({
          organizationId: clerkOrgID,
          limit: PAGE,
          offset,
        }),
      ({ publicUserData: data, role }) => {
        const email = data?.identifier ?? "";
        const name = [data?.firstName, data?.lastName]
          .filter((p) => p != null && p !== "")
          .join(" ");
        return { userID: data?.userId ?? "", name: name || email, email, role };
      },
    );
    return members.filter((m) => m.userID !== "");
  },
  users: async () =>
    await paged(
      async (offset) =>
        await client.users.getUserList({
          limit: PAGE,
          offset,
          orderBy: "-created_at",
        }),
      toAccount,
    ),
  account: async (userID) => {
    const user = await client.users.getUser(userID).catch((e: unknown) => {
      if (isNotFound(e)) return null;
      throw errors.fromUnknown(e);
    });
    return user == null ? null : toAccount(user);
  },
  deleteAccount: async (userID) => {
    await client.users.deleteUser(userID).catch((e: unknown) => {
      if (!isNotFound(e)) throw errors.fromUnknown(e);
    });
  },
  teams: async () =>
    await paged(
      async (offset) =>
        await client.organizations.getOrganizationList({
          limit: PAGE,
          offset,
          includeMembersCount: true,
        }),
      (o) => {
        if (o.membersCount == null)
          throw new Error(`Clerk listed organization ${o.id} without its member count`);
        return { clerkOrgID: o.id, name: o.name, members: o.membersCount };
      },
    ),
  team: async (clerkOrgID) => {
    const org = await client.organizations
      .getOrganization({ organizationId: clerkOrgID })
      .catch((e: unknown) => {
        if (isNotFound(e)) return null;
        throw errors.fromUnknown(e);
      });
    return org == null ? null : { clerkOrgID: org.id, name: org.name };
  },
  names: async (userIDs) => {
    const named: Record<string, string> = {};
    for (let i = 0; i < userIDs.length; i += PAGE) {
      const { data } = await client.users.getUserList({
        userId: userIDs.slice(i, i + PAGE),
        limit: PAGE,
      });
      for (const u of data)
        named[u.id] = u.fullName ?? u.primaryEmailAddress?.emailAddress ?? u.id;
    }
    return named;
  },
  addresses: async (userID) => {
    const user = await client.users.getUser(userID);
    return user.emailAddresses
      .filter((e) => e.verification?.status === "verified")
      .map((e) => e.emailAddress.toLowerCase());
  },
  add: async (clerkOrgID, userID, role) => {
    await client.organizations
      .createOrganizationMembership({
        organizationId: clerkOrgID,
        userId: userID,
        role,
      })
      .catch((e: unknown) => {
        throw refused(e);
      });
  },
  setRole: async (clerkOrgID, userID, role) => {
    await client.organizations
      .updateOrganizationMembership({
        organizationId: clerkOrgID,
        userId: userID,
        role,
      })
      .catch((e: unknown) => {
        throw refused(e);
      });
  },
  remove: async (clerkOrgID, userID) => {
    await client.organizations
      .deleteOrganizationMembership({ organizationId: clerkOrgID, userId: userID })
      .catch((e: unknown) => {
        throw refused(e);
      });
  },
  create: async (name) => {
    const org = await client.organizations
      .createOrganization({ name })
      .catch((e: unknown) => {
        throw refused(e);
      });
    return { clerkOrgID: org.id, name: org.name };
  },
  delete: async (clerkOrgID) => {
    await client.organizations.deleteOrganization(clerkOrgID).catch((e: unknown) => {
      if (!isNotFound(e)) throw errors.fromUnknown(e);
    });
  },
  rename: async (clerkOrgID, name) => {
    await client.organizations
      .updateOrganization(clerkOrgID, { name })
      .catch((e: unknown) => {
        throw refused(e);
      });
  },
});

/** Records are what a {@link memory} directory serves. */
export interface Records {
  /** people maps a Clerk user id to the person, who joined at the epoch if unset. */
  people: Record<string, Person & { joinedAt?: Date }>;
  /** members maps a Clerk user id to the teams the user belongs to. */
  members: Record<string, Team[]>;
  organizations: Listed[];
}

const accountOf = (people: Records["people"], userID: string): Account => {
  const { name, email, joinedAt = new Date(0) } = people[userID];
  return { userID, name, email, joinedAt };
};

/** memory serves a directory from records instead of Clerk. For tests. */
export const memory = ({
  people = {},
  members = {},
  organizations = [],
}: Partial<Records> = {}): Directory & Records => ({
  people,
  members,
  organizations,
  person: async (userID) => {
    const person = people[userID];
    if (person == null) throw new Error(`no Clerk user ${userID}`);
    return person;
  },
  memberships: async (userID) => members[userID] ?? [],
  roster: async (clerkOrgID) =>
    Object.entries(members).flatMap(([userID, teams]) => {
      const team = teams.find((t) => t.clerkOrgID === clerkOrgID);
      if (team == null) return [];
      const { name = "", email = "" } = people[userID] ?? {};
      return [{ userID, name, email, role: team.role }];
    }),
  users: async () =>
    Object.keys(people)
      .map((userID) => accountOf(people, userID))
      .sort((a, b) => b.joinedAt.getTime() - a.joinedAt.getTime()),
  account: async (userID) =>
    people[userID] == null ? null : accountOf(people, userID),
  deleteAccount: async (userID) => {
    delete people[userID];
    delete members[userID];
  },
  teams: async () =>
    organizations.map((o) => ({
      ...o,
      members: Object.values(members).filter((teams) =>
        teams.some((t) => t.clerkOrgID === o.clerkOrgID),
      ).length,
    })),
  team: async (clerkOrgID) =>
    organizations.find((o) => o.clerkOrgID === clerkOrgID) ?? null,
  names: async (userIDs) =>
    Object.fromEntries(
      userIDs.flatMap((id) => (people[id] == null ? [] : [[id, people[id].name]])),
    ),
  addresses: async (userID) => {
    const email = people[userID]?.email ?? "";
    return email === "" ? [] : [email.toLowerCase()];
  },
  add: async (clerkOrgID, userID, role) => {
    const team = organizations.find((o) => o.clerkOrgID === clerkOrgID);
    if (team == null) throw new Error(`no Clerk organization ${clerkOrgID}`);
    members[userID] = [...(members[userID] ?? []), { ...team, role }];
  },
  setRole: async (clerkOrgID, userID, role) => {
    members[userID] = (members[userID] ?? []).map((t) =>
      t.clerkOrgID === clerkOrgID ? { ...t, role } : t,
    );
  },
  remove: async (clerkOrgID, userID) => {
    members[userID] = (members[userID] ?? []).filter(
      (t) => t.clerkOrgID !== clerkOrgID,
    );
  },
  create: async (name) => {
    const team = { clerkOrgID: `org_${organizations.length}`, name };
    organizations.unshift(team);
    return team;
  },
  delete: async (clerkOrgID) => {
    const i = organizations.findIndex((o) => o.clerkOrgID === clerkOrgID);
    if (i !== -1) organizations.splice(i, 1);
    for (const [userID, teams] of Object.entries(members))
      members[userID] = teams.filter((t) => t.clerkOrgID !== clerkOrgID);
  },
  rename: async (clerkOrgID, name) => {
    const team = organizations.find((o) => o.clerkOrgID === clerkOrgID);
    if (team == null) throw new Error(`no Clerk organization ${clerkOrgID}`);
    team.name = name;
    for (const teams of Object.values(members))
      for (const t of teams) if (t.clerkOrgID === clerkOrgID) t.name = name;
  },
});
