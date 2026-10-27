// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { and, count, desc, eq, inArray, isNull, ne, type SQL } from "drizzle-orm";

import { type Reader, schema } from "@/server/db";
import { deny } from "@/server/license/deny";

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

/**
 * listAll returns every license with its organization and seats in use, newest first.
 * Staff only.
 */
export const listAll = async (db: Reader): Promise<(Owned & Held)[]> =>
  await db
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
    .groupBy(schema.license.key, schema.organization.key)
    .orderBy(desc(schema.license.issuedAt));

/**
 * listActivatable returns the licenses of `orgs` that a machine can activate against
 * at `now`, newest first.
 */
export const listActivatable = async (
  db: Reader,
  orgs: string[],
  now: Date,
): Promise<Owned[]> => {
  if (orgs.length === 0) return [];
  const rows = await db
    .select({ license: schema.license, organization: schema.organization })
    .from(schema.license)
    .innerJoin(
      schema.organization,
      eq(schema.license.organization, schema.organization.key),
    )
    .where(
      and(
        inArray(schema.license.organization, orgs),
        isNull(schema.license.revokedAt),
        ne(schema.license.edition, "desktop"),
      ),
    )
    .orderBy(desc(schema.license.issuedAt));
  return rows.filter((r) => deny(r.license, now) == null);
};
