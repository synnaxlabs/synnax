// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Portal } from "@/portal";

interface HarnessProps extends Omit<Portal.UseNodesProps, "children"> {
  onRender?: (ref: Map<string, Portal.Node>) => void;
  host?: string;
}

const Harness = ({ onRender, host, ...rest }: HarnessProps): ReactElement => {
  const [ref, nodes] = Portal.useNodes({
    ...rest,
    children: (key) => <p>content-{key}</p>,
  });
  onRender?.(ref.current);
  const hostNode = host != null ? ref.current.get(host) : undefined;
  return (
    <>
      {nodes}
      {hostNode != null && <Portal.Out node={hostNode} />}
    </>
  );
};

describe("Portal.useNodes", () => {
  it("should render children into a detached node per key", () => {
    let nodes!: Map<string, Portal.Node>;
    const { queryByText } = render(
      <Harness keys={["a", "b"]} onRender={(r) => (nodes = r)} />,
    );
    expect(nodes.get("a")?.el.textContent).toEqual("content-a");
    expect(nodes.get("b")?.el.textContent).toEqual("content-b");
    expect(queryByText("content-a")).toBeNull();
  });

  it("should attach a node's content to the document when hosted", () => {
    const { getByText } = render(<Harness keys={["a"]} host="a" />);
    expect(getByText("content-a")).toBeTruthy();
  });

  it("should keep the same node instance across re-renders", () => {
    let first!: Map<string, Portal.Node>;
    const { rerender } = render(<Harness keys={["a"]} onRender={(r) => (first = r)} />);
    const node = first.get("a");
    let second!: Map<string, Portal.Node>;
    rerender(<Harness keys={["a"]} onRender={(r) => (second = r)} />);
    expect(second.get("a")).toBe(node);
  });

  it("should release nodes whose keys disappear", () => {
    let nodes!: Map<string, Portal.Node>;
    const { rerender } = render(
      <Harness keys={["a", "b"]} onRender={(r) => (nodes = r)} />,
    );
    rerender(<Harness keys={["a"]} onRender={(r) => (nodes = r)} />);
    expect(nodes.has("b")).toBe(false);
    expect(nodes.has("a")).toBe(true);
  });

  it("should apply attrs to created node elements", () => {
    let nodes!: Map<string, Portal.Node>;
    render(
      <Harness
        keys={["a"]}
        attrs={{ style: "width: 100%;" }}
        onRender={(r) => (nodes = r)}
      />,
    );
    expect(nodes.get("a")?.el.getAttribute("style")).toEqual("width: 100%;");
  });

  it("should invoke onClick with the key when the hosted element is clicked", () => {
    const onClick = vi.fn();
    const { getByText } = render(<Harness keys={["a"]} host="a" onClick={onClick} />);
    fireEvent.click(getByText("content-a"));
    expect(onClick).toHaveBeenCalledTimes(1);
    expect(onClick).toHaveBeenCalledWith("a");
  });

  it("should invoke the latest onClick handler after re-renders", () => {
    const first = vi.fn();
    const second = vi.fn();
    const { getByText, rerender } = render(
      <Harness keys={["a"]} host="a" onClick={first} />,
    );
    rerender(<Harness keys={["a"]} host="a" onClick={second} />);
    fireEvent.click(getByText("content-a"));
    expect(first).not.toHaveBeenCalled();
    expect(second).toHaveBeenCalledWith("a");
  });
});
