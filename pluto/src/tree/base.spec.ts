// Copyright 2026 Synnax Labs, Inc.
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
  filterShape,
  findNode,
  findNodeParent,
  findNodes,
  flatten,
  getAllNodesOfMinDepth,
  getDepth,
  getDescendants,
  getNodeShape,
  moveNode,
  type Node,
  removeNode,
  setNode,
  type Shape,
  shouldExpand,
  updateNode,
  updateNodeChildren,
} from "@/tree/base";

describe("tree/base", () => {
  const simpleTree: Node[] = [
    {
      key: "root1",
      children: [
        { key: "child1" },
        {
          key: "child2",
          children: [{ key: "grandchild1" }, { key: "grandchild2" }],
        },
      ],
    },
    { key: "root2", children: [{ key: "child3" }] },
  ];

  describe("shouldExpand", () => {
    it("should return true when node key is in expanded list", () => {
      const node = { key: "test" };
      expect(shouldExpand(node, ["test", "other"])).toBe(true);
    });

    it("should return false when node key is not in expanded list", () => {
      const node = { key: "test" };
      expect(shouldExpand(node, ["other"])).toBe(false);
    });
  });

  describe("flatten", () => {
    it("should flatten nodes without expansion", () => {
      const result = flatten({ nodes: simpleTree, expanded: [] });
      expect(result.keys).toEqual(["root1", "root2"]);
      expect(result.nodes).toEqual([
        { depth: 0, expanded: false, hasChildren: true },
        { depth: 0, expanded: false, hasChildren: true },
      ]);
    });

    it("should flatten with expanded nodes", () => {
      const result = flatten({ nodes: simpleTree, expanded: ["root1", "child2"] });
      expect(result.keys).toEqual([
        "root1",
        "child1",
        "child2",
        "grandchild1",
        "grandchild2",
        "root2",
      ]);
      expect(result.nodes).toEqual([
        { depth: 0, expanded: true, hasChildren: true },
        { depth: 1, expanded: false, hasChildren: false },
        { depth: 1, expanded: true, hasChildren: true },
        { depth: 2, expanded: false, hasChildren: false },
        { depth: 2, expanded: false, hasChildren: false },
        { depth: 0, expanded: false, hasChildren: true },
      ]);
    });

    it("should sort nodes when comparator provided", () => {
      const nodes: Node[] = [
        { key: "b", children: [{ key: "d" }, { key: "c" }] },
        { key: "a" },
      ];
      const result = flatten({
        nodes,
        expanded: ["b"],
        sort: (a, b) => a.key.localeCompare(b.key),
      });
      expect(result.keys).toEqual(["a", "b", "c", "d"]);
    });

    it("should handle custom initial depth", () => {
      const result = flatten({ nodes: simpleTree, expanded: [], depth: 2 });
      expect(result.nodes[0].depth).toBe(2);
      expect(result.nodes[1].depth).toBe(2);
    });
  });

  describe("moveNode", () => {
    it("should move node to root when destination is null", () => {
      const tree = deepCopy(simpleTree);
      const result = moveNode({ tree, destination: null, keys: "child1" });
      expect(result[0].key).toBe("child1");
      expect(result[1].key).toBe("root1");
      expect(result.filter((n) => n.key === "child1").length).toBe(1);
    });

    it("should move multiple nodes to destination", () => {
      const tree = deepCopy(simpleTree);
      moveNode({ tree, destination: "root2", keys: ["child1", "child2"] });
      expect(tree[1].children?.map((c) => c.key)).toContain("child1");
      expect(tree[1].children?.map((c) => c.key)).toContain("child2");
      expect(tree[0].children?.length).toBe(0);
    });

    it("should handle non-existent node gracefully", () => {
      const tree = deepCopy(simpleTree);
      const result = moveNode({ tree, destination: "root2", keys: "nonexistent" });
      expect(result).toEqual(tree);
    });
  });

  describe("removeNode", () => {
    it("should remove node from root", () => {
      const tree = deepCopy(simpleTree);
      const result = removeNode({ tree, keys: "root1" });
      expect(result.length).toBe(1);
      expect(result[0].key).toBe("root2");
    });

    it("should remove nested node", () => {
      const tree = deepCopy(simpleTree);
      removeNode({ tree, keys: "child1" });
      expect(tree[0].children?.find((c) => c.key === "child1")).toBeUndefined();
      expect(tree[0].children?.length).toBe(1);
    });

    it("should remove multiple nodes", () => {
      const tree = deepCopy(simpleTree);
      removeNode({ tree, keys: ["child1", "child2"] });
      expect(tree[0].children?.length).toBe(0);
    });

    it("should remove from specific parent when provided", () => {
      const tree = deepCopy(simpleTree);
      removeNode({ tree, keys: "child1", parent: "root1" });
      expect(tree[0].children?.find((c) => c.key === "child1")).toBeUndefined();
    });

    it("should remove from root when parent is null", () => {
      const tree = deepCopy(simpleTree);
      const result = removeNode({ tree, keys: "root1", parent: null });
      expect(result.length).toBe(1);
      expect(result[0].key).toBe("root2");
    });
  });

  describe("setNode", () => {
    it("should add node to root when destination is null", () => {
      const tree = deepCopy(simpleTree);
      const result = setNode({
        tree,
        destination: null,
        additions: { key: "newRoot" },
      });
      expect(result[0].key).toBe("newRoot");
      expect(result.length).toBe(3);
    });

    it("should add node to existing parent", () => {
      const tree = deepCopy(simpleTree);
      setNode({ tree, destination: "root2", additions: { key: "newChild" } });
      expect(tree[1].children?.find((c) => c.key === "newChild")).toBeDefined();
    });

    it("should replace duplicate keys", () => {
      const tree = deepCopy(simpleTree);
      setNode({
        tree,
        destination: "root1",
        additions: { key: "child1", children: [{ key: "new" }] },
      });
      const child1 = tree[0].children?.find((c) => c.key === "child1");
      expect(child1?.children?.[0].key).toBe("new");
    });

    it("should throw when destination not found and throwOnMissing is true", () => {
      const tree = deepCopy(simpleTree);
      expect(() =>
        setNode({ tree, destination: "nonexistent", additions: { key: "new" } }),
      ).toThrow("Could not find node with key nonexistent");
    });

    it("should not throw when destination not found and throwOnMissing is false", () => {
      const tree = deepCopy(simpleTree);
      const result = setNode({
        tree,
        destination: "nonexistent",
        additions: { key: "new" },
        throwOnMissing: false,
      });
      expect(result).toEqual(tree);
    });

    it("should handle multiple additions with deduplication", () => {
      const tree = deepCopy(simpleTree);
      const result = setNode({
        tree,
        destination: null,
        additions: [{ key: "new1" }, { key: "new1" }, { key: "new2" }],
      });
      expect(result.filter((n) => n.key === "new1").length).toBe(1);
      expect(result.find((n) => n.key === "new2")).toBeDefined();
    });
  });

  describe("updateNode", () => {
    it("should update root node", () => {
      const tree = deepCopy(simpleTree);
      updateNode({
        tree,
        key: "root1",
        updater: (node) => ({ ...node, key: "updatedRoot1" }),
      });
      expect(tree[0].key).toBe("updatedRoot1");
    });

    it("should update nested node", () => {
      const tree = deepCopy(simpleTree);
      updateNode({
        tree,
        key: "child1",
        updater: (node) => ({ ...node, key: "updatedChild1" }),
      });
      expect(tree[0].children?.[0].key).toBe("updatedChild1");
    });

    it("should throw when node not found and throwOnMissing is true", () => {
      const tree = deepCopy(simpleTree);
      expect(() =>
        updateNode({
          tree,
          key: "nonexistent",
          updater: (node) => node,
        }),
      ).toThrow("Could not find node with key nonexistent");
    });

    it("should not throw when node not found and throwOnMissing is false", () => {
      const tree = deepCopy(simpleTree);
      const result = updateNode({
        tree,
        key: "nonexistent",
        updater: (node) => node,
        throwOnMissing: false,
      });
      expect(result).toEqual(tree);
    });
  });

  describe("updateNodeChildren", () => {
    it("should update children of a node", () => {
      const tree = deepCopy(simpleTree);
      updateNodeChildren({
        tree,
        parent: "root1",
        updater: (children) => [...children, { key: "newChild" }],
      });
      expect(tree[0].children?.find((c) => c.key === "newChild")).toBeDefined();
    });

    it("should handle node with no children", () => {
      const tree = deepCopy(simpleTree);
      updateNodeChildren({
        tree,
        parent: "child1",
        updater: () => [{ key: "newGrandchild" }],
      });
      const child1 = findNode({ tree, key: "child1" });
      expect(child1?.children?.[0].key).toBe("newGrandchild");
    });

    it("should throw when parent not found and throwOnMissing is true", () => {
      const tree = deepCopy(simpleTree);
      expect(() =>
        updateNodeChildren({
          tree,
          parent: "nonexistent",
          updater: (children) => children,
        }),
      ).toThrow("Could not find node with key nonexistent");
    });

    it("should not throw when parent not found and throwOnMissing is false", () => {
      const tree = deepCopy(simpleTree);
      const result = updateNodeChildren({
        tree,
        parent: "nonexistent",
        updater: (children) => children,
        throwOnMissing: false,
      });
      expect(result).toEqual(tree);
    });
  });

  describe("findNode", () => {
    it("should find root node", () => {
      const result = findNode({ tree: simpleTree, key: "root1" });
      expect(result?.key).toBe("root1");
    });

    it("should find nested node", () => {
      const result = findNode({ tree: simpleTree, key: "grandchild1" });
      expect(result?.key).toBe("grandchild1");
    });

    it("should return null for non-existent node", () => {
      const result = findNode({ tree: simpleTree, key: "nonexistent" });
      expect(result).toBeNull();
    });
  });

  describe("findNodes", () => {
    it("should find multiple nodes", () => {
      const result = findNodes({
        tree: simpleTree,
        keys: ["root1", "child1", "grandchild1"],
      });
      expect(result.map((n) => n.key)).toEqual(["root1", "child1", "grandchild1"]);
    });

    it("should skip non-existent nodes", () => {
      const result = findNodes({
        tree: simpleTree,
        keys: ["root1", "nonexistent", "child1"],
      });
      expect(result.map((n) => n.key)).toEqual(["root1", "child1"]);
    });

    it("should return empty array for all non-existent keys", () => {
      const result = findNodes({ tree: simpleTree, keys: ["nonexistent"] });
      expect(result).toEqual([]);
    });
  });

  describe("findNodeParent", () => {
    it("should find parent of direct child", () => {
      const result = findNodeParent({ tree: simpleTree, key: "child1" });
      expect(result?.key).toBe("root1");
    });

    it("should find parent of nested child", () => {
      const result = findNodeParent({ tree: simpleTree, key: "grandchild1" });
      expect(result?.key).toBe("child2");
    });

    it("should return null for root node", () => {
      const result = findNodeParent({ tree: simpleTree, key: "root1" });
      expect(result).toBeNull();
    });

    it("should return null for non-existent node", () => {
      const result = findNodeParent({ tree: simpleTree, key: "nonexistent" });
      expect(result).toBeNull();
    });
  });

  describe("deepCopy", () => {
    it("should create independent copy", () => {
      const copy = deepCopy(simpleTree);
      copy[0].key = "modified";
      copy[0].children![0].key = "modifiedChild";
      expect(simpleTree[0].key).toBe("root1");
      expect(simpleTree[0].children![0].key).toBe("child1");
    });

    it("should handle nodes without children", () => {
      const nodes = [{ key: "solo" }];
      const copy = deepCopy(nodes);
      expect(copy[0].children).toBeUndefined();
    });
  });

  describe("getDescendants", () => {
    it("should get all descendants including self", () => {
      const result = getDescendants(...simpleTree);
      const keys = result.map((n) => n.key);
      expect(keys).toEqual([
        "root1",
        "child1",
        "child2",
        "grandchild1",
        "grandchild2",
        "root2",
        "child3",
      ]);
    });

    it("should handle single node", () => {
      const result = getDescendants({ key: "solo" });
      expect(result.map((n) => n.key)).toEqual(["solo"]);
    });

    it("should handle multiple root nodes", () => {
      const result = getDescendants(simpleTree[0], simpleTree[1]);
      expect(result.length).toBe(7);
    });
  });

  describe("filterShape", () => {
    it("should filter by key match", () => {
      const shape: Shape = {
        keys: ["a", "b", "c"],
        nodes: [
          { depth: 0, expanded: false, hasChildren: false },
          { depth: 1, expanded: false, hasChildren: false },
          { depth: 0, expanded: false, hasChildren: false },
        ],
      };
      const result = filterShape(shape, (key) => key === "a" || key === "c");
      expect(result.keys).toEqual(["a", "c"]);
      expect(result.nodes).toHaveLength(2);
    });

    it("should filter by depth", () => {
      const shape: Shape = {
        keys: ["a", "b", "c"],
        nodes: [
          { depth: 0, expanded: false, hasChildren: false },
          { depth: 1, expanded: false, hasChildren: false },
          { depth: 2, expanded: false, hasChildren: false },
        ],
      };
      const result = filterShape(shape, (_, depth) => depth > 0);
      expect(result.keys).toEqual(["b", "c"]);
      expect(result.nodes.map((n) => n.depth)).toEqual([1, 2]);
    });
  });

  describe("getAllNodesOfMinDepth", () => {
    it("should return nodes at minimum depth", () => {
      const shape: Shape = {
        keys: ["a", "b", "c", "d"],
        nodes: [
          { depth: 1, expanded: false, hasChildren: false },
          { depth: 0, expanded: false, hasChildren: false },
          { depth: 0, expanded: false, hasChildren: false },
          { depth: 2, expanded: false, hasChildren: false },
        ],
      };
      const result = getAllNodesOfMinDepth(shape);
      expect(result).toEqual(["b", "c"]);
    });

    it("should handle single depth level", () => {
      const shape: Shape = {
        keys: ["a", "b"],
        nodes: [
          { depth: 5, expanded: false, hasChildren: false },
          { depth: 5, expanded: false, hasChildren: false },
        ],
      };
      const result = getAllNodesOfMinDepth(shape);
      expect(result).toEqual(["a", "b"]);
    });
  });

  describe("getDepth", () => {
    it("should return depth of existing key", () => {
      const shape: Shape<string> = {
        keys: ["a", "b", "c"],
        nodes: [
          { depth: 0, expanded: false, hasChildren: false },
          { depth: 1, expanded: false, hasChildren: false },
          { depth: 2, expanded: false, hasChildren: false },
        ],
      };
      expect(getDepth("b", shape)).toBe(1);
    });
  });

  describe("getNodeShape", () => {
    it("should return shape for existing key", () => {
      const shape: Shape = {
        keys: ["a", "b"],
        nodes: [
          { depth: 0, expanded: true, hasChildren: true },
          { depth: 1, expanded: false, hasChildren: false },
        ],
      };
      const result = getNodeShape(shape, "a");
      expect(result).toEqual({ depth: 0, expanded: true, hasChildren: true });
    });

    it("should return shape for last key", () => {
      const shape: Shape = {
        keys: ["a", "b"],
        nodes: [
          { depth: 0, expanded: false, hasChildren: false },
          { depth: 1, expanded: true, hasChildren: true },
        ],
      };
      const result = getNodeShape(shape, "b");
      expect(result).toEqual({ depth: 1, expanded: true, hasChildren: true });
    });
  });
});
