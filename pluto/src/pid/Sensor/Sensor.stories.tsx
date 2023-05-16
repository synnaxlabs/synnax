// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { ComponentMeta, ComponentStory } from "@storybook/react"
import { Background, Controls, Node, ReactFlow } from "reactflow"

import { Sensor } from "."
import { SensorNumeric } from "./SensorNumeric"

const story: ComponentMeta<typeof Sensor.Numeric> = {
  title: "PID/Sensor",
  component: Sensor.Numeric,
}

const nodeTypes = {
  numericSensor: Sensor.Numeric,
}

const nodes: Node[] = [
  {
    id: 'sensor-1',
    type: "numericSensor",
    position: { x: 0, y: 0 },
    data: { label: "Cryo Fill", units: "psi" }
  }
]

export const Numeric: ComponentStory<typeof SensorNumeric> = () => (
  <div style={{ width: "100%", height: "100%" }}>
    <ReactFlow nodeTypes={nodeTypes} nodes={nodes}>
      <Background />
      <Controls />
    </ReactFlow>
  </div>
)

// eslint-disable-next-line import/no-default-export
export default story
