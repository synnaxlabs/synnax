// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type license } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import { License } from "@/platform/license";

const BASE: license.License = {
  jti: "6d1f7a0e-1c3b-4e2a-9f7d-2a1b3c4d5e6f",
  iat: 1_700_000_000,
  v: 1,
  org: "0f8fad5b-d9cb-469f-a165-70867728950e",
  ed: "e",
  fp: [],
  fs: 1,
  n: 2,
  ch: 0,
};

// 2027-03-01T00:00:00Z
const EXP = 1_803_859_200;

describe("License.describeTerm", () => {
  it("should describe a subscription by its expiry", () => {
    expect(License.describeTerm({ ...BASE, exp: EXP })).toBe("Expires 2027-03-01");
  });

  it("should describe a perpetual license by its version ceiling", () => {
    expect(License.describeTerm({ ...BASE, mv: "0.62" })).toBe(
      "Perpetual, covers versions up to 0.62",
    );
  });

  it("should describe a subscription with a version fallback", () => {
    expect(License.describeTerm({ ...BASE, exp: EXP, mv: "0.62" })).toBe(
      "Subscription until 2027-03-01, then versions up to 0.62",
    );
  });

  it("should call a license with neither bound perpetual", () => {
    expect(License.describeTerm(BASE)).toBe("Perpetual");
  });
});

describe("License.describeChannels", () => {
  it("should treat a zero cap as unlimited", () => {
    expect(License.describeChannels(BASE)).toBe("Unlimited");
  });

  it("should state the cap", () => {
    expect(License.describeChannels({ ...BASE, ch: 500 })).toBe("Up to 500");
  });
});

describe("License.editionLabel", () => {
  it("should name the known editions", () => {
    expect(License.editionLabel(BASE)).toBe("Enterprise");
    expect(License.editionLabel({ ...BASE, ed: "d" })).toBe("Desktop");
  });

  it("should pass an unknown edition code through", () => {
    expect(License.editionLabel({ ...BASE, ed: "x" })).toBe("x");
  });
});

describe("License.joinFingerprint", () => {
  it("should separate hashes so a single-line paste keeps them apart", () => {
    expect(License.joinFingerprint(["aa", "bb"])).toBe("aa, bb");
  });
});
