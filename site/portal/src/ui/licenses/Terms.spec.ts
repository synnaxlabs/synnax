// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Licenses } from "@/ui/licenses";

const schema = Licenses.termsSchema.check(Licenses.checkTerms);

const VALID: Licenses.TermsValues = {
  ...Licenses.ZERO_TERMS,
  label: "Test stand",
  expiresAt: "2027-01-01",
};

const issues = (values: Partial<Licenses.TermsValues>) =>
  schema
    .safeParse({ ...VALID, ...values })
    .error?.issues.map(({ path, message }) => ({ path, message })) ?? [];

describe("Terms", () => {
  describe("checkTerms", () => {
    it("should require a channel count when the channel limit is on", () => {
      expect(issues({ channelsLimited: true, channels: 0 })).toEqual([
        { path: ["channels"], message: "At least one channel" },
      ]);
    });

    it("should reject a fractional channel count when the channel limit is on", () => {
      expect(issues({ channelsLimited: true, channels: 1.5 })).toEqual([
        { path: ["channels"], message: "Use a whole number" },
      ]);
    });

    it("should ignore the channel count when the channel limit is off", () => {
      expect(issues({ channels: 0 })).toEqual([]);
      expect(issues({ channels: 1.5 })).toEqual([]);
    });

    it("should ignore the maximum version on a subscription", () => {
      expect(issues({ maxVersion: "0.62.1" })).toEqual([]);
    });

    it("should require a maximum version on a perpetual license", () => {
      expect(issues({ term: "perpetual", expiresAt: "" })).toEqual([
        {
          path: ["maxVersion"],
          message: "A perpetual license needs a maximum version",
        },
      ]);
    });

    it("should require a maximum version on a perpetual license with updates", () => {
      expect(issues({ term: "updates" })).toEqual([
        {
          path: ["maxVersion"],
          message: "A perpetual license needs a maximum version",
        },
      ]);
    });

    it("should require the date updates end on a license with updates", () => {
      expect(issues({ term: "updates", expiresAt: "", maxVersion: "0.62" })).toEqual([
        { path: ["expiresAt"], message: "Enter the date updates end" },
      ]);
    });

    it("should reject a version that is not major.minor", () => {
      expect(issues({ term: "perpetual", maxVersion: "0.62.1" })).toEqual([
        { path: ["maxVersion"], message: "Use major.minor, like 0.62" },
      ]);
    });
  });
});
