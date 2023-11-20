// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useState } from "react";

import type { Meta, StoryFn } from "@storybook/react";

import { Static } from "@/telem/static";
import { Canvas } from "@/vis/canvas";
import { LinePlot } from "@/vis/lineplot";
import { Tooltip } from "@/vis/lineplot/tooltip/Tooltip";

const story: Meta<typeof LinePlot.LinePlot> = {
  title: "LinePlot",
  component: LinePlot.LinePlot,
};

const LENGTH = 5000;
const DIV = 1000;
const MULT = 1000;

const xData = Float32Array.from({ length: LENGTH }, (_, i) => i);
const yData = Float32Array.from(
  { length: LENGTH },
  (_, i) => Math.sin(i / DIV) * MULT + Math.random(),
);
const xData2 = Float32Array.from({ length: LENGTH }, (_, i) => i + 0.25);
const yData2 = Float32Array.from(
  { length: LENGTH },
  (_, i) => Math.sin(i / DIV) * MULT + Math.random() + 200,
);
const xData3 = Float32Array.from({ length: LENGTH }, (_, i) => i + 0.5);
const yData3 = Float32Array.from(
  { length: LENGTH },
  (_, i) => Math.sin(i / DIV) * MULT + Math.random() + 400,
);
const Example = (): ReactElement => {
  const telem = Static.useStaticXY({
    x: [xData],
    y: [yData],
  });
  const telem2 = Static.useStaticXY({
    x: [xData2],
    y: [yData2],
  });
  const telem3 = Static.useStaticXY({
    x: [xData3],
    y: [yData3],
  });
  const [label, setLabel] = useState("Line");
  const [xLabel, setXLabel] = useState("X");
  return (
    <Canvas.Canvas
      style={{
        width: "100%",
        height: "100%",
        position: "fixed",
        top: 0,
        left: 0,
      }}
    >
      <LinePlot.LinePlot>
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
            location="right"
            showGrid
          >
            <LinePlot.Line x={telem} color="#F733FF" strokeWidth={2} label="Line" />
            <LinePlot.Line
              x={telem2}
              color="#20e530"
              strokeWidth={2}
              label="Line 2"
            />
            <LinePlot.Line
              x={telem3}
              color="#e54420"
              strokeWidth={2}
              label="Line 3"
              downsample={10}
            />
            <LinePlot.Viewport />
          </LinePlot.YAxis>
        </LinePlot.XAxis>
        <LinePlot.Legend />
        <Tooltip />
      </LinePlot.LinePlot>
    </Canvas.Canvas>
  );
};

export const Default: StoryFn = () => <Example />;

// eslint-disable-next-line import/no-default-export
export default story;
