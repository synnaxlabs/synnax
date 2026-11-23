// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schema, type Store } from "@/server/db";
import {
  ADMIN_ROLE,
  type Directory,
  type Member,
  MEMBER_ROLE,
} from "@/server/directory";
import { badRequest, notFound } from "@/server/errors";

/** Team is a team organization with the Clerk organization that backs it. */
export type Team = schema.Organization & { clerkOrgID: string };

/** Change is who changes a team's members, and when. */
export interface Change {
  team: Team;
  /** actor is the Clerk user id of the team admin or staff member making the change. */
  actor: string;
  now: Date;
}

/** requireRole refuses a role that is not admin or member with a 400. */
export const requireRole = (role: string): void => {
  if (role !== ADMIN_ROLE && role !== MEMBER_ROLE) throw badRequest("Choose a role");
};

/**
 * record writes the event of a change and applies the change in Clerk in one
 * transaction. A failed event write applies nothing, and a failed change leaves no
 * event.
 */
const record = async (
  store: Store,
  { team, actor, now }: Change,
  kind: schema.EventKind,
  detail: Record<string, unknown>,
  apply: () => Promise<void>,
): Promise<void> =>
  await store.transact(async (tx) => {
    await tx
      .insert(schema.event)
      .values({ at: now, kind, actor, organization: team.key, detail });
    await apply();
  });

export interface MemberArgs extends Change {
  /** userID is the Clerk user id of the member changed. */
  userID: string;
}

const find = async (
  directory: Directory,
  { team, userID }: MemberArgs,
): Promise<{ roster: Member[]; member: Member }> => {
  const roster = await directory.roster(team.clerkOrgID);
  const member = roster.find((m) => m.userID === userID);
  if (member == null) throw notFound("Member");
  return { roster, member };
};

/** keepAnAdmin refuses a change that would leave the team with no admin. */
const keepAnAdmin = (roster: Member[], member: Member): void => {
  if (member.role !== ADMIN_ROLE) return;
  if (roster.filter((m) => m.role === ADMIN_ROLE).length > 1) return;
  throw badRequest("A team needs an admin. Make another member an admin first.");
};

export interface ChangeRoleArgs extends MemberArgs {
  role: string;
}

/**
 * changeRole gives a member a new role. Nothing changes when the member already has it.
 * @throws {HTTPError} 400 for an unknown role or when it would leave the team with no
 * admin, 404 when the user is not a member.
 */
export const changeRole = async (
  store: Store,
  directory: Directory,
  { role, ...args }: ChangeRoleArgs,
): Promise<void> => {
  requireRole(role);
  const { roster, member } = await find(directory, args);
  if (member.role === role) return;
  keepAnAdmin(roster, member);
  await record(
    store,
    args,
    "change_role",
    { member: member.userID, email: member.email, from: member.role, to: role },
    async () => await directory.setRole(args.team.clerkOrgID, member.userID, role),
  );
};

/**
 * remove takes a member off the team. Their machines keep their seats.
 * @throws {HTTPError} 400 when it would leave the team with no admin, 404 when the
 * user is not a member.
 */
export const remove = async (
  store: Store,
  directory: Directory,
  args: MemberArgs,
): Promise<void> => {
  const { roster, member } = await find(directory, args);
  keepAnAdmin(roster, member);
  await record(
    store,
    args,
    "remove_member",
    { member: member.userID, email: member.email },
    async () => await directory.remove(args.team.clerkOrgID, member.userID),
  );
};
