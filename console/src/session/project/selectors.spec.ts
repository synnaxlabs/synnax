// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Project } from "@/session/project";

describe("project selectors", () => {
  const selected = "00000000-0000-0000-0000-000000000001";
  const withSelected = {
    [Project.SLICE_NAME]: { ...Project.ZERO_SLICE_STATE, selected },
  };
  const withoutSelected = { [Project.SLICE_NAME]: Project.ZERO_SLICE_STATE };

  describe("with a selected project", () => {
    it("should return the selected project from both selectors", () => {
      expect(Project.selectSelected(withSelected)).toEqual(selected);
      expect(Project.selectOptionalSelected(withSelected)).toEqual(selected);
    });
  });

  describe("without a selected project", () => {
    it("should return undefined from the optional selector", () => {
      expect(Project.selectOptionalSelected(withoutSelected)).toBeUndefined();
    });

    it("should throw from the required selector", () => {
      expect(() => Project.selectSelected(withoutSelected)).toThrow();
    });
  });
});
