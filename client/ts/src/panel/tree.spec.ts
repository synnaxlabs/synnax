// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { type ontology } from "@/ontology";
import { panel } from "@/panel";

const leaf = (...tabKeys: string[]): panel.Node => ({
  variant: "leaf",
  tabs: tabKeys.map((key) => ({
    variant: "view",
    key,
    type: "selector",
    name: "",
    args: {},
  })),
});

// root splits into [a, b] | [c]; the left side splits again into [a] / [b].
const TREE: panel.Node = {
  variant: "split",
  direction: "x",
  size: 0.5,
  first: {
    variant: "split",
    direction: "y",
    size: 0.5,
    first: leaf("a"),
    last: leaf("b"),
  },
  last: leaf("c"),
};

describe("tree", () => {
  describe("childPath", () => {
    it("should derive child path keys from the parent", () => {
      expect(panel.childNodeKey(panel.ROOT_NODE_KEY, "first")).toEqual(2);
      expect(panel.childNodeKey(panel.ROOT_NODE_KEY, "last")).toEqual(3);
      expect(panel.childNodeKey(2, "first")).toEqual(4);
      expect(panel.childNodeKey(2, "last")).toEqual(5);
    });
  });

  describe("splitSide", () => {
    it("should match the side the reducer places the new empty leaf on", () => {
      expect(panel.splitSide("left")).toEqual("first");
      expect(panel.splitSide("top")).toEqual("first");
      expect(panel.splitSide("right")).toEqual("last");
      expect(panel.splitSide("bottom")).toEqual("last");
    });
  });

  describe("walkPath", () => {
    it("should return the root for ROOT_PATH", () => {
      expect(panel.findNode(TREE, panel.ROOT_NODE_KEY)).toBe(TREE);
    });

    it("should walk nested splits", () => {
      expect(panel.findNode(TREE, 4)).toEqual(leaf("a"));
      expect(panel.findNode(TREE, 5)).toEqual(leaf("b"));
      expect(panel.findNode(TREE, 3)).toEqual(leaf("c"));
    });

    it("should return null for a path that does not exist", () => {
      expect(panel.findNode(TREE, 6)).toBeUndefined();
      expect(panel.findNode(undefined, panel.ROOT_NODE_KEY)).toBeUndefined();
    });
  });

  describe("findTab", () => {
    it("should find a tab anywhere in the tree", () => {
      expect(panel.findTab(TREE, "b")?.key).toEqual("b");
      expect(panel.findTab(TREE, "c")?.key).toEqual("c");
    });

    it("should return null when the tab is absent", () => {
      expect(panel.findTab(TREE, "nope")).toBeUndefined();
    });
  });

  describe("findTabByResource", () => {
    const lp: ontology.ID = { type: "lineplot", key: "lp-1" };
    const withResource: panel.Node = {
      variant: "split",
      direction: "x",
      size: 0.5,
      first: leaf("a"),
      last: {
        variant: "leaf",
        tabs: [{ variant: "resource", key: "r", resource: lp }],
      },
    };

    it("should find the tab backing the resource anywhere in the tree", () => {
      expect(panel.findTabByResource(withResource, lp)?.key).toEqual("r");
    });

    it("should return null when no tab backs the resource", () => {
      expect(
        panel.findTabByResource(withResource, { type: "schematic", key: "lp-1" }),
      ).toBeUndefined();
      expect(panel.findTabByResource(TREE, lp)).toBeUndefined();
    });
  });

  describe("firstTab", () => {
    it("should return the first tab in traversal order", () => {
      expect(panel.firstTab(TREE)?.key).toEqual("a");
    });

    it("should return null for an empty tree", () => {
      expect(panel.firstTab(leaf())).toBeUndefined();
    });
  });

  describe("tabLeafPath", () => {
    it("should return the path of the leaf holding the tab", () => {
      expect(panel.tabLeafPath(TREE, "a")).toEqual(4);
      expect(panel.tabLeafPath(TREE, "b")).toEqual(5);
      expect(panel.tabLeafPath(TREE, "c")).toEqual(3);
    });

    it("should return null when the tab is absent", () => {
      expect(panel.tabLeafPath(TREE, "nope")).toBeUndefined();
    });
  });

  describe("firstLeafPath", () => {
    it("should return the first leaf in traversal order", () => {
      expect(panel.firstLeafPath(TREE)).toEqual(4);
      expect(panel.firstLeafPath(leaf("a"))).toEqual(panel.ROOT_NODE_KEY);
    });
  });
});
