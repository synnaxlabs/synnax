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
  type Activation,
  activation,
  event,
  type License,
  license,
} from "@/server/db/schema";
import { badRequest, notFound } from "@/server/errors";
import { build } from "@/server/license/claims";
import { sign, type Signer } from "@/server/license/sign";

export type Denial = "revoked" | "expired" | "no_seats";

export const DENIAL_MESSAGES: Record<Denial, string> = {
  revoked: "This license has been revoked.",
  expired: "This license has expired.",
  no_seats: "Every seat on this license is taken. Release a machine to free one.",
};

export type Decision =
  { ok: true; existing?: Activation } | { ok: false; reason: Denial };

export interface DecideArgs {
  license: License;
  activations: Activation[];
  fingerprint: string[];
  now: Date;
}

/**
 * decide applies the seat rules to an activation request. A machine that already
 * holds a seat, found by any shared host hash, keeps it. An expired subscription with
 * a fallback version still activates, because the token remains valid up to that
 * version.
 */
export const decide = ({
  license,
  activations,
  fingerprint,
  now,
}: DecideArgs): Decision => {
  if (license.revokedAt != null) return { ok: false, reason: "revoked" };
  if (
    license.expiresAt != null &&
    license.expiresAt <= now &&
    license.maxVersion == null
  )
    return { ok: false, reason: "expired" };
  const active = activations.filter((a) => a.releasedAt == null);
  const existing = active.find((a) =>
    a.fingerprint.some((h) => fingerprint.includes(h)),
  );
  if (existing != null) return { ok: true, existing };
  if (active.length >= license.nodes) return { ok: false, reason: "no_seats" };
  return { ok: true };
};

export interface ActivateArgs {
  licenseKey: string;
  fingerprint: string[];
  actor: string;
  now: Date;
}

export type Result =
  { ok: true; activation: Activation; token: string } | { ok: false; reason: Denial };

/**
 * activate grants or refreshes a seat for a machine and signs its token. The seat
 * check and the ledger write run in one transaction under a row lock on the license,
 * so concurrent requests cannot oversubscribe it. A denial is recorded as an event and
 * returned, not thrown.
 */
export const activate = async (
  store: Store,
  signer: Signer,
  { licenseKey, fingerprint, actor, now }: ActivateArgs,
): Promise<Result> =>
  await store.transact(async (tx) => {
    const [lic] = await tx
      .select()
      .from(license)
      .where(eq(license.key, licenseKey))
      .for("update");
    if (lic == null) throw notFound("License");
    const activations = await tx
      .select()
      .from(activation)
      .where(eq(activation.license, licenseKey));
    const decision = decide({ license: lic, activations, fingerprint, now });
    if (!decision.ok) {
      await tx.insert(event).values({
        kind: "activate_denied",
        actor,
        organization: lic.organization,
        license: lic.key,
        detail: { reason: decision.reason, fingerprint },
      });
      return decision;
    }
    const [act] =
      decision.existing == null
        ? await tx
            .insert(activation)
            .values({ license: lic.key, fingerprint, firstSeen: now, lastSeen: now })
            .returning()
        : await tx
            .update(activation)
            .set({ lastSeen: now })
            .where(eq(activation.key, decision.existing.key))
            .returning();
    await tx.insert(event).values({
      kind: "activate",
      actor,
      organization: lic.organization,
      license: lic.key,
      activation: act.key,
      detail: { fingerprint },
    });
    const token = await sign(signer, build({ license: lic, fingerprint, now }));
    return { ok: true, activation: act, token };
  });

export interface ReissueArgs {
  activationKey: string;
  actor: string;
  now: Date;
}

/**
 * reissue signs a fresh token for a machine that already holds a seat, for a download
 * after the activation page has been left. Throws when the seat was released or the
 * license no longer activates.
 */
export const reissue = async (
  store: Store,
  signer: Signer,
  { activationKey, actor, now }: ReissueArgs,
): Promise<{ license: License; token: string }> => {
  const [row] = await store.query
    .select({ activation, license })
    .from(activation)
    .innerJoin(license, eq(activation.license, license.key))
    .where(eq(activation.key, activationKey));
  if (row == null || row.activation.releasedAt != null) throw notFound("Activation");
  const decision = decide({
    license: row.license,
    activations: [row.activation],
    fingerprint: row.activation.fingerprint,
    now,
  });
  if (!decision.ok) throw badRequest(DENIAL_MESSAGES[decision.reason]);
  await store.query
    .update(activation)
    .set({ lastSeen: now })
    .where(eq(activation.key, activationKey));
  await store.query.insert(event).values({
    kind: "token",
    actor,
    organization: row.license.organization,
    license: row.license.key,
    activation: activationKey,
    detail: {},
  });
  const token = await sign(
    signer,
    build({ license: row.license, fingerprint: row.activation.fingerprint, now }),
  );
  return { license: row.license, token };
};

export interface ReleaseArgs {
  activationKey: string;
  actor: string;
  now: Date;
}

/** release frees a machine's seat so another machine, or the same one, can activate. */
export const release = async (
  store: Store,
  { activationKey, actor, now }: ReleaseArgs,
): Promise<void> => {
  const [row] = await store.query
    .select({ activation, license })
    .from(activation)
    .innerJoin(license, eq(activation.license, license.key))
    .where(eq(activation.key, activationKey));
  if (row == null) throw notFound("Activation");
  if (row.activation.releasedAt != null) return;
  await store.query
    .update(activation)
    .set({ releasedAt: now })
    .where(eq(activation.key, activationKey));
  await store.query.insert(event).values({
    kind: "release",
    actor,
    organization: row.license.organization,
    license: row.license.key,
    activation: activationKey,
    detail: {},
  });
};
