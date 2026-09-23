// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type license as client } from "@synnaxlabs/client";

import { type Edition, type License } from "@/server/db/schema";

export const CLAIMS_VERSION = 1;
export const FINGERPRINT_SCHEME = 1;

const EDITION_CODES: Record<Edition, string> = { desktop: "d", enterprise: "e" };

export interface BuildArgs {
  license: License;
  /** Machine fingerprints to bind the token to. Empty for a floating token. */
  fingerprint: string[];
  now: Date;
}

const seconds = (date: Date): number => Math.floor(date.getTime() / 1000);

/** build assembles the claim set a Core verifies from a stored license. */
export const build = ({ license, fingerprint, now }: BuildArgs): client.License => ({
  jti: license.key,
  iat: seconds(now),
  exp: license.expiresAt == null ? undefined : seconds(license.expiresAt),
  v: CLAIMS_VERSION,
  org: license.organization,
  ed: EDITION_CODES[license.edition],
  fingerprints: fingerprint,
  fingerprintScheme: FINGERPRINT_SCHEME,
  n: license.nodes,
  ch: license.channels,
  mv: license.maxVersion ?? undefined,
});
