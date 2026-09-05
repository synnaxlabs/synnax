// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render } from "@testing-library/react";
import { type Node, ReactFlowProvider, useStoreApi } from "@xyflow/react";
import { type ReactElement } from "react";
import { assert, describe, expect, it, vi } from "vitest";

import { Haul } from "@/haul";
import { type Config } from "@/schematic/node/groupBox/config";
import { defaultConfig } from "@/schematic/node/groupBox/external";
import { Symbol } from "@/schematic/node/groupBox/Symbol";
import { type NodeProps } from "@/schematic/node/spec";

const initialNodes: Node[] = [
  { id: "g1", position: { x: -20, y: -40 }, data: {} },
  { id: "m1", position: { x: 0, y: 0 }, measured: { width: 40, height: 20 }, data: {} },
  {
    id: "m2",
    position: { x: 100, y: 60 },
    measured: { width: 50, height: 40 },
    data: {},
  },
];

const renderSymbol = (
  props: Partial<NodeProps<Config>> = {},
): ReturnType<typeof render> =>
  render(
    <ReactFlowProvider initialNodes={initialNodes}>
      <Haul.Provider>
        <div data-id="g1">
          <Symbol
            nodeKey="g1"
            selected={false}
            onConfigChange={vi.fn()}
            config={{ ...defaultConfig(), members: ["m1", "m2"] }}
            {...props}
          />
        </div>
      </Haul.Provider>
    </ReactFlowProvider>,
  );

const box = (container: HTMLElement): HTMLElement => {
  const el = container.querySelector<HTMLElement>(".pluto-group-box");
  assert(el != null);
  return el;
};

const chip = (container: HTMLElement): HTMLElement => {
  const el = container.querySelector<HTMLElement>(".pluto-group-box__lock");
  assert(el != null);
  return el;
};

describe("GroupBox.Symbol", () => {
  it("should size the box to its members' bounds plus padding", () => {
    // Far member edge: m2 at (100, 60) measuring 50x40. Box anchored at (-20, -40):
    // width = 150 - (-20) + 20, height = 100 - (-40) + 20.
    const { container } = renderSymbol();
    expect(box(container).style.width).toBe("190px");
    expect(box(container).style.height).toBe("160px");
  });

  it("should fall back to the padding rectangle without members", () => {
    const { container } = renderSymbol({ config: defaultConfig() });
    expect(box(container).style.width).toBe("40px");
    expect(box(container).style.height).toBe("60px");
  });

  it("should track a member growing and shrinking", () => {
    const resized = (m2: { width: number; height: number }): Node[] =>
      initialNodes.map((n) => (n.id === "m2" ? { ...n, measured: m2 } : n));
    const Resize = ({
      label,
      nodes,
    }: {
      label: string;
      nodes: Node[];
    }): ReactElement => {
      const store = useStoreApi();
      return <button onClick={() => store.getState().setNodes(nodes)}>{label}</button>;
    };
    const { container, getByText } = render(
      <ReactFlowProvider initialNodes={initialNodes}>
        <Haul.Provider>
          <div data-id="g1">
            <Symbol
              nodeKey="g1"
              selected={false}
              onConfigChange={vi.fn()}
              config={{ ...defaultConfig(), members: ["m1", "m2"] }}
            />
          </div>
          <Resize label="grow" nodes={resized({ width: 80, height: 90 })} />
          <Resize label="shrink" nodes={resized({ width: 10, height: 10 })} />
        </Haul.Provider>
      </ReactFlowProvider>,
    );
    expect(box(container).style.width).toBe("190px");
    fireEvent.click(getByText("grow"));
    expect(box(container).style.width).toBe("220px");
    expect(box(container).style.height).toBe("210px");
    fireEvent.click(getByText("shrink"));
    expect(box(container).style.width).toBe("150px");
    expect(box(container).style.height).toBe("130px");
  });

  it("should show the lock chip when selected", () => {
    const { container } = renderSymbol({ selected: true });
    expect(container.querySelector(".pluto-group-box__lock")).not.toBeNull();
  });

  it("should hide the lock chip when not selected", () => {
    const { container } = renderSymbol();
    expect(container.querySelector(".pluto-group-box__lock")).toBeNull();
  });

  it("should hide the lock chip on a nested group", () => {
    const { container } = renderSymbol({ selected: true, draggable: false });
    expect(container.querySelector(".pluto-group-box__lock")).toBeNull();
  });

  it("should keep the chip on a locked, undraggable box", () => {
    const { container } = renderSymbol({
      selected: true,
      draggable: false,
      config: { ...defaultConfig(), locked: true },
    });
    expect(container.querySelector(".pluto-group-box__lock")).not.toBeNull();
  });

  it("should show the open padlock while unlocked and lock on click", () => {
    const onConfigChange = vi.fn();
    const { container } = renderSymbol({ selected: true, onConfigChange });
    expect(container.querySelector(".pluto-icon--unlock")).not.toBeNull();
    expect(container.querySelector(".pluto-icon--lock")).toBeNull();
    fireEvent.click(chip(container));
    expect(onConfigChange).toHaveBeenCalledWith({ locked: true });
  });

  it("should show the closed padlock while locked and unlock on click", () => {
    const onConfigChange = vi.fn();
    const { container } = renderSymbol({
      selected: true,
      onConfigChange,
      config: { ...defaultConfig(), locked: true },
    });
    expect(container.querySelector(".pluto-icon--lock")).not.toBeNull();
    fireEvent.click(chip(container));
    expect(onConfigChange).toHaveBeenCalledWith({ locked: false });
  });
});
