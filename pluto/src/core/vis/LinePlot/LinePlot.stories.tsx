// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useState } from "react";

import type { Meta, StoryFn } from "@storybook/react";

import { Canvas } from "../Canvas";

import { LinePlot } from "@/core/vis/LinePlot";
import { Telem } from "@/telem";

const story: Meta<typeof LinePlot> = {
  title: "Core/Vis/LinePlot",
  component: LinePlot,
};

const LENGTH = 5000;
const DIV = 1000;
const MULT = 1000;

const xData = Float32Array.from({ length: LENGTH }, (_, i) => i);
const yData = Float32Array.from(
  { length: LENGTH },
  (_, i) => Math.sin(i / DIV) * MULT + Math.random()
);
const xData2 = Float32Array.from({ length: LENGTH }, (_, i) => i);
const yData2 = Float32Array.from(
  { length: LENGTH },
  (_, i) => Math.sin(i / DIV) * MULT + Math.random() + 2
);
const xData3 = Float32Array.from({ length: LENGTH }, (_, i) => i);
const yData3 = Float32Array.from(
  { length: LENGTH },
  (_, i) => Math.sin(i / DIV) * MULT + Math.random() + 4
);
const Example = (): ReactElement => {
  const telem = Telem.Static.useXY({
    x: [xData],
    y: [yData],
  });
  const telem2 = Telem.Static.useXY({
    x: [xData2],
    y: [yData2],
  });
  const telem3 = Telem.Static.useXY({
    x: [xData3],
    y: [yData3],
  });
  const [label, setLabel] = useState("Line");
  const [xLabel, setXLabel] = useState("X");
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
      <LinePlot style={{ padding: "1rem" }}>
        <LinePlot.XAxis
          type="linear"
          label={xLabel}
          location="bottom"
          showGrid
          onLabelChange={setXLabel}
        >
          <LinePlot.YAxis
            type="linear"
            label={label}
            onLabelChange={setLabel}
            location="left"
            showGrid
          >
            <LinePlot.Line telem={telem} color="#F733FF" strokeWidth={2} label="Line" />
            <LinePlot.Line
              telem={telem2}
              color="#20e530"
              strokeWidth={2}
              label="Line 2"
            />
            <LinePlot.Line
              telem={telem3}
              color="#e54420"
              strokeWidth={2}
              label="Line 3"
              downsample={10}
            />
          </LinePlot.YAxis>
        </LinePlot.XAxis>
        <LinePlot.Legend />
      </LinePlot>
    </Canvas>
  );
};

export const Default: StoryFn = () => <Example />;

// eslint-disable-next-line import/no-default-export
export default story;
