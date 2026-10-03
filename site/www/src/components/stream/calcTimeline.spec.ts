// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";
import { describe, expect, it } from "vitest";

import { CALC_EXAMPLES } from "@/components/stream/calcTimeline";

describe("calcTimeline", () => {
  it("should give each example its own ID", () => {
    expect(new Set(CALC_EXAMPLES.map((e) => e.id)).size).toBe(CALC_EXAMPLES.length);
  });

  describe.each(CALC_EXAMPLES.map((e) => [e.id, e] as const))("%s", (_, example) => {
    const ids = new Set(example.diagram.nodes.map((n) => n.id));

    it("should name every channel of its diagram in its code", () => {
      const channels = example.diagram.nodes.filter((n) => n.icon === Icon.Channel);
      expect(channels.length).toBeGreaterThan(0);
      for (const { label } of channels) expect(example.code).toContain(label);
    });

    it("should highlight only lines that exist in its code", () => {
      const count = example.code.split("\n").length;
      for (const { activeLines } of example.steps)
        for (const line of activeLines) {
          expect(line).toBeGreaterThanOrEqual(1);
          expect(line).toBeLessThanOrEqual(count);
        }
    });

    it("should refer only to nodes in its diagram", () => {
      for (const { state } of example.steps) {
        const refs = [
          ...(state.activeNodes ?? []),
          ...Object.keys(state.nodeValues ?? {}),
          ...(state.excludedNodes ?? []),
          ...(state.alarmNodes ?? []),
        ];
        for (const ref of refs) expect(ids).toContain(ref);
      }
    });

    it("should hold each step for a positive time with a node active", () => {
      for (const { duration, state } of example.steps) {
        expect(duration).toBeGreaterThan(0);
        expect(state.activeNodes?.length).toBeGreaterThan(0);
      }
    });
  });
});
