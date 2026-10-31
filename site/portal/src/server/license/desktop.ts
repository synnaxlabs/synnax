// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createHash, randomBytes } from "node:crypto";

import { and, desc, eq, inArray, isNull, max } from "drizzle-orm";

import { type Reader, schema, type Store } from "@/server/db";
import { badRequest, forbidden, notFound } from "@/server/errors";
import { build } from "@/server/license/claims";
import { sign, type Signer } from "@/server/license/sign";
import { ensurePersonal } from "@/server/organization";
import { DESKTOP } from "@/server/usage";

const DAY_MS = 24 * 60 * 60 * 1000;

/** TERM_MS is how long a desktop license runs before it needs a renewal. */
export const TERM_MS = 30 * DAY_MS;

const CHANNEL_LIMIT = 50;

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
  key: string;
  secret: string;
  activation: schema.Activation;
  license: schema.License;
  /** previous is the user's last Desktop use before this login, or null for none. */
  previous: Date | null;
}

export interface Machine {
  activation: schema.Activation;
  license: schema.License;
}

/** lastUse returns the newest Desktop use of an organization, or null for none. */
const lastUse = async (db: Reader, org: string): Promise<Date | null> => {
  const [row] = await db
    .select({ at: max(schema.event.at) })
    .from(schema.event)
    .where(
      and(eq(schema.event.organization, org), inArray(schema.event.kind, DESKTOP)),
    );
  return row?.at ?? null;
};

/** superseded picks the linked machines that share a hash with the one logging in. */
export const superseded = (machines: Machine[], fingerprint: string[]): Machine[] =>
  machines.filter(({ activation: a }) =>
    a.fingerprint.some((h) => fingerprint.includes(h)),
  );

/**
 * machinesFor returns the Desktop machines of an organization that hold a seat, the
 * most recently renewed first.
 */
export const machinesFor = async (
  db: Reader,
  organization: string,
): Promise<Machine[]> =>
  await db
    .select({ activation: schema.activation, license: schema.license })
    .from(schema.activation)
    .innerJoin(schema.license, eq(schema.activation.license, schema.license.key))
    .where(
      and(
        eq(schema.license.organization, organization),
        eq(schema.license.edition, "desktop"),
        isNull(schema.activation.releasedAt),
      ),
    )
    .orderBy(desc(schema.activation.lastSeen));

/**
 * link issues a desktop license for the user's personal organization, bound to one
 * machine, and returns its license key beside the secret that renews it. A machine that
 * logs in again replaces its earlier link, so it stays one entry. Nothing is stored
 * when signing fails.
 */
export const link = async (
  store: Store,
  signer: Signer,
  { userID, userName, fingerprint, machineName, now }: LinkArgs,
): Promise<Linked> => {
  const org = await ensurePersonal(store, { userID, name: userName });
  const secret = mintSecret();
  const linked = await store.transact(async (tx) => {
    // Serializes a user's links, so a concurrent one sees this link and supersedes it.
    await tx
      .select({ key: schema.organization.key })
      .from(schema.organization)
      .where(eq(schema.organization.key, org.key))
      .for("update");
    const previous = await lastUse(tx, org.key);
    const machines = await machinesFor(tx, org.key);
    for (const old of superseded(machines, fingerprint)) {
      await tx
        .update(schema.activation)
        .set({ releasedAt: now, renewalSecretHash: null })
        .where(eq(schema.activation.key, old.activation.key));
      await tx
        .update(schema.license)
        .set({ revokedAt: old.license.revokedAt ?? now })
        .where(eq(schema.license.key, old.license.key));
      await tx.insert(schema.event).values({
        at: now,
        kind: "unlink",
        actor: userID,
        organization: org.key,
        license: old.license.key,
        activation: old.activation.key,
        detail: { reason: "superseded" },
      });
    }
    const [lic] = await tx
      .insert(schema.license)
      .values({
        organization: org.key,
        edition: "desktop",
        term: "subscription",
        nodes: 1,
        channels: CHANNEL_LIMIT,
        label: machineName,
        expiresAt: expiryFrom(now),
        issuedBy: userID,
        issuedAt: now,
      })
      .returning();
    const [act] = await tx
      .insert(schema.activation)
      .values({
        license: lic.key,
        fingerprint,
        name: machineName,
        renewalSecretHash: hashSecret(secret),
        firstSeen: now,
        lastSeen: now,
      })
      .returning();
    await tx.insert(schema.event).values({
      at: now,
      kind: "link",
      actor: userID,
      organization: org.key,
      license: lic.key,
      activation: act.key,
      detail: { name: machineName, fingerprint },
    });
    const signed = await sign(signer, build({ license: lic, fingerprint, now }));
    return { key: signed, license: lic, activation: act, previous };
  });
  return { secret, ...linked };
};

/**
 * UNLINKED is the code of a refusal that means the machine is logged out. The Desktop
 * app logs out on this code alone, so a 403 from a proxy does not log a user out.
 */
const UNLINKED = "unlinked";

const LOGGED_OUT = "This machine was logged out. Log in again.";

/**
 * resolve finds the machine a renewal secret belongs to. Throws a 403 with the code
 * `unlinked` when the secret is unknown or the machine was unlinked.
 */
export const resolve = async (store: Store, secret: string): Promise<Machine> => {
  const [row] = await store.query
    .select({ activation: schema.activation, license: schema.license })
    .from(schema.activation)
    .innerJoin(schema.license, eq(schema.activation.license, schema.license.key))
    .where(eq(schema.activation.renewalSecretHash, hashSecret(secret)));
  if (row == null) throw forbidden("This machine is not logged in", UNLINKED);
  if (row.activation.releasedAt != null || row.license.revokedAt != null)
    throw forbidden(LOGGED_OUT, UNLINKED);
  return row;
};

export interface RenewArgs {
  machine: Machine;
  now: Date;
}

export interface Renewed {
  key: string;
  license: schema.License;
  /** owner is the Clerk user ID of the user the machine is linked for. */
  owner: string;
  /** previous is the user's last Desktop use before this renewal, or null for none. */
  previous: Date | null;
}

/**
 * renew slides the expiry of a machine's license and returns a fresh license key.
 * Throws a 403 with the code `unlinked` when the machine was unlinked after `resolve`
 * found it. Nothing is stored when signing fails.
 */
export const renew = async (
  store: Store,
  signer: Signer,
  { machine: row, now }: RenewArgs,
): Promise<Renewed> =>
  await store.transact(async (tx) => {
    // Serializes the user's Desktop use with their links, so one is the day's first.
    const [org] = await tx
      .select({ owner: schema.organization.ownerUserID })
      .from(schema.organization)
      .where(eq(schema.organization.key, row.license.organization))
      .for("update");
    if (org?.owner == null)
      throw new Error(`desktop license ${row.license.key} has no personal owner`);
    const previous = await lastUse(tx, row.license.organization);
    const [lic] = await tx
      .update(schema.license)
      .set({ expiresAt: expiryFrom(now) })
      .where(
        and(eq(schema.license.key, row.license.key), isNull(schema.license.revokedAt)),
      )
      .returning();
    if (lic == null) throw forbidden(LOGGED_OUT, UNLINKED);
    await tx
      .update(schema.activation)
      .set({ lastSeen: now })
      .where(eq(schema.activation.key, row.activation.key));
    await tx.insert(schema.event).values({
      at: now,
      kind: "renew",
      actor: row.activation.key,
      organization: lic.organization,
      license: lic.key,
      activation: row.activation.key,
      detail: {},
    });
    const signed = await sign(
      signer,
      build({ license: lic, fingerprint: row.activation.fingerprint, now }),
    );
    return { key: signed, license: lic, owner: org.owner, previous };
  });

export interface UnlinkArgs {
  activationKey: string;
  actor: string;
  now: Date;
}

/**
 * unlink releases a Desktop machine's seat, forgets its renewal secret, and revokes
 * its license, so its next renewal is refused.
 * @throws {HTTPError} 400 for an enterprise machine, whose license others share.
 */
export const unlink = async (
  store: Store,
  { activationKey, actor, now }: UnlinkArgs,
): Promise<void> => {
  const [row] = await store.query
    .select({ activation: schema.activation, license: schema.license })
    .from(schema.activation)
    .innerJoin(schema.license, eq(schema.activation.license, schema.license.key))
    .where(eq(schema.activation.key, activationKey));
  if (row == null) throw notFound("Activation");
  if (row.license.edition !== "desktop")
    throw badRequest("Only a Synnax Desktop machine can be logged out");
  await store.transact(async (tx) => {
    // Locks the personal organization first, as link and renew do, so no two Desktop
    // writes deadlock.
    await tx
      .select({ key: schema.organization.key })
      .from(schema.organization)
      .where(eq(schema.organization.key, row.license.organization))
      .for("update");
    await tx
      .update(schema.activation)
      .set({ releasedAt: row.activation.releasedAt ?? now, renewalSecretHash: null })
      .where(eq(schema.activation.key, activationKey));
    await tx
      .update(schema.license)
      .set({ revokedAt: row.license.revokedAt ?? now })
      .where(eq(schema.license.key, row.license.key));
    await tx.insert(schema.event).values({
      at: now,
      kind: "unlink",
      actor,
      organization: row.license.organization,
      license: row.license.key,
      activation: activationKey,
      detail: {},
    });
  });
};
