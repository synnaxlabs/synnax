// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import { type Node, ReactFlowProvider } from "@xyflow/react";
import { describe, expect, it, vi } from "vitest";

import { Jumps } from "@/schematic/edge/common/jumps";
import { resolveSpec, type Variant } from "@/schematic/edge/registry";
import { diagram } from "@/vis/diagram/aether";

const SIZE = { width: 40, height: 40 };

const NODES: Node[] = [
  { id: "a", position: { x: 0, y: 0 }, measured: SIZE, data: {} },
  { id: "b", position: { x: 400, y: 0 }, measured: SIZE, data: {} },
];

const VARIANTS: Variant[] = [
  "pipe",
  "electric",
  "secondary",
  "jacketed",
  "hydraulic",
  "pneumatic",
  "data",
];

const renderEdge = (variant: Variant, strokeWidth?: number): SVGElement => {
  const spec = resolveSpec(variant);
  const { container } = render(
    <ReactFlowProvider defaultNodes={NODES}>
      <Jumps.Context value={Jumps.createStore()}>
        <svg>
          <spec.Edge
            edgeKey="edge"
            source={diagram.createEndpoint(40, 20, "right")}
            target={diagram.createEndpoint(400, 20, "left")}
            sourceNode="a"
            targetNode="b"
            selected={false}
            config={{ ...spec.defaultConfig(), strokeWidth }}
            onChange={vi.fn()}
          />
        </svg>
      </Jumps.Context>
    </ReactFlowProvider>,
  );
  return container.querySelector("svg") as SVGElement;
};

const edgeWidths = (svg: SVGElement): string[] =>
  Array.from(svg.querySelectorAll<SVGPathElement>(".react-flow__edge-path")).map(
    (el) => el.style.strokeWidth,
  );

const symbolWidths = (svg: SVGElement): Array<string | null> =>
  Array.from(
    svg.querySelectorAll("[stroke-width]:not(.react-flow__edge-interaction)"),
  ).map((el) => el.getAttribute("stroke-width"));

describe("edge registry", () => {
  it.each(VARIANTS)("should draw every %s line at the stroke width", (variant) => {
    const widths = edgeWidths(renderEdge(variant, 6));
    expect(widths.length).toBeGreaterThan(0);
    expect(new Set(widths)).toEqual(new Set(["6"]));
  });

  it.each(VARIANTS)("should draw a %s with no stroke width at 2px", (variant) => {
    expect(new Set(edgeWidths(renderEdge(variant)))).toEqual(new Set(["2"]));
  });

  it.each(["hydraulic", "pneumatic", "data"] satisfies Variant[])(
    "should draw the %s symbols at the stroke width",
    (variant) => {
      const widths = symbolWidths(renderEdge(variant, 6));
      expect(widths.length).toBeGreaterThan(0);
      expect(new Set(widths)).toEqual(new Set(["6"]));
    },
  );

  it("should keep the jacket clear of a wide core line", () => {
    const paths = (strokeWidth: number) =>
      Array.from(
        renderEdge("jacketed", strokeWidth).querySelectorAll(".react-flow__edge-path"),
      ).map((el) => el.getAttribute("d"));
    const [, atDefault] = paths(2);
    const [jacket, core] = paths(8);
    expect(core).toEqual(atDefault);
    expect(jacket).toMatch(/^M\S*?,32L/);
  });

  it("should keep the default jacket offset for an edge with no stroke width", () => {
    const jacket = renderEdge("jacketed").querySelector(".react-flow__edge-path");
    expect(jacket?.getAttribute("d")).toMatch(/^M\S*?,26L/);
  });
});
