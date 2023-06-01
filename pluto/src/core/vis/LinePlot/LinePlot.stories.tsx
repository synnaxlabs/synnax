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

import { VisCanvas } from "../Canvas";

import { LinePlot } from "@/core/vis/LinePlot";
import { Pluto } from "@/Pluto";
import { useStaticTelem } from "@/telem/useStaticTelem";

const story: Meta<typeof LinePlot> = {
  title: "Vis/LinePlot",
  component: LinePlot,
};

const Example = (): ReactElement => {
  const telem = useStaticTelem({
    x: [new Int32Array([1, 2, 3])],
    y: [new Int32Array([1, 2, 3])],
  });
  return (
    <Pluto>
      <VisCanvas
        style={{
          width: "100%",
          height: "100%",
          position: "fixed",
          top: 0,
          left: 0,
        }}
      >
        <LinePlot style={{ height: "50%", padding: 10 }}>
          <LinePlot.XAxis
            type="linear"
            label="Time"
            color="#FFFFFF"
            location="bottom"
            bound={{ lower: 500, upper: 1000 }}
            showGrid
          >
            <LinePlot.YAxis
              type="linear"
              label="Value"
              color="#FFFFFF"
              location="right"
              bound={{ lower: 500, upper: 1000 }}
              showGrid
            />
            <LinePlot.YAxis
              type="linear"
              label="Value"
              color="#FFFFFF"
              location="left"
              bound={{ lower: 500, upper: 1000 }}
            />
            <LinePlot.YAxis
              type="linear"
              label="Value"
              color="#FFFFFF"
              location="left"
              bound={{ lower: 2000, upper: 90000 }}
            />
            <LinePlot.YAxis
              type="linear"
              label="Value"
              color="#FFFFFF"
              location="left"
              bound={{ lower: 500, upper: 1000 }}
            />
          </LinePlot.XAxis>
        </LinePlot>
        <LinePlot style={{ height: "50%", padding: 10 }}>
          <LinePlot.XAxis
            type="linear"
            label="Time"
            color="#FFFFFF"
            location="bottom"
            bound={{ lower: 500, upper: 1000 }}
            showGrid
          >
            <LinePlot.YAxis
              type="linear"
              label="Value"
              color="#FFFFFF"
              location="right"
              bound={{ lower: 500, upper: 1000 }}
              showGrid
            />
            <LinePlot.YAxis
              type="linear"
              label="Value"
              color="#FFFFFF"
              location="left"
              bound={{ lower: 500, upper: 1000 }}
            />
            <LinePlot.YAxis
              type="linear"
              label="Value"
              color="#FFFFFF"
              location="left"
              bound={{ lower: 2000, upper: 90000 }}
            />
            <LinePlot.YAxis
              type="linear"
              label="Value"
              color="#FFFFFF"
              location="left"
              bound={{ lower: 500, upper: 1000 }}
            />
          </LinePlot.XAxis>
        </LinePlot>
      </VisCanvas>
    </Pluto>
  );
};

export const Primary: StoryFn<typeof LinePlot> = () => <Example />;

// eslint-disable-next-line import/no-default-export
export default story;
