// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  and,
  countDistinct,
  eq,
  gte,
  inArray,
  lt,
  max,
  min,
  notInArray,
  type SQL,
  sql,
} from "drizzle-orm";

import { type Reader, type Store, type Tx } from "@/server/db/db";
import {
  event,
  invitation,
  type Organization,
  organization,
  referral,
} from "@/server/db/schema";
import { type Account, type Directory, isInternal } from "@/server/directory";
import { notFound } from "@/server/errors";
import { type Machine, machinesFor } from "@/server/license/desktop";
import { mirrorTeams, purge } from "@/server/organization";
import {
  DAY_MS,
  dayOf,
  DESKTOP,
  NONE,
  type Period,
  since,
  startsDay,
  type Usage,
} from "@/server/usage";

const desktopEvents = (...where: (SQL | undefined)[]): SQL | undefined =>
  and(eq(organization.kind, "personal"), inArray(event.kind, DESKTOP), ...where);

/**
 * usageOf reads the Desktop usage of every user, or of `userIDs`, at `now`. A user
 * with no usage is left out.
 */
export const usageOf = async (
  db: Reader,
  now: Date,
  userIDs?: string[],
): Promise<Map<string, Usage>> => {
  const rows = await db
    .select({
      userID: organization.ownerUserID,
      lastLogin:
        sql<Date | null>`max(${event.at}) filter (where ${eq(event.kind, "link")})`.mapWith(
          event.at,
        ),
      lastActive: max(event.at),
      days: sql<number>`count(distinct (${event.at} at time zone 'UTC')::date) filter (where ${gte(event.at, since("month", now))})`.mapWith(
        Number,
      ),
    })
    .from(event)
    .innerJoin(organization, eq(event.organization, organization.key))
    .where(
      desktopEvents(
        userIDs == null ? undefined : inArray(organization.ownerUserID, userIDs),
      ),
    )
    .groupBy(organization.ownerUserID);
  return new Map(
    rows.flatMap(({ userID, ...usage }) => (userID == null ? [] : [[userID, usage]])),
  );
};

/** Entry is one account and how it used Synnax Desktop. */
export interface Entry {
  account: Account;
  usage: Usage;
}

/** list returns every account in Clerk with its usage at `now`. Staff only. */
export const list = async (
  store: Store,
  directory: Directory,
  now: Date,
): Promise<Entry[]> => {
  const [accounts, usage] = await Promise.all([
    directory.users(),
    usageOf(store.query, now),
  ]);
  return accounts.map((account) => ({
    account,
    usage: usage.get(account.userID) ?? NONE,
  }));
};

/** Membership is one team a user belongs to. */
export interface Membership {
  organization: Organization;
  role: string;
}

/** Detail is everything the staff console shows about one account. */
export interface Detail extends Entry {
  /** personal is the user's personal organization, or null before their first visit. */
  personal: Organization | null;
  /** machines are the user's Desktop machines that hold a seat. */
  machines: Machine[];
  teams: Membership[];
}

/**
 * retrieve reads one account, its usage at `now`, its machines, and its teams. Staff
 * only.
 * @throws {HTTPError} 404 when Clerk does not know the user.
 */
export const retrieve = async (
  store: Store,
  directory: Directory,
  userID: string,
  now: Date,
): Promise<Detail> => {
  const account = await directory.account(userID);
  if (account == null) throw notFound("User");
  const [[personal], usage, memberships] = await Promise.all([
    store.query.select().from(organization).where(eq(organization.ownerUserID, userID)),
    usageOf(store.query, now, [userID]),
    directory.memberships(userID),
  ]);
  const [machines, teams] = await Promise.all([
    personal == null ? [] : machinesFor(store.query, personal.key),
    mirrorTeams(store.query, memberships),
  ]);
  return {
    account,
    usage: usage.get(userID) ?? NONE,
    personal: personal ?? null,
    machines,
    teams: teams.map((org, i) => ({ organization: org, role: memberships[i].role })),
  };
};

/** purgeRecords deletes a user's personal organization and claimed referral. */
const purgeRecords = async (tx: Tx, userID: string): Promise<void> => {
  const [personal] = await tx
    .select()
    .from(organization)
    .where(eq(organization.ownerUserID, userID));
  if (personal != null) await purge(tx, personal);
  await tx.delete(referral).where(eq(referral.userID, userID));
};

/**
 * remove deletes a user from Clerk, which takes them off their teams, and deletes their
 * personal organization, the referral they claimed, and the invitations sent to their
 * addresses. Their events on a team stay. Nothing is deleted when Clerk fails.
 */
export const remove = async (
  store: Store,
  directory: Directory,
  userID: string,
): Promise<void> => {
  const addresses = await directory.addresses(userID);
  await store.transact(async (tx) => {
    await purgeRecords(tx, userID);
    await tx.delete(invitation).where(inArray(invitation.email, addresses));
    await directory.deleteAccount(userID);
  });
};

/**
 * forget deletes the personal organization of a user Clerk deleted, with its Desktop
 * licenses, machines, and history, and the referral they claimed. It also deletes what
 * a request in flight wrote after {@link remove}.
 */
export const forget = async (store: Store, userID: string): Promise<void> =>
  await store.transact(async (tx) => await purgeRecords(tx, userID));

export interface ArrivalArgs {
  userID: string;
  /** machine is the name of the machine the user is on. */
  machine: string;
  /** previous is the user's last Desktop use before `now`, or null for none. */
  previous: Date | null;
  now: Date;
}

/** Arrival is a user's first Desktop use of a UTC day. */
export interface Arrival extends Omit<ArrivalArgs, "userID" | "now"> {
  account: Account;
  at: Date;
  /** days counts the user's active days among the last 30, today included. */
  days: number;
}

/**
 * arrival reads a user's Desktop use at `now`, after it is recorded, when it is their
 * first of the UTC day. Returns null for a later use, or when Clerk does not know the
 * user.
 */
export const arrival = async (
  store: Store,
  directory: Directory,
  { userID, machine, previous, now }: ArrivalArgs,
): Promise<Arrival | null> => {
  if (!startsDay(previous, now)) return null;
  const [account, usage] = await Promise.all([
    directory.account(userID),
    usageOf(store.query, now, [userID]),
  ]);
  if (account == null) return null;
  return { account, machine, previous, at: now, days: usage.get(userID)?.days ?? 0 };
};

/** Summary counts how Synnax Desktop was used up to the end of one UTC day. */
export interface Summary {
  /** day is the start of the UTC day. */
  day: Date;
  /** active counts the users active in each period that ends with the day. */
  active: Record<Period, number>;
  /** accounts counts the accounts created on the day. */
  accounts: number;
  /** firstLogins counts the users whose first Desktop login was on the day. */
  firstLogins: number;
}

/**
 * summarize counts the use of the UTC day that holds `day`, leaving out Synnax Labs
 * users.
 */
export const summarize = async (
  store: Store,
  directory: Directory,
  day: Date,
): Promise<Summary> => {
  const start = dayOf(day);
  const end = new Date(start.getTime() + DAY_MS);
  const accounts = await directory.users();
  const external = notInArray(
    organization.ownerUserID,
    accounts.filter((a) => isInternal(a.email)).map((a) => a.userID),
  );
  const activeIn = async (period: Period): Promise<number> => {
    const [row] = await store.query
      .select({ n: countDistinct(organization.ownerUserID) })
      .from(event)
      .innerJoin(organization, eq(event.organization, organization.key))
      .where(
        desktopEvents(external, gte(event.at, since(period, start)), lt(event.at, end)),
      );
    return row.n;
  };
  const [daily, weekly, monthly, firsts] = await Promise.all([
    activeIn("day"),
    activeIn("week"),
    activeIn("month"),
    store.query
      .select({ userID: organization.ownerUserID })
      .from(event)
      .innerJoin(organization, eq(event.organization, organization.key))
      .where(and(eq(organization.kind, "personal"), eq(event.kind, "link"), external))
      .groupBy(organization.ownerUserID)
      .having(and(gte(min(event.at), start), lt(min(event.at), end))),
  ]);
  return {
    day: start,
    active: { day: daily, week: weekly, month: monthly },
    accounts: accounts.filter(
      (a) => !isInternal(a.email) && a.joinedAt >= start && a.joinedAt < end,
    ).length,
    firstLogins: firsts.length,
  };
};
