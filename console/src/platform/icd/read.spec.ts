// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { ICD } from "@/platform/icd";

describe("ICD", () => {
  describe("mostRead", () => {
    it("should return undefined when no source reads an ICD", () => {
      expect(ICD.mostRead([])).toBeUndefined();
    });

    it("should return the ICD that the most sources read", () => {
      expect(ICD.mostRead([{ icd: "a" }, { icd: "b" }, { icd: "b" }])).toBe("b");
    });

    it("should return the first ICD that a source reads when counts tie", () => {
      expect(
        ICD.mostRead([{ icd: "a" }, { icd: "b" }, { icd: "b" }, { icd: "a" }]),
      ).toBe("a");
    });
  });
});
