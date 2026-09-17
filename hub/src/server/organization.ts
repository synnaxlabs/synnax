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

export interface EnsurePersonalArgs {
  userID: string;
  name: string;
}

/**
 * ensurePersonal returns the user's personal organization, creating it on first
 * sight. The Clerk webhook creates it eagerly; this covers a user who signed in before
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

/** isMember reports whether a membership covers an organization. */
export const isMember = (
  org: Organization,
  { userID, clerkOrgIDs }: Membership,
): boolean =>
  org.kind === "personal"
    ? org.ownerUserID === userID
    : org.clerkOrgID != null && clerkOrgIDs.includes(org.clerkOrgID);
