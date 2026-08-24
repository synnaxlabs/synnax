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
  });

  describe("edgeChangesToActions", () => {
    it("maps an add change to add_edge carrying the edge key and a continuous kind", () => {
      const source = handle("a", "out");
      const target = handle("b", "in");
      const edge = { key: "e1", source, target };
      expect(edgeChangesToActions([{ type: "add", edge }])).toEqual([
        arc.addEdge({
          edge: { key: "e1", source, target, kind: arc.ir.EdgeKind.continuous },
        }),
      ]);
    });
    it("maps a remove change to remove_edge by key", () => {
      expect(edgeChangesToActions([{ type: "remove", key: "e1" }])).toEqual([
        arc.removeEdge({ key: "e1" }),
      ]);
    });
    it("maps a reconnect change to reconnect_edge", () => {
      const source = handle("a", "out");
      const target = handle("c", "in");
      expect(
        edgeChangesToActions([{ type: "reconnect", key: "e1", source, target }]),
      ).toEqual([arc.reconnectEdge({ key: "e1", source, target })]);
    });
  });
});
