// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { type schema } from "@/server/db";
import { expiring, issued, restored, revoked } from "@/server/license/notice";
import { LICENSE } from "@/server/license/testutil";

const TEAM: schema.Organization = {
  key: LICENSE.organization,
  kind: "team",
  name: "Acme",
  clerkOrgID: "org_acme",
  ownerUserID: null,
  createdAt: new Date("2026-09-01T00:00:00Z"),
};

describe("notice", () => {
  it("should name the license, its organization, and its term", () => {
    expect(revoked(LICENSE, TEAM)).toMatchObject({
      facts: [
        { label: "License", value: "Test rig" },
        { label: "Organization", value: "Acme" },
        { label: "Term", value: "Until 2027-03-01" },
      ],
      action: { label: "View license", href: `/licenses/${LICENSE.key}` },
      reason: "You get this email because you are an admin of Acme.",
    });
  });

  it("should describe a perpetual term by its last version", () => {
    const perpetual: schema.License = {
      ...LICENSE,
      term: "perpetual",
      expiresAt: null,
      maxVersion: "0.60",
    };
    expect(restored(perpetual, TEAM).facts).toContainEqual({
      label: "Term",
      value: "Perpetual, up to v0.60",
    });
  });

  it("should tell a personal owner the license is theirs", () => {
    const personal: schema.Organization = {
      ...TEAM,
      kind: "personal",
      clerkOrgID: null,
    };
    expect(revoked(LICENSE, personal).reason).toBe(
      "You get this email because the license is yours.",
    );
  });

  describe("issued", () => {
    it("should tell the admins how to put the license to use", () => {
      expect(issued(LICENSE, TEAM)).toMatchObject({
        subject: 'Your Synnax license "Test rig" is ready',
        heading: "Your new license is ready",
        paragraphs: [
          "Activate a machine in the portal to give a Core its license key.",
          "Any member of Acme can activate machines, up to the license's seats.",
        ],
      });
    });

    it("should list the license's full terms", () => {
      expect(issued({ ...LICENSE, channels: 500 }, TEAM).facts).toEqual([
        { label: "License", value: "Test rig" },
        { label: "Organization", value: "Acme" },
        { label: "Term", value: "Until 2027-03-01" },
        { label: "Seats", value: "2" },
        { label: "Channels per Core", value: "500" },
      ]);
    });
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
