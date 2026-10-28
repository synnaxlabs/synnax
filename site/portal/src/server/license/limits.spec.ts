// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { channels, filename, term } from "@/server/license/limits";
import { LICENSE } from "@/server/license/testutil";

describe("limits", () => {
  describe("filename", () => {
    it("should turn the license label into a file name", () => {
      expect(filename("Test rig")).toBe("Test-rig.lic");
    });

    it("should collapse and trim runs of other characters", () => {
      expect(filename("  Acme / Rig #2!  ")).toBe("Acme-Rig-2.lic");
    });

    it("should fall back to synnax for a label with nothing usable", () => {
      expect(filename("")).toBe("synnax.lic");
      expect(filename("///")).toBe("synnax.lic");
    });

    it("should add the name of the machine the key is for", () => {
      expect(filename("Test rig", "Stand #3")).toBe("Test-rig-Stand-3.lic");
    });

    it("should leave out a machine with no name", () => {
      expect(filename("Test rig", null)).toBe("Test-rig.lic");
    });
  });

  describe("term", () => {
    it("should give a subscription's expiry", () => {
      expect(term(LICENSE)).toBe("Until 2027-03-01");
    });

    it("should give the version a subscription keeps after it expires", () => {
      expect(term({ ...LICENSE, maxVersion: "0.60" })).toBe(
        "Until 2027-03-01, then up to v0.60",
      );
    });

    it("should give a perpetual license's last version", () => {
      expect(
        term({ ...LICENSE, term: "perpetual", expiresAt: null, maxVersion: "0.62" }),
      ).toBe("Perpetual, up to v0.62");
    });
  });

  describe("channels", () => {
    it("should call a cap of 0 unlimited", () => {
      expect(channels(0)).toBe("Unlimited");
    });

    it("should give any other cap as a number", () => {
      expect(channels(500)).toBe("500");
    });
  });
});
