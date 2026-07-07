// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { uuid } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { type ontology } from "@/ontology";
import { panel } from "@/panel";

const a = uuid.create();
const b = uuid.create();
const c = uuid.create();
const z = uuid.create();
const missing = uuid.create();

const viewTab = (key: string, type: string = "selector"): panel.Tab => ({
  variant: "view",
  key,
  type,
  args: {},
});

const resource = (key: string): ontology.ID => ({ type: "lineplot", key });

const resourceTab = (key: string, resourceKey: string = key): panel.Tab => ({
  variant: "resource",
  key,
  resource: resource(resourceKey),
});

const leaf = (...tabKeys: string[]): panel.Node => ({
  variant: "leaf",
  tabs: tabKeys.map((key) => viewTab(key)),
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
    it("should collapse the source split when moving the last tab out of a side", () => {
      const { next } = panel.reduceAll(state(split("x", 0.5, leaf(a), leaf(b))), [
        panel.moveTab({ key: a, targetLeaf: 3, index: 0 }),
      ]);
      expect(next.root.variant).toEqual("leaf");
      expect(tabKeys(next.root)).toEqual([a, b]);
    });

    it("should split the target leaf and move the tab into the new sibling when location is present", () => {
      const { next } = panel.reduceAll(state(leaf(a, b)), [
        panel.moveTab({ key: b, targetLeaf: panel.ROOT_PATH, location: "right" }),
      ]);
      const root = asSplit(next.root);
      expect(root.direction).toEqual("x");
      expect(tabKeys(root.first)).toEqual([a]);
      expect(tabKeys(root.last)).toEqual([b]);
    });

    it("should place the new sibling first for a top location", () => {
      const { next } = panel.reduceAll(state(leaf(a, b)), [
        panel.moveTab({ key: b, targetLeaf: panel.ROOT_PATH, location: "top" }),
      ]);
      const root = asSplit(next.root);
      expect(root.direction).toEqual("y");
      expect(tabKeys(root.first)).toEqual([b]);
      expect(tabKeys(root.last)).toEqual([a]);
    });

    it("should no-op when moving a leaf's only tab to an edge of its own leaf", () => {
      const { next } = panel.reduceAll(state(leaf(a)), [
        panel.moveTab({ key: a, targetLeaf: panel.ROOT_PATH, location: "left" }),
      ]);
      expect(next.root.variant).toEqual("leaf");
      expect(tabKeys(next.root)).toEqual([a]);
    });

    it("should place the tab directly in the target leaf for a center location", () => {
      const { next } = panel.reduceAll(state(split("x", 0.5, leaf(a), leaf(b))), [
        panel.moveTab({ key: a, targetLeaf: 3, location: "center" }),
      ]);
      expect(next.root.variant).toEqual("leaf");
      expect(tabKeys(next.root)).toEqual([b, a]);
    });
  });

  describe("splitTab", () => {
    it("should split the tab off into a new sibling pane to the right for direction x", () => {
      const { next } = panel.reduceAll(state(leaf(a, b)), [
        panel.splitTab({ key: b, direction: "x" }),
      ]);
      const root = asSplit(next.root);
      expect(root.direction).toEqual("x");
      expect(tabKeys(root.first)).toEqual([a]);
      expect(tabKeys(root.last)).toEqual([b]);
    });

    it("should split the tab off into a new sibling pane below for direction y", () => {
      const { next } = panel.reduceAll(state(leaf(a, b)), [
        panel.splitTab({ key: b, direction: "y" }),
      ]);
      const root = asSplit(next.root);
      expect(root.direction).toEqual("y");
      expect(tabKeys(root.first)).toEqual([a]);
      expect(tabKeys(root.last)).toEqual([b]);
    });

    it("should resolve the tab's own leaf in a nested tree", () => {
      const { next } = panel.reduceAll(state(split("x", 0.5, leaf(a, b), leaf(c))), [
        panel.splitTab({ key: a, direction: "x" }),
      ]);
      const root = asSplit(next.root);
      const firstChild = asSplit(root.first);
      expect(tabKeys(firstChild.first)).toEqual([b]);
      expect(tabKeys(firstChild.last)).toEqual([a]);
      expect(tabKeys(root.last)).toEqual([c]);
    });

    it("should no-op when the tab is the only tab in its leaf", () => {
      const prev = state(leaf(a));
      const { next } = panel.reduceAll(prev, [
        panel.splitTab({ key: a, direction: "x" }),
      ]);
      expect(next).toBe(prev);
    });

    it("should no-op when no tab matches the key", () => {
      const prev = state(leaf(a, b));
      const { next } = panel.reduceAll(prev, [
        panel.splitTab({ key: z, direction: "x" }),
      ]);
      expect(next).toBe(prev);
    });
  });

  describe("resizeSplit", () => {
    it("should return the same state reference when the size is unchanged", () => {
      const prev = state(split("x", 0.5, leaf(a), leaf(b)));
      const { next } = panel.reduceAll(prev, [
        panel.resizeSplit({ split: panel.ROOT_PATH, size: 0.5 }),
      ]);
      expect(next).toBe(prev);
    });

    it("should resize the split when the size differs", () => {
      const prev = state(split("x", 0.5, leaf(a), leaf(b)));
      const { next } = panel.reduceAll(prev, [
        panel.resizeSplit({ split: panel.ROOT_PATH, size: 0.7 }),
      ]);
      expect(next).not.toBe(prev);
      expect(asSplit(next.root).size).toEqual(0.7);
    });
  });

  describe("insertTab", () => {
    it("should split the target leaf and insert into the new sibling when location is present", () => {
      const { next } = panel.reduceAll(state(leaf(a)), [
        panel.insertTab({
          tab: viewTab(b),
          targetLeaf: panel.ROOT_PATH,
          location: "bottom",
        }),
      ]);
      const root = asSplit(next.root);
      expect(root.direction).toEqual("y");
      expect(tabKeys(root.first)).toEqual([a]);
      expect(tabKeys(root.last)).toEqual([b]);
    });

    it("should insert directly into the target leaf for a center location", () => {
      const { next } = panel.reduceAll(state(leaf(a)), [
        panel.insertTab({
          tab: viewTab(b),
          targetLeaf: panel.ROOT_PATH,
          location: "center",
        }),
      ]);
      expect(next.root.variant).toEqual("leaf");
      expect(tabKeys(next.root)).toEqual([a, b]);
    });

    it("should degrade an edge insert into an empty leaf to a direct insert", () => {
      const { next } = panel.reduceAll(state(leaf()), [
        panel.insertTab({
          tab: viewTab(a),
          targetLeaf: panel.ROOT_PATH,
          location: "right",
        }),
      ]);
      expect(next.root.variant).toEqual("leaf");
      expect(tabKeys(next.root)).toEqual([a]);
    });

    it("should insert into the leaf holding targetTab when set", () => {
      const { next } = panel.reduceAll(state(split("x", 0.5, leaf(a), leaf(b))), [
        panel.insertTab({
          tab: viewTab(c),
          targetTab: b,
        }),
      ]);
      const root = asSplit(next.root);
      expect(tabKeys(root.first)).toEqual([a]);
      expect(tabKeys(root.last)).toEqual([b, c]);
    });

    it("should no-op when targetTab matches no tab", () => {
      const prev = state(leaf(a));
      const { next } = panel.reduceAll(prev, [
        panel.insertTab({
          tab: viewTab(b),
          targetTab: z,
        }),
      ]);
      expect(next).toBe(prev);
    });

    it("should default to the first leaf in traversal order when no target is set", () => {
      const { next } = panel.reduceAll(state(split("x", 0.5, leaf(a), leaf(b))), [
        panel.insertTab({ tab: viewTab(c) }),
      ]);
      const root = asSplit(next.root);
      expect(tabKeys(root.first)).toEqual([a, c]);
      expect(tabKeys(root.last)).toEqual([b]);
    });

    it("should default to the root leaf when no target is set on a single-leaf tree", () => {
      const { next } = panel.reduceAll(state(leaf(a)), [
        panel.insertTab({ tab: viewTab(b) }),
      ]);
      expect(next.root.variant).toEqual("leaf");
      expect(tabKeys(next.root)).toEqual([a, b]);
    });

    it("should refresh an existing tab's content in place when no placement is given", () => {
      const { next } = panel.reduceAll(state(split("x", 0.5, leaf(a), leaf(b))), [
        panel.insertTab({ tab: resourceTab(b) }),
      ]);
      const root = asSplit(next.root);
      expect(tabKeys(root.first)).toEqual([a]);
      expect(tabKeys(root.last)).toEqual([b]);
      expect(panel.findTab(next.root, b)).toEqual(resourceTab(b));
    });

    it("should not duplicate an existing tab when no placement is given", () => {
      const { next } = panel.reduceAll(state(leaf(a, b)), [
        panel.insertTab({ tab: resourceTab(a) }),
      ]);
      expect(tabKeys(next.root)).toEqual([a, b]);
    });

    it("should no-op when the resource already backs a different tab", () => {
      const prev = state({ variant: "leaf", tabs: [resourceTab(a), viewTab(b)] });
      const { next } = panel.reduceAll(prev, [
        panel.insertTab({ tab: resourceTab(c, a) }),
      ]);
      expect(next).toBe(prev);
    });

    it("should no-op even when the duplicate insert carries a placement", () => {
      const prev = state({ variant: "leaf", tabs: [resourceTab(a), viewTab(b)] });
      const { next } = panel.reduceAll(prev, [
        panel.insertTab({
          tab: resourceTab(c, a),
          targetLeaf: panel.ROOT_PATH,
          location: "right",
        }),
      ]);
      expect(next).toBe(prev);
    });

    it("should relocate an existing tab and refresh its content when a placement is given", () => {
      const { next } = panel.reduceAll(state(leaf(a, b)), [
        panel.insertTab({
          tab: resourceTab(b),
          targetLeaf: panel.ROOT_PATH,
          location: "right",
        }),
      ]);
      const root = asSplit(next.root);
      expect(root.direction).toEqual("x");
      expect(tabKeys(root.first)).toEqual([a]);
      expect(tabKeys(root.last)).toEqual([b]);
      expect(panel.findTab(next.root, b)).toEqual(resourceTab(b));
    });

    it("should reorder an existing tab within its leaf when only an index is given", () => {
      const { next } = panel.reduceAll(state(leaf(a, b, c)), [
        panel.insertTab({ tab: viewTab(c), index: 0 }),
      ]);
      expect(tabKeys(next.root)).toEqual([c, a, b]);
    });

    it("should move an existing tab into the leaf holding targetTab without duplicating it", () => {
      const { next } = panel.reduceAll(state(split("x", 0.5, leaf(a), leaf(b))), [
        panel.insertTab({ tab: viewTab(a), targetTab: b }),
      ]);
      expect(next.root.variant).toEqual("leaf");
      expect(tabKeys(next.root)).toEqual([b, a]);
    });
  });

  describe("removeTab", () => {
    it("should collapse the split when removing a side's last tab", () => {
      const { next } = panel.reduceAll(state(split("x", 0.5, leaf(a), leaf(b, c))), [
        panel.removeTab({ key: a }),
      ]);
      expect(next.root.variant).toEqual("leaf");
      expect(tabKeys(next.root)).toEqual([b, c]);
    });
  });

  describe("setTabResource", () => {
    it("should swap a view tab to the resource in place", () => {
      const { next } = panel.reduceAll(state(leaf(a, b)), [
        panel.setTabResource({ key: a, resource: resource(c) }),
      ]);
      expect(next.root.variant).toEqual("leaf");
      expect(tabKeys(next.root)).toEqual([a, b]);
      expect(panel.findTab(next.root, a)).toEqual(resourceTab(a, c));
    });

    it("should no-op when the resource already backs a different tab", () => {
      const prev = state({ variant: "leaf", tabs: [resourceTab(a), viewTab(b)] });
      const { next } = panel.reduceAll(prev, [
        panel.setTabResource({ key: b, resource: resource(a) }),
      ]);
      expect(next).toBe(prev);
    });

    it("should be a no-op when no tab matches the key", () => {
      const prev = state(leaf(a, b));
      const { next } = panel.reduceAll(prev, [
        panel.setTabResource({ key: missing, resource: resource(c) }),
      ]);
      expect(next).toBe(prev);
    });
  });

  describe("setTabView", () => {
    it("should swap a resource tab to the view in place", () => {
      const view = { type: "docs", args: {} };
      const { next } = panel.reduceAll(
        state({ variant: "leaf", tabs: [resourceTab(a), viewTab(b)] }),
        [panel.setTabView({ key: a, view })],
      );
      expect(next.root.variant).toEqual("leaf");
      expect(tabKeys(next.root)).toEqual([a, b]);
      expect(panel.findTab(next.root, a)).toEqual({ variant: "view", key: a, ...view });
    });

    it("should be a no-op when no tab matches the key", () => {
      const prev = state(leaf(a, b));
      const { next } = panel.reduceAll(prev, [
        panel.setTabView({ key: missing, view: { type: "docs" } }),
      ]);
      expect(next).toBe(prev);
    });
  });
});
