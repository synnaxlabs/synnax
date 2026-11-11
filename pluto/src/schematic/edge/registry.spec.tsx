// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { render } from "@testing-library/react";
import { type Node, ReactFlowProvider } from "@xyflow/react";
import { describe, expect, it, vi } from "vitest";

import { Edge } from "@/schematic/edge";
import { diagram } from "@/vis/diagram/aether";

const SIZE = { width: 40, height: 40 };

const NODES: Node[] = [
  { id: "a", position: { x: 0, y: 0 }, measured: SIZE, data: {} },
  { id: "b", position: { x: 400, y: 0 }, measured: SIZE, data: {} },
];

const VARIANTS: Edge.Variant[] = [
  "pipe",
  "electric",
  "secondary",
  "jacketed",
  "hydraulic",
  "pneumatic",
  "data",
];

interface RenderOptions {
  strokeWidth?: number;
  crossings?: Edge.Path.Crossing[];
}

const renderEdge = (
  variant: Edge.Variant,
  { strokeWidth, crossings = [] }: RenderOptions = {},
): SVGElement => {
  const spec = Edge.resolveSpec(variant);
  const jumps = Edge.Jumps.createStore();
  jumps.commit(new Map([["edge", crossings]]));
  const edge = schematic.edgeZ.parse({
    key: "edge",
    source: { node: "a", param: "out" },
    target: { node: "b", param: "in" },
    variant,
    strokeWidth,
  });
  const { container } = render(
    <ReactFlowProvider defaultNodes={NODES}>
      <Edge.Jumps.Context value={jumps}>
        <svg>
          <spec.Edge
            edgeKey="edge"
            source={diagram.createEndpoint(40, 20, "right")}
            target={diagram.createEndpoint(400, 20, "left")}
            sourceNode="a"
            targetNode="b"
            selected={false}
            edge={edge}
            onChange={vi.fn()}
          />
        </svg>
      </Edge.Jumps.Context>
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
    const widths = edgeWidths(renderEdge(variant, { strokeWidth: 6 }));
    expect(widths.length).toBeGreaterThan(0);
    expect(new Set(widths)).toEqual(new Set(["6"]));
  });

  it.each(VARIANTS)("should draw a %s with no stroke width at 2px", (variant) => {
    expect(new Set(edgeWidths(renderEdge(variant)))).toEqual(new Set(["2"]));
  });

  it.each(["hydraulic", "pneumatic", "data"] satisfies Edge.Variant[])(
    "should draw the %s symbols at the stroke width",
    (variant) => {
      const widths = symbolWidths(renderEdge(variant, { strokeWidth: 6 }));
      expect(widths.length).toBeGreaterThan(0);
      expect(new Set(widths)).toEqual(new Set(["6"]));
    },
  );

  it("should keep the jacket clear of a wide core line", () => {
    const paths = (strokeWidth: number) =>
      Array.from(
        renderEdge("jacketed", { strokeWidth }).querySelectorAll(
          ".react-flow__edge-path",
        ),
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

  it("should hop every jacketed line over a crossing at the widest stroke", () => {
    const paths = renderEdge("jacketed", {
      strokeWidth: 20,
      crossings: [{ x: 200, y: 20, radius: 23 }],
    }).querySelectorAll(".react-flow__edge-path");
    expect(paths).toHaveLength(3);
    paths.forEach((el) => expect(el.getAttribute("d")).toContain("A23,23"));
  });

  it("should keep data symbols apart at the widest stroke", () => {
    const strokeWidth = 20;
    const circles = Array.from(
      renderEdge("data", { strokeWidth }).querySelectorAll("circle"),
    );
    expect(circles.length).toBeGreaterThan(1);
    const diameter = 2 * Number(circles[0].getAttribute("r")) + strokeWidth;
    const centers = circles.map((c) => Number(c.getAttribute("cx")));
    centers
      .slice(1)
      .forEach((cx, i) => expect(cx - centers[i]).toBeGreaterThan(diameter));
  });
});

describe("renderedWidth", () => {
  it.each(VARIANTS.filter((v) => v !== "jacketed"))(
    "should be the stroke width for a %s edge",
    (variant) => {
      expect(Edge.REGISTRY[variant].renderedWidth(6)).toBe(6);
    },
  );

  it.each([2, 8, 20])(
    "should include both jacket lines of a jacketed edge at %i px",
    (strokeWidth) => {
      const jacketGap = 4;
      const outer = jacketGap + strokeWidth + strokeWidth / 2;
      expect(Edge.REGISTRY.jacketed.renderedWidth(strokeWidth)).toBe(2 * outer);
    },
  );

  it.each([8, 20])("should hop a thin line clear of a %i px jacketed edge", (sw) => {
    const width = Edge.REGISTRY.jacketed.renderedWidth(sw);
    const crossings = Edge.Jumps.findCrossings([
      {
        key: "jacketed",
        order: 0,
        strokeWidth: sw,
        width,
        points: [
          { x: 50, y: 0 },
          { x: 50, y: 100 },
        ],
      },
      {
        key: "thin",
        order: 1,
        strokeWidth: 2,
        width: 2,
        points: [
          { x: 0, y: 50 },
          { x: 100, y: 50 },
        ],
      },
    ]);
    const [hop] = crossings.get("thin") ?? [];
    expect(hop.radius).toBeGreaterThan(width / 2);
  });

  it("should size the hop of a jacketed edge by its own stroke", () => {
    const crossings = Edge.Jumps.findCrossings([
      {
        key: "pipe",
        order: 0,
        strokeWidth: 2,
        width: 2,
        points: [
          { x: 50, y: 0 },
          { x: 50, y: 100 },
        ],
      },
      {
        key: "jacketed",
        order: 1,
        strokeWidth: 2,
        width: Edge.REGISTRY.jacketed.renderedWidth(2),
        points: [
          { x: 0, y: 50 },
          { x: 100, y: 50 },
        ],
      },
    ]);
    expect(crossings.get("jacketed")).toEqual([{ x: 50, y: 50, radius: 5 }]);
  });
});
