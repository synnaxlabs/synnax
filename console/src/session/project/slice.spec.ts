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

describe("project slice", () => {
  const selected = "00000000-0000-0000-0000-000000000001";

  describe("select", () => {
    it("should set the selected project", () => {
      const next = Project.reducer(Project.ZERO_SLICE_STATE, Project.select(selected));
      expect(next.selected).toEqual(selected);
    });
  });

  describe("clearSelection", () => {
    it("should clear the selected project", () => {
      const next = Project.reducer(
        { ...Project.ZERO_SLICE_STATE, selected },
        Project.clearSelection(),
      );
      expect(next.selected).toBeUndefined();
    });
  });
});
