// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { id } from "@/id";

describe("id", () => {
  describe("generate", () => {
    it("should generate a string", () => {
      const newID = id.generate();
      expect(typeof newID).toBe("string");
    });

    it("should generate a string of length 11", () => {
      const newID = id.generate();
      expect(newID.length).toBe(11);
    });

    it("should only contain alphanumeric characters", () => {
      const newID = id.generate();
      expect(newID).toMatch(/^[0-9A-Za-z]+$/);
    });

    it("should generate unique IDs", () => {
      const ids = new Set<string>();
      for (let i = 0; i < 100; i++) {
        const newID = id.generate();
        expect(ids.has(newID)).toBe(false);
        ids.add(newID);
      }
    });
  });
});
