// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { deriveNodeState, resolveTraceColor } from "@/components/stream/diagrams/theme";

describe("theme", () => {
  describe("deriveNodeState", () => {
    it("should mark a node in no list as inactive", () => {
      expect(deriveNodeState("p1", [], [], [])).toBe("inactive");
    });

    it("should mark an active node as active", () => {
      expect(deriveNodeState("p1", ["p1"], [], [])).toBe("active");
    });

    it("should rank an alarm above activity", () => {
      expect(deriveNodeState("p1", ["p1"], [], ["p1"])).toBe("alarm");
    });

    it("should rank an exclusion above an alarm and activity", () => {
      expect(deriveNodeState("p1", ["p1"], ["p1"], ["p1"])).toBe("excluded");
    });

    it("should treat missing alarm nodes as none", () => {
      expect(deriveNodeState("p1", ["p1"], [])).toBe("active");
    });
  });

  describe("resolveTraceColor", () => {
    it("should dim an idle trace", () => {
      expect(resolveTraceColor(false, false)).toBe("var(--pluto-gray-l2)");
    });

    it("should color a flowing trace in the primary color", () => {
      expect(resolveTraceColor(true, false)).toBe("var(--pluto-primary-z-15)");
    });

    it("should rank an exclusion above flow", () => {
      expect(resolveTraceColor(true, true)).toBe("var(--pluto-error-z-15)");
    });
  });
});
