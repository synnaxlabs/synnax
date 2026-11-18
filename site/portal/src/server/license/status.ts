// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type License } from "@/server/db/schema";

export type Status = "active" | "expiring" | "expired" | "revoked";

export const STATUSES: Status[] = ["active", "expiring", "expired", "revoked"];

export const isStatus = (value: string): value is Status =>
  STATUSES.some((s) => s === value);

/** EXPIRING is how long before its expiry a license starts asking to be renewed. */
export const EXPIRING = 30 * 24 * 60 * 60 * 1000;

/** statusOf derives a license's state from its dates at `now`. */
export const statusOf = (lic: License, now: Date): Status => {
  if (lic.revokedAt != null) return "revoked";
  if (lic.expiresAt == null) return "active";
  const left = lic.expiresAt.getTime() - now.getTime();
  if (left <= 0) return "expired";
  // A Desktop license renews itself on a short term, so a near expiry is normal there.
  if (lic.edition === "desktop" || left > EXPIRING) return "active";
  return "expiring";
};
