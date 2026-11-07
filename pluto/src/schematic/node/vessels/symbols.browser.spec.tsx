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

import { Haul } from "@synnaxlabs/lyra/haul";
import { renderThemed } from "@synnaxlabs/lyra/styletest";
import { direction, type location, type xy } from "@synnaxlabs/x";
import {
  type Edge,
  type EdgeProps,
  type EdgeTypes,
  Handle,
  type Node,
  type NodeTypes,
  Position,
  ReactFlow,
  ReactFlowProvider,
} from "@xyflow/react";
import { type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { CSS } from "@/css";
import { createConfig, resolveSpec } from "@/schematic/node/registry";

const ORIENTATIONS: location.Outer[] = ["left", "right", "top", "bottom"];
const SCALES = [1, 2];
const TOLERANCE = 0.1;
const SYMBOL_POSITION = { x: 100, y: 100 };
const NOOP = (): void => {};

interface Bar {
  handle: string;
  /** The axis of the symbol's drawing that runs across the bar. */
  across: direction.Direction;
  /** The center of the bar along that axis, in drawing units. */
  center: number;
}

const JUNCTIONS: Array<{ variant: "t_junction" | "cross_junction"; bars: Bar[] }> = [
  {
    variant: "t_junction",
    bars: [
      { handle: "1", across: "y", center: 4.5 },
      { handle: "2", across: "y", center: 4.5 },
      { handle: "3", across: "x", center: 19.5 },
    ],
  },
  {
    variant: "cross_junction",
    bars: [
      { handle: "1", across: "y", center: 19.5 },
      { handle: "2", across: "y", center: 19.5 },
      { handle: "3", across: "x", center: 19.5 },
      { handle: "4", across: "x", center: 19.5 },
    ],
  },
];

const createEdgeRecorder = (): {
  ends: Map<string, xy.XY>;
  edgeTypes: EdgeTypes;
} => {
  const ends = new Map<string, xy.XY>();
  const Recorder = ({ id, sourceX, sourceY }: EdgeProps): null => {
    ends.set(id, { x: sourceX, y: sourceY });
    return null;
  };
  return { ends, edgeTypes: { recorder: Recorder } };
};

const Target = (): ReactElement => (
  <div>
    <Handle id="target" type="target" position={Position.Left} />
  </div>
);

interface RenderProps {
  variant: "t_junction" | "cross_junction";
  orientation: location.Outer;
  scale: number;
  bars: Bar[];
}

const renderJunction = async ({ variant, orientation, scale, bars }: RenderProps) => {
  const { ends, edgeTypes } = createEdgeRecorder();
  const Junction = resolveSpec(variant).Node;
  const config = { ...createConfig({ variant }), orientation, scale };
  const nodeTypes: NodeTypes = {
    symbol: ({ id }) => (
      <Junction nodeKey={id} selected={false} config={config} onConfigChange={NOOP} />
    ),
    target: Target,
  };
  const nodes: Node[] = [
    { id: "symbol", type: "symbol", position: SYMBOL_POSITION, data: {} },
    { id: "target", type: "target", position: { x: 400, y: 400 }, data: {} },
  ];
  const edges: Edge[] = bars.map(({ handle }) => ({
    id: handle,
    type: "recorder",
    source: "symbol",
    sourceHandle: handle,
    target: "target",
    targetHandle: "target",
  }));
  const { container } = await renderThemed(
    <div style={{ width: 800, height: 800 }}>
      <Haul.Provider>
        <ReactFlowProvider>
          <ReactFlow
            className={CSS.B("schematic")}
            nodes={nodes}
            edges={edges}
            nodeTypes={nodeTypes}
            edgeTypes={edgeTypes}
            defaultViewport={{ x: 0, y: 0, zoom: 1 }}
          />
        </ReactFlowProvider>
      </Haul.Provider>
    </div>,
  );
  await vi.waitFor(() => expect(ends.size).toBe(bars.length));
  const flow = container.querySelector(".react-flow");
  const path = container.querySelector(".react-flow__node-symbol svg path");
  if (flow == null || !(path instanceof SVGPathElement))
    throw new Error("junction not rendered");
  return { ends, flow, path };
};

/**
 * @returns the bar's center in flow coordinates, and the axis of the flow that runs
 * across the bar in this orientation.
 */
const barCenter = (
  path: SVGPathElement,
  flow: Element,
  { across, center }: Bar,
): { position: xy.XY; across: direction.Direction } => {
  const matrix = path.getScreenCTM();
  if (matrix == null) throw new Error("symbol has no screen matrix");
  const drawn = new DOMPoint(
    across === "x" ? center : 0,
    across === "y" ? center : 0,
  ).matrixTransform(matrix);
  const { left, top } = flow.getBoundingClientRect();
  const rotated = Math.abs(matrix.a) < Math.abs(matrix.b);
  return {
    position: { x: drawn.x - left, y: drawn.y - top },
    across: rotated ? direction.swap(across) : across,
  };
};

describe("Vessel symbols", () => {
  describe.each(JUNCTIONS)("$variant", ({ variant, bars }) => {
    describe.each(ORIENTATIONS)("%s", (orientation) => {
      it.each(SCALES)(
        "should center its connection points on its bars at scale %d",
        async (scale) => {
          const { ends, flow, path } = await renderJunction({
            variant,
            orientation,
            scale,
            bars,
          });
          for (const bar of bars) {
            const { position, across } = barCenter(path, flow, bar);
            const end = ends.get(bar.handle);
            if (end == null) throw new Error(`no edge from handle ${bar.handle}`);
            const offset = Math.abs(end[across] - position[across]);
            expect(offset, `handle ${bar.handle}`).toBeLessThan(TOLERANCE);
          }
        },
      );
    });
  });
});
