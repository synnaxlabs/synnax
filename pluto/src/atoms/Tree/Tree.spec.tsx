import { useState } from "react";

import { fireEvent, render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Tree, TreeLeaf } from ".";

const ControlledTree = ({ data }: { data: TreeLeaf[] }): JSX.Element => {
  const [value, setValue] = useState<readonly string[]>([]);
  return <Tree value={value} onChange={setValue} data={data} />;
};

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
    const { getByText, queryByText } = render(<ControlledTree data={tree} />);
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
    const { getByText } = render(<ControlledTree data={tree} />);
    const node = getByText("Test");
    expect(node).toBeTruthy();
    fireEvent.click(node);
    expect(getByText("Test Child")).toBeTruthy();
  });
});
