// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { Haul } from "@synnaxlabs/lyra/haul";
import { render } from "@testing-library/react";
import { ReactFlowProvider } from "@xyflow/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { Symbol } from "@/schematic/node/general/polygon/Symbol";

// React Flow resizes a node by pointer drags that jsdom cannot run, so the spec calls
// the resize callback that the control receives.
const mocks = vi.hoisted(() => ({
  onResize: [] as Array<(e: unknown, d: { width: number; height: number }) => void>,
}));

vi.mock("@xyflow/react", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  return {
    ...actual,
    NodeResizeControl: ({
      onResize,
    }: {
      onResize: (e: unknown, d: { width: number; height: number }) => void;
    }) => {
      mocks.onResize.push(onResize);
      return null;
    },
  };
});

describe("Polygon.Symbol", () => {
  beforeEach(() => {
    mocks.onResize = [];
  });

  it("should round the side length of a resize to whole pixels", () => {
    const onConfigChange = vi.fn();
    render(
      <ReactFlowProvider>
        <Haul.Provider>
          <div data-id="p1">
            <Symbol
              nodeKey="p1"
              selected
              onConfigChange={onConfigChange}
              config={schematic.polygonNodeConfigZ.parse({ variant: "polygon" })}
            />
          </div>
        </Haul.Provider>
      </ReactFlowProvider>,
    );
    expect(mocks.onResize.length).toBeGreaterThan(0);
    mocks.onResize[0]({}, { width: 101, height: 101 });
    // A hexagon: 101 * sin(π / 6) / 0.8 = 63.125.
    expect(onConfigChange).toHaveBeenLastCalledWith({ sideLength: 63 });
  });
});
