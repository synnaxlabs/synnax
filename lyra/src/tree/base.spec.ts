// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { compare } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Tree } from "@/tree";

describe("tree/base", () => {
  const simpleTree: Tree.Node[] = [
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
      expect(Tree.shouldExpand(node, ["test", "other"])).toBe(true);
    });

    it("should return false when node key is not in expanded list", () => {
      const node = { key: "test" };
      expect(Tree.shouldExpand(node, ["other"])).toBe(false);
    });
  });

  describe("flatten", () => {
    it("should flatten nodes without expansion", () => {
      const result = Tree.flatten({ nodes: simpleTree, expanded: [] });
      expect(result.keys).toEqual(["root1", "root2"]);
      expect(result.nodes).toEqual([
        { depth: 0, expanded: false, hasChildren: true },
        { depth: 0, expanded: false, hasChildren: true },
      ]);
    });

    it("should flatten with expanded nodes", () => {
      const result = Tree.flatten({ nodes: simpleTree, expanded: ["root1", "child2"] });
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
      const nodes: Tree.Node[] = [
        { key: "b", children: [{ key: "d" }, { key: "c" }] },
        { key: "a" },
      ];
      const result = Tree.flatten({
        nodes,
        expanded: ["b"],
        sort: compare.by((n) => n.key),
      });
      expect(result.keys).toEqual(["a", "b", "c", "d"]);
    });

    it("should sort copies and leave the caller's arrays in order", () => {
      const children: Tree.Node[] = [{ key: "d" }, { key: "c" }];
      const nodes: Tree.Node[] = [{ key: "b", children }, { key: "a" }];
      Tree.flatten({ nodes, expanded: ["b"], sort: compare.by((n) => n.key) });
      expect(nodes.map(({ key }) => key)).toEqual(["b", "a"]);
      expect(children.map(({ key }) => key)).toEqual(["d", "c"]);
    });

    it("should throw when two nodes share a key", () => {
      const nodes: Tree.Node[] = [{ key: "a", children: [{ key: "b" }] }, { key: "b" }];
      expect(() => Tree.flatten({ nodes, expanded: ["a"] })).toThrow(
        "tree has two nodes with key b",
      );
    });

    it("should not throw for a repeated key under a collapsed node", () => {
      const nodes: Tree.Node[] = [{ key: "a", children: [{ key: "b" }] }, { key: "b" }];
      expect(Tree.flatten({ nodes, expanded: [] }).keys).toEqual(["a", "b"]);
    });
  });

  describe("setNode", () => {
    it("should add node to root when destination is null", () => {
      const tree = structuredClone(simpleTree);
      const result = Tree.setNode({
        tree,
        destination: null,
        additions: { key: "newRoot" },
      });
      expect(result[0].key).toBe("newRoot");
      expect(result.length).toBe(3);
    });

    it("should add node to existing parent", () => {
      const tree = structuredClone(simpleTree);
      Tree.setNode({ tree, destination: "root2", additions: { key: "newChild" } });
      expect(tree[1].children?.find((c) => c.key === "newChild")).toBeDefined();
    });

    it("should replace duplicate keys", () => {
      const tree = structuredClone(simpleTree);
      Tree.setNode({
        tree,
        destination: "root1",
        additions: { key: "child1", children: [{ key: "new" }] },
      });
      const child1 = tree[0].children?.find((c) => c.key === "child1");
      expect(child1?.children?.[0].key).toBe("new");
    });

    it("should throw when destination not found and throwOnMissing is true", () => {
      const tree = structuredClone(simpleTree);
      expect(() =>
        Tree.setNode({ tree, destination: "nonexistent", additions: { key: "new" } }),
      ).toThrow("Could not find node with key nonexistent");
    });

    it("should not throw when destination not found and throwOnMissing is false", () => {
      const tree = structuredClone(simpleTree);
      const result = Tree.setNode({
        tree,
        destination: "nonexistent",
        additions: { key: "new" },
        throwOnMissing: false,
      });
      expect(result).toEqual(tree);
    });

    it("should handle multiple additions with deduplication", () => {
      const tree = structuredClone(simpleTree);
      const result = Tree.setNode({
        tree,
        destination: null,
        additions: [{ key: "new1" }, { key: "new1" }, { key: "new2" }],
      });
      expect(result.filter((n) => n.key === "new1").length).toBe(1);
      expect(result.find((n) => n.key === "new2")).toBeDefined();
    });
  });

  describe("findNode", () => {
    it("should find root node", () => {
      const result = Tree.findNode({ tree: simpleTree, key: "root1" });
      expect(result?.key).toBe("root1");
    });

    it("should find nested node", () => {
      const result = Tree.findNode({ tree: simpleTree, key: "grandchild1" });
      expect(result?.key).toBe("grandchild1");
    });

    it("should return null for non-existent node", () => {
      const result = Tree.findNode({ tree: simpleTree, key: "nonexistent" });
      expect(result).toBeNull();
    });
  });

  describe("filterShape", () => {
    it("should filter by key match", () => {
      const shape: Tree.Shape = {
        keys: ["a", "b", "c"],
        nodes: [
          { depth: 0, expanded: false, hasChildren: false },
          { depth: 1, expanded: false, hasChildren: false },
          { depth: 0, expanded: false, hasChildren: false },
        ],
      };
      const result = Tree.filterShape(shape, (key) => key === "a" || key === "c");
      expect(result.keys).toEqual(["a", "c"]);
      expect(result.nodes).toHaveLength(2);
    });

    it("should filter by depth", () => {
      const shape: Tree.Shape = {
        keys: ["a", "b", "c"],
        nodes: [
          { depth: 0, expanded: false, hasChildren: false },
          { depth: 1, expanded: false, hasChildren: false },
          { depth: 2, expanded: false, hasChildren: false },
        ],
      };
      const result = Tree.filterShape(shape, (_, depth) => depth > 0);
      expect(result.keys).toEqual(["b", "c"]);
      expect(result.nodes.map((n) => n.depth)).toEqual([1, 2]);
    });
  });

  describe("getAllNodesOfMinDepth", () => {
    it("should return nodes at minimum depth", () => {
      const shape: Tree.Shape = {
        keys: ["a", "b", "c", "d"],
        nodes: [
          { depth: 1, expanded: false, hasChildren: false },
          { depth: 0, expanded: false, hasChildren: false },
          { depth: 0, expanded: false, hasChildren: false },
          { depth: 2, expanded: false, hasChildren: false },
        ],
      };
      const result = Tree.getAllNodesOfMinDepth(shape);
      expect(result).toEqual(["b", "c"]);
    });

    it("should handle single depth level", () => {
      const shape: Tree.Shape = {
        keys: ["a", "b"],
        nodes: [
          { depth: 5, expanded: false, hasChildren: false },
          { depth: 5, expanded: false, hasChildren: false },
        ],
      };
      const result = Tree.getAllNodesOfMinDepth(shape);
      expect(result).toEqual(["a", "b"]);
    });
  });
});
