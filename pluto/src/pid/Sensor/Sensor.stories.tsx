// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Meta, StoryFn } from "@storybook/react";
import { Background, Controls, Node, ReactFlow, useViewport } from "reactflow";

import { PIDProvider } from "../PIDProvider";
import { ValveBody } from "../Valve/ValveBody";

import { SensorNumeric } from "./SensorNumeric";

import { Sensor } from ".";

const story: Meta<typeof Sensor.Numeric> = {
  title: "PID/Sensor",
  component: Sensor.Numeric,
};

const nodeTypes = {
  numericSensor: Sensor.Numeric,
  valveBody: (data) => {
    const viewPort = useViewport();
    return (
      <ValveBody
        id={data.id}
        dimensions={{
          height: 25 * (viewPort.zoom / window.devicePixelRatio),
          width: 50 * (viewPort.zoom / window.devicePixelRatio),
        }}
        position={{
          x: (data.xPos + viewPort.x) / window.devicePixelRatio,
          y: (data.yPos + viewPort.y) / window.devicePixelRatio,
        }}
        stroke="white"
        fill=""
      />
    );
  },
};

const nodes: Node[] = [
  {
    id: "sensor-1",
    type: "valveBody",
    position: { x: 50, y: 50 },
    data: { label: "Cryo Fill", units: "psi" },
  },
  {
    id: "sensor-2",
    type: "valveBody",
    position: { x: 100, y: 100 },
    data: { label: "Cryo Fill", units: "psi" },
  },
];

export const Numeric: StoryFn<typeof SensorNumeric> = () => (
  <div style={{ width: window.innerWidth, height: window.innerHeight }}>
    <PIDProvider engine="canvas" renderers={{}}>
      <ReactFlow nodeTypes={nodeTypes} nodes={nodes}>
        <Background />

        <Controls />
      </ReactFlow>
    </PIDProvider>
  </div>
);

// eslint-disable-next-line import/no-default-export
export default story;
