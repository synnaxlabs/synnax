// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { and, desc, eq, inArray, lt, type SQL } from "drizzle-orm";

import { schema, type Store } from "@/server/db";

/** listForLicense returns a license's audit events, newest first. */
export const listForLicense = async (
  store: Store,
  licenseKey: string,
  limit = 50,
): Promise<schema.Event[]> =>
  await store.query
    .select()
    .from(schema.event)
    .where(eq(schema.event.license, licenseKey))
    .orderBy(desc(schema.event.at))
    .limit(limit);

/** CATEGORIES group event kinds the way the staff activity log filters them. */
const CATEGORIES = {
  licenses: ["issue", "amend", "revoke", "restore", "download", "expiry_notice"],
  machines: [
    "activate",
    "activate_denied",
    "release",
    "rename",
    "link",
    "renew",
    "unlink",
  ],
  members: [
    "invite",
    "revoke_invitation",
    "accept_invitation",
    "change_role",
    "remove_member",
  ],
  organizations: ["register", "rename_organization"],
} as const satisfies Record<string, schema.EventKind[]>;

export type Category = keyof typeof CATEGORIES;

export const isCategory = (value: string): value is Category =>
  Object.hasOwn(CATEGORIES, value);

/** Entry is an event with the names of the records it concerns. */
export interface Entry {
  event: schema.Event;
  /** organization is the name of the organization, or null for none. */
  organization: string | null;
  /** license is the label of the license, or null for none. */
  license: string | null;
  /** machine is the activation the event concerns, or null for none. */
  machine: { name: string | null; fingerprint: string[] } | null;
}

export interface Query {
  organization?: string;
  category?: Category;
  /** actors keeps only the events these Clerk user ids made. */
  actors?: string[];
  /** before keeps only the events older than this event key. */
  before?: number;
  limit?: number;
}

export interface Page {
  /** entries are newest first. */
  entries: Entry[];
  /** next is the `before` that reads the page after this one, or null for the last. */
  next: number | null;
}

/** list reads a page of audit events, newest first. Staff only. */
export const list = async (
  store: Store,
  { organization: org, category, actors, before, limit = 50 }: Query,
): Promise<Page> => {
  const where: SQL[] = [];
  if (org != null) where.push(eq(schema.event.organization, org));
  if (category != null)
    where.push(inArray(schema.event.kind, [...CATEGORIES[category]]));
  if (actors != null) where.push(inArray(schema.event.actor, actors));
  if (before != null) where.push(lt(schema.event.key, before));
  const rows = await store.query
    .select({
      event: schema.event,
      organization: schema.organization.name,
      license: schema.license.label,
      machineName: schema.activation.name,
      fingerprint: schema.activation.fingerprint,
    })
    .from(schema.event)
    .leftJoin(
      schema.organization,
      eq(schema.event.organization, schema.organization.key),
    )
    .leftJoin(schema.license, eq(schema.event.license, schema.license.key))
    .leftJoin(schema.activation, eq(schema.event.activation, schema.activation.key))
    .where(and(...where))
    .orderBy(desc(schema.event.key))
    .limit(limit + 1);
  const entries = rows.slice(0, limit).map((r) => ({
    event: r.event,
    organization: r.organization,
    license: r.license,
    machine:
      r.fingerprint == null
        ? null
        : { name: r.machineName, fingerprint: r.fingerprint },
  }));
  return {
    entries,
    next: rows.length > limit ? entries[entries.length - 1].event.key : null,
  };
};
