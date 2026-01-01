// Copyright 2026 Synnax Labs, Inc.
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
  describe("create", () => {
    it("should create a string", () => {
      const newID = id.create();
      expect(typeof newID).toBe("string");
    });

    it("should create a string of length 11", () => {
      const newID = id.create();
      expect(newID.length).toBe(id.LENGTH);
    });

    it("should only contain alphanumeric characters", () => {
      const newID = id.create();
      expect(newID).toMatch(/^[0-9A-Za-z]+$/);
    });

    it("should create unique IDs", () => {
      const ids = new Set<string>();
      for (let i = 0; i < 100; i++) {
        const newID = id.create();
        expect(ids.has(newID)).toBe(false);
        ids.add(newID);
      }
    });
  });
});
