// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import * as definitions from "@/components/stream/diagrams/definitions";
import type { NodeDef } from "@/components/stream/diagrams/types";

const DIAGRAMS = Object.entries(definitions);

const overlap = (a: NodeDef, b: NodeDef): boolean =>
  Math.abs(a.x - b.x) < (a.w + b.w) / 2 && Math.abs(a.y - b.y) < (a.h + b.h) / 2;

describe("definitions", () => {
  describe.each(DIAGRAMS)("%s", (_, diagram) => {
    const nodes = new Map(diagram.nodes.map((n) => [n.id, n]));

    it("should give each node its own ID", () => {
      expect(nodes.size).toBe(diagram.nodes.length);
    });

    it("should connect only nodes that exist", () => {
      for (const { from, to } of diagram.edges) {
        expect(nodes.has(from)).toBe(true);
        expect(nodes.has(to)).toBe(true);
      }
    });

    it("should run every edge from left to right", () => {
      for (const { from, to } of diagram.edges)
        expect(nodes.get(from)!.x).toBeLessThan(nodes.get(to)!.x);
    });

    it("should connect every node", () => {
      const connected = new Set(diagram.edges.flatMap((e) => [e.from, e.to]));
      expect(connected).toEqual(new Set(nodes.keys()));
    });

    it("should keep every node inside the view box", () => {
      const [minX, minY, width, height] = diagram.viewBox.split(" ").map(Number);
      for (const n of diagram.nodes) {
        expect(n.x - n.w / 2).toBeGreaterThanOrEqual(minX);
        expect(n.y - n.h / 2).toBeGreaterThanOrEqual(minY);
        expect(n.x + n.w / 2).toBeLessThanOrEqual(minX + width);
        expect(n.y + n.h / 2).toBeLessThanOrEqual(minY + height);
      }
    });

    it("should not overlap two nodes", () => {
      for (const [i, a] of diagram.nodes.entries())
        for (const b of diagram.nodes.slice(i + 1)) expect(overlap(a, b)).toBe(false);
    });
  });
});
