// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { and, eq, isNotNull, isNull } from "drizzle-orm";

import { schema, type Store } from "@/server/db";
import { badRequest, notFound } from "@/server/errors";
import { requireEnterprise } from "@/server/license/activate";
import { build } from "@/server/license/claims";
import { DENIAL_MESSAGES, deny } from "@/server/license/deny";
import { VERSION_PATTERN } from "@/server/license/limits";
import { sign, type Signer } from "@/server/license/sign";

/** Terms are the fields staff set when issuing a license and may change later. */
export interface Terms {
  term: schema.Term;
  nodes: number;
  channels: number;
  label: string;
  /** expiresAt is required on a subscription and forbidden on a perpetual license. */
  expiresAt?: Date;
  /** maxVersion is required on a perpetual license and optional on a subscription. */
  maxVersion?: string;
}

export interface IssueArgs extends Terms {
  organization: string;
  edition: schema.Edition;
  actor: string;
  now: Date;
}

/** readTerms reads the terms a staff form posts. {@link validate} checks them. */
export const readTerms = (body: Record<string, string>): Terms => {
  const term = body.term === "perpetual" ? "perpetual" : "subscription";
  return {
    term,
    nodes: Number(body.nodes),
    channels: Number(body.channels || "0"),
    label: (body.label ?? "").trim(),
    expiresAt:
      term === "subscription" && body.expiresAt
        ? new Date(`${body.expiresAt}T00:00:00Z`)
        : undefined,
    maxVersion: body.maxVersion?.trim() || undefined,
  };
};

/** validate checks the term rules a license must satisfy and throws a 400 if not. */
export const validate = (args: Terms & { now: Date }): void => {
  if (args.label === "") throw badRequest("Give the license a label");
  if (!Number.isInteger(args.nodes) || args.nodes < 1)
    throw badRequest("Nodes must be a whole number of at least 1");
  if (!Number.isInteger(args.channels) || args.channels < 0)
    throw badRequest("Channels must be a whole number, 0 for unlimited");
  if (args.maxVersion != null && !VERSION_PATTERN.test(args.maxVersion))
    throw badRequest('Maximum version must look like "0.62"');
  if (args.term === "subscription") {
    if (args.expiresAt == null) throw badRequest("A subscription needs an expiry");
    if (Number.isNaN(args.expiresAt.getTime()))
      throw badRequest("The expiry must be a valid date");
    if (args.expiresAt <= args.now)
      throw badRequest("The expiry must be in the future");
  } else {
    if (args.expiresAt != null) throw badRequest("A perpetual license has no expiry");
    if (args.maxVersion == null)
      throw badRequest("A perpetual license needs a maximum version");
  }
};

export const issue = async (store: Store, args: IssueArgs): Promise<schema.License> => {
  validate(args);
  return await store.transact(async (tx) => {
    const [row] = await tx
      .insert(schema.license)
      .values({
        organization: args.organization,
        edition: args.edition,
        term: args.term,
        nodes: args.nodes,
        channels: args.channels,
        label: args.label,
        expiresAt: args.expiresAt,
        maxVersion: args.maxVersion,
        issuedBy: args.actor,
        issuedAt: args.now,
      })
      .returning();
    await tx.insert(schema.event).values({
      at: args.now,
      kind: "issue",
      actor: args.actor,
      organization: row.organization,
      license: row.key,
      detail: { nodes: row.nodes, channels: row.channels, term: row.term },
    });
    return row;
  });
};

export interface AmendArgs extends Terms {
  licenseKey: string;
  actor: string;
  now: Date;
}

/** CHANGEABLE are the license fields an amendment may alter. */
const CHANGEABLE = [
  "term",
  "nodes",
  "channels",
  "label",
  "expiresAt",
  "maxVersion",
] as const;

const readable = (value: unknown): unknown =>
  value instanceof Date ? value.toISOString() : value;

/** changes lists what an amendment altered, each field as its before and after. */
export const changes = (
  before: schema.License,
  after: schema.License,
): Record<string, unknown> => {
  const out: Record<string, unknown> = {};
  for (const field of CHANGEABLE) {
    const from = readable(before[field]);
    const to = readable(after[field]);
    if (from !== to) out[field] = { from, to };
  }
  return out;
};

/**
 * amend changes the terms of a license already issued, keeping its key so seats and
 * history survive. Machines pick the new terms up on their next license key.
 * @throws {HTTPError} 400 when the license is revoked or the seat count would drop
 * below the machines holding one.
 */
export const amend = async (
  store: Store,
  { licenseKey, actor, now, ...terms }: AmendArgs,
): Promise<schema.License> => {
  validate({ ...terms, now });
  return await store.transact(async (tx) => {
    // Holds the lock activate takes, so no seat is granted between the count and
    // the update.
    const [before] = await tx
      .select()
      .from(schema.license)
      .where(eq(schema.license.key, licenseKey))
      .for("update");
    if (before == null) throw notFound("License");
    requireEnterprise(before);
    if (before.revokedAt != null)
      throw badRequest("A revoked license cannot be changed");
    const held = await tx
      .select()
      .from(schema.activation)
      .where(
        and(
          eq(schema.activation.license, licenseKey),
          isNull(schema.activation.releasedAt),
        ),
      );
    if (terms.nodes < held.length)
      throw badRequest(
        `${held.length} machines hold a seat. Release one before lowering the limit ` +
          `to ${terms.nodes}.`,
      );
    const [after] = await tx
      .update(schema.license)
      .set({
        term: terms.term,
        nodes: terms.nodes,
        channels: terms.channels,
        label: terms.label,
        expiresAt: terms.expiresAt ?? null,
        maxVersion: terms.maxVersion ?? null,
      })
      .where(eq(schema.license.key, licenseKey))
      .returning();
    await tx.insert(schema.event).values({
      at: now,
      kind: "amend",
      actor,
      organization: after.organization,
      license: after.key,
      detail: changes(before, after),
    });
    return after;
  });
};

export interface RevokeArgs {
  licenseKey: string;
  actor: string;
  now: Date;
}

/**
 * revoke stops a license from activating. A Core checks its license offline, so a Core
 * that already holds it keeps running as its terms allow, even after a restart.
 * @throws {HTTPError} 400 when the license is already revoked.
 */
export const revoke = async (
  store: Store,
  { licenseKey, actor, now }: RevokeArgs,
): Promise<schema.License> =>
  await store.transact(async (tx) => {
    const [row] = await tx
      .update(schema.license)
      .set({ revokedAt: now })
      .where(and(eq(schema.license.key, licenseKey), isNull(schema.license.revokedAt)))
      .returning();
    if (row == null) {
      const [existing] = await tx
        .select({ key: schema.license.key })
        .from(schema.license)
        .where(eq(schema.license.key, licenseKey));
      if (existing == null) throw notFound("License");
      throw badRequest("This license is already revoked");
    }
    await tx.insert(schema.event).values({
      at: now,
      kind: "revoke",
      actor,
      organization: row.organization,
      license: row.key,
      detail: {},
    });
    return row;
  });

export interface RestoreArgs {
  licenseKey: string;
  actor: string;
  now: Date;
}

/**
 * restore undoes a revocation. The license keeps its key, terms, and seats, so the
 * machines that held one need not activate again.
 * @throws {HTTPError} 400 when the license is a Synnax Desktop license or is not
 * revoked.
 */
export const restore = async (
  store: Store,
  { licenseKey, actor, now }: RestoreArgs,
): Promise<schema.License> =>
  await store.transact(async (tx) => {
    const [row] = await tx
      .update(schema.license)
      .set({ revokedAt: null })
      .where(
        and(
          eq(schema.license.key, licenseKey),
          eq(schema.license.edition, "enterprise"),
          isNotNull(schema.license.revokedAt),
        ),
      )
      .returning();
    if (row == null) {
      const [existing] = await tx
        .select()
        .from(schema.license)
        .where(eq(schema.license.key, licenseKey));
      if (existing == null) throw notFound("License");
      requireEnterprise(existing);
      throw badRequest("This license is not revoked");
    }
    await tx.insert(schema.event).values({
      at: now,
      kind: "restore",
      actor,
      organization: row.organization,
      license: row.key,
      detail: {},
    });
    return row;
  });

/**
 * remove deletes a license with its seats and its history. A Core checks its license
 * offline, so a Core that already holds it keeps running as its terms allow.
 * @throws {HTTPError} 404 when the license does not exist.
 */
export const remove = async (store: Store, licenseKey: string): Promise<void> =>
  await store.transact(async (tx) => {
    // Holds the lock activate takes, so no machine takes a seat during the delete.
    const [row] = await tx
      .select({ key: schema.license.key })
      .from(schema.license)
      .where(eq(schema.license.key, licenseKey))
      .for("update");
    if (row == null) throw notFound("License");
    await tx.delete(schema.event).where(eq(schema.event.license, licenseKey));
    await tx.delete(schema.activation).where(eq(schema.activation.license, licenseKey));
    await tx.delete(schema.license).where(eq(schema.license.key, licenseKey));
  });

export interface FloatingArgs {
  licenseKey: string;
  actor: string;
  now: Date;
}

/**
 * floating signs a license key bound to no host, for CI runners with random hardware.
 * Staff only; the ledger records the issuance but holds no seat.
 */
export const floating = async (
  store: Store,
  signer: Signer,
  { licenseKey, actor, now }: FloatingArgs,
): Promise<string> => {
  const [row] = await store.query
    .select()
    .from(schema.license)
    .where(eq(schema.license.key, licenseKey));
  if (row == null) throw notFound("License");
  requireEnterprise(row);
  const denial = deny(row, now);
  if (denial != null) throw badRequest(DENIAL_MESSAGES[denial]);
  const signed = await sign(signer, build({ license: row, fingerprint: [], now }));
  await store.query.insert(schema.event).values({
    at: now,
    kind: "download",
    actor,
    organization: row.organization,
    license: row.key,
    detail: { floating: true },
  });
  return signed;
};
