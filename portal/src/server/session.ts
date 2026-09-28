// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { clerkClient, type OrganizationMembership } from "@clerk/astro/server";
import { type APIContext } from "astro";

import { unauthorized } from "@/server/errors";
import { type Membership } from "@/server/organization";

/** Team is one Clerk organization the user belongs to. */
export interface Team {
  clerkOrgID: string;
  name: string;
  role: string;
}

/** Session is what the portal knows about the signed-in user for one request. */
export interface Session extends Membership {
  email: string;
  name: string;
  /** teams are the Clerk organizations the user is a member of. */
  teams: Team[];
  /** staff is true for admins of the Synnax Labs team organization. */
  staff: boolean;
}

export const ADMIN_ROLE = "org:admin";
const STAFF_ROLE = ADMIN_ROLE;

/**
 * resolve reads the signed-in user and their organization memberships from Clerk.
 * Throws a 401 when nobody is signed in.
 */
export const resolve = async (
  context: APIContext,
  staffOrgID: string,
): Promise<Session> => {
  const { userId } = context.locals.auth();
  if (userId == null) throw unauthorized();
  const clerk = clerkClient(context);
  const [user, memberships] = await Promise.all([
    clerk.users.getUser(userId),
    allMemberships(clerk, userId),
  ]);
  const teams = memberships.map((m) => ({
    clerkOrgID: m.organization.id,
    name: m.organization.name,
    role: m.role,
  }));
  const clerkOrgIDs = teams.map((t) => t.clerkOrgID);
  const staff = teams.some((t) => t.clerkOrgID === staffOrgID && t.role === STAFF_ROLE);
  const email =
    user.emailAddresses.find((e) => e.id === user.primaryEmailAddressId)
      ?.emailAddress ??
    user.emailAddresses[0]?.emailAddress ??
    "";
  return {
    userID: userId,
    email,
    name: displayName(user, email),
    teams,
    clerkOrgIDs,
    staff,
  };
};

const PAGE = 100;

const allMemberships = async (
  clerk: ReturnType<typeof clerkClient>,
  userId: string,
): Promise<OrganizationMembership[]> => {
  const all: OrganizationMembership[] = [];
  for (let offset = 0; ; offset += PAGE) {
    const page = await clerk.users.getOrganizationMembershipList({
      userId,
      limit: PAGE,
      offset,
    });
    all.push(...page.data);
    if (page.data.length < PAGE) return all;
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

/**
 * emails returns the addresses to notify for an organization: its owner for a
 * personal organization, its admins for a team.
 */
export const emails = async (
  context: APIContext,
  org: { kind: string; ownerUserID: string | null; clerkOrgID: string | null },
): Promise<string[]> => {
  const clerk = clerkClient(context);
  if (org.kind === "personal") {
    if (org.ownerUserID == null) return [];
    const user = await clerk.users.getUser(org.ownerUserID);
    const primary = user.emailAddresses.find(
      (e) => e.id === user.primaryEmailAddressId,
    );
    return primary == null ? [] : [primary.emailAddress];
  }
  if (org.clerkOrgID == null) return [];
  const members = await clerk.organizations.getOrganizationMembershipList({
    organizationId: org.clerkOrgID,
    limit: 100,
  });
  return members.data
    .filter((m) => m.role === STAFF_ROLE)
    .map((m) => m.publicUserData?.identifier)
    .filter((e): e is string => e != null && e !== "");
};
