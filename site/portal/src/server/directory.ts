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

/** Invitation asks someone by email to join a team. */
export interface Invitation {
  clerkOrgID: string;
  email: string;
  role: string;
  /** inviterUserID is the admin who sends the invitation. */
  inviterUserID: string;
  /** redirectURL is the absolute URL the link in the email lands on. */
  redirectURL: string;
}

/** Pending is a team invitation that its recipient has not accepted. */
export interface Pending extends Listed {
  invitationID: string;
}

/** Sent is an invitation to a team that its recipient has not accepted yet. */
export interface Sent {
  invitationID: string;
  email: string;
  role: string;
  sentAt: Date;
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
  teams: () => Promise<Listing[]>;
  /** team returns null when Clerk does not know the organization. */
  team: (clerkOrgID: string) => Promise<Listed | null>;
  /** names resolves user ids to the name each reads as, leaving out unknown ids. */
  names: (userIDs: string[]) => Promise<Record<string, string>>;
  /**
   * invite emails an invitation to join a team.
   * @throws {HTTPError} 400 with Clerk's reason when Clerk refuses the invitation.
   */
  invite: (invitation: Invitation) => Promise<void>;
  /** pending returns the team invitations the user has not accepted. */
  pending: (userID: string) => Promise<Pending[]>;
  /** outstanding returns a team's invitations no one has accepted, newest first. */
  outstanding: (clerkOrgID: string) => Promise<Sent[]>;
  /**
   * revokeInvitation cancels an invitation, so its link no longer joins the team.
   * @throws {HTTPError} 400 with Clerk's reason when Clerk refuses.
   */
  revokeInvitation: (clerkOrgID: string, invitationID: string) => Promise<void>;
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

const displayName = (
  user: { firstName: string | null; lastName: string | null; username: string | null },
  email: string,
): string => {
  const full = [user.firstName, user.lastName].filter((p) => p != null && p !== "");
  if (full.length > 0) return full.join(" ");
  return user.username ?? email;
};

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
  invite: async ({ clerkOrgID, email, role, inviterUserID, redirectURL }) => {
    await client.organizations
      .createOrganizationInvitation({
        organizationId: clerkOrgID,
        emailAddress: email,
        role,
        inviterUserId: inviterUserID,
        redirectUrl: redirectURL,
      })
      .catch((e: unknown) => {
        throw refused(e);
      });
  },
  pending: async (userID) =>
    await paged(
      async (offset) =>
        await client.users.getOrganizationInvitationList({
          userId: userID,
          status: "pending",
          limit: PAGE,
          offset,
        }),
      ({ id, organizationId, publicOrganizationData: org }) => {
        if (org == null)
          throw new Error(`Clerk sent invitation ${id} without its team`);
        return { invitationID: id, clerkOrgID: organizationId, name: org.name };
      },
    ),
  outstanding: async (clerkOrgID) =>
    await paged(
      async (offset) =>
        await client.organizations.getOrganizationInvitationList({
          organizationId: clerkOrgID,
          status: ["pending"],
          limit: PAGE,
          offset,
        }),
      (i) => ({
        invitationID: i.id,
        email: i.emailAddress,
        role: i.role,
        sentAt: new Date(i.createdAt),
      }),
    ),
  revokeInvitation: async (clerkOrgID, invitationID) => {
    await client.organizations
      .revokeOrganizationInvitation({
        organizationId: clerkOrgID,
        invitationId: invitationID,
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
  /** people maps a Clerk user id to the person. */
  people: Record<string, Person>;
  /** members maps a Clerk user id to the teams the user belongs to. */
  members: Record<string, Team[]>;
  organizations: Listed[];
  /** invitations are every invitation sent, oldest first. */
  invitations: Invitation[];
  /** invited maps a Clerk user id to the invitations the user has not accepted. */
  invited: Record<string, Pending[]>;
  /** revoked are the revoked invitation ids. An invitation's id is `inv_<index>`. */
  revoked: string[];
}

/** memory serves a directory from records instead of Clerk. For tests. */
export const memory = ({
  people = {},
  members = {},
  organizations = [],
  invitations = [],
  invited = {},
  revoked = [],
}: Partial<Records> = {}): Directory & Records => ({
  people,
  members,
  organizations,
  invitations,
  invited,
  revoked,
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
  invite: async (invitation) => {
    invitations.push(invitation);
  },
  pending: async (userID) => invited[userID] ?? [],
  outstanding: async (clerkOrgID) =>
    invitations
      .map((inv, i) => ({ ...inv, invitationID: `inv_${i}` }))
      .filter(
        (inv) => inv.clerkOrgID === clerkOrgID && !revoked.includes(inv.invitationID),
      )
      .map(({ invitationID, email, role }) => ({
        invitationID,
        email,
        role,
        sentAt: new Date(0),
      }))
      .reverse(),
  revokeInvitation: async (_, invitationID) => {
    revoked.push(invitationID);
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
  rename: async (clerkOrgID, name) => {
    const team = organizations.find((o) => o.clerkOrgID === clerkOrgID);
    if (team == null) throw new Error(`no Clerk organization ${clerkOrgID}`);
    team.name = name;
    for (const teams of Object.values(members))
      for (const t of teams) if (t.clerkOrgID === clerkOrgID) t.name = name;
  },
});
