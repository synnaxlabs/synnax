// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { filename } from "@/server/license/limits";

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
  });
});
