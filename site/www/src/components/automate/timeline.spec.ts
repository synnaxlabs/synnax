// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { EXAMPLES } from "@/components/automate/timeline";

describe("timeline", () => {
  describe.each(EXAMPLES.map((e) => [e.id, e] as const))("%s", (_, example) => {
    it("should highlight only lines that exist in its code", () => {
      const count = example.code.split("\n").length;
      for (const { activeLines } of example.steps) {
        expect(activeLines.length).toBeGreaterThan(0);
        for (const line of activeLines) {
          expect(line).toBeGreaterThanOrEqual(1);
          expect(line).toBeLessThanOrEqual(count);
        }
      }
    });

    it("should hold each step for a positive time", () => {
      for (const { duration } of example.steps) expect(duration).toBeGreaterThan(0);
    });

    it("should visit each stage that its code declares", () => {
      const stages = new Set(example.steps.map((s) => s.state.stage));
      for (const [, stage] of example.code.matchAll(/^\s*stage (\w+)/gm))
        expect(stages).toContain(stage);
    });
  });
});
