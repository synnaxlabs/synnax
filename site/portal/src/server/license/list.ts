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
  count,
  desc,
  eq,
  inArray,
  isNull,
  ne,
  type SQL,
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

/**
 * listAll returns every license with its organization and seats in use, newest first.
 * Staff only.
 */
export const listAll = async (db: Reader): Promise<(Owned & Held)[]> =>
  await db
    .select({ license, organization, seats: count(activation.key) })
    .from(license)
    .innerJoin(organization, eq(license.organization, organization.key))
    .leftJoin(
      activation,
      and(eq(activation.license, license.key), isNull(activation.releasedAt)),
    )
    .groupBy(license.key, organization.key)
    .orderBy(desc(license.issuedAt));

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
