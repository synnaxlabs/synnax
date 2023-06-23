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
import { Rate, TimeRange, TimeSpan } from "@synnaxlabs/x";

import { VisCanvas } from "@/core/vis/Canvas";
import { Line } from "@/core/vis/Line/Line";
import { LinePlot } from "@/core/vis/LinePlot";
import { RangeTelem } from "@/telem/range/main";
import { StaticTelem } from "@/telem/static/main";

const story: Meta<typeof LinePlot> = {
  title: "Vis/LinePlot",
  component: LinePlot,
};

const LENGTH = 5000;
const DIV = 100;

const xData = Float32Array.from({ length: LENGTH }, (_, i) => i);
const yData = Float32Array.from(
  { length: LENGTH },
  (_, i) => Math.sin(i / DIV) * 20 + Math.random()
);
const yData2 = Float32Array.from(
  { length: LENGTH },
  (_, i) => Math.sin(i / DIV) * 20 - 2 + Math.random()
);
const yData3 = Float32Array.from(
  { length: LENGTH },
  (_, i) => Math.sin(i / DIV) * 20 - 4 + Math.random()
);
const xData2 = Float32Array.from({ length: LENGTH }, (_, i) => i);
const xData3 = Float32Array.from({ length: LENGTH }, (_, i) => i);

const Example = (): ReactElement => {
  const telem = RangeTelem.useDynamicXY({
    x: 65537,
    y: 65538,
    span: TimeSpan.minutes(15),
  });
  // const telem2 = StaticTelem.useIterativeXY
  //   x: [xData2],
  //   y: [yData2],
  //   rate: Rate.hz(25),
  // });
  // const telem3 = StaticTelem.useIterativeXY({
  //   x: [xData3],
  //   y: [yData3],
  //   rate: Rate.hz(30),
  // });
  return (
    <VisCanvas
      style={{
        width: "100%",
        height: "100%",
        position: "fixed",
        top: 0,
        left: 0,
      }}
    >
      <LinePlot>
        <LinePlot.XAxis type="time" label="Time" location="bottom" showGrid>
          <LinePlot.YAxis type="linear" label="Value" location="left" showGrid>
            <Line telem={telem} color="#F733FF" strokeWidth={2} />
            {/* <Line telem={telem2} color="#fcba03" strokeWidth={2} />
            <Line telem={telem3} color="#3ad6cc" strokeWidth={2} /> */}
          </LinePlot.YAxis>
        </LinePlot.XAxis>
      </LinePlot>
    </VisCanvas>
  );
};


// eslint-disable-next-line import/no-default-export
export default story;
