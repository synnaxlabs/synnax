// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createHmac } from "node:crypto";

import { describe, expect, it } from "vitest";

import { type Organization } from "@/server/db/schema";
import { activationOf, HASH_A, LICENSE, NOW } from "@/server/license/testutil";
import {
  build,
  CARD_KEYS,
  TTL_SECONDS,
  verify,
  type View,
} from "@/server/support/card";

const SECRET = "s3cret";
const BODY = JSON.stringify({ cardKeys: ["licenses"] });
const sign = (secret: string, body: string) =>
  createHmac("sha256", secret).update(body).digest("hex");

const ORG: Organization = {
  key: LICENSE.organization,
  kind: "team",
  name: "Streeling",
  clerkOrgID: "org_1",
  plainTenantID: null,
  ownerUserID: null,
  createdAt: NOW,
};

const view = (overrides: Partial<View> = {}): View => ({
  organizations: [ORG],
  licenses: [{ ...LICENSE, seats: 1, organizationName: ORG.name }],
  activations: [{ ...activationOf([HASH_A]), label: LICENSE.label }],
  site: "https://docs.synnaxlabs.com",
  now: NOW,
  ...overrides,
});

describe("card.verify", () => {
  it("should accept the signature of the exact body", () => {
    expect(verify(SECRET, BODY, sign(SECRET, BODY))).toBe(true);
  });

  it("should reject a missing, foreign, or truncated signature", () => {
    expect(verify(SECRET, BODY, null)).toBe(false);
    expect(verify(SECRET, BODY, sign("other", BODY))).toBe(false);
    expect(verify(SECRET, BODY, sign(SECRET, BODY).slice(1))).toBe(false);
    expect(verify(SECRET, `${BODY} `, sign(SECRET, BODY))).toBe(false);
  });
});

describe("card.build", () => {
  it("should answer every requested key, unknown ones with an empty card", () => {
    const cards = build([...CARD_KEYS, "billing"], view());
    expect(cards.map((c) => c.key)).toEqual([...CARD_KEYS, "billing"]);
    expect(cards.every((c) => c.timeToLiveSeconds === TTL_SECONDS)).toBe(true);
    expect(cards[3].components).toEqual([]);
    expect(cards[0].components.length).toBeGreaterThan(0);
  });

  it("should badge a license by its state", () => {
    const badge = (lic: View["licenses"][number]) =>
      JSON.stringify(build(["licenses"], view({ licenses: [lic] }))).match(
        /"badgeLabel":"(\w+)"/,
      )?.[1];
    const lic = view().licenses[0];
    expect(badge(lic)).toBe("Active");
    expect(badge({ ...lic, revokedAt: NOW })).toBe("Revoked");
    expect(badge({ ...lic, expiresAt: new Date(NOW.getTime() - 1) })).toBe("Expired");
  });

  it("should link each organization and license into the portal", () => {
    const text = JSON.stringify(build(["organizations", "licenses"], view()));
    expect(text).toContain(`https://docs.synnaxlabs.com/licenses?org=${ORG.key}`);
    expect(text).toContain(`https://docs.synnaxlabs.com/licenses/${LICENSE.key}`);
  });

  it("should say so when there is nothing to show", () => {
    const cards = build(
      [...CARD_KEYS],
      view({ organizations: [], licenses: [], activations: [] }),
    );
    expect(JSON.stringify(cards)).toContain("No licenses");
    expect(JSON.stringify(cards)).toContain("No activations");
  });
});
