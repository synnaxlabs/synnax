// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/Schematic.css";
import "@xyflow/react/dist/base.css";

import { renderThemed } from "@synnaxlabs/lyra/styletest";
import { describe, expect, it } from "vitest";

import { CSS } from "@/css";
import { Base } from "@/schematic/edge/common/base";

const renderedStrokeWidth = async (strokeWidth?: number): Promise<string> => {
  const { container } = await renderThemed(
    <div className={CSS.B("schematic")}>
      <svg>
        <g className="react-flow__edge">
          <Base.Base path="M0 10 L100 10" strokeWidth={strokeWidth} />
        </g>
      </svg>
    </div>,
  );
  const path = container.querySelector<SVGPathElement>("path.react-flow__edge-path");
  if (path == null) throw new Error("expected an edge path element");
  return getComputedStyle(path).strokeWidth;
};

describe("edge Base layout", () => {
  it("should draw at the default stroke width inside a schematic", async () => {
    expect(await renderedStrokeWidth()).toBe(`${Base.DEFAULT_STROKE_WIDTH}px`);
  });

  it("should draw at a wide stroke width inside a schematic", async () => {
    expect(await renderedStrokeWidth(8)).toBe("8px");
  });
});
