// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import {
  deepCopy,
  findNode,
  findNodeParent,
  findNodes,
  flatten,
  getAllNodesOfMinDepth,
  getDescendants,
  moveNode,
  type Node,
  type NodeWithPosition,
  removeNode,
  setNode,
  sortAndSplice,
  updateNode,
} from "@/tree/core";

describe("Tree", () => {
  describe("sortAndSplice", () => {
    it("should correctly temporarily force position of a node higher up than it should be", () => {
      const nodes: Node[] = [
        { key: "1", name: "1" },
        { key: "2", name: "2" },
        { key: "3", name: "3", forcePosition: 1 },
      ];
      const result = sortAndSplice(nodes, true);
      expect(result).toEqual([
        { key: "1", name: "1" },
        { key: "3", name: "3", forcePosition: 1 },
        { key: "2", name: "2" },
      ]);
    });
    it("should correctly temporarily force position of a node lower down than it should be", () => {
      const nodes: Node[] = [
        { key: "1", name: "1", forcePosition: 1 },
        { key: "2", name: "2" },
        { key: "3", name: "3" },
      ];
      const result = sortAndSplice(nodes);
      expect(result).toEqual([
        { key: "2", name: "2" },
        { key: "1", name: "1", forcePosition: 1 },
        { key: "3", name: "3" },
      ]);
    });
  });

  describe("flatten", () => {
    it("should correctly flatten a tree structure", () => {
      const nodes: Node[] = [
        {
          key: "1",
          name: "parent1",
          children: [
            { key: "1-1", name: "child1" },
            { key: "1-2", name: "child2" },
          ],
        },
        { key: "2", name: "parent2" },
      ];
      const expanded = ["1"];
      const result = flatten({ nodes, expanded });
      expect(result).toEqual([
        {
          key: "1",
          name: "parent1",
          depth: 0,
          expanded: true,
          index: 0,
          path: "1/",
          children: [
            { key: "1-1", name: "child1" },
            { key: "1-2", name: "child2" },
          ],
        },
        {
          key: "1-1",
          name: "child1",
          depth: 1,
          expanded: false,
          index: 0,
          path: "1/1-1/",
        },
        {
          key: "1-2",
          name: "child2",
          depth: 1,
          expanded: false,
          index: 1,
          path: "1/1-2/",
        },
        { key: "2", name: "parent2", depth: 0, expanded: false, index: 1, path: "2/" },
      ]);
    });

    it("should respect sort option", () => {
      const nodes: Node[] = [
        { key: "2", name: "B" },
        { key: "1", name: "A" },
      ];
      const result = flatten({ nodes, expanded: [], sort: true });
      expect(result[0].name).toBe("A");
      expect(result[1].name).toBe("B");
    });
  });

  describe("moveNode", () => {
    it("should move nodes to root when destination is null", () => {
      const tree: Node[] = [
        {
          key: "1",
          name: "parent",
          children: [{ key: "2", name: "child" }],
        },
      ];
      const result = moveNode({ tree, destination: null, keys: "2" });
      expect(result).toHaveLength(2);
      expect(result.find((n) => n.key === "2")).toBeDefined();
    });

    it("should move node to new parent", () => {
      const tree: Node[] = [
        { key: "1", name: "parent1" },
        { key: "2", name: "parent2", children: [] },
      ];
      const result = moveNode({ tree, destination: "2", keys: "1" });
      expect(result[0].children?.[0].key).toBe("1");
    });
  });

  describe("removeNode", () => {
    it("should remove node from root level", () => {
      const tree: Node[] = [
        { key: "1", name: "node1" },
        { key: "2", name: "node2" },
      ];
      const result = removeNode({ tree, keys: "1" });
      expect(result).toHaveLength(1);
      expect(result[0].key).toBe("2");
    });

    it("should remove node from nested level", () => {
      const tree: Node[] = [
        {
          key: "1",
          name: "parent",
          children: [{ key: "2", name: "child" }],
        },
      ];
      const result = removeNode({ tree, keys: "2" });
      expect(result[0].children).toHaveLength(0);
    });
  });

  describe("setNode", () => {
    it("should add nodes to root when destination is null", () => {
      const tree: Node[] = [{ key: "1", name: "existing" }];
      const addition: Node = { key: "2", name: "new" };
      const result = setNode({ tree, destination: null, additions: addition });
      expect(result).toHaveLength(2);
      expect(result.find((n) => n.key === "2")).toBeDefined();
    });

    it("should add nodes to specified parent", () => {
      const tree: Node[] = [{ key: "1", name: "parent", children: [] }];
      const addition: Node = { key: "2", name: "child" };
      const result = setNode({ tree, destination: "1", additions: addition });
      expect(result[0].children).toHaveLength(1);
      expect(result[0].children?.[0].key).toBe("2");
    });
  });

  describe("updateNode", () => {
    it("should update node properties", () => {
      const tree: Node[] = [{ key: "1", name: "old" }];
      const result = updateNode({
        tree,
        key: "1",
        updater: (node) => ({ ...node, name: "new" }),
      });
      expect(result[0].name).toBe("new");
    });

    it("should throw on missing node when throwOnMissing is true", () => {
      const tree: Node[] = [];
      expect(() =>
        updateNode({
          tree,
          key: "missing",
          updater: (node) => node,
        }),
      ).toThrow();
    });
  });

  describe("findNode", () => {
    it("should find node and return position info", () => {
      const tree: Node[] = [
        {
          key: "1",
          name: "parent",
          children: [{ key: "2", name: "child" }],
        },
      ];
      const result = findNode({ tree, key: "2" });
      expect(result?.depth).toBe(1);
      expect(result?.position).toBe(0);
    });

    it("should return null for non-existent node", () => {
      const tree: Node[] = [{ key: "1", name: "node" }];
      const result = findNode({ tree, key: "missing" });
      expect(result).toBeNull();
    });
  });

  describe("findNodes", () => {
    it("should find multiple nodes", () => {
      const tree: Node[] = [
        { key: "1", name: "node1" },
        { key: "2", name: "node2" },
      ];
      const result = findNodes({ tree, keys: ["1", "2"] });
      expect(result).toHaveLength(2);
    });
  });

  describe("findNodeParent", () => {
    it("should find parent of nested node", () => {
      const tree: Node[] = [
        {
          key: "1",
          name: "parent",
          children: [{ key: "2", name: "child" }],
        },
      ];
      const result = findNodeParent({ tree, key: "2" });
      expect(result?.key).toBe("1");
    });

    it("should return null for root level node", () => {
      const tree: Node[] = [{ key: "1", name: "root" }];
      const result = findNodeParent({ tree, key: "1" });
      expect(result).toBeNull();
    });
  });

  describe("deepCopy", () => {
    it("should create deep copy of tree structure", () => {
      const original: Node[] = [
        {
          key: "1",
          name: "parent",
          children: [{ key: "2", name: "child" }],
        },
      ];
      const copy = deepCopy(original);
      copy[0].name = "modified";
      expect(original[0].name).toBe("parent");
    });
  });

  describe("getDescendants", () => {
    it("should return all descendants of nodes", () => {
      const node: Node = {
        key: "1",
        name: "parent",
        children: [
          { key: "2", name: "child1" },
          {
            key: "3",
            name: "child2",
            children: [{ key: "4", name: "grandchild" }],
          },
        ],
      };
      const result = getDescendants(node);
      expect(result).toHaveLength(4);
      expect(result.map((n) => n.key)).toEqual(["1", "2", "3", "4"]);
    });
  });

  describe("getAllNodesOfMinDepth", () => {
    it("should return nodes at minimum depth", () => {
      const nodes: NodeWithPosition[] = [
        { key: "1", name: "root1", depth: 0, position: 0 },
        { key: "2", name: "root2", depth: 0, position: 1 },
        { key: "3", name: "child", depth: 1, position: 0 },
      ];
      const result = getAllNodesOfMinDepth(nodes);
      expect(result).toHaveLength(2);
      expect(result.every((n) => n.depth === 0)).toBe(true);
    });

    it("should return empty array for empty input", () => {
      const result = getAllNodesOfMinDepth([]);
      expect(result).toHaveLength(0);
    });
  });
});
