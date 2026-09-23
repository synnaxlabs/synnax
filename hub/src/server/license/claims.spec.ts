// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { build } from "@/server/license/claims";
import { CLAIMS, HASH_A, LICENSE, NOW } from "@/server/license/testutil";

describe("claims.build", () => {
  it("should map a subscription license to the claim set", () => {
    expect(build({ license: LICENSE, fingerprint: [HASH_A], now: NOW })).toEqual({
      ...CLAIMS,
      mv: undefined,
    });
  });

  it("should leave exp off a perpetual license and carry its ceiling", () => {
    const claims = build({
      license: { ...LICENSE, term: "perpetual", expiresAt: null, maxVersion: "0.62" },
      fingerprint: [],
      now: NOW,
    });
    expect(claims.exp).toBeUndefined();
    expect(claims.mv).toBe("0.62");
    expect(claims.fingerprints).toEqual([]);
  });

  it("should code the desktop edition as d", () => {
    expect(
      build({ license: { ...LICENSE, edition: "desktop" }, fingerprint: [], now: NOW })
        .ed,
    ).toBe("d");
  });
});
