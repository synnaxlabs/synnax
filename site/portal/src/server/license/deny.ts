// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type License } from "@/server/db/schema";

export type Denial = "revoked" | "expired" | "no_seats";

export const DENIAL_MESSAGES: Record<Denial, string> = {
  revoked: "This license has been revoked.",
  expired: "This license has expired.",
  no_seats: "Every seat on this license is taken. Release a machine to free one.",
};

/**
 * deny returns why a license can no longer issue a license key at `now`, or undefined
 * when it can. An expired subscription with a fallback version still issues.
 */
export const deny = (license: License, now: Date): Denial | undefined => {
  if (license.revokedAt != null) return "revoked";
  if (
    license.expiresAt != null &&
    license.expiresAt <= now &&
    license.maxVersion == null
  )
    return "expired";
  return undefined;
};
