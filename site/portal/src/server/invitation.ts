// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { and, desc, eq, gt, inArray } from "drizzle-orm";
import { z } from "zod";

import { type Reader, type Store } from "@/server/db/db";
import {
  event,
  type Invitation,
  invitation,
  type Organization,
  organization,
} from "@/server/db/schema";
import { type Directory, roleName } from "@/server/directory";
import { badRequest, forbidden, notFound } from "@/server/errors";
import { type Mailer, type Notice } from "@/server/mail";
import { type Change, requireRole, type Team } from "@/server/member";
import { type Session } from "@/server/session";

const DAY_MS = 24 * 60 * 60 * 1000;

/** OPEN_MS is how long an invitation stays open after it is sent. */
export const OPEN_MS = 30 * DAY_MS;

/** Invited is an invitation with the team it joins. */
export interface Invited {
  invitation: Invitation;
  team: Team;
}

const isKey = (key: string): boolean => z.uuid().safeParse(key).success;

const invited = (row: { invitation: Invitation; organization: Organization }) => {
  const { invitation: inv, organization: org } = row;
  if (org.clerkOrgID == null)
    throw new Error(`team ${org.key} has no Clerk organization`);
  return { invitation: inv, team: { ...org, clerkOrgID: org.clerkOrgID } };
};

const joined = (db: Reader) =>
  db
    .select({ invitation, organization })
    .from(invitation)
    .innerJoin(organization, eq(invitation.organization, organization.key));

/** retrieve returns an invitation with its team, or null for an unknown key. */
export const retrieve = async (db: Reader, key: string): Promise<Invited | null> => {
  if (!isKey(key)) return null;
  const [row] = await joined(db).where(eq(invitation.key, key));
  return row == null ? null : invited(row);
};

/** addressedTo returns the open invitations to any of `addresses`, newest first. */
export const addressedTo = async (
  db: Reader,
  addresses: string[],
  now: Date,
): Promise<Invited[]> => {
  const rows = await joined(db)
    .where(and(inArray(invitation.email, addresses), gt(invitation.expiresAt, now)))
    .orderBy(desc(invitation.sentAt));
  return rows.map(invited);
};

/** list returns a team's invitations, expired ones included, newest first. */
export const list = async (db: Reader, team: string): Promise<Invitation[]> =>
  await db
    .select()
    .from(invitation)
    .where(eq(invitation.organization, team))
    .orderBy(desc(invitation.sentAt));

const notice = (inv: Invitation, team: Team, inviter: string): Notice => ({
  subject: `Join ${team.name} on Synnax`,
  heading: `Join ${team.name} on Synnax`,
  paragraphs: [
    `${inviter} invited you to join ${team.name} in the Synnax portal.`,
    `Log in or create an account with ${inv.email} to accept.`,
  ],
  facts: [
    { label: "Team", value: team.name },
    { label: "Role", value: roleName(inv.role) },
    { label: "Expires", value: inv.expiresAt.toISOString().slice(0, 10) },
  ],
  action: { label: "Accept invitation", href: `/invitations/${inv.key}` },
  reason: "If you did not expect this invitation, you can ignore it.",
});

export interface InviteArgs extends Change {
  email: string;
  role: string;
}

/**
 * invite emails an invitation to join the team. Inviting an address that has an
 * invitation sends it again with the new role and a new expiry. Nothing is saved when
 * the email fails.
 * @throws {HTTPError} 400 for an invalid email, an unknown role, or a team member.
 */
export const invite = async (
  store: Store,
  directory: Directory,
  mail: Mailer,
  { email, role, team, actor, now }: InviteArgs,
): Promise<Invitation> => {
  const address = email.trim().toLowerCase();
  if (!z.email().safeParse(address).success) throw badRequest("Enter an email address");
  requireRole(role);
  const roster = await directory.roster(team.clerkOrgID);
  if (roster.some((m) => m.email.toLowerCase() === address))
    throw badRequest(`${address} is already on ${team.name}`);
  // Staff are not on the teams they invite to, so their email names Synnax Labs.
  const inviter = roster.find((m) => m.userID === actor)?.name ?? "Synnax Labs";
  const expiresAt = new Date(now.getTime() + OPEN_MS);
  return await store.transact(async (tx) => {
    const [row] = await tx
      .insert(invitation)
      .values({ organization: team.key, email: address, role, sentAt: now, expiresAt })
      .onConflictDoUpdate({
        target: [invitation.organization, invitation.email],
        set: { role, sentAt: now, expiresAt },
      })
      .returning();
    await tx.insert(event).values({
      at: now,
      kind: "invite",
      actor,
      organization: team.key,
      detail: { email: address, role },
    });
    await mail.send({ to: [address], ...notice(row, team, inviter) });
    return row;
  });
};

export interface RevokeArgs extends Change {
  key: string;
}

/**
 * revoke deletes an invitation, so its link no longer joins the team.
 * @throws {HTTPError} 404 when the team has no such invitation.
 */
export const revoke = async (
  store: Store,
  { key, team, actor, now }: RevokeArgs,
): Promise<void> => {
  if (!isKey(key)) throw notFound("Invitation");
  await store.transact(async (tx) => {
    const [row] = await tx
      .delete(invitation)
      .where(and(eq(invitation.key, key), eq(invitation.organization, team.key)))
      .returning();
    if (row == null) throw notFound("Invitation");
    await tx.insert(event).values({
      at: now,
      kind: "revoke_invitation",
      actor,
      organization: team.key,
      detail: { email: row.email, role: row.role },
    });
  });
};

export interface AcceptArgs {
  key: string;
  session: Session;
  now: Date;
}

/**
 * accept puts the logged-in user on the invitation's team with its role, then deletes
 * the invitation. A user already on the team keeps their role.
 * @throws {HTTPError} 404 for an unknown invitation, 400 for an expired one, 403 for
 * one to an address the user has not verified, and 400 when Clerk refuses.
 */
export const accept = async (
  store: Store,
  directory: Directory,
  { key, session, now }: AcceptArgs,
): Promise<Team> => {
  const found = await retrieve(store.query, key);
  if (found == null) throw notFound("Invitation");
  const { invitation: inv, team } = found;
  if (inv.expiresAt <= now)
    throw badRequest("This invitation expired. Ask a team admin to send it again.");
  const addresses = await directory.addresses(session.userID);
  if (!addresses.includes(inv.email))
    throw forbidden(
      `This invitation is for ${inv.email}. Log in with that email to accept it.`,
    );
  await store.transact(async (tx) => {
    // Deleting first makes the invitation single use when two accepts race.
    const [row] = await tx
      .delete(invitation)
      .where(eq(invitation.key, inv.key))
      .returning();
    if (row == null) throw notFound("Invitation");
    await tx.insert(event).values({
      at: now,
      kind: "accept_invitation",
      actor: session.userID,
      organization: team.key,
      detail: { email: row.email, role: row.role },
    });
    if (!session.clerkOrgIDs.includes(team.clerkOrgID))
      await directory.add(team.clerkOrgID, session.userID, row.role);
  });
  return team;
};
