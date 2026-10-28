// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { randomBytes } from "node:crypto";

import { and, desc, eq, isNull, notExists } from "drizzle-orm";

import { type Capture } from "@/server/analytics";
import { type Reader, type Store } from "@/server/db/db";
import { type Referral, referral } from "@/server/db/schema";
import { type Account, type Directory } from "@/server/directory";
import { badRequest, notFound } from "@/server/errors";
import { MAX_NAME_LENGTH } from "@/server/license/limits";
import { type Mailer } from "@/server/mail";
import { type Session } from "@/server/session";
import { NONE, type Usage } from "@/server/usage";
import { usageOf } from "@/server/user";
import { referralPath } from "@/shell";

/** KEY is the shape of a referral's key: 72 random bits, short enough to read. */
const KEY = /^[A-Za-z0-9_-]{12}$/;

const mintKey = (): string => randomBytes(9).toString("base64url");

/** FALLBACK_REFERRER names the sender of a link whose staff member Clerk deleted. */
const FALLBACK_REFERRER = "Synnax Labs";

export interface CreateArgs {
  /** recipient is who the link is for, such as their name. */
  recipient: string;
  actor: string;
  now: Date;
}

/**
 * create makes a referral link. Staff only.
 * @throws {HTTPError} 400 for an empty or too long recipient.
 */
export const create = async (
  store: Store,
  { recipient, actor, now }: CreateArgs,
): Promise<Referral> => {
  const trimmed = recipient.trim();
  if (trimmed === "") throw badRequest("Say who the link is for");
  if (trimmed.length > MAX_NAME_LENGTH)
    throw badRequest(`The name must be at most ${MAX_NAME_LENGTH} characters`);
  const [row] = await store.query
    .insert(referral)
    .values({ key: mintKey(), recipient: trimmed, createdBy: actor, createdAt: now })
    .returning();
  return row;
};

/** retrieve returns a referral, or null for an unknown key. */
export const retrieve = async (db: Reader, key: string): Promise<Referral | null> => {
  if (!KEY.test(key)) return null;
  const [row] = await db.select().from(referral).where(eq(referral.key, key));
  return row ?? null;
};

/** referrerOf returns the name of the staff member who made a referral. */
export const referrerOf = async (
  directory: Directory,
  { createdBy }: Referral,
): Promise<string> =>
  (await directory.names([createdBy]))[createdBy] ?? FALLBACK_REFERRER;

/** Viewer is a logged-in visitor of a referral link. */
export interface Viewer {
  email: string;
  staff: boolean;
  /** installed is true once the visitor logged in to Synnax Desktop. */
  installed: boolean;
}

/** View is what the page of a referral link shows. */
export interface View {
  key: string;
  referrer: string;
  /** viewer is the logged-in visitor, or null for one who is not logged in. */
  viewer: Viewer | null;
}

export interface ViewArgs {
  key: string;
  session: Session | null;
  now: Date;
}

/** view reads what the page of a referral link shows, or null for an unknown key. */
export const view = async (
  store: Store,
  directory: Directory,
  { key, session, now }: ViewArgs,
): Promise<View | null> => {
  const found = await retrieve(store.query, key);
  if (found == null) return null;
  const referrer = await referrerOf(directory, found);
  if (session == null) return { key, referrer, viewer: null };
  const usage = await usageOf(store.query, now, [session.userID]);
  return {
    key,
    referrer,
    viewer: {
      email: session.email,
      staff: session.staff,
      installed: usage.get(session.userID)?.lastLogin != null,
    },
  };
};

export interface VisitArgs {
  key: string;
  session: Session | null;
  now: Date;
}

/**
 * visit records that someone opened a referral link. A logged-in visitor claims it
 * when nobody has and they hold no other referral. Staff count as no visit, so they
 * can check a link before they send it. Returns the referral when this visit claimed
 * it, and null otherwise.
 * @throws {HTTPError} 404 for an unknown key.
 */
export const visit = async (
  store: Store,
  { key, session, now }: VisitArgs,
): Promise<Referral | null> => {
  const found = await retrieve(store.query, key);
  if (found == null) throw notFound("Referral");
  if (session?.staff === true) return null;
  if (found.openedAt == null)
    await store.query
      .update(referral)
      .set({ openedAt: now })
      .where(and(eq(referral.key, key), isNull(referral.openedAt)));
  if (session == null) return null;
  const [claimed] = await store.query
    .update(referral)
    .set({ userID: session.userID, claimedAt: now })
    .where(
      and(
        eq(referral.key, key),
        isNull(referral.userID),
        notExists(
          store.query
            .select({ key: referral.key })
            .from(referral)
            .where(eq(referral.userID, session.userID)),
        ),
      ),
    )
    .returning();
  return claimed ?? null;
};

/** mailLink emails a referral link to the logged-in user, to open on a computer. */
export const mailLink = async (
  mail: Mailer,
  { key }: Referral,
  { email }: Session,
): Promise<void> =>
  await mail.send({
    to: [email],
    subject: "Install Synnax Desktop on your computer",
    heading: "Install Synnax Desktop on your computer",
    paragraphs: [
      "Open this email on your Mac or Windows computer to download Synnax Desktop. " +
        "Then log in to the app with this account.",
    ],
    facts: [
      { label: "Account", value: email },
      { label: "Runs on", value: "macOS on Apple silicon and Windows" },
    ],
    action: { label: "Continue setup", href: referralPath(key) },
    reason: "You get this email because you asked for the download link.",
  });

/**
 * Stage is how far the recipient of a referral got: the link is sent, they opened it,
 * they logged in to the portal from it, or they logged in to Synnax Desktop.
 */
export type Stage = "sent" | "opened" | "signed_up" | "installed";

/** stageOf reads the stage of a referral from its claimant's Desktop usage. */
export const stageOf = (ref: Referral, usage: Usage): Stage => {
  if (ref.userID == null) return ref.openedAt == null ? "sent" : "opened";
  return usage.lastLogin == null ? "signed_up" : "installed";
};

/** Entry is a referral with who made it and how far its recipient got. */
export interface Entry {
  referral: Referral;
  referrer: string;
  /** account is the user who claimed the link, or null before a claim. */
  account: Account | null;
  usage: Usage;
  stage: Stage;
}

/** list returns every referral, the newest first. Staff only. */
export const list = async (
  store: Store,
  directory: Directory,
  now: Date,
): Promise<Entry[]> => {
  const rows = await store.query
    .select()
    .from(referral)
    .orderBy(desc(referral.createdAt));
  const claimants = rows.flatMap((r) => (r.userID == null ? [] : [r.userID]));
  const [accounts, usage] = await Promise.all([
    directory.users(),
    usageOf(store.query, now, claimants),
  ]);
  const byID = new Map(accounts.map((a) => [a.userID, a]));
  return rows.map((r) => {
    const used = (r.userID == null ? null : usage.get(r.userID)) ?? NONE;
    return {
      referral: r,
      referrer: byID.get(r.createdBy)?.name ?? FALLBACK_REFERRER,
      account: r.userID == null ? null : (byID.get(r.userID) ?? null),
      usage: used,
      stage: stageOf(r, used),
    };
  });
};

/** Referred is a user's referral with the name of the staff member who made it. */
export interface Referred {
  referral: Referral;
  referrer: string;
}

/** referredBy returns the referral a user claimed, or null for none. */
export const referredBy = async (
  store: Store,
  directory: Directory,
  userID: string,
): Promise<Referred | null> => {
  const [found] = await store.query
    .select()
    .from(referral)
    .where(eq(referral.userID, userID));
  if (found == null) return null;
  return { referral: found, referrer: await referrerOf(directory, found) };
};

/**
 * CONVERTED is the PostHog event of a referred user's first login to Synnax Desktop.
 * It sets the referral on their person once, so an insight can filter or break down by
 * who referred them.
 */
export const CONVERTED = "referral_converted";

/** converted is the {@link CONVERTED} event of a referred user. */
export const converted = (
  { referral: ref, referrer }: Referred,
  userID: string,
  now: Date,
): Capture => {
  const properties = {
    referral: ref.key,
    referred_by: ref.createdBy,
    referred_by_name: referrer,
  };
  return {
    userID,
    event: CONVERTED,
    at: now,
    properties,
    once: { ...properties, referred_at: ref.createdAt.toISOString() },
  };
};
