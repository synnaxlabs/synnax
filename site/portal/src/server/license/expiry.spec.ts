// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { type Event } from "@/server/db/schema";
import { dueNotice } from "@/server/license/expiry";
import { LICENSE, NOW } from "@/server/license/testutil";

const DAY = 24 * 60 * 60 * 1000;
const expiringIn = (days: number) => ({
  ...LICENSE,
  expiresAt: new Date(NOW.getTime() + days * DAY),
});
const notice = (days: number): Event => ({
  key: days,
  at: NOW,
  kind: "expiry_notice",
  actor: "system",
  organization: LICENSE.organization,
  license: LICENSE.key,
  activation: null,
  detail: { days },
});

describe("expiry.dueNotice", () => {
  it("should be silent far from expiry", () => {
    expect(dueNotice(expiringIn(45), [], NOW)).toBeNull();
  });

  it("should send the 30 day warning inside its window once", () => {
    expect(dueNotice(expiringIn(29.5), [], NOW)).toBe(30);
    expect(dueNotice(expiringIn(29.5), [notice(30)], NOW)).toBeNull();
  });

  it("should send the most urgent unsent warning", () => {
    expect(dueNotice(expiringIn(6), [notice(30)], NOW)).toBe(7);
    expect(dueNotice(expiringIn(0.5), [notice(30), notice(7)], NOW)).toBe(1);
  });

  it("should skip an earlier window a later one has overtaken", () => {
    expect(dueNotice(expiringIn(6), [notice(7)], NOW)).toBeNull();
  });

  it("should be silent once expired, revoked, or perpetual", () => {
    expect(dueNotice(expiringIn(-1), [], NOW)).toBeNull();
    expect(dueNotice({ ...expiringIn(5), revokedAt: NOW }, [], NOW)).toBeNull();
    expect(dueNotice({ ...LICENSE, expiresAt: null }, [], NOW)).toBeNull();
  });
});
