// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Link } from "@/session/link";

describe("link slice", () => {
  describe("beginProjectWait", () => {
    it("should mark a link as awaiting a project selection", () => {
      const next = Link.reducer(Link.ZERO_SLICE_STATE, Link.beginProjectWait());
      expect(next.awaitingProject).toBe(true);
    });
  });

  describe("endProjectWait", () => {
    it("should clear the awaiting flag", () => {
      const next = Link.reducer(
        { ...Link.ZERO_SLICE_STATE, awaitingProject: true },
        Link.endProjectWait(),
      );
      expect(next.awaitingProject).toBe(false);
    });

    it("should clear the flag on a single end even after repeated begins", () => {
      const begun = Link.reducer(Link.ZERO_SLICE_STATE, Link.beginProjectWait());
      const again = Link.reducer(begun, Link.beginProjectWait());
      const next = Link.reducer(again, Link.endProjectWait());
      expect(next.awaitingProject).toBe(false);
    });
  });
});
