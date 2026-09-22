// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { describeEvent, statusOf, usable } from "@/portal/ui/format";
import { type Event, type License } from "@/server/db/schema";
import { LICENSE, NOW } from "@/server/license/testutil";

const licenseOf = (overrides: Partial<License>): License => ({
  ...LICENSE,
  ...overrides,
});

const days = (n: number): Date => new Date(NOW.getTime() + n * 24 * 60 * 60 * 1000);

describe("format.statusOf", () => {
  it("should call a revoked license revoked whatever its expiry", () => {
    const lic = licenseOf({ revokedAt: NOW, expiresAt: days(90) });
    expect(statusOf(lic, NOW)).toBe("revoked");
  });

  it("should call a perpetual license active", () => {
    expect(statusOf(licenseOf({ expiresAt: null }), NOW)).toBe("active");
  });

  it("should call a license past its expiry expired", () => {
    expect(statusOf(licenseOf({ expiresAt: days(-1) }), NOW)).toBe("expired");
  });

  it("should call a license inside the renewal window expiring", () => {
    expect(statusOf(licenseOf({ expiresAt: days(29) }), NOW)).toBe("expiring");
  });

  it("should call a license outside the renewal window active", () => {
    expect(statusOf(licenseOf({ expiresAt: days(31) }), NOW)).toBe("active");
  });

  it("should not warn on a Desktop license, which renews itself", () => {
    const lic = licenseOf({ edition: "desktop", expiresAt: days(2) });
    expect(statusOf(lic, NOW)).toBe("active");
  });
});

describe("format.usable", () => {
  it("should grant a seat while active or expiring", () => {
    expect(usable("active")).toBe(true);
    expect(usable("expiring")).toBe(true);
  });

  it("should refuse a seat once expired or revoked", () => {
    expect(usable("expired")).toBe(false);
    expect(usable("revoked")).toBe(false);
  });
});

const MACHINE = "9b0b5f3c-0d3f-4a4e-9f21-6a0b2f1c7a10";
const USER = "user_2abc";

const eventOf = (overrides: Partial<Event>): Event => ({
  key: 1,
  at: NOW,
  kind: "release",
  actor: USER,
  organization: LICENSE.organization,
  license: LICENSE.key,
  activation: null,
  detail: {},
  ...overrides,
});

const NAMES = { machines: { [MACHINE]: "Test stand" }, actors: { [USER]: "Ada" } };

describe("format.describeEvent", () => {
  it("should name the machine an event concerns and the person who acted", () => {
    const e = eventOf({ kind: "release", activation: MACHINE });
    expect(describeEvent(e, NAMES)).toBe("Seat released: Test stand by Ada");
  });

  it("should leave out a machine the event does not name", () => {
    expect(describeEvent(eventOf({ kind: "issue" }), NAMES)).toBe(
      "License issued by Ada",
    );
  });

  it("should name a machine acting on itself once", () => {
    const e = eventOf({ kind: "renew", actor: MACHINE, activation: MACHINE });
    expect(describeEvent(e, NAMES)).toBe("License renewed: Test stand");
  });

  it("should leave out an actor no name is known for", () => {
    const e = eventOf({ kind: "expiry_notice", actor: "cron" });
    expect(describeEvent(e, NAMES)).toBe("Expiry notice sent");
  });

  it("should carry the reason an event was given", () => {
    const e = eventOf({
      kind: "unlink",
      activation: MACHINE,
      detail: { reason: "superseded" },
    });
    expect(describeEvent(e, NAMES)).toBe(
      "Machine unlinked: Test stand by Ada (superseded)",
    );
  });
});
