// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type license as client } from "@synnaxlabs/client";

import { type Activation, type License } from "@/server/db/schema";

export const NOW = new Date("2026-09-17T12:00:00Z");

export const LICENSE: License = {
  key: "0f2c6a7e-6b1a-4c1d-9c3e-1f1f0b7a2d11",
  organization: "64156293-6534-416c-99d1-8db73a22ca6a",
  edition: "enterprise",
  term: "subscription",
  nodes: 2,
  channels: 0,
  expiresAt: new Date("2027-03-01T00:00:00Z"),
  maxVersion: null,
  label: "Test rig",
  issuedBy: "user_staff",
  issuedAt: new Date("2026-09-01T00:00:00Z"),
  revokedAt: null,
};

export const HASH_A = "a".repeat(64);
export const HASH_B = "b".repeat(64);
export const HASH_C = "c".repeat(64);

export const activationOf = (
  fingerprint: string[],
  overrides: Partial<Activation> = {},
): Activation => ({
  key: crypto.randomUUID(),
  license: LICENSE.key,
  fingerprint,
  firstSeen: NOW,
  lastSeen: NOW,
  releasedAt: null,
  ...overrides,
});

export const CLAIMS: client.License = {
  jti: LICENSE.key,
  iat: 1_789_646_400,
  exp: 1_803_859_200,
  v: 1,
  org: LICENSE.organization,
  ed: "e",
  fingerprints: [HASH_A],
  fingerprintScheme: 1,
  n: 2,
  ch: 0,
};
