// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createHash, randomBytes } from "node:crypto";

import { eq } from "drizzle-orm";

import { type Store } from "@/server/db/db";
import {
  type Activation,
  activation,
  event,
  type License,
  license,
} from "@/server/db/schema";
import { forbidden, notFound } from "@/server/errors";
import { build } from "@/server/license/claims";
import { sign, type Signer } from "@/server/license/sign";
import { ensurePersonal } from "@/server/organization";

const DAY_MS = 24 * 60 * 60 * 1000;

/** TERM_MS is how long a desktop license runs before it needs a renewal. */
export const TERM_MS = 30 * DAY_MS;

/** mintSecret returns a fresh renewal secret for one machine. */
export const mintSecret = (): string => randomBytes(32).toString("base64url");

/** hashSecret is what the activation row stores in place of the secret. */
export const hashSecret = (secret: string): string =>
  createHash("sha256").update(secret).digest("hex");

/** expiryFrom is when a desktop license issued or renewed at now stops applying. */
export const expiryFrom = (now: Date): Date => new Date(now.getTime() + TERM_MS);

export interface LinkArgs {
  userID: string;
  userName: string;
  fingerprint: string[];
  /** machineName is the hostname the Desktop app reported. */
  machineName: string;
  now: Date;
}

export interface Linked {
  token: string;
  secret: string;
  activation: Activation;
  license: License;
}

/**
 * link issues a desktop license for the user's personal organization, bound to one
 * machine, and returns its token beside the secret that renews it.
 */
export const link = async (
  store: Store,
  signer: Signer,
  { userID, userName, fingerprint, machineName, now }: LinkArgs,
): Promise<Linked> => {
  const org = await ensurePersonal(store, { userID, name: userName });
  const secret = mintSecret();
  const linked = await store.transact(async (tx) => {
    const [lic] = await tx
      .insert(license)
      .values({
        organization: org.key,
        edition: "desktop",
        term: "subscription",
        nodes: 1,
        channels: 0,
        label: machineName,
        expiresAt: expiryFrom(now),
        issuedBy: userID,
        issuedAt: now,
      })
      .returning();
    const [act] = await tx
      .insert(activation)
      .values({
        license: lic.key,
        fingerprint,
        name: machineName,
        renewalSecretHash: hashSecret(secret),
        firstSeen: now,
        lastSeen: now,
      })
      .returning();
    await tx.insert(event).values({
      kind: "link",
      actor: userID,
      organization: org.key,
      license: lic.key,
      activation: act.key,
      detail: { name: machineName, fingerprint },
    });
    return { license: lic, activation: act };
  });
  const token = await sign(
    signer,
    build({ license: linked.license, fingerprint, now }),
  );
  return { token, secret, ...linked };
};

export interface Machine {
  activation: Activation;
  license: License;
}

/**
 * resolve finds the machine a renewal secret belongs to. Throws a 403 when the secret
 * is unknown or the machine was unlinked.
 */
export const resolve = async (store: Store, secret: string): Promise<Machine> => {
  const [row] = await store.query
    .select({ activation, license })
    .from(activation)
    .innerJoin(license, eq(activation.license, license.key))
    .where(eq(activation.renewalSecretHash, hashSecret(secret)));
  if (row == null) throw forbidden("This machine is not linked to an account");
  if (row.activation.releasedAt != null || row.license.revokedAt != null)
    throw forbidden("This machine was unlinked. Sign in again.");
  return row;
};

export interface RenewArgs {
  machine: Machine;
  now: Date;
}

/** renew slides the expiry of a machine's license and returns a fresh token. */
export const renew = async (
  store: Store,
  signer: Signer,
  { machine: row, now }: RenewArgs,
): Promise<{ token: string; license: License }> => {
  const [lic] = await store.query
    .update(license)
    .set({ expiresAt: expiryFrom(now) })
    .where(eq(license.key, row.license.key))
    .returning();
  await store.query
    .update(activation)
    .set({ lastSeen: now })
    .where(eq(activation.key, row.activation.key));
  await store.query.insert(event).values({
    kind: "renew",
    actor: row.activation.key,
    organization: lic.organization,
    license: lic.key,
    activation: row.activation.key,
    detail: {},
  });
  const token = await sign(
    signer,
    build({ license: lic, fingerprint: row.activation.fingerprint, now }),
  );
  return { token, license: lic };
};

export interface UnlinkArgs {
  activationKey: string;
  actor: string;
  now: Date;
}

/**
 * unlink releases a Desktop machine's seat, forgets its renewal secret, and revokes
 * its license, so its next renewal is refused.
 */
export const unlink = async (
  store: Store,
  { activationKey, actor, now }: UnlinkArgs,
): Promise<void> => {
  const [row] = await store.query
    .select({ activation, license })
    .from(activation)
    .innerJoin(license, eq(activation.license, license.key))
    .where(eq(activation.key, activationKey));
  if (row == null) throw notFound("Activation");
  await store.transact(async (tx) => {
    await tx
      .update(activation)
      .set({ releasedAt: row.activation.releasedAt ?? now, renewalSecretHash: null })
      .where(eq(activation.key, activationKey));
    await tx
      .update(license)
      .set({ revokedAt: row.license.revokedAt ?? now })
      .where(eq(license.key, row.license.key));
    await tx.insert(event).values({
      kind: "unlink",
      actor,
      organization: row.license.organization,
      license: row.license.key,
      activation: activationKey,
      detail: {},
    });
  });
};
