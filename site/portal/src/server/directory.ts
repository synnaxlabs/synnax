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

/** Person is a Clerk user as the portal names and mails them. */
export interface Person {
  email: string;
  name: string;
  /** image is the URL of the user's picture, absent when they have not set one. */
  image?: string;
}

/** Listed is one organization in the Clerk dashboard, keyed by its Clerk id. */
export interface Listed {
  clerkOrgID: string;
  name: string;
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

/** Directory reads the people and organizations Clerk holds. */
export interface Directory {
  /** person throws when Clerk does not know the user. */
  person: (userID: string) => Promise<Person>;
  memberships: (userID: string) => Promise<Team[]>;
  /** roster returns every member of a team. */
  roster: (clerkOrgID: string) => Promise<Member[]>;
  /** teams returns every organization in Clerk, newest first. */
  teams: () => Promise<Listed[]>;
  /** team returns null when Clerk does not know the organization. */
  team: (clerkOrgID: string) => Promise<Listed | null>;
  /** names resolves user ids to the name each reads as, leaving out unknown ids. */
  names: (userIDs: string[]) => Promise<Record<string, string>>;
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

const displayName = (
  user: { firstName: string | null; lastName: string | null; username: string | null },
  email: string,
): string => {
  const full = [user.firstName, user.lastName].filter((p) => p != null && p !== "");
  if (full.length > 0) return full.join(" ");
  return user.username ?? email;
};

export const clerk = (client: Client): Directory => ({
  person: async (userID) => {
    const user = await client.users.getUser(userID);
    const email =
      user.emailAddresses.find((e) => e.id === user.primaryEmailAddressId)
        ?.emailAddress ??
      user.emailAddresses[0]?.emailAddress ??
      "";
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
  teams: async () =>
    await paged(
      async (offset) =>
        await client.organizations.getOrganizationList({ limit: PAGE, offset }),
      (o) => ({ clerkOrgID: o.id, name: o.name }),
    ),
  team: async (clerkOrgID) => {
    const org = await client.organizations
      .getOrganization({ organizationId: clerkOrgID })
      .catch((e: unknown) => {
        if (e instanceof Error && "status" in e && e.status === 404) return null;
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
});

/** Records are what a {@link memory} directory serves. */
export interface Records {
  /** people maps a Clerk user id to the person. */
  people: Record<string, Person>;
  /** members maps a Clerk user id to the teams the user belongs to. */
  members: Record<string, Team[]>;
  organizations: Listed[];
}

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
  teams: async () => organizations,
  team: async (clerkOrgID) =>
    organizations.find((o) => o.clerkOrgID === clerkOrgID) ?? null,
  names: async (userIDs) =>
    Object.fromEntries(
      userIDs.flatMap((id) => (people[id] == null ? [] : [[id, people[id].name]])),
    ),
});
