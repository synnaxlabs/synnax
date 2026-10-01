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
import { type Denial, DENIAL_MESSAGES, deny } from "@/server/license/deny";
import { sign, type Signer } from "@/server/license/sign";

/**
 * requireEnterprise throws a 400 when `lic` is a Synnax Desktop license. Only the
 * Desktop app activates, renews, and releases those.
 */
export const requireEnterprise = (lic: License): void => {
  if (lic.edition === "desktop")
    throw badRequest("A Synnax Desktop license is managed from the Desktop app");
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
 * holds a seat, found by any shared host hash, keeps it.
 */
export const decide = ({
  license,
  activations,
  fingerprint,
  now,
}: DecideArgs): Decision => {
  const denial = deny(license, now);
  if (denial != null) return { ok: false, reason: denial };
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
  name: string;
  actor: string;
  now: Date;
}

export type Result =
  { ok: true; activation: Activation; key: string } | { ok: false; reason: Denial };

/**
 * activate grants or refreshes a seat for a machine and signs its license key. The seat
 * check and the ledger write run in one transaction under a row lock on the license,
 * so concurrent requests cannot oversubscribe it. A denial is recorded as an event and
 * returned, not thrown.
 */
export const activate = async (
  store: Store,
  signer: Signer,
  { licenseKey, fingerprint, name, actor, now }: ActivateArgs,
): Promise<Result> =>
  await store.transact(async (tx) => {
    const [lic] = await tx
      .select()
      .from(license)
      .where(eq(license.key, licenseKey))
      .for("update");
    if (lic == null) throw notFound("License");
    requireEnterprise(lic);
    const activations = await tx
      .select()
      .from(activation)
      .where(eq(activation.license, licenseKey));
    const decision = decide({ license: lic, activations, fingerprint, now });
    if (!decision.ok) {
      await tx.insert(event).values({
        at: now,
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
            .values({
              license: lic.key,
              fingerprint,
              name,
              firstSeen: now,
              lastSeen: now,
            })
            .returning()
        : await tx
            .update(activation)
            .set({ lastSeen: now, name })
            .where(eq(activation.key, decision.existing.key))
            .returning();
    await tx.insert(event).values({
      at: now,
      kind: "activate",
      actor,
      organization: lic.organization,
      license: lic.key,
      activation: act.key,
      detail: { fingerprint },
    });
    const signed = await sign(signer, build({ license: lic, fingerprint, now }));
    return { ok: true, activation: act, key: signed };
  });

export interface ReissueArgs {
  activationKey: string;
  actor: string;
  now: Date;
}

/**
 * reissue signs a fresh license key for a machine that already holds a seat, for a
 * download after the activation page has been left. Throws when the seat was released
 * or the license no longer activates.
 */
export const reissue = async (
  store: Store,
  signer: Signer,
  { activationKey, actor, now }: ReissueArgs,
): Promise<{ license: License; key: string }> => {
  const [row] = await store.query
    .select({ activation, license })
    .from(activation)
    .innerJoin(license, eq(activation.license, license.key))
    .where(eq(activation.key, activationKey));
  if (row == null || row.activation.releasedAt != null) throw notFound("Activation");
  requireEnterprise(row.license);
  const decision = decide({
    license: row.license,
    activations: [row.activation],
    fingerprint: row.activation.fingerprint,
    now,
  });
  if (!decision.ok) throw badRequest(DENIAL_MESSAGES[decision.reason]);
  const signed = await sign(
    signer,
    build({ license: row.license, fingerprint: row.activation.fingerprint, now }),
  );
  await store.query
    .update(activation)
    .set({ lastSeen: now })
    .where(eq(activation.key, activationKey));
  await store.query.insert(event).values({
    at: now,
    kind: "download",
    actor,
    organization: row.license.organization,
    license: row.license.key,
    activation: activationKey,
    detail: {},
  });
  return { license: row.license, key: signed };
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
  requireEnterprise(row.license);
  if (row.activation.releasedAt != null) return;
  await store.query
    .update(activation)
    .set({ releasedAt: now })
    .where(eq(activation.key, activationKey));
  await store.query.insert(event).values({
    at: now,
    kind: "release",
    actor,
    organization: row.license.organization,
    license: row.license.key,
    activation: activationKey,
    detail: {},
  });
};
