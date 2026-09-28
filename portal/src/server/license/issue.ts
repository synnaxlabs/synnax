// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { eq } from "drizzle-orm";

import { type Store } from "@/server/db/db";
import {
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

export interface IssueArgs {
  organization: string;
  edition: Edition;
  term: Term;
  nodes: number;
  channels: number;
  label: string;
  /** expiresAt is required on a subscription and forbidden on a perpetual license. */
  expiresAt?: Date;
  /** maxVersion is required on a perpetual license and optional on a subscription. */
  maxVersion?: string;
  actor: string;
  now: Date;
}

/** validate checks the term rules an issuance must satisfy and throws a 400 if not. */
export const validate = (args: IssueArgs): void => {
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
