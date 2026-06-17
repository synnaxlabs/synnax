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

import { edgeChangesToActions, nodeChangesToActions } from "@/arc/Graph";
import { edgeKey, edgesToDiagram, nodeProps, parseEdgeKey } from "@/arc/translate";

const handle = (node: string, param: string): arc.ir.Handle => ({ node, param });

describe("arc translate", () => {
  describe("edge key", () => {
    it("round-trips a source and target handle through the diagram key", () => {
      const source = handle("a", "out");
      const target = handle("b", "in");
      expect(parseEdgeKey(edgeKey(source, target))).toEqual({ source, target });
    });
  });

  describe("edgesToDiagram", () => {
    it("derives a keyed diagram edge for each server edge", () => {
      const edges: arc.ir.Edge[] = [
        { source: handle("a", "out"), target: handle("b", "in"), kind: 1 },
      ];
      const out = edgesToDiagram(edges);
      expect(out.length).toEqual(1);
      expect(out[0].source).toEqual(handle("a", "out"));
      expect(out[0].target).toEqual(handle("b", "in"));
      expect(parseEdgeKey(out[0].key)).toEqual({
        source: handle("a", "out"),
        target: handle("b", "in"),
      });
    });
  });

  describe("nodeProps", () => {
    it("surfaces the function type under key and merges config", () => {
      const node: arc.graph.Node = {
        key: "n1",
        type: "stage",
        config: { gain: 2 },
        position: { x: 0, y: 0 },
      };
      expect(nodeProps(node)).toEqual({ key: "stage", gain: 2 });
    });
  });
});

describe("arc gesture converters", () => {
  describe("nodeChangesToActions", () => {
    it("maps a position change to set_node_position", () => {
      const actions = nodeChangesToActions([
        { type: "position", key: "n1", position: { x: 5, y: 6 }, dragging: false },
      ]);
      expect(actions).toEqual([
        arc.setNodePosition({ key: "n1", position: { x: 5, y: 6 } }),
      ]);
    });
    it("maps a remove change to remove_node", () => {
      const actions = nodeChangesToActions([{ type: "remove", key: "n1" }]);
      expect(actions).toEqual([arc.removeNode({ key: "n1" })]);
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
      const edge = {
        key: edgeKey(handle("a", "out"), handle("b", "in")),
        source: handle("a", "out"),
        target: handle("b", "in"),
      };
      expect(edgeChangesToActions([{ type: "add", edge }])).toEqual([
        arc.addEdge({
          edge: {
            source: handle("a", "out"),
            target: handle("b", "in"),
            kind: arc.ir.EdgeKind.continuous,
          },
        }),
      ]);
    });
    it("maps a remove change back to remove_edge via the key", () => {
      const key = edgeKey(handle("a", "out"), handle("b", "in"));
      expect(edgeChangesToActions([{ type: "remove", key }])).toEqual([
        arc.removeEdge({ source: handle("a", "out"), target: handle("b", "in") }),
      ]);
    });
  });
});
