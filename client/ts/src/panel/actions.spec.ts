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
  leaf: { tabs: tabKeys.map((key) => ({ key })) },
});

const state = (root: panel.Node): panel.Panel => ({
  key: "00000000-0000-0000-0000-000000000000",
  name: "panel",
  root,
});

const tabKeys = (node: panel.Node | undefined): string[] =>
  node?.leaf?.tabs.map((t) => t.key) ?? [];

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
      expect(next.root.split).toBeDefined();
      expect(tabKeys(next.root.split?.first)).toEqual(["a"]);
      expect(tabKeys(next.root.split?.last)).toEqual(["b"]);
    });

    it("should collapse the source split when moving the last tab out of a side", () => {
      const { next } = panel.reduceAll(
        state({
          split: {
            direction: "x",
            size: 0.5,
            first: leaf("a"),
            last: leaf("b"),
          },
        }),
        [panel.moveTab({ key: "a", targetLeaf: 3, index: 0 })],
      );
      expect(next.root.split).toBeUndefined();
      expect(tabKeys(next.root)).toEqual(["a", "b"]);
    });
  });

  describe("removeTab", () => {
    it("should collapse the split when removing a side's last tab", () => {
      const { next } = panel.reduceAll(
        state({
          split: {
            direction: "x",
            size: 0.5,
            first: leaf("a"),
            last: leaf("b", "c"),
          },
        }),
        [panel.removeTab({ key: "a" })],
      );
      expect(next.root.split).toBeUndefined();
      expect(tabKeys(next.root)).toEqual(["b", "c"]);
    });
  });
});
