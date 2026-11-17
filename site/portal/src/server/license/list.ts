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
  arrayOverlaps,
  asc,
  count,
  desc,
  eq,
  gt,
  inArray,
  isNotNull,
  isNull,
  lte,
  ne,
  or,
  type SQL,
  sql,
} from "drizzle-orm";

import { type Reader } from "@/server/db/db";
import {
  type Activation,
  activation,
  type License,
  license,
  type Organization,
  organization,
} from "@/server/db/schema";
import { deny } from "@/server/license/deny";
import { EXPIRING, type Status } from "@/server/license/status";

/** Held is a license beside the number of machines holding one of its seats. */
export interface Held {
  license: License;
  seats: number;
}

/** Owned is a license beside the organization it belongs to. */
export interface Owned {
  license: License;
  organization: Organization;
}

const listHeld = async (db: Reader, where?: SQL): Promise<Held[]> =>
  await db
    .select({ license, seats: count(activation.key) })
    .from(license)
    .leftJoin(
      activation,
      and(eq(activation.license, license.key), isNull(activation.releasedAt)),
    )
    .where(where)
    .groupBy(license.key)
    .orderBy(desc(license.issuedAt));

/** listForOrganization returns an organization's licenses, newest first. */
export const listForOrganization = async (db: Reader, org: string): Promise<Held[]> =>
  await listHeld(db, eq(license.organization, org));

/** listAllHeld returns every license and its seats in use, newest first. Staff only. */
export const listAllHeld = async (db: Reader): Promise<Held[]> => await listHeld(db);

const PAGE_SIZE = 50;

export interface Query {
  /** search keeps the licenses whose label, organization name, or key holds it. */
  search?: string;
  status?: Status;
  /** after is the key of the last license on the page before this one. */
  after?: string;
}

export interface Page {
  entries: (Owned & Held)[];
  /** total counts every license that matches the search and status. */
  total: number;
  /** next is the `after` that reads the page after this one, or null on the last. */
  next: string | null;
}

const live = isNull(license.revokedAt);

/** STATUS_WHERE matches the licenses that statusOf reads as each status. */
const STATUS_WHERE: Record<Status, (now: Date, soon: Date) => SQL | undefined> = {
  revoked: () => isNotNull(license.revokedAt),
  expired: (now) => and(live, lte(license.expiresAt, now)),
  expiring: (now, soon) =>
    and(
      live,
      gt(license.expiresAt, now),
      lte(license.expiresAt, soon),
      ne(license.edition, "desktop"),
    ),
  active: (now, soon) =>
    and(
      live,
      or(
        isNull(license.expiresAt),
        gt(license.expiresAt, soon),
        and(gt(license.expiresAt, now), eq(license.edition, "desktop")),
      ),
    ),
};

const matches = ({ search = "", status }: Query, now: Date): SQL | undefined => {
  const needle = search.trim().toLowerCase();
  return and(
    status == null
      ? undefined
      : STATUS_WHERE[status](now, new Date(now.getTime() + EXPIRING)),
    needle === ""
      ? undefined
      : sql`strpos(lower(${license.label} || ' ' || ${organization.name} || ' ' ||
          ${license.key}::text), ${needle}) > 0`,
  );
};

/**
 * listPage reads a page of every organization's licenses with their organizations and
 * seats in use. Licenses are newest first, or soonest to expire first under the
 * expiring status. Staff only.
 */
export const listPage = async (db: Reader, query: Query, now: Date): Promise<Page> => {
  const where = matches(query, now);
  const soonest = query.status === "expiring";
  // Expiring licenses all have an expiry, so the row comparison never meets a null.
  const cursor =
    query.after == null
      ? undefined
      : soonest
        ? sql`(${license.expiresAt}, ${license.key}) > (select c.expires_at, c.key
            from ${license} c where c.key = ${query.after})`
        : sql`(${license.issuedAt}, ${license.key}) < (select c.issued_at, c.key
            from ${license} c where c.key = ${query.after})`;
  const rows = await db
    .select({ license, organization, seats: count(activation.key) })
    .from(license)
    .innerJoin(organization, eq(license.organization, organization.key))
    .leftJoin(
      activation,
      and(eq(activation.license, license.key), isNull(activation.releasedAt)),
    )
    .where(and(where, cursor))
    .groupBy(license.key, organization.key)
    .orderBy(
      ...(soonest
        ? [asc(license.expiresAt), asc(license.key)]
        : [desc(license.issuedAt), desc(license.key)]),
    )
    .limit(PAGE_SIZE + 1);
  const [{ total }] = await db
    .select({ total: count() })
    .from(license)
    .innerJoin(organization, eq(license.organization, organization.key))
    .where(where);
  const entries = rows.slice(0, PAGE_SIZE);
  return {
    entries,
    total,
    next: rows.length > PAGE_SIZE ? entries[entries.length - 1].license.key : null,
  };
};

const listActivatableWhere = async (
  db: Reader,
  where: SQL,
  now: Date,
): Promise<(Owned & Held)[]> => {
  const rows = await db
    .select({ license, organization, seats: count(activation.key) })
    .from(license)
    .innerJoin(organization, eq(license.organization, organization.key))
    .leftJoin(
      activation,
      and(eq(activation.license, license.key), isNull(activation.releasedAt)),
    )
    .where(and(where, isNull(license.revokedAt), ne(license.edition, "desktop")))
    .groupBy(license.key, organization.key)
    .orderBy(desc(license.issuedAt));
  return rows.filter((r) => deny(r.license, now) == null);
};

/**
 * listActivatable returns the licenses of `orgs` that a machine can activate against
 * at `now`, with their seats in use, newest first.
 */
export const listActivatable = async (
  db: Reader,
  orgs: string[],
  now: Date,
): Promise<(Owned & Held)[]> => {
  if (orgs.length === 0) return [];
  return await listActivatableWhere(db, inArray(license.organization, orgs), now);
};

/**
 * findActivatable returns the license `key` with its organization and seats in use,
 * or undefined when a machine cannot activate against it at `now`. Staff only.
 */
export const findActivatable = async (
  db: Reader,
  key: string,
  now: Date,
): Promise<(Owned & Held) | undefined> => {
  const [found] = await listActivatableWhere(db, eq(license.key, key), now);
  return found;
};

/**
 * findSeat returns the seat a machine holds on one of `licenses`: an activation that is
 * not released and shares a hash with `fingerprint`.
 */
export const findSeat = async (
  db: Reader,
  licenses: string[],
  fingerprint: string[],
): Promise<Activation | undefined> => {
  if (licenses.length === 0) return undefined;
  const [seat] = await db
    .select()
    .from(activation)
    .where(
      and(
        inArray(activation.license, licenses),
        isNull(activation.releasedAt),
        arrayOverlaps(activation.fingerprint, fingerprint),
      ),
    )
    .orderBy(desc(activation.lastSeen))
    .limit(1);
  return seat;
};
