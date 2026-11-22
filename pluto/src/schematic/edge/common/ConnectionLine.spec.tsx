// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box } from "@synnaxlabs/x";
import { render } from "@testing-library/react";
import { ReactFlowProvider } from "@xyflow/react";
import { describe, expect, it } from "vitest";

import { ConnectionLine } from "@/schematic/edge/common";
import { diagram } from "@/vis/diagram/aether";

const renderLine = (status: "valid" | "invalid" | null): SVGPathElement => {
  const { container } = render(
    <ReactFlowProvider>
      <svg>
        <ConnectionLine
          source={diagram.createEndpoint(40, 20, "right")}
          target={diagram.createEndpoint(400, 20, "left")}
          sourceBox={box.construct(0, 0, 40, 40)}
          targetBox={box.construct(400, 0, 40, 40)}
          status={status}
          style={{}}
        />
      </svg>
    </ReactFlowProvider>,
  );
  const path = container.querySelector<SVGPathElement>(".react-flow__edge-path");
  if (path == null) throw new Error("expected a connection line path");
  return path;
};

describe("Edge.ConnectionLine", () => {
  it("should draw a path", () => {
    expect(renderLine(null).getAttribute("d")).toMatch(/^M/);
  });

  it("should stroke a valid connection in gray", () => {
    expect(renderLine("valid").style.stroke).toBe("var(--pluto-gray-l11)");
  });

  it("should stroke an invalid connection in the error color", () => {
    expect(renderLine("invalid").style.stroke).toBe("var(--pluto-error-z)");
  });
});
