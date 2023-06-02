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

import { VisCanvas } from "@/core/vis/Canvas";
import { Line } from "@/core/vis/Line/LineC";
import { LinePlot } from "@/core/vis/LinePlot";
import { Pluto } from "@/Pluto";
import { useDynamicTelem } from "@/telem/useStaticTelem";

const story: Meta<typeof LinePlot> = {
  title: "Vis/LinePlot",
  component: LinePlot,
};

const Example = (): ReactElement => {
  const telem = useDynamicTelem({
    x: [Float32Array.from({ length: 1000 }, (_, i) => i)],
    y: [
      Float32Array.from(
        { length: 1000 },
        (_, i) => Math.sin(i / 100) * 20 + Math.random()
      ),
    ],
    updateRate: 30,
  });
  const telem2 = useDynamicTelem({
    x: [Float32Array.from({ length: 1000 }, (_, i) => i)],
    y: [
      Float32Array.from(
        { length: 1000 },
        (_, i) => Math.cos(i / 100) * 20 + Math.random()
      ),
    ],
    updateRate: 30,
  });

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
      <LinePlot style={{ padding: 10 }}>
        <LinePlot.XAxis type="linear" label="Time" location="bottom" showGrid>
          <LinePlot.YAxis type="linear" label="Value" location="left" showGrid>
            <Line telem={telem} color="#F733FF" strokeWidth={10} />
            <Line telem={telem2} color="#D5E04E" strokeWidth={10} />
          </LinePlot.YAxis>
        </LinePlot.XAxis>
      </LinePlot>
    </VisCanvas>
  );
};

export const Primary: StoryFn<typeof LinePlot> = () => (
  <Pluto>
    <Example />
  </Pluto>
);

// eslint-disable-next-line import/no-default-export
export default story;
