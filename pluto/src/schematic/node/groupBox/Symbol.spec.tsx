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
import { assert, describe, expect, it, vi } from "vitest";

import { Haul } from "@/haul";
import { type Config } from "@/schematic/node/groupBox/config";
import { defaultConfig } from "@/schematic/node/groupBox/external";
import { Symbol } from "@/schematic/node/groupBox/Symbol";
import { type NodeProps } from "@/schematic/node/spec";

const renderSymbol = (
  props: Partial<NodeProps<Config>> = {},
): ReturnType<typeof render> =>
  render(
    <ReactFlowProvider>
      <Haul.Provider>
        <div data-id="g1">
          <Symbol
            nodeKey="g1"
            selected={false}
            onConfigChange={vi.fn()}
            config={defaultConfig()}
            {...props}
          />
        </div>
      </Haul.Provider>
    </ReactFlowProvider>,
  );

describe("GroupBox.Symbol", () => {
  it("should size the box to the configured dimensions", () => {
    const { container } = renderSymbol();
    const box = container.querySelector<HTMLElement>(".pluto-group-box");
    assert(box != null);
    expect(box.style.width).toBe("100px");
    expect(box.style.height).toBe("100px");
  });

  it("should show the move anchor when selected", () => {
    const { container } = renderSymbol({ selected: true });
    expect(container.querySelector(".pluto-group-box__move")).not.toBeNull();
  });

  it("should hide the move anchor when not selected", () => {
    const { container } = renderSymbol();
    expect(container.querySelector(".pluto-group-box__move")).toBeNull();
  });

  it("should hide the move anchor on a nested group", () => {
    const { container } = renderSymbol({ selected: true, draggable: false });
    expect(container.querySelector(".pluto-group-box__move")).toBeNull();
  });
});
