// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { errors } from "@synnaxlabs/x";
import { eq, inArray, or, sql } from "drizzle-orm";
import { z } from "zod";

import { type Query, type Store, type Tx } from "@/server/db/db";
import {
  activation,
  event,
  invitation,
  license,
  type Organization,
  organization,
} from "@/server/db/schema";
import { ADMIN_ROLE, type Directory, type Listed } from "@/server/directory";
import { badRequest, HTTPError } from "@/server/errors";
import { invite } from "@/server/invitation";
import { MAX_NAME_LENGTH } from "@/server/license/limits";
import { type Held, listAllHeld } from "@/server/license/list";
import { type Mailer } from "@/server/mail";
import { type Team } from "@/server/member";
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

/**
 * mirrorTeams records each Clerk organization in `teams` as a team organization, once,
 * renaming any whose name changed. It returns the rows in the order given.
 */
export const mirrorTeams = async (
  db: Query | Tx,
  teams: Listed[],
): Promise<Organization[]> => {
  if (teams.length === 0) return [];
  const rows = await db
    .insert(organization)
    .values(
      teams.map(({ clerkOrgID, name }) => ({
        kind: "team" as const,
        name,
        clerkOrgID,
      })),
    )
    .onConflictDoUpdate({
      target: organization.clerkOrgID,
      set: { name: sql`excluded.name` },
    })
    .returning();
  const byClerk = new Map(rows.map((r) => [r.clerkOrgID, r]));
  return teams.map(({ clerkOrgID }) => {
    const row = byClerk.get(clerkOrgID);
    if (row == null) throw new Error(`no row returned for ${clerkOrgID}`);
    return row;
  });
};

export interface RegisterArgs {
  name: string;
  /** email is the address of the first admin, who is invited. */
  email: string;
  actor: string;
  now: Date;
}

const readName = (name: string): string => {
  const trimmed = name.trim();
  if (trimmed === "") throw badRequest("Name the organization");
  if (trimmed.length > MAX_NAME_LENGTH)
    throw badRequest(`The name must be at most ${MAX_NAME_LENGTH} characters`);
  return trimmed;
};

const requireUnique = async (directory: Directory, name: string): Promise<void> => {
  const lower = name.toLowerCase();
  const teams = await directory.teams();
  if (teams.some((t) => t.name.toLowerCase() === lower))
    throw badRequest(`An organization named ${name} exists`);
};

/**
 * register creates a team with no members and invites its first admin. Staff never
 * join it. Nothing is registered when Clerk or the event write fails.
 * @throws {HTTPError} 400 for an empty or taken name or an invalid email, and 502 when
 * the invitation fails. The team stays registered after a failed invitation.
 */
export const register = async (
  store: Store,
  directory: Directory,
  mail: Mailer,
  { name, email, actor, now }: RegisterArgs,
): Promise<Team> => {
  const trimmed = readName(name);
  const address = email.trim();
  if (!z.email().safeParse(address).success)
    throw badRequest("Enter the admin's email address");
  await requireUnique(directory, trimmed);
  const listed = await directory.create(trimmed);
  const team = await store
    .transact(async (tx) => {
      const [row] = await mirrorTeams(tx, [listed]);
      await tx.insert(event).values({
        at: now,
        kind: "register",
        actor,
        organization: row.key,
        detail: { name: trimmed },
      });
      return { ...row, clerkOrgID: listed.clerkOrgID };
    })
    .catch(async (err: unknown) => {
      // Deleting the Clerk team frees its name, so a retry registers it again.
      await directory.delete(listed.clerkOrgID);
      throw errors.fromUnknown(err);
    });
  try {
    await invite(store, directory, mail, {
      team,
      actor,
      now,
      email: address,
      role: ADMIN_ROLE,
    });
  } catch (err) {
    throw new HTTPError(
      502,
      `${trimmed} is registered, but its invitation failed. Invite the admin from ` +
        "its Members tab.",
      undefined,
      { cause: err },
    );
  }
  return team;
};

export interface RenameArgs {
  team: Team;
  name: string;
  actor: string;
  now: Date;
}

/**
 * rename changes a team's name in Clerk and in the portal, and records it. Nothing
 * changes when Clerk or the event write fails.
 * @throws {HTTPError} 400 for an empty or taken name.
 */
export const rename = async (
  store: Store,
  directory: Directory,
  { team, name, actor, now }: RenameArgs,
): Promise<Organization> => {
  const trimmed = readName(name);
  if (trimmed === team.name) return team;
  if (trimmed.toLowerCase() !== team.name.toLowerCase())
    await requireUnique(directory, trimmed);
  return await store.transact(async (tx) => {
    const [row] = await tx
      .update(organization)
      .set({ name: trimmed })
      .where(eq(organization.key, team.key))
      .returning();
    await tx.insert(event).values({
      at: now,
      kind: "rename_organization",
      actor,
      organization: team.key,
      detail: { name: trimmed, from: team.name },
    });
    await directory.rename(team.clerkOrgID, trimmed);
    return row;
  });
};

/**
 * remove deletes a team in Clerk with everything the portal holds for it: its
 * licenses and their seats, its invitations, and its history. Nothing is deleted when
 * Clerk fails.
 */
export const remove = async (
  store: Store,
  directory: Directory,
  team: Team,
): Promise<void> =>
  await store.transact(async (tx) => {
    // Locks the licenses, then the team, in the order activate takes them, so no record
    // joins the team during the delete.
    const licenses = await tx
      .select({ key: license.key })
      .from(license)
      .where(eq(license.organization, team.key))
      .for("update");
    await tx
      .select({ key: organization.key })
      .from(organization)
      .where(eq(organization.key, team.key))
      .for("update");
    await tx.delete(event).where(eq(event.organization, team.key));
    await tx.delete(activation).where(
      inArray(
        activation.license,
        licenses.map((l) => l.key),
      ),
    );
    await tx.delete(license).where(eq(license.organization, team.key));
    await tx.delete(invitation).where(eq(invitation.organization, team.key));
    await tx.delete(organization).where(eq(organization.key, team.key));
    await directory.delete(team.clerkOrgID);
  });

/**
 * adoptTeam mirrors a Clerk organization into the portal's tables and returns the
 * row, so a license can be issued to an organization the webhook has not delivered.
 *
 * @throws {HTTPError} 400 when Clerk does not know the organization.
 */
export const adoptTeam = async (
  store: Store,
  directory: Directory,
  clerkOrgID: string,
): Promise<Organization> => {
  const team = await directory.team(clerkOrgID);
  if (team == null) throw badRequest(`Clerk has no organization ${clerkOrgID}`);
  const [row] = await mirrorTeams(store.query, [team]);
  return row;
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
  await mirrorTeams(store.query, session.teams);
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

/** Summary is one team as the staff console lists it. */
export interface Summary {
  organization: Organization;
  members: number;
  licenses: Held[];
}

/**
 * listTeams returns every team in Clerk with its member count and licenses, recording
 * any team the portal has not seen. Staff only.
 */
export const listTeams = async (
  store: Store,
  directory: Directory,
): Promise<Summary[]> => {
  const teams = await directory.teams();
  const rows = await mirrorTeams(store.query, teams);
  const held = await listAllHeld(store.query);
  return rows.map((org, i) => ({
    organization: org,
    members: teams[i].members,
    licenses: held.filter((h) => h.license.organization === org.key),
  }));
};

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
