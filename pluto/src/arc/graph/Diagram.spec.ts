// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import { edgeChangesToActions, nodeChangesToActions } from "@/arc/graph/Diagram";

const handle = (node: string, param: string): arc.ir.Handle => ({ node, param });

describe("arc gesture converters", () => {
  describe("nodeChangesToActions", () => {
    it("maps a position change to set_node_position", () => {
      expect(
        nodeChangesToActions([
          { type: "position", key: "n1", position: { x: 5, y: 6 }, dragging: false },
        ]),
      ).toEqual([arc.setNodePosition({ key: "n1", position: { x: 5, y: 6 } })]);
    });
    it("maps a remove change to remove_node", () => {
      expect(nodeChangesToActions([{ type: "remove", key: "n1" }])).toEqual([
        arc.removeNode({ key: "n1" }),
      ]);
    });
    it("drops dimension changes, which Arc does not persist", () => {
      expect(
        nodeChangesToActions([
          { type: "dimensions", key: "n1", dimensions: { width: 1, height: 1 } },
        ]),
      ).toEqual([]);
    });
  });

  describe("edgeChangesToActions", () => {
    it("maps an add change to add_edge with a continuous kind", () => {
      const source = handle("a", "out");
      const target = handle("b", "in");
      const edge = { key: arc.ir.edgeKey(source, target), source, target };
      expect(edgeChangesToActions([{ type: "add", edge }])).toEqual([
        arc.addEdge({ edge: { source, target, kind: arc.ir.EdgeKind.continuous } }),
      ]);
    });
    it("recovers the endpoints from the diagram key on remove", () => {
      const source = handle("a", "out");
      const target = handle("b", "in");
      expect(
        edgeChangesToActions([{ type: "remove", key: arc.ir.edgeKey(source, target) }]),
      ).toEqual([arc.removeEdge({ source, target })]);
    });
  });
});
