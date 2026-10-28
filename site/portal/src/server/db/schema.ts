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
  unique,
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
  "amend",
  "activate",
  "activate_denied",
  "download",
  "release",
  "rename",
  "revoke",
  "restore",
  "expiry_notice",
  "link",
  "renew",
  "unlink",
  "register",
  "rename_organization",
  "invite",
  "revoke_invitation",
  "accept_invitation",
  "change_role",
  "remove_member",
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
    name: text("name"),
    /** renewalSecretHash is the SHA-256 of the secret a Desktop machine renews with. */
    renewalSecretHash: text("renewal_secret_hash").unique(),
    firstSeen: timestamp("first_seen", { withTimezone: true }).notNull().defaultNow(),
    lastSeen: timestamp("last_seen", { withTimezone: true }).notNull().defaultNow(),
    releasedAt: timestamp("released_at", { withTimezone: true }),
  },
  (t) => [index("activation_license_idx").on(t.license)],
);
export type Activation = typeof activation.$inferSelect;

/** invitation asks someone to join a team. Accepting or revoking it deletes it. */
export const invitation = pgTable(
  "invitation",
  {
    key: uuid("key").primaryKey().defaultRandom(),
    organization: uuid("organization")
      .notNull()
      .references(() => organization.key),
    /** email is the lowercase address the invitation is for. */
    email: text("email").notNull(),
    role: text("role").notNull(),
    sentAt: timestamp("sent_at", { withTimezone: true }).notNull().defaultNow(),
    expiresAt: timestamp("expires_at", { withTimezone: true }).notNull(),
  },
  (t) => [
    unique("invitation_organization_email_unique").on(t.organization, t.email),
    index("invitation_email_idx").on(t.email),
  ],
);
export type Invitation = typeof invitation.$inferSelect;

/**
 * referral is a link that staff send to invite someone to Synnax Desktop. The first
 * user who opens it while logged in claims it.
 */
export const referral = pgTable("referral", {
  /** key is the random code at the end of the link. */
  key: text("key").primaryKey(),
  /** recipient is who staff made the link for, in their own words. */
  recipient: text("recipient").notNull(),
  /** createdBy is the Clerk user ID of the staff member who made the link. */
  createdBy: text("created_by").notNull(),
  createdAt: timestamp("created_at", { withTimezone: true }).notNull().defaultNow(),
  openedAt: timestamp("opened_at", { withTimezone: true }),
  /** userID is the Clerk user ID of the user who claimed the link. */
  userID: text("user_id").unique(),
  claimedAt: timestamp("claimed_at", { withTimezone: true }),
});
export type Referral = typeof referral.$inferSelect;

/**
 * handoff holds a license key until the Console that asked for it collects it. The
 * Console proves that it asked with the code whose SHA-256 is codeHash.
 */
export const handoff = pgTable("handoff", {
  codeHash: text("code_hash").primaryKey(),
  key: text("key").notNull(),
  expiresAt: timestamp("expires_at", { withTimezone: true }).notNull(),
});

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
