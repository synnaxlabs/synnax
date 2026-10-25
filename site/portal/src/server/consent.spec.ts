// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { ASK, modeFor } from "@/server/consent";

describe("consent", () => {
  describe("ASK", () => {
    it("should hold the EU, the rest of the EEA, the UK, and Switzerland", () => {
      expect(ASK.size).toBe(32);
      for (const country of ["DE", "FR", "GR", "IE", "IS", "LI", "NO", "GB", "CH"])
        expect(ASK.has(country)).toBe(true);
    });
  });

  describe("modeFor", () => {
    it("should ask a visitor from a country on the list", () => {
      expect(modeFor("DE")).toBe("ask");
      expect(modeFor("GB")).toBe("ask");
      expect(modeFor("CH")).toBe("ask");
    });

    it("should give notice to a visitor from any other country", () => {
      expect(modeFor("US")).toBe("notice");
      expect(modeFor("JP")).toBe("notice");
      expect(modeFor("BR")).toBe("notice");
    });

    it("should ask when the country is missing", () => {
      expect(modeFor(null)).toBe("ask");
      expect(modeFor("")).toBe("ask");
    });

    it("should ask when the country is unknown", () => {
      expect(modeFor("XX")).toBe("ask");
      expect(modeFor("ZZ")).toBe("ask");
      expect(modeFor("T1")).toBe("ask");
      expect(modeFor("us")).toBe("ask");
      expect(modeFor("USA")).toBe("ask");
    });
  });
});
