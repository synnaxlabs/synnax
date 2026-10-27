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
  type Node,
  Position as RFPosition,
  ReactFlow,
  ReactFlowProvider,
} from "@xyflow/react";
import { type ReactElement } from "react";
import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";

import { Handle } from "@/arc/graph/handle";

const SourceNode = (): ReactElement => <Handle.Source location="right" id="out" />;
const NODE_TYPES = { source: SourceNode };
const NODES: Node[] = [{ id: "a", type: "source", position: { x: 0, y: 0 }, data: {} }];

const queryHandle = (container: HTMLElement): HTMLElement | null =>
  container.querySelector(".react-flow__handle");

describe("Handle", () => {
  // jsdom lays nothing out; React Flow sizes its container from these.
  beforeAll(() => {
    vi.spyOn(HTMLElement.prototype, "offsetWidth", "get").mockReturnValue(500);
    vi.spyOn(HTMLElement.prototype, "offsetHeight", "get").mockReturnValue(500);
  });
  afterAll(() => vi.restoreAllMocks());

  it("should render inside a React Flow node", () => {
    const { container } = render(
      <ReactFlowProvider>
        <ReactFlow defaultNodes={NODES} nodeTypes={NODE_TYPES} />
      </ReactFlowProvider>,
    );
    expect(queryHandle(container)?.getAttribute("data-handlepos")).toBe(
      RFPosition.Right,
    );
  });

  it("should render nothing in a flow outside a node", () => {
    const { container } = render(
      <ReactFlowProvider>
        <Handle.Source location="right" id="out" />
      </ReactFlowProvider>,
    );
    expect(queryHandle(container)).toBeNull();
  });

  it("should render nothing outside a flow", () => {
    const { container } = render(<Handle.Source location="right" id="out" />);
    expect(queryHandle(container)).toBeNull();
  });
});
