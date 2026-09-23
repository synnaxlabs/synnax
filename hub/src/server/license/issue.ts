// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { and, eq, isNull } from "drizzle-orm";

import { type Store } from "@/server/db/db";
import {
  activation,
  type Edition,
  event,
  type License,
  license,
  type Term,
} from "@/server/db/schema";
import { badRequest, notFound } from "@/server/errors";
import { build } from "@/server/license/claims";
import { sign, type Signer } from "@/server/license/sign";

const MINOR_VERSION = /^\d+\.\d+$/;

/** Terms are the fields staff set when issuing a license and may change later. */
export interface Terms {
  term: Term;
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
  edition: Edition;
  actor: string;
  now: Date;
}

/** validate checks the term rules a license must satisfy and throws a 400 if not. */
export const validate = (args: Terms & { now: Date }): void => {
  if (!Number.isInteger(args.nodes) || args.nodes < 1)
    throw badRequest("Nodes must be a whole number of at least 1");
  if (!Number.isInteger(args.channels) || args.channels < 0)
    throw badRequest("Channels must be a whole number, 0 for unlimited");
  if (args.maxVersion != null && !MINOR_VERSION.test(args.maxVersion))
    throw badRequest('Maximum version must look like "0.62"');
  if (args.term === "subscription") {
    if (args.expiresAt == null) throw badRequest("A subscription needs an expiry");
    if (args.expiresAt <= args.now)
      throw badRequest("The expiry must be in the future");
  } else {
    if (args.expiresAt != null) throw badRequest("A perpetual license has no expiry");
    if (args.maxVersion == null)
      throw badRequest("A perpetual license needs a maximum version");
  }
};

/** issue records a new license for an organization. */
export const issue = async (store: Store, args: IssueArgs): Promise<License> => {
  validate(args);
  const [row] = await store.query
    .insert(license)
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
  await store.query.insert(event).values({
    kind: "issue",
    actor: args.actor,
    organization: row.organization,
    license: row.key,
    detail: { nodes: row.nodes, channels: row.channels, term: row.term },
  });
  return row;
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
export const changes = (before: License, after: License): Record<string, unknown> => {
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
 * history survive. Machines pick the new terms up on their next token.
 * @throws {HTTPError} 400 when the license is revoked or the seat count would drop
 * below the machines holding one.
 */
export const amend = async (
  store: Store,
  { licenseKey, actor, now, ...terms }: AmendArgs,
): Promise<License> => {
  validate({ ...terms, now });
  const [before] = await store.query
    .select()
    .from(license)
    .where(eq(license.key, licenseKey));
  if (before == null) throw notFound("License");
  if (before.revokedAt != null) throw badRequest("A revoked license cannot be changed");
  const held = await store.query
    .select()
    .from(activation)
    .where(and(eq(activation.license, licenseKey), isNull(activation.releasedAt)));
  if (terms.nodes < held.length)
    throw badRequest(
      `${held.length} machines hold a seat. Release one before lowering the limit ` +
        `to ${terms.nodes}.`,
    );
  const [after] = await store.query
    .update(license)
    .set({
      term: terms.term,
      nodes: terms.nodes,
      channels: terms.channels,
      label: terms.label,
      expiresAt: terms.expiresAt ?? null,
      maxVersion: terms.maxVersion ?? null,
    })
    .where(eq(license.key, licenseKey))
    .returning();
  await store.query.insert(event).values({
    kind: "amend",
    actor,
    organization: after.organization,
    license: after.key,
    detail: changes(before, after),
  });
  return after;
};

export interface RevokeArgs {
  licenseKey: string;
  actor: string;
  now: Date;
}

/** revoke stops a license from activating. Running Cores keep it until they restart. */
export const revoke = async (
  store: Store,
  { licenseKey, actor, now }: RevokeArgs,
): Promise<License> => {
  const [row] = await store.query
    .update(license)
    .set({ revokedAt: now })
    .where(eq(license.key, licenseKey))
    .returning();
  if (row == null) throw notFound("License");
  await store.query.insert(event).values({
    kind: "revoke",
    actor,
    organization: row.organization,
    license: row.key,
    detail: {},
  });
  return row;
};

export interface FloatingArgs {
  licenseKey: string;
  actor: string;
  now: Date;
}

/**
 * floating signs a token bound to no host, for CI runners with random hardware. Staff
 * only; the ledger records the issuance but holds no seat.
 */
export const floating = async (
  store: Store,
  signer: Signer,
  { licenseKey, actor, now }: FloatingArgs,
): Promise<string> => {
  const [row] = await store.query
    .select()
    .from(license)
    .where(eq(license.key, licenseKey));
  if (row == null) throw notFound("License");
  if (row.revokedAt != null) throw badRequest("This license has been revoked");
  await store.query.insert(event).values({
    kind: "token",
    actor,
    organization: row.organization,
    license: row.key,
    detail: { floating: true },
  });
  return await sign(signer, build({ license: row, fingerprint: [], now }));
};
