// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import { ReactFlowProvider } from "@xyflow/react";
import { describe, expect, it, vi } from "vitest";

import { Haul } from "@/haul";
import { type Config } from "@/schematic/node/general/box/config";
import { Symbol } from "@/schematic/node/general/box/Symbol";
import { type NodeProps } from "@/schematic/node/spec";

const NODE_KEY = "b1";

const CONFIG: Config = {
  variant: "box",
  dimensions: { width: 125, height: 200 },
  borderRadius: 3,
  strokeWidth: 2,
};

const renderSymbol = (
  props: Partial<NodeProps<Config>> = {},
): ReturnType<typeof render> =>
  render(
    <ReactFlowProvider>
      <Haul.Provider>
        <div data-id={NODE_KEY}>
          <Symbol
            nodeKey={NODE_KEY}
            selected={false}
            onConfigChange={vi.fn()}
            config={CONFIG}
            {...props}
          />
        </div>
      </Haul.Provider>
    </ReactFlowProvider>,
  );

const frame = (container: HTMLElement): SVGRectElement | null =>
  container.querySelector<SVGRectElement>(".pluto-box-frame rect");

// Which element takes the pointer is decided by pointer-events rules in box.css, which
// jsdom cannot exercise. These cover the structure those rules key on.
describe("Box.Symbol", () => {
  it("should render the frame while unselected so the border can select the box", () => {
    const { container } = renderSymbol();
    expect(frame(container)).not.toBeNull();
  });

  it("should stretch the frame over the whole box", () => {
    const { container } = renderSymbol();
    expect(frame(container)?.getAttribute("width")).toBe("100%");
    expect(frame(container)?.getAttribute("height")).toBe("100%");
  });

  it("should round the frame with the border radius so the band follows the corners", () => {
    const { container } = renderSymbol({ config: { ...CONFIG, borderRadius: 12 } });
    expect(frame(container)?.getAttribute("rx")).toBe("12");
  });

  it("should leave the frame square without a border radius", () => {
    const { container } = renderSymbol({
      config: { ...CONFIG, borderRadius: undefined },
    });
    expect(frame(container)?.getAttribute("rx")).toBeNull();
  });

  it("should put the frame inside the drag handle so the border drags the node", () => {
    const { container } = renderSymbol();
    expect(
      container.querySelector(".pluto-drag-handle .pluto-box-frame"),
    ).not.toBeNull();
  });

  it("should mark the primitive as a box for the hollow node rule", () => {
    const { container } = renderSymbol();
    expect(container.querySelector(".pluto-box")).not.toBeNull();
  });

  it("should keep the resize controls when selected", () => {
    const { container } = renderSymbol({ selected: true });
    expect(container.querySelectorAll(".react-flow__resize-control")).toHaveLength(8);
  });
});
