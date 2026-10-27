// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { type Organization } from "@/server/db/schema";
import { expiring, restored, revoked } from "@/server/license/notice";
import { LICENSE } from "@/server/license/testutil";

const TEAM: Organization = {
  key: LICENSE.organization,
  kind: "team",
  name: "Acme",
  clerkOrgID: "org_acme",
  ownerUserID: null,
  createdAt: new Date("2026-09-01T00:00:00Z"),
};

describe("notice", () => {
  it("should name the license, its organization, and its expiry", () => {
    expect(revoked(LICENSE, TEAM)).toMatchObject({
      facts: [
        { label: "License", value: "Test rig" },
        { label: "Organization", value: "Acme" },
        { label: "Expires", value: "2027-03-01" },
      ],
      action: { label: "View license", href: `/licenses/${LICENSE.key}` },
      reason: "You get this email because you are an admin of Acme.",
    });
  });

  it("should leave out the expiry of a license that never expires", () => {
    const { facts } = restored({ ...LICENSE, expiresAt: null }, TEAM);
    expect(facts.map((f) => f.label)).toEqual(["License", "Organization"]);
  });

  it("should tell a personal owner the license is theirs", () => {
    const personal: Organization = { ...TEAM, kind: "personal", clerkOrgID: null };
    expect(revoked(LICENSE, personal).reason).toBe(
      "You get this email because the license is yours.",
    );
  });

  describe("expiring", () => {
    it("should count the days in the subject and heading", () => {
      expect(expiring(LICENSE, TEAM, 1)).toMatchObject({
        subject: "Your Synnax license expires in 1 day",
        heading: "Your license expires in 1 day",
      });
      expect(expiring(LICENSE, TEAM, 30).subject).toBe(
        "Your Synnax license expires in 30 days",
      );
    });

    it("should say what happens after expiry", () => {
      expect(expiring(LICENSE, TEAM, 7).paragraphs[0]).toBe(
        "After it expires, Cores running under it refuse to start.",
      );
      expect(expiring({ ...LICENSE, maxVersion: "0.60" }, TEAM, 7).paragraphs[0]).toBe(
        "After it expires, it covers Core versions up to 0.60.",
      );
    });
  });
});
