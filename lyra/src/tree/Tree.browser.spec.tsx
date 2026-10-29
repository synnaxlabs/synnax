// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { List } from "@/list";
import { renderThemed } from "@/styletest";
import { Tree } from "@/tree";

const NODES: Tree.Node[] = [
  { key: "parent", children: [{ key: "child", children: [{ key: "grandchild" }] }] },
  { key: "sibling" },
];

const getItem = List.createGetItem(
  (key: string) => ({ key, name: key }),
  (keys: string[]) => keys.map((key) => ({ key, name: key })),
);

const Component = (): ReactElement => {
  const props = Tree.use({ nodes: NODES, initialExpanded: ["parent", "child"] });
  return (
    <Tree.Tree {...props} getItem={getItem}>
      {({ key, ...rest }) => (
        <Tree.Item key={key} {...rest}>
          {key}
        </Tree.Item>
      )}
    </Tree.Tree>
  );
};

const renderTree = async (): Promise<(key: string) => DOMRect> => {
  const { getByText } = await renderThemed(<Component />);
  return (key) => {
    const row = getByText(key, { exact: true }).element();
    const text = Array.from(row.childNodes).find(
      (node) => node.nodeType === Node.TEXT_NODE,
    )!;
    const range = document.createRange();
    range.selectNode(text);
    return range.getBoundingClientRect();
  };
};

describe("Tree", () => {
  it("should indent a child's text past its parent's text", async () => {
    const rect = await renderTree();
    expect(rect("child").left).toBeGreaterThan(rect("parent").left);
  });

  it("should line up a leaf's text with its parent's text", async () => {
    const rect = await renderTree();
    expect(rect("grandchild").left).toBe(rect("child").left);
  });

  it("should stack rows at one pitch without gaps or overlaps", async () => {
    await renderTree();
    const rows = ["parent", "child", "grandchild", "sibling"].map((key) =>
      document.getElementById(key)!.getBoundingClientRect(),
    );
    for (let i = 1; i < rows.length; i++) {
      expect(rows[i].height).toBe(rows[0].height);
      expect(rows[i].top).toBe(rows[i - 1].bottom);
    }
  });
});
