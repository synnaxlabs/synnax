// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Event, type License } from "@/server/db/schema";

export const date = (d: Date | string | null | undefined): string =>
  d == null ? "" : new Date(d).toISOString().slice(0, 10);

export const dateTime = (d: Date | string): string =>
  `${new Date(d).toISOString().slice(0, 16).replace("T", " ")} UTC`;

export const shortHash = (hashes: string[]): string =>
  hashes.length === 0
    ? "Floating"
    : `${hashes[0].slice(0, 12)}${hashes.length > 1 ? ` +${hashes.length - 1}` : ""}`;

/** machineName is what a machine reads as: its name, or its hashes before one. */
export const machineName = (a: {
  name: string | null;
  fingerprint: string[];
}): string => a.name ?? shortHash(a.fingerprint);

export type LicenseStatus = "active" | "expiring" | "expired" | "revoked";

/** EXPIRING is how long before its expiry a license starts asking to be renewed. */
const EXPIRING = 30 * 24 * 60 * 60 * 1000;

/** statusOf derives a license's state from its dates at `now`. */
export const statusOf = (lic: License, now: Date): LicenseStatus => {
  if (lic.revokedAt != null) return "revoked";
  if (lic.expiresAt == null) return "active";
  const left = new Date(lic.expiresAt).getTime() - now.getTime();
  if (left <= 0) return "expired";
  // A Desktop license renews itself on a short term, so a near expiry is normal there.
  if (lic.edition === "desktop" || left > EXPIRING) return "active";
  return "expiring";
};

/** usable is true while a license in `status` can still grant a seat. */
export const usable = (status: LicenseStatus): boolean =>
  status === "active" || status === "expiring";

export const term = (lic: License): string => {
  if (lic.term === "perpetual") return `Perpetual, up to v${lic.maxVersion}`;
  const until = `Until ${date(lic.expiresAt)}`;
  return lic.maxVersion == null ? until : `${until}, then up to v${lic.maxVersion}`;
};

export const edition = (e: License["edition"]): string =>
  e === "desktop" ? "Desktop" : "Enterprise";

export const channels = (n: number): string => (n === 0 ? "Unlimited" : String(n));

const EVENT_LABELS: Record<Event["kind"], string> = {
  issue: "License issued",
  amend: "License changed",
  activate: "Machine activated",
  activate_denied: "Activation denied",
  token: "Token downloaded",
  release: "Seat released",
  revoke: "License revoked",
  expiry_notice: "Expiry notice sent",
  link: "Machine linked",
  renew: "License renewed",
  unlink: "Machine unlinked",
};

interface Narrator {
  /** machines is the name each activation key reads as. */
  machines: Record<string, string>;
  /** actors is the name each Clerk user id reads as. */
  actors: Record<string, string>;
}

/** describeEvent writes one event as a line: what happened, to what, by whom. */
export const describeEvent = (e: Event, { machines, actors }: Narrator): string => {
  const on = e.activation == null ? undefined : machines[e.activation];
  // A machine renewing itself is its own actor, so it is only named once.
  const by = actors[e.actor] ?? machines[e.actor];
  const detail = e.detail as Record<string, unknown>;
  const reason = typeof detail.reason === "string" ? ` (${detail.reason})` : "";
  const subject = on == null ? "" : `: ${on}`;
  const actor = by == null || by === on ? "" : ` by ${by}`;
  return `${EVENT_LABELS[e.kind]}${subject}${actor}${reason}`;
};
