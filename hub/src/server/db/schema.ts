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
  "thread",
  "feedback",
] as const;
export type EventKind = (typeof EVENT_KINDS)[number];

export const organization = pgTable("organization", {
  key: uuid("key").primaryKey().defaultRandom(),
  kind: text("kind", { enum: ORGANIZATION_KINDS }).notNull(),
  name: text("name").notNull(),
  clerkOrgID: text("clerk_org_id").unique(),
  linearCustomerID: text("linear_customer_id"),
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

export const THREAD_KINDS = ["support", "feedback"] as const;
export type ThreadKind = (typeof THREAD_KINDS)[number];

export const SENDERS = ["customer", "staff"] as const;
export type Sender = (typeof SENDERS)[number];

/** thread is a support conversation. Its issue in Linear is where staff triage it. */
export const thread = pgTable(
  "thread",
  {
    key: uuid("key").primaryKey().defaultRandom(),
    kind: text("kind", { enum: THREAD_KINDS }).notNull(),
    organization: uuid("organization").references(() => organization.key),
    title: text("title").notNull(),
    /** contact is the email staff replies are sent to, empty when unknown. */
    contact: text("contact").notNull().default(""),
    /** createdBy is the user id of the member who opened it, empty when signed out. */
    createdBy: text("created_by").notNull().default(""),
    issueID: text("issue_id").notNull(),
    issueIdentifier: text("issue_identifier").notNull(),
    issueURL: text("issue_url").notNull(),
    createdAt: timestamp("created_at", { withTimezone: true }).notNull().defaultNow(),
    updatedAt: timestamp("updated_at", { withTimezone: true }).notNull().defaultNow(),
  },
  (t) => [index("thread_organization_idx").on(t.organization)],
);
export type Thread = typeof thread.$inferSelect;

export const message = pgTable(
  "message",
  {
    key: serial("key").primaryKey(),
    thread: uuid("thread")
      .notNull()
      .references(() => thread.key),
    sender: text("sender", { enum: SENDERS }).notNull(),
    author: text("author").notNull(),
    text: text("text").notNull(),
    at: timestamp("at", { withTimezone: true }).notNull().defaultNow(),
  },
  (t) => [index("message_thread_idx").on(t.thread)],
);
export type Message = typeof message.$inferSelect;
