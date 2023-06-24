// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import type { Meta, StoryFn } from "@storybook/react";
import { Rate } from "@synnaxlabs/x";

import { Canvas } from "../Canvas";

import { Line } from "@/core/vis/Line/Line";
import { LinePlot } from "@/core/vis/LinePlot";
import { Telem } from "@/telem";

const story: Meta<typeof LinePlot> = {
  title: "Core/Vis/LinePlot",
  component: LinePlot,
};

const LENGTH = 50000;
const DIV = 100;

const xData = Float32Array.from({ length: LENGTH }, (_, i) => i);
const yData = Float32Array.from(
  { length: LENGTH },
  (_, i) => Math.sin(i / DIV) * 20 + Math.random()
);
const Example = (): ReactElement => {
  const telem2 = Telem.Static.useIterativeXY({
    x: [xData],
    y: [yData],
    rate: Rate.hz(500),
  });
  return (
    <Canvas
      style={{
        width: "100%",
        height: "100%",
        position: "fixed",
        top: 0,
        left: 0,
      }}
    >
      <LinePlot>
        <LinePlot.XAxis type="linear" label="Time" location="bottom" showGrid>
          <LinePlot.YAxis type="linear" label="Value" location="left" showGrid>
            <Line telem={telem2} color="#F733FF" strokeWidth={2} />
            {/* <Line telem={telem2} color="#fcba03" strokeWidth={2} />
            <Line telem={telem3} color="#3ad6cc" strokeWidth={2} /> */}
          </LinePlot.YAxis>
        </LinePlot.XAxis>
      </LinePlot>
    </Canvas>
  );
};

export const Default: StoryFn = () => <Example />;

// eslint-disable-next-line import/no-default-export
export default story;
