// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Mosaic } from "@/mosaic";

describe("tree", () => {
  describe("insertTab", () => {
    it("should insert a tab into the center of an empty tree", () => {
      const tree = Mosaic.insertTab({ key: 1, tabs: [] }, "tab1", "center", 1);
      expect(tree).toEqual({ key: 1, tabs: ["tab1"], selected: "tab1" });
    });

    it("shouldn't split an empty tree, instead put the tab in the center", () => {
      const tree = Mosaic.insertTab({ key: 1, tabs: [] }, "tab1", "right", 1);
      expect(tree).toEqual({ key: 1, tabs: ["tab1"], selected: "tab1" });
    });

    it("should split a tree with one tab", () => {
      const tree: Mosaic.Node = { key: 1, tabs: ["tab1"], selected: "tab1" };
      const nextTree = Mosaic.insertTab(tree, "tab2", "right", 1);
      expect(nextTree).toEqual({
        key: 1,
        direction: "x",
        first: { key: 2, tabs: ["tab1"], selected: "tab1" },
        last: { key: 3, tabs: ["tab2"], selected: "tab2" },
      });
    });

    it("should insert into the center of a valid leaf when no key is provided", () => {
      const tree: Mosaic.Node = {
        key: 1,
        direction: "x",
        first: { key: 2, tabs: ["tab1"], selected: "tab1" },
        last: { key: 3, tabs: ["tab2"], selected: "tab2" },
      };
      const nextTree = Mosaic.insertTab(tree, "tab3");
      expect(nextTree).toEqual({
        key: 1,
        direction: "x",
        first: { key: 2, tabs: ["tab1", "tab3"], selected: "tab3" },
        last: { key: 3, tabs: ["tab2"], selected: "tab2" },
      });
    });

    it("should insert a tab at the given index within a leaf", () => {
      const tree: Mosaic.Node = {
        key: 1,
        tabs: ["tab1", "tab2", "tab3"],
        selected: "tab1",
      };
      const nextTree = Mosaic.insertTab(tree, "tab4", "center", 1, 1);
      expect(nextTree).toEqual({
        key: 1,
        tabs: ["tab1", "tab4", "tab2", "tab3"],
        selected: "tab4",
      });
    });

    it("should append the tab when the index is out of bounds", () => {
      const tree: Mosaic.Node = { key: 1, tabs: ["tab1"], selected: "tab1" };
      const nextTree = Mosaic.insertTab(tree, "tab2", "center", 1, 5);
      expect(nextTree).toEqual({
        key: 1,
        tabs: ["tab1", "tab2"],
        selected: "tab2",
      });
    });

    it("should not mutate the input tree", () => {
      const tree: Mosaic.Node = { key: 1, tabs: ["tab1"], selected: "tab1" };
      Mosaic.insertTab(tree, "tab2", "center", 1);
      expect(tree.tabs).toEqual(["tab1"]);
    });
  });

  describe("removeTab", () => {
    it("should remove a tab from the center of a tree", () => {
      const tree: Mosaic.Node = {
        key: 1,
        direction: "x",
        first: { key: 2, tabs: ["tab1"], selected: "tab1" },
        last: { key: 3, tabs: ["tab2", "tab3"], selected: "tab2" },
      };
      const [nextTree] = Mosaic.removeTab(tree, "tab2");
      expect(nextTree).toEqual({
        key: 1,
        direction: "x",
        first: { key: 2, tabs: ["tab1"], selected: "tab1" },
        last: { key: 3, tabs: ["tab3"], selected: "tab3" },
      });
    });

    it("should remove a tab from the side of a tree and garbage collect", () => {
      const tree: Mosaic.Node = {
        key: 1,
        direction: "x",
        first: { key: 2, tabs: ["tab1"], selected: "tab1" },
        last: { key: 3, tabs: ["tab2", "tab3"], selected: "tab2" },
      };
      const [nextTree] = Mosaic.removeTab(tree, "tab1");
      expect(nextTree).toEqual({
        key: 1,
        tabs: ["tab2", "tab3"],
        selected: "tab2",
      });
    });
  });

  describe("moveTab", () => {
    it("should move a tab from one side of a leaf to another", () => {
      const tree: Mosaic.Node = {
        key: 1,
        direction: "x",
        first: { key: 2, tabs: ["tab1"], selected: "tab1" },
        last: { key: 3, tabs: ["tab2", "tab3"], selected: "tab2" },
      };
      const [nextTree] = Mosaic.moveTab(tree, "tab2", "center", 2);
      expect(nextTree).toEqual({
        key: 1,
        direction: "x",
        first: { key: 2, tabs: ["tab1", "tab2"], selected: "tab2" },
        last: { key: 3, tabs: ["tab3"], selected: "tab3" },
      });
    });

    it("should move a tab from the first leaf of the root to the second leaf", () => {
      const initialTree: Mosaic.Node = {
        key: 1,
        direction: "x",
        first: { key: 2, tabs: ["1"] },
        last: { key: 3, tabs: ["2", "3"] },
      };
      const [nextTree] = Mosaic.moveTab(initialTree, "1", "center", 2);
      expect(nextTree).toEqual({
        key: 1,
        selected: "1",
        size: undefined,
        tabs: ["2", "3", "1"],
      });
    });

    it("should move a tab to the given index in the target leaf", () => {
      const tree: Mosaic.Node = {
        key: 1,
        direction: "x",
        first: { key: 2, tabs: ["tab1"], selected: "tab1" },
        last: { key: 3, tabs: ["tab2", "tab3"], selected: "tab2" },
      };
      const [nextTree] = Mosaic.moveTab(tree, "tab1", "center", 3, 1);
      expect(nextTree).toEqual({
        key: 1,
        selected: "tab1",
        size: undefined,
        tabs: ["tab2", "tab1", "tab3"],
      });
    });

    it("should maintain correct key hierarchy after moves and garbage collection", () => {
      const initialTree: Mosaic.Node = {
        key: 1,
        direction: "y",
        last: { key: 3, tabs: ["labjack"], selected: "labjack" },
        first: {
          key: 2,
          direction: "x",
          last: { key: 5, tabs: ["component-b"], selected: "component-b", size: 0.5 },
          first: { key: 4, tabs: ["component-a"], selected: "component-a" },
        },
      };

      const [result] = Mosaic.moveTab(initialTree, "labjack", "bottom", 5);

      const verifyKeyHierarchy = (node: Mosaic.Node): boolean => {
        let valid = true;
        if (node.first) {
          valid &&= node.first.key === node.key * 2;
          valid &&= verifyKeyHierarchy(node.first);
        }
        if (node.last) {
          valid &&= node.last.key === node.key * 2 + 1;
          valid &&= verifyKeyHierarchy(node.last);
        }
        return valid;
      };

      expect(result.key).toBe(1);
      expect(verifyKeyHierarchy(result)).toBe(true);
    });
  });

  describe("resizeNode", () => {
    it("should resize a node", () => {
      const tree: Mosaic.Node = {
        key: 1,
        direction: "x",
        first: { key: 2, tabs: ["tab1"], selected: "tab1" },
        last: { key: 3, tabs: ["tab2"], selected: "tab2" },
      };
      const nextTree = Mosaic.resizeNode(tree, 2, 100);
      expect(nextTree).toEqual({
        key: 1,
        direction: "x",
        first: { key: 2, tabs: ["tab1"], selected: "tab1", size: 100 },
        last: { key: 3, tabs: ["tab2"], selected: "tab2" },
      });
    });
  });

  describe("selectTab", () => {
    it("should select a tab", () => {
      const tree: Mosaic.Node = {
        key: 1,
        tabs: ["tab1", "tab2"],
        selected: "tab1",
      };
      const nextTree = Mosaic.selectTab(tree, "tab2");
      expect(nextTree).toEqual({
        key: 1,
        tabs: ["tab1", "tab2"],
        selected: "tab2",
      });
    });
  });

  describe("autoSelectTabs", () => {
    it("should select a tab for every leaf missing a selection", () => {
      const tree: Mosaic.Node = {
        key: 1,
        direction: "x",
        first: { key: 2, tabs: ["tab1", "tab2"] },
        last: { key: 3, tabs: ["tab3"], selected: "tab3" },
      };
      const [nextTree, selected] = Mosaic.autoSelectTabs(tree);
      expect(nextTree.first?.selected).toEqual("tab2");
      expect(nextTree.last?.selected).toEqual("tab3");
      expect(selected).toEqual(["tab2", "tab3"]);
    });
  });

  describe("findTabNode", () => {
    it("should find a tab", () => {
      const tree: Mosaic.Node = {
        key: 1,
        tabs: ["tab1", "tab2"],
        selected: "tab1",
      };
      expect(Mosaic.findTabNode(tree, "tab2")?.key).toEqual(1);
    });

    it("should find a tab in a nested tree", () => {
      const tree: Mosaic.Node = {
        key: 1,
        direction: "x",
        first: { key: 2, tabs: ["tab1"], selected: "tab1" },
        last: { key: 3, tabs: ["tab2", "tab3"], selected: "tab2" },
      };
      expect(Mosaic.findTabNode(tree, "tab3")?.key).toEqual(3);
    });

    it("should return undefined if the tab is not found", () => {
      const tree: Mosaic.Node = {
        key: 1,
        tabs: ["tab1", "tab2"],
        selected: "tab1",
      };
      expect(Mosaic.findTabNode(tree, "tab3")).toBeUndefined();
    });
  });

  describe("split", () => {
    it("should split a tree vertically", () => {
      const tree: Mosaic.Node = {
        key: 1,
        tabs: ["tab1", "tab2"],
        selected: "tab1",
      };
      const nextTree = Mosaic.split(tree, "tab2", "y");
      expect(nextTree).toEqual({
        key: 1,
        direction: "y",
        first: { key: 2, tabs: ["tab1"], selected: "tab1" },
        last: { key: 3, tabs: ["tab2"], selected: "tab2" },
      });
    });

    it("should split a nested tree vertically", () => {
      const tree: Mosaic.Node = {
        key: 1,
        direction: "x",
        first: { key: 2, tabs: ["tab1"], selected: "tab1" },
        last: {
          key: 3,
          direction: "y",
          first: { key: 6, tabs: ["tab2"], selected: "tab2" },
          last: { key: 7, tabs: ["tab3", "tab4"], selected: "tab3" },
        },
      };
      const nextTree = Mosaic.split(tree, "tab3", "y");
      expect(nextTree).toEqual({
        key: 1,
        direction: "x",
        first: { key: 2, tabs: ["tab1"], selected: "tab1" },
        last: {
          key: 3,
          direction: "y",
          first: { key: 6, tabs: ["tab2"], selected: "tab2" },
          last: {
            key: 7,
            direction: "y",
            first: { key: 14, selected: "tab4", tabs: ["tab4"] },
            last: { key: 15, selected: "tab3", tabs: ["tab3"] },
          },
        },
      });
    });
  });

  describe("canSplit", () => {
    it("should return true when the tab's leaf has more than one tab", () => {
      const tree: Mosaic.Node = {
        key: 1,
        tabs: ["tab1", "tab2"],
        selected: "tab1",
      };
      expect(Mosaic.canSplit(tree, "tab1")).toBe(true);
    });

    it("should return false when the tab's leaf has a single tab", () => {
      const tree: Mosaic.Node = { key: 1, tabs: ["tab1"], selected: "tab1" };
      expect(Mosaic.canSplit(tree, "tab1")).toBe(false);
    });

    it("should return false when the tab is not in the tree", () => {
      const tree: Mosaic.Node = { key: 1, tabs: ["tab1"], selected: "tab1" };
      expect(Mosaic.canSplit(tree, "tab2")).toBe(false);
    });
  });

  describe("isEmpty", () => {
    it("should return true for a leaf with no tabs", () => {
      expect(Mosaic.isEmpty({ key: 1, tabs: [] })).toBe(true);
    });

    it("should return false for a leaf with tabs", () => {
      expect(Mosaic.isEmpty({ key: 1, tabs: ["tab1"] })).toBe(false);
    });

    it("should return false for a split node", () => {
      expect(
        Mosaic.isEmpty({
          key: 1,
          direction: "x",
          first: { key: 2, tabs: [] },
          last: { key: 3, tabs: ["tab1"] },
        }),
      ).toBe(false);
    });
  });

  describe("findSelected", () => {
    it("should return the first selected tab in the tree", () => {
      const tree: Mosaic.Node = {
        key: 1,
        direction: "x",
        first: { key: 2, tabs: ["tab1"] },
        last: { key: 3, tabs: ["tab2"], selected: "tab2" },
      };
      expect(Mosaic.findSelected(tree)).toEqual("tab2");
    });

    it("should return null when nothing is selected", () => {
      expect(Mosaic.findSelected({ key: 1, tabs: ["tab1"] })).toBeNull();
    });
  });

  describe("nodeZ", () => {
    it("should parse a valid structure-only tree", () => {
      const tree: Mosaic.Node = {
        key: 1,
        direction: "x",
        size: 0.4,
        first: { key: 2, tabs: ["tab1"], selected: "tab1" },
        last: { key: 3, tabs: ["tab2", "tab3"], selected: "tab2" },
      };
      expect(Mosaic.nodeZ.parse(tree)).toEqual(tree);
    });

    it("should reject tabs that are not strings", () => {
      expect(
        Mosaic.nodeZ.safeParse({ key: 1, tabs: [{ tabKey: "tab1" }] }).success,
      ).toBe(false);
    });
  });
});
