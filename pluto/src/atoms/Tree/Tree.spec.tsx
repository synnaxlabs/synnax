// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Tree, TreeLeaf } from ".";

describe("Tree", () => {
  it("should render a tree", () => {
    const tree: TreeLeaf[] = [
      {
        title: "Test",
        key: "test",
        hasChildren: true,
        children: [
          {
            title: "Test Child",
            key: "test-child",
            hasChildren: false,
          },
        ],
      },
    ];
    const { getByText, queryByText } = render(<Tree data={tree} />);
    expect(getByText("Test")).toBeTruthy();
    expect(queryByText("Test Child")).toBeFalsy();
  });
  it("should expand a tree node when a user clicks on it", () => {
    const tree: TreeLeaf[] = [
      {
        title: "Test",
        key: "test",
        hasChildren: true,
        children: [
          {
            title: "Test Child",
            key: "test-child",
            hasChildren: false,
          },
        ],
      },
    ];
    const { getByText } = render(<Tree data={tree} />);
    const node = getByText("Test");
    expect(node).toBeTruthy();
    fireEvent.click(node);
    expect(getByText("Test Child")).toBeTruthy();
  });
});
