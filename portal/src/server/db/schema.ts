// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  index,
  integer,
  jsonb,
  pgTable,
  serial,
  text,
  timestamp,
  uuid,
} from "drizzle-orm/pg-core";

export const ORGANIZATION_KINDS = ["personal", "team"] as const;
export type OrganizationKind = (typeof ORGANIZATION_KINDS)[number];

export const EDITIONS = ["desktop", "enterprise"] as const;
export type Edition = (typeof EDITIONS)[number];

export const TERMS = ["subscription", "perpetual"] as const;
export type Term = (typeof TERMS)[number];

export const EVENT_KINDS = [
  "issue",
  "activate",
  "activate_denied",
  "token",
  "release",
  "revoke",
  "expiry_notice",
] as const;
export type EventKind = (typeof EVENT_KINDS)[number];

export const organization = pgTable("organization", {
  key: uuid("key").primaryKey().defaultRandom(),
  kind: text("kind", { enum: ORGANIZATION_KINDS }).notNull(),
  name: text("name").notNull(),
  clerkOrgID: text("clerk_org_id").unique(),
  ownerUserID: text("owner_user_id").unique(),
  createdAt: timestamp("created_at", { withTimezone: true }).notNull().defaultNow(),
});
export type Organization = typeof organization.$inferSelect;

export const license = pgTable(
  "license",
  {
    key: uuid("key").primaryKey().defaultRandom(),
    organization: uuid("organization")
      .notNull()
      .references(() => organization.key),
    edition: text("edition", { enum: EDITIONS }).notNull(),
    term: text("term", { enum: TERMS }).notNull(),
    nodes: integer("nodes").notNull(),
    channels: integer("channels").notNull().default(0),
    expiresAt: timestamp("expires_at", { withTimezone: true }),
    maxVersion: text("max_version"),
    label: text("label").notNull().default(""),
    issuedBy: text("issued_by").notNull(),
    issuedAt: timestamp("issued_at", { withTimezone: true }).notNull().defaultNow(),
    revokedAt: timestamp("revoked_at", { withTimezone: true }),
  },
  (t) => [index("license_organization_idx").on(t.organization)],
);
export type License = typeof license.$inferSelect;

export const activation = pgTable(
  "activation",
  {
    key: uuid("key").primaryKey().defaultRandom(),
    license: uuid("license")
      .notNull()
      .references(() => license.key),
    fingerprint: text("fingerprint").array().notNull(),
    firstSeen: timestamp("first_seen", { withTimezone: true }).notNull().defaultNow(),
    lastSeen: timestamp("last_seen", { withTimezone: true }).notNull().defaultNow(),
    releasedAt: timestamp("released_at", { withTimezone: true }),
  },
  (t) => [index("activation_license_idx").on(t.license)],
);
export type Activation = typeof activation.$inferSelect;

export const event = pgTable(
  "event",
  {
    key: serial("key").primaryKey(),
    at: timestamp("at", { withTimezone: true }).notNull().defaultNow(),
    kind: text("kind", { enum: EVENT_KINDS }).notNull(),
    actor: text("actor").notNull(),
    organization: uuid("organization").references(() => organization.key),
    license: uuid("license").references(() => license.key),
    activation: uuid("activation").references(() => activation.key),
    detail: jsonb("detail").notNull().default({}),
  },
  (t) => [
    index("event_license_idx").on(t.license),
    index("event_actor_at_idx").on(t.actor, t.at),
    index("event_organization_at_idx").on(t.organization, t.at),
  ],
);
export type Event = typeof event.$inferSelect;
