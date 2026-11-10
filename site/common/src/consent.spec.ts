// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { modeFor } from "./consent";

const EU =
  "AT BE BG CY CZ DE DK EE ES FI FR GR HR HU IE IT LT LU LV MT NL PL PT RO SE SI SK";

describe("consent", () => {
  describe("modeFor", () => {
    it("should ask a visitor from the EU, the EEA, the UK, or Switzerland", () => {
      for (const country of `${EU} IS LI NO GB CH`.split(" "))
        expect(modeFor(country)).toBe("ask");
    });

    it("should give notice to a visitor from any other country", () => {
      expect(modeFor("US")).toBe("notice");
      expect(modeFor("JP")).toBe("notice");
      expect(modeFor("BR")).toBe("notice");
    });

    it("should ask for the reserved codes of the EU and the UK", () => {
      expect(modeFor("EU")).toBe("ask");
      expect(modeFor("UK")).toBe("ask");
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
