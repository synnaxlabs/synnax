// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { errors } from "@synnaxlabs/x";
import { and, eq, isNotNull, isNull, ne } from "drizzle-orm";

import { type Query, schema, type Store } from "@/server/db";
import { expiring } from "@/server/license/notice";
import { type Mailer } from "@/server/mail";

/** NOTICE_DAYS are the days before expiry that get a warning, most urgent last. */
export const NOTICE_DAYS = [30, 7, 1] as const;

const DAY_MS = 24 * 60 * 60 * 1000;

const daysLeft = (expiresAt: Date, now: Date): number =>
  Math.ceil((expiresAt.getTime() - now.getTime()) / DAY_MS);

const sentDays = (notices: schema.Event[]): Set<number> =>
  new Set(
    notices.map((e) => (e.detail as { days?: number }).days).filter((d) => d != null),
  );

/**
 * dueNotice returns the warning a license is due, as days before expiry, or null when
 * none is. The most urgent unsent window wins, so a missed earlier window never sends
 * after a later one has become due.
 */
export const dueNotice = (
  lic: schema.License,
  notices: schema.Event[],
  now: Date,
): number | null => {
  if (lic.expiresAt == null || lic.revokedAt != null) return null;
  const left = daysLeft(lic.expiresAt, now);
  if (left <= 0) return null;
  const due = [...NOTICE_DAYS].reverse().find((d) => left <= d);
  if (due == null || sentDays(notices).has(due)) return null;
  return due;
};

export interface SweepArgs {
  store: Store;
  mail: Mailer;
  /** recipients resolves the addresses to warn for an organization. */
  recipients: (organizationKey: string) => Promise<string[]>;
  now: Date;
}

export interface Sent {
  license: string;
  days: number;
  to: string[];
}

/**
 * sweep mails every due expiry warning and records each one as an event. An
 * organization with nobody to mail is skipped and retried on the next sweep. Each
 * license is warned under a row lock, so overlapping sweeps send a warning once. A
 * failed send leaves its warning due.
 *
 * @throws {AggregateError} after warning the rest, when any send failed.
 */
export const sweep = async ({
  store,
  mail,
  recipients,
  now,
}: SweepArgs): Promise<Sent[]> => {
  const rows = await store.query
    .select({ license: schema.license, organization: schema.organization })
    .from(schema.license)
    .innerJoin(
      schema.organization,
      eq(schema.license.organization, schema.organization.key),
    )
    .where(
      and(
        isNotNull(schema.license.expiresAt),
        isNull(schema.license.revokedAt),
        // A desktop license renews itself while the app runs; nobody is warned.
        ne(schema.license.edition, "desktop"),
      ),
    );
  const sent: Sent[] = [];
  const failures: unknown[] = [];
  for (const { license: lic, organization: org } of rows) {
    const { key } = lic;
    if (dueNotice(lic, await notices(store.query, key), now) == null) continue;
    const to = await recipients(org.key);
    if (to.length === 0) continue;
    try {
      const notice = await store.transact(async (tx): Promise<Sent | null> => {
        const [locked] = await tx
          .select()
          .from(schema.license)
          .where(eq(schema.license.key, key))
          .for("update");
        const days = dueNotice(locked, await notices(tx, key), now);
        if (days == null) return null;
        await tx.insert(schema.event).values({
          at: now,
          kind: "expiry_notice",
          actor: "system",
          organization: org.key,
          license: key,
          detail: { days, to },
        });
        await mail.send({ to, ...expiring(locked, org, days) });
        return { license: key, days, to };
      });
      if (notice != null) sent.push(notice);
    } catch (e) {
      failures.push(e);
    }
  }
  if (failures.length > 0) {
    const reasons = failures.map((e) => errors.fromUnknown(e).message).join("; ");
    throw new AggregateError(
      failures,
      `${failures.length} of ${sent.length + failures.length} expiry warnings failed ` +
        `to send: ${reasons}`,
    );
  }
  return sent;
};

const notices = async (q: Query, licenseKey: string): Promise<schema.Event[]> =>
  await q
    .select()
    .from(schema.event)
    .where(
      and(eq(schema.event.license, licenseKey), eq(schema.event.kind, "expiry_notice")),
    );
