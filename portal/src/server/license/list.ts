// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { and, count, desc, eq, inArray, isNull } from "drizzle-orm";

import { type Reader } from "@/server/db/db";
import {
  activation,
  type License,
  license,
  type Organization,
  organization,
} from "@/server/db/schema";

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

/** listForOrganization returns an organization's licenses, newest first. */
export const listForOrganization = async (db: Reader, org: string): Promise<Held[]> =>
  await db
    .select({ license, seats: count(activation.key) })
    .from(license)
    .leftJoin(
      activation,
      and(eq(activation.license, license.key), isNull(activation.releasedAt)),
    )
    .where(eq(license.organization, org))
    .groupBy(license.key)
    .orderBy(desc(license.issuedAt));

/** listAll returns every license with its organization, newest first. Staff only. */
export const listAll = async (db: Reader): Promise<Owned[]> =>
  await db
    .select({ license, organization })
    .from(license)
    .innerJoin(organization, eq(license.organization, organization.key))
    .orderBy(desc(license.issuedAt));

/**
 * listActivatable returns the licenses of `orgs` that a machine can still activate
 * against, newest first.
 */
export const listActivatable = async (db: Reader, orgs: string[]): Promise<Owned[]> => {
  if (orgs.length === 0) return [];
  return await db
    .select({ license, organization })
    .from(license)
    .innerJoin(organization, eq(license.organization, organization.key))
    .where(and(inArray(license.organization, orgs), isNull(license.revokedAt)))
    .orderBy(desc(license.issuedAt));
};
