// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import {
  NodeResizeControl,
  ReactFlowProvider,
  type ResizeControlProps,
  type ResizeDragEvent,
} from "@xyflow/react";
import { type FC, type ReactElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { Aether } from "@/aether";
import { Haul } from "@/haul";
import { Symbol as LightSymbol } from "@/schematic/node/general/light/Symbol";
import { Theming } from "@/theming";
import { Context as DiagramContext, ZERO_CONTEXT_VALUE } from "@/vis/diagram/Context";

const NODE_KEY = "light-1";

interface RecordedControl {
  triggerResize: (width: number, height: number) => void;
}

const resizeControls = new Map<string, RecordedControl>();

const RESIZE_EVENT = {} as ResizeDragEvent;

// Renders the real NodeResizeControl and records a trigger so the test can drive a
// resize the DOM cannot perform.
const SpyResizeControl: FC<ResizeControlProps> = (props) => {
  resizeControls.set(props.position ?? "", {
    triggerResize: (width, height) =>
      props.onResize?.(RESIZE_EVENT, { x: 0, y: 0, width, height, direction: [] }),
  });
  return <NodeResizeControl {...props} />;
};

const diagramCtx = { ...ZERO_CONTEXT_VALUE, resizeControl: SpyResizeControl };

const Wrap = ({ children }: { children: ReactNode }): ReactElement => (
  <ReactFlowProvider>
    <Theming.Provider>
      <Haul.Provider>
        <Aether.Provider workerEnabled={false}>
          <DiagramContext value={diagramCtx}>
            <div data-id={NODE_KEY}>{children}</div>
          </DiagramContext>
        </Aether.Provider>
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
    resizeControls.get("right")?.triggerResize(256, 999);
    expect(onConfigChange).toHaveBeenCalledWith({ scale: 5 });
  });
});
