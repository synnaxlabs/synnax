// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { deny } from "@/server/license/deny";
import { LICENSE, NOW } from "@/server/license/testutil";

describe("deny", () => {
  it("should allow a license inside its term", () => {
    expect(deny(LICENSE, NOW)).toBeUndefined();
  });

  it("should refuse a revoked license", () => {
    expect(deny({ ...LICENSE, revokedAt: NOW }, NOW)).toBe("revoked");
  });

  it("should refuse an expired subscription without a fallback", () => {
    expect(deny({ ...LICENSE, expiresAt: new Date(NOW.getTime() - 1) }, NOW)).toBe(
      "expired",
    );
  });

  it("should allow an expired subscription that has a fallback", () => {
    expect(
      deny(
        { ...LICENSE, expiresAt: new Date(NOW.getTime() - 1), maxVersion: "0.60" },
        NOW,
      ),
    ).toBeUndefined();
  });
});
