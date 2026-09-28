// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { and, eq, isNotNull, isNull } from "drizzle-orm";

import { type Store } from "@/server/db/db";
import {
  type Event,
  event,
  type License,
  license,
  organization,
} from "@/server/db/schema";
import { type Mailer } from "@/server/mail";

/** NOTICE_DAYS are the days before expiry at which a warning is sent, most urgent last. */
export const NOTICE_DAYS = [30, 7, 1] as const;

const DAY_MS = 24 * 60 * 60 * 1000;

const daysLeft = (expiresAt: Date, now: Date): number =>
  Math.ceil((expiresAt.getTime() - now.getTime()) / DAY_MS);

const sentDays = (notices: Event[]): Set<number> =>
  new Set(
    notices.map((e) => (e.detail as { days?: number }).days).filter((d) => d != null),
  );

/**
 * dueNotice returns the warning a license is due, as days before expiry, or null when
 * none is. The most urgent unsent window wins, so a missed earlier window never sends
 * after a later one has become due.
 */
export const dueNotice = (lic: License, notices: Event[], now: Date): number | null => {
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
 * organization with nobody to mail is skipped and retried on the next sweep.
 */
export const sweep = async ({
  store,
  mail,
  recipients,
  now,
}: SweepArgs): Promise<Sent[]> => {
  const rows = await store.query
    .select({ license, organization })
    .from(license)
    .innerJoin(organization, eq(license.organization, organization.key))
    .where(and(isNotNull(license.expiresAt), isNull(license.revokedAt)));
  const sent: Sent[] = [];
  for (const row of rows) {
    const notices = await store.query
      .select()
      .from(event)
      .where(and(eq(event.license, row.license.key), eq(event.kind, "expiry_notice")));
    const days = dueNotice(row.license, notices, now);
    if (days == null) continue;
    const to = await recipients(row.organization.key);
    if (to.length === 0) continue;
    await mail.send({
      to,
      subject: `Your Synnax license expires in ${days} ${days === 1 ? "day" : "days"}`,
      text: expiryText(row.license, row.organization.name, days),
    });
    await store.query.insert(event).values({
      kind: "expiry_notice",
      actor: "system",
      organization: row.organization.key,
      license: row.license.key,
      detail: { days, to },
    });
    sent.push({ license: row.license.key, days, to });
  }
  return sent;
};

const expiryText = (lic: License, organizationName: string, days: number): string =>
  [
    `The Synnax license "${lic.label}" for ${organizationName} expires in ${days} ` +
      `${days === 1 ? "day" : "days"}, on ${lic.expiresAt?.toISOString().slice(0, 10)}.`,
    "",
    lic.maxVersion == null
      ? "Cores running under it will refuse to start after that date."
      : `After that date it covers Core versions up to ${lic.maxVersion}.`,
    "",
    "Reply to this email to renew.",
  ].join("\n");

export const revocationText = (lic: License, organizationName: string): string =>
  [
    `The Synnax license "${lic.label}" for ${organizationName} has been revoked.`,
    "",
    "Cores holding it keep running until their next restart. Contact Synnax Labs if " +
      "you believe this is a mistake.",
  ].join("\n");
