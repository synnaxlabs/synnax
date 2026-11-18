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
import { type location, type xy } from "@synnaxlabs/x";
import {
  type Edge,
  type EdgeProps,
  Handle,
  type Node as RFNode,
  type NodeTypes,
  Position,
  ReactFlow,
  ReactFlowProvider,
} from "@xyflow/react";
import { type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { CSS } from "@/css";
import { Node } from "@/schematic/node";

const VARIANT = "electric_regulator_motorized";
const ORIENTATIONS: location.Outer[] = ["left", "right", "top", "bottom"];
const SCALES = [1, 2];
const TOLERANCE = 0.1;
// Stored edges use handle "2", so it stays at the left of the motor.
const MOTOR_LEFT_HANDLE = "2";
const MOTOR_LEFT: xy.XY = { x: 31, y: 14 };
const NOOP = (): void => {};

const Target = (): ReactElement => (
  <div>
    <Handle id="target" type="target" position={Position.Left} />
  </div>
);

interface RenderProps {
  orientation: location.Outer;
  scale: number;
}

const renderRegulator = async ({ orientation, scale }: RenderProps) => {
  let end: xy.XY | null = null;
  const Recorder = ({ sourceX, sourceY }: EdgeProps): null => {
    end = { x: sourceX, y: sourceY };
    return null;
  };
  const Regulator = Node.resolveSpec(VARIANT).Node;
  const config = { ...Node.createConfig({ variant: VARIANT }), orientation, scale };
  const nodeTypes: NodeTypes = {
    symbol: ({ id }) => (
      <Regulator nodeKey={id} selected={false} config={config} onConfigChange={NOOP} />
    ),
    target: Target,
  };
  const nodes: RFNode[] = [
    { id: "symbol", type: "symbol", position: { x: 100, y: 100 }, data: {} },
    { id: "target", type: "target", position: { x: 400, y: 400 }, data: {} },
  ];
  const edges: Edge[] = [
    {
      id: "edge",
      type: "recorder",
      source: "symbol",
      sourceHandle: MOTOR_LEFT_HANDLE,
      target: "target",
      targetHandle: "target",
    },
  ];
  const { container } = await renderThemed(
    <div style={{ width: 800, height: 800 }}>
      <Haul.Provider>
        <ReactFlowProvider>
          <ReactFlow
            className={CSS.B("schematic")}
            nodes={nodes}
            edges={edges}
            nodeTypes={nodeTypes}
            edgeTypes={{ recorder: Recorder }}
            defaultViewport={{ x: 0, y: 0, zoom: 1 }}
          />
        </ReactFlowProvider>
      </Haul.Provider>
    </div>,
  );
  await vi.waitFor(() => expect(end).not.toBeNull());
  const flow = container.querySelector(".react-flow");
  const motor = container.querySelector(".react-flow__node-symbol svg circle");
  const handle = container.querySelector(
    ".react-flow__node-symbol .react-flow__handle",
  );
  if (flow == null || !(motor instanceof SVGCircleElement) || handle == null)
    throw new Error("regulator not rendered");
  const matrix = motor.getScreenCTM();
  if (matrix == null) throw new Error("regulator has no screen matrix");
  const origin = flow.getBoundingClientRect();
  const point = new DOMPoint(MOTOR_LEFT.x, MOTOR_LEFT.y).matrixTransform(matrix);
  return {
    end: end!,
    motorLeft: { x: point.x - origin.left, y: point.y - origin.top },
    handle: handle.getBoundingClientRect(),
  };
};

describe("ElectricRegulatorMotorized", () => {
  describe.each(ORIENTATIONS)("%s", (orientation) => {
    it.each(SCALES)(
      "should attach an edge on handle 2 at the left of its motor at scale %d",
      async (scale) => {
        const { end, motorLeft, handle } = await renderRegulator({
          orientation,
          scale,
        });
        expect(Math.abs(end.x - motorLeft.x)).toBeLessThanOrEqual(
          handle.width / 2 + TOLERANCE,
        );
        expect(Math.abs(end.y - motorLeft.y)).toBeLessThanOrEqual(
          handle.height / 2 + TOLERANCE,
        );
      },
    );
  });
});
