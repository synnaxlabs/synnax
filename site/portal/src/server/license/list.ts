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

import { type Reader, schema } from "@/server/db";
import { deny } from "@/server/license/deny";
import { EXPIRING, type Status } from "@/server/license/status";

/** Held is a license beside the number of machines holding one of its seats. */
export interface Held {
  license: schema.License;
  seats: number;
}

/** Owned is a license beside the organization it belongs to. */
export interface Owned {
  license: schema.License;
  organization: schema.Organization;
}

const listHeld = async (db: Reader, where?: SQL): Promise<Held[]> =>
  await db
    .select({ license: schema.license, seats: count(schema.activation.key) })
    .from(schema.license)
    .leftJoin(
      schema.activation,
      and(
        eq(schema.activation.license, schema.license.key),
        isNull(schema.activation.releasedAt),
      ),
    )
    .where(where)
    .groupBy(schema.license.key)
    .orderBy(desc(schema.license.issuedAt));

/** listForOrganization returns an organization's licenses, newest first. */
export const listForOrganization = async (db: Reader, org: string): Promise<Held[]> =>
  await listHeld(db, eq(schema.license.organization, org));

/** listAllHeld returns every license and its seats in use, newest first. Staff only. */
export const listAllHeld = async (db: Reader): Promise<Held[]> => await listHeld(db);

const PAGE_SIZE = 50;

export interface Query {
  /** search keeps the licenses whose label, organization name, or key holds it. */
  search?: string;
  status?: Status;
  /** after is the last license on the page before this one. */
  after?: Cursor;
}

/** Cursor is the last license on a page, which the page after it starts from. */
export interface Cursor {
  key: string;
  /**
   * expiresAt is the license's expiry when its page was read. The expiring status
   * needs it, since a renewal can change the expiry before the next page is read.
   */
  expiresAt?: Date;
}

export interface Page {
  entries: (Owned & Held)[];
  /** total counts every license that matches the search and status. */
  total: number;
  /** next is the `after` that reads the page after this one, or null on the last. */
  next: Cursor | null;
}

const live = isNull(schema.license.revokedAt);

/** STATUS_WHERE matches the licenses that statusOf reads as each status. */
const STATUS_WHERE: Record<Status, (now: Date, soon: Date) => SQL | undefined> = {
  revoked: () => isNotNull(schema.license.revokedAt),
  expired: (now) => and(live, lte(schema.license.expiresAt, now)),
  expiring: (now, soon) =>
    and(
      live,
      gt(schema.license.expiresAt, now),
      lte(schema.license.expiresAt, soon),
      ne(schema.license.edition, "desktop"),
    ),
  active: (now, soon) =>
    and(
      live,
      or(
        isNull(schema.license.expiresAt),
        gt(schema.license.expiresAt, soon),
        and(gt(schema.license.expiresAt, now), eq(schema.license.edition, "desktop")),
      ),
    ),
};

const matches = ({ search = "", status }: Query, now: Date): SQL | undefined => {
  const needle = search.trim();
  return and(
    status == null
      ? undefined
      : STATUS_WHERE[status](now, new Date(now.getTime() + EXPIRING)),
    needle === ""
      ? undefined
      : sql`strpos(lower(${schema.license.label} || ' ' ||
          ${schema.organization.name} || ' ' || ${schema.license.key}::text),
          lower(${needle})) > 0`,
  );
};

const after = (cursor: Cursor | undefined, soonest: boolean): SQL | undefined => {
  if (cursor == null) return undefined;
  if (!soonest)
    return sql`(${schema.license.issuedAt}, ${schema.license.key}) <
      (select c.issued_at, c.key from ${schema.license} c where c.key = ${cursor.key})`;
  if (cursor.expiresAt == null)
    throw new Error("A cursor in the expiring order needs an expiry");
  return sql`(${schema.license.expiresAt}, ${schema.license.key}) >
    (${cursor.expiresAt.toISOString()}::timestamptz, ${cursor.key}::uuid)`;
};

/**
 * listPage reads a page of every organization's licenses with their organizations and
 * seats in use. Licenses are newest first, or soonest to expire first under the
 * expiring status. Staff only.
 */
export const listPage = async (db: Reader, query: Query, now: Date): Promise<Page> => {
  const where = matches(query, now);
  const soonest = query.status === "expiring";
  const rows = await db
    .select({
      license: schema.license,
      organization: schema.organization,
      seats: count(schema.activation.key),
    })
    .from(schema.license)
    .innerJoin(
      schema.organization,
      eq(schema.license.organization, schema.organization.key),
    )
    .leftJoin(
      schema.activation,
      and(
        eq(schema.activation.license, schema.license.key),
        isNull(schema.activation.releasedAt),
      ),
    )
    .where(and(where, after(query.after, soonest)))
    .groupBy(schema.license.key, schema.organization.key)
    .orderBy(
      ...(soonest
        ? [asc(schema.license.expiresAt), asc(schema.license.key)]
        : [desc(schema.license.issuedAt), desc(schema.license.key)]),
    )
    .limit(PAGE_SIZE + 1);
  const [{ total }] = await db
    .select({ total: count() })
    .from(schema.license)
    .innerJoin(
      schema.organization,
      eq(schema.license.organization, schema.organization.key),
    )
    .where(where);
  const entries = rows.slice(0, PAGE_SIZE);
  if (rows.length <= PAGE_SIZE) return { entries, total, next: null };
  const { key, expiresAt } = entries[entries.length - 1].license;
  const next: Cursor = { key };
  if (soonest && expiresAt != null) next.expiresAt = expiresAt;
  return { entries, total, next };
};

const listActivatableWhere = async (
  db: Reader,
  where: SQL,
  now: Date,
): Promise<(Owned & Held)[]> => {
  const rows = await db
    .select({
      license: schema.license,
      organization: schema.organization,
      seats: count(schema.activation.key),
    })
    .from(schema.license)
    .innerJoin(
      schema.organization,
      eq(schema.license.organization, schema.organization.key),
    )
    .leftJoin(
      schema.activation,
      and(
        eq(schema.activation.license, schema.license.key),
        isNull(schema.activation.releasedAt),
      ),
    )
    .where(
      and(
        where,
        isNull(schema.license.revokedAt),
        ne(schema.license.edition, "desktop"),
      ),
    )
    .groupBy(schema.license.key, schema.organization.key)
    .orderBy(desc(schema.license.issuedAt));
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
  return await listActivatableWhere(
    db,
    inArray(schema.license.organization, orgs),
    now,
  );
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
  const [found] = await listActivatableWhere(db, eq(schema.license.key, key), now);
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
): Promise<schema.Activation | undefined> => {
  if (licenses.length === 0) return undefined;
  const [seat] = await db
    .select()
    .from(schema.activation)
    .where(
      and(
        inArray(schema.activation.license, licenses),
        isNull(schema.activation.releasedAt),
        arrayOverlaps(schema.activation.fingerprint, fingerprint),
      ),
    )
    .orderBy(desc(schema.activation.lastSeen))
    .limit(1);
  return seat;
};
