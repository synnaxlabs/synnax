// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "vitest";

import { type Haul } from "@/haul";
import { List } from "@/list";
import { mockGeometry } from "@/testutil/dom";
import { type Node } from "@/tree/base";
import { Item } from "@/tree/Item";
import {
  canDropHaulItem,
  createHaulItem,
  filterHaulItems,
  HAUL_TYPE,
  isHaulItem,
  Tree,
  use,
} from "@/tree/Tree";

const KEY = "node-1";
const OTHER: Haul.Item = { type: "other_type", key: "other" };

describe("tree haul utilities", () => {
  describe("createHaulItem", () => {
    it("creates an item with the tree HAUL_TYPE", () => {
      expect(createHaulItem(KEY).type).toEqual(HAUL_TYPE);
    });

    it("creates an item with the provided key", () => {
      expect(createHaulItem(KEY).key).toEqual(KEY);
    });
  });

  describe("isHaulItem", () => {
    it("returns true for an item of the tree kind", () => {
      expect(isHaulItem(createHaulItem(KEY))).toBe(true);
    });

    it("returns false for an item of another kind", () => {
      expect(isHaulItem(OTHER)).toBe(false);
    });
  });

  describe("filterHaulItems", () => {
    it("keeps tree items and drops items of other kinds", () => {
      const item = createHaulItem(KEY);
      expect(filterHaulItems([item, OTHER])).toEqual([item]);
    });
  });

  describe("canDropHaulItem", () => {
    it("returns true when at least one item is a tree item", () => {
      expect(
        canDropHaulItem({ source: OTHER, items: [createHaulItem(KEY), OTHER] }),
      ).toBe(true);
    });

    it("returns false when no item is a tree item", () => {
      expect(canDropHaulItem({ source: OTHER, items: [OTHER] })).toBe(false);
    });
  });
});

describe("Tree", () => {
  beforeAll(() => mockGeometry(100, 100));

  const NODES: Node[] = Array.from({ length: 200 }, (_, i) => ({
    key: `node-${String(i).padStart(3, "0")}`,
  }));

  const getItem = List.createGetItem(
    (key: string) => ({ key, name: key }),
    (keys: string[]) => keys.map((key) => ({ key, name: key })),
  );

  const Component = () => {
    const props = use({ nodes: NODES });
    return (
      <Tree {...props} getItem={getItem}>
        {({ key, ...rest }) => (
          <Item key={key} {...rest}>
            {key}
          </Item>
        )}
      </Tree>
    );
  };

  const renderTree = () => render(<Component />);

  it("should window its rows by default", () => {
    const { container } = renderTree();
    const rows = container.querySelectorAll(".pluto-tree__item");
    expect(rows.length).toBeGreaterThan(0);
    expect(rows.length).toBeLessThan(NODES.length);
  });

  it("should space its rows by the tree's item height", () => {
    const { container } = renderTree();
    const rows = Array.from(
      container.querySelectorAll<HTMLElement>(".pluto-tree__item"),
    );
    rows.forEach((row, index) =>
      expect(row.style.transform).toBe(`translateY(${index * 27}px)`),
    );
  });

  it("should reserve scroll height for every node", () => {
    const { container } = renderTree();
    const virtualizer = container.querySelector<HTMLElement>(
      ".pluto-list__virtualizer",
    );
    expect(virtualizer?.style.minHeight).toBe(`${NODES.length * 27}px`);
  });
});
