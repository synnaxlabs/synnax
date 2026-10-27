// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { and, desc, eq, inArray, lt, type SQL } from "drizzle-orm";

import { type Store } from "@/server/db/db";
import {
  activation,
  type Event,
  event,
  type EventKind,
  license,
  organization,
} from "@/server/db/schema";

/** listForLicense returns a license's audit events, newest first. */
export const listForLicense = async (
  store: Store,
  licenseKey: string,
  limit = 50,
): Promise<Event[]> =>
  await store.query
    .select()
    .from(event)
    .where(eq(event.license, licenseKey))
    .orderBy(desc(event.at))
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
  members: ["invite", "revoke_invitation", "change_role", "remove_member"],
  organizations: ["register", "rename_organization"],
} as const satisfies Record<string, EventKind[]>;

export type Category = keyof typeof CATEGORIES;

export const isCategory = (value: string): value is Category =>
  Object.hasOwn(CATEGORIES, value);

/** Entry is an event with the names of the records it concerns. */
export interface Entry {
  event: Event;
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
  if (org != null) where.push(eq(event.organization, org));
  if (category != null) where.push(inArray(event.kind, [...CATEGORIES[category]]));
  if (actors != null) where.push(inArray(event.actor, actors));
  if (before != null) where.push(lt(event.key, before));
  const rows = await store.query
    .select({
      event,
      organization: organization.name,
      license: license.label,
      machineName: activation.name,
      fingerprint: activation.fingerprint,
    })
    .from(event)
    .leftJoin(organization, eq(event.organization, organization.key))
    .leftJoin(license, eq(event.license, license.key))
    .leftJoin(activation, eq(event.activation, activation.key))
    .where(and(...where))
    .orderBy(desc(event.key))
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
