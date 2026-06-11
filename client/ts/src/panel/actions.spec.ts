// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { panel } from "@/panel";

const leaf = (...tabKeys: string[]): panel.Node => ({
  variant: "leaf",
  tabs: tabKeys.map((key) => ({ variant: "empty", key })),
});

const split = (
  direction: "x" | "y",
  size: number,
  first: panel.Node,
  last: panel.Node,
): panel.Node => ({ variant: "split", direction, size, first, last });

const state = (root: panel.Node): panel.Panel => ({
  key: "00000000-0000-0000-0000-000000000000",
  name: "panel",
  root,
});

const asSplit = (node: panel.Node): panel.NodeSplit => {
  expect(node.variant).toEqual("split");
  return node as panel.NodeSplit;
};

const tabKeys = (node: panel.Node | undefined): string[] =>
  node?.variant === "leaf" ? node.tabs.map((t) => t.key) : [];

describe("reduceAll", () => {
  describe("moveTab", () => {
    it("should move a tab into the empty side of a freshly split leaf", () => {
      const { next } = panel.reduceAll(state(leaf("a", "b")), [
        panel.splitLeaf({ leaf: panel.ROOT_PATH, location: "right", size: 0.5 }),
        panel.moveTab({
          key: "b",
          targetLeaf: panel.childPath(panel.ROOT_PATH, "last"),
          index: 0,
        }),
      ]);
      const root = asSplit(next.root);
      expect(tabKeys(root.first)).toEqual(["a"]);
      expect(tabKeys(root.last)).toEqual(["b"]);
    });

    it("should collapse the source split when moving the last tab out of a side", () => {
      const { next } = panel.reduceAll(state(split("x", 0.5, leaf("a"), leaf("b"))), [
        panel.moveTab({ key: "a", targetLeaf: 3, index: 0 }),
      ]);
      expect(next.root.variant).toEqual("leaf");
      expect(tabKeys(next.root)).toEqual(["a", "b"]);
    });

    it("should split the target leaf and move the tab into the new sibling when location is present", () => {
      const { next } = panel.reduceAll(state(leaf("a", "b")), [
        panel.moveTab({ key: "b", targetLeaf: panel.ROOT_PATH, location: "right" }),
      ]);
      const root = asSplit(next.root);
      expect(root.direction).toEqual("x");
      expect(tabKeys(root.first)).toEqual(["a"]);
      expect(tabKeys(root.last)).toEqual(["b"]);
    });

    it("should place the new sibling first for a top location", () => {
      const { next } = panel.reduceAll(state(leaf("a", "b")), [
        panel.moveTab({ key: "b", targetLeaf: panel.ROOT_PATH, location: "top" }),
      ]);
      const root = asSplit(next.root);
      expect(root.direction).toEqual("y");
      expect(tabKeys(root.first)).toEqual(["b"]);
      expect(tabKeys(root.last)).toEqual(["a"]);
    });

    it("should no-op when moving a leaf's only tab to an edge of its own leaf", () => {
      const { next } = panel.reduceAll(state(leaf("a")), [
        panel.moveTab({ key: "a", targetLeaf: panel.ROOT_PATH, location: "left" }),
      ]);
      expect(next.root.variant).toEqual("leaf");
      expect(tabKeys(next.root)).toEqual(["a"]);
    });

    it("should place the tab directly in the target leaf for a center location", () => {
      const { next } = panel.reduceAll(state(split("x", 0.5, leaf("a"), leaf("b"))), [
        panel.moveTab({ key: "a", targetLeaf: 3, location: "center" }),
      ]);
      expect(next.root.variant).toEqual("leaf");
      expect(tabKeys(next.root)).toEqual(["b", "a"]);
    });
  });

  describe("resizeSplit", () => {
    it("should return the same state reference when the size is unchanged", () => {
      const prev = state(split("x", 0.5, leaf("a"), leaf("b")));
      const { next } = panel.reduceAll(prev, [
        panel.resizeSplit({ split: panel.ROOT_PATH, size: 0.5 }),
      ]);
      expect(next).toBe(prev);
    });

    it("should resize the split when the size differs", () => {
      const prev = state(split("x", 0.5, leaf("a"), leaf("b")));
      const { next } = panel.reduceAll(prev, [
        panel.resizeSplit({ split: panel.ROOT_PATH, size: 0.7 }),
      ]);
      expect(next).not.toBe(prev);
      expect(asSplit(next.root).size).toEqual(0.7);
    });
  });

  describe("insertTab", () => {
    it("should split the target leaf and insert into the new sibling when location is present", () => {
      const { next } = panel.reduceAll(state(leaf("a")), [
        panel.insertTab({
          tab: { variant: "empty", key: "b" },
          targetLeaf: panel.ROOT_PATH,
          location: "bottom",
        }),
      ]);
      const root = asSplit(next.root);
      expect(root.direction).toEqual("y");
      expect(tabKeys(root.first)).toEqual(["a"]);
      expect(tabKeys(root.last)).toEqual(["b"]);
    });

    it("should insert directly into the target leaf for a center location", () => {
      const { next } = panel.reduceAll(state(leaf("a")), [
        panel.insertTab({
          tab: { variant: "empty", key: "b" },
          targetLeaf: panel.ROOT_PATH,
          location: "center",
        }),
      ]);
      expect(next.root.variant).toEqual("leaf");
      expect(tabKeys(next.root)).toEqual(["a", "b"]);
    });

    it("should degrade an edge insert into an empty leaf to a direct insert", () => {
      const { next } = panel.reduceAll(state(leaf()), [
        panel.insertTab({
          tab: { variant: "empty", key: "a" },
          targetLeaf: panel.ROOT_PATH,
          location: "right",
        }),
      ]);
      expect(next.root.variant).toEqual("leaf");
      expect(tabKeys(next.root)).toEqual(["a"]);
    });
  });

  describe("removeTab", () => {
    it("should collapse the split when removing a side's last tab", () => {
      const { next } = panel.reduceAll(
        state(split("x", 0.5, leaf("a"), leaf("b", "c"))),
        [panel.removeTab({ key: "a" })],
      );
      expect(next.root.variant).toEqual("leaf");
      expect(tabKeys(next.root)).toEqual(["b", "c"]);
    });
  });

  describe("setTabResource", () => {
    it("should swap the tab variant in place without changing its position", () => {
      const resource = { type: "lineplot", key: "lp-1" } as const;
      const { next } = panel.reduceAll(state(leaf("a", "b")), [
        panel.setTabResource({ key: "a", resource }),
      ]);
      expect(next.root.variant).toEqual("leaf");
      expect(tabKeys(next.root)).toEqual(["a", "b"]);
      const tab = panel.findTab(next.root, "a");
      expect(tab?.variant).toEqual("resource");
      if (tab?.variant === "resource") expect(tab.resource).toEqual(resource);
    });
  });

  describe("setTabView", () => {
    it("should swap the tab variant in place without changing its position", () => {
      const view = { type: "docs", args: {} };
      const { next } = panel.reduceAll(state(leaf("a")), [
        panel.setTabView({ key: "a", view }),
      ]);
      const tab = panel.findTab(next.root, "a");
      expect(tab?.variant).toEqual("view");
      if (tab?.variant === "view") expect(tab.view).toEqual(view);
    });
  });
});
