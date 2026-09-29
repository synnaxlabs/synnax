// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { eq, inArray, or } from "drizzle-orm";

import { type Store } from "@/server/db/db";
import { type Organization, organization } from "@/server/db/schema";
import { type Directory } from "@/server/directory";
import { type Session } from "@/server/session";

export interface EnsurePersonalArgs {
  userID: string;
  name: string;
}

/**
 * ensurePersonal returns the user's personal organization, creating it on first
 * sight. The Clerk webhook creates it eagerly; this covers a user who logged in before
 * the webhook fired.
 */
export const ensurePersonal = async (
  store: Store,
  { userID, name }: EnsurePersonalArgs,
): Promise<Organization> => {
  const [existing] = await store.query
    .select()
    .from(organization)
    .where(eq(organization.ownerUserID, userID));
  if (existing != null) return existing;
  const [created] = await store.query
    .insert(organization)
    .values({ kind: "personal", name, ownerUserID: userID })
    .onConflictDoNothing()
    .returning();
  if (created != null) return created;
  const [raced] = await store.query
    .select()
    .from(organization)
    .where(eq(organization.ownerUserID, userID));
  return raced;
};

export interface MirrorTeamArgs {
  clerkOrgID: string;
  name: string;
}

/** mirrorTeam records a Clerk organization as a team organization, once. */
export const mirrorTeam = async (
  store: Store,
  { clerkOrgID, name }: MirrorTeamArgs,
): Promise<Organization> => {
  const [created] = await store.query
    .insert(organization)
    .values({ kind: "team", name, clerkOrgID })
    .onConflictDoUpdate({ target: organization.clerkOrgID, set: { name } })
    .returning();
  return created;
};

/**
 * adoptTeam mirrors a Clerk organization into the portal's tables and returns the
 * row, so a license can be issued to an organization the webhook has not delivered.
 */
export const adoptTeam = async (
  store: Store,
  directory: Directory,
  clerkOrgID: string,
): Promise<Organization> => await mirrorTeam(store, await directory.team(clerkOrgID));

export interface Membership {
  userID: string;
  clerkOrgIDs: string[];
}

/** listForMember returns every organization a user belongs to, personal first. */
export const listForMember = async (
  store: Store,
  { userID, clerkOrgIDs }: Membership,
): Promise<Organization[]> => {
  const rows = await store.query
    .select()
    .from(organization)
    .where(
      clerkOrgIDs.length === 0
        ? eq(organization.ownerUserID, userID)
        : or(
            eq(organization.ownerUserID, userID),
            inArray(organization.clerkOrgID, clerkOrgIDs),
          ),
    );
  return rows.sort(
    (a, b) => Number(b.kind === "personal") - Number(a.kind === "personal"),
  );
};

/**
 * organizationsFor returns the organizations a session may act for: the personal one,
 * created on first sight, then every team, mirrored from the session's Clerk
 * memberships so a team created in the Clerk dashboard is usable before its webhook
 * lands.
 */
export const organizationsFor = async (
  store: Store,
  session: Session,
): Promise<Organization[]> => {
  await ensurePersonal(store, { userID: session.userID, name: session.name });
  for (const team of session.teams) await mirrorTeam(store, team);
  return await listForMember(store, session);
};

/**
 * pick chooses the scope a page acts for. `requested`, from `?org=`, selects any of
 * the user's organizations and returns null when it is not one of them. Without it,
 * the `remembered` scope applies while the user still belongs to it. Otherwise a team
 * member acts for their first team and anyone else for their personal organization.
 */
export const pick = (
  organizations: Organization[],
  requested: string | null,
  remembered: string | null,
): Organization | null => {
  if (requested != null) return organizations.find((o) => o.key === requested) ?? null;
  return (
    organizations.find((o) => o.key === remembered) ??
    organizations.find((o) => o.kind === "team") ??
    organizations.find((o) => o.kind === "personal") ??
    null
  );
};

/** listAll returns every organization, for staff. */
export const listAll = async (store: Store): Promise<Organization[]> =>
  await store.query.select().from(organization).orderBy(organization.name);

export const retrieve = async (
  store: Store,
  key: string,
): Promise<Organization | undefined> => {
  const [row] = await store.query
    .select()
    .from(organization)
    .where(eq(organization.key, key));
  return row;
};

export const isMember = (
  org: Organization,
  { userID, clerkOrgIDs }: Membership,
): boolean =>
  org.kind === "personal"
    ? org.ownerUserID === userID
    : org.clerkOrgID != null && clerkOrgIDs.includes(org.clerkOrgID);
