// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import { type ControlPosition, ReactFlowProvider } from "@xyflow/react";
import { type FC, type ReactElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { Haul } from "@/haul";
import { Symbol as LightSymbol } from "@/schematic/node/general/light/Symbol";
import { Theming } from "@/theming";

const NODE_KEY = "light-1";

interface SpyResizeControlProps {
  position: ControlPosition;
  onResize?: (event: unknown, params: { width: number; height: number }) => void;
}

const { resizeControls } = vi.hoisted(() => ({
  resizeControls: new Map<string, SpyResizeControlProps>(),
}));

// NodeResizeControl renders for real; the spy only records onResize so the test
// can drive the resize callback the DOM does not expose.
vi.mock("@xyflow/react", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  const Real = actual.NodeResizeControl as FC<SpyResizeControlProps>;
  return {
    ...actual,
    NodeResizeControl: (props: SpyResizeControlProps): ReactElement => {
      resizeControls.set(props.position, props);
      return <Real {...props} />;
    },
  };
});

// Light's telemetry source is irrelevant to resizing; stub the hook so the Symbol
// renders without an Aether provider.
vi.mock("@/vis/light", () => ({ Light: { use: () => ({ enabled: false }) } }));

const Wrap = ({ children }: { children: ReactNode }): ReactElement => (
  <ReactFlowProvider>
    <Theming.Provider>
      <Haul.Provider>
        <div data-id={NODE_KEY}>{children}</div>
      </Haul.Provider>
    </Theming.Provider>
  </ReactFlowProvider>
);

describe("Light resize", () => {
  beforeEach(() => resizeControls.clear());

  it("should set scale to the dragged width over the scale-1 width", () => {
    const onConfigChange = vi.fn();
    render(
      <Wrap>
        <LightSymbol
          nodeKey={NODE_KEY}
          selected
          onConfigChange={onConfigChange}
          config={{ variant: "light", scale: 1 }}
        />
      </Wrap>,
    );
    // 256 = WIDTH_PER_SCALE (51.2) * 5; the differing height is ignored because the
    // symbol scales uniformly off width.
    resizeControls.get("right")?.onResize?.(null, { width: 256, height: 999 });
    expect(onConfigChange).toHaveBeenCalledWith({ scale: 5 });
  });
});
