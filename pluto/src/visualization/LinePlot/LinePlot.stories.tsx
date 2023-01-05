// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { ComponentMeta, ComponentStory } from "@storybook/react";

import { LinePlot } from "./LinePlot";
import { Axis, Series } from "./types";

import { Autosize } from "@/core/Autosize";

const story: ComponentMeta<typeof LinePlot> = {
  title: "Visualization/Line Plot",
  component: LinePlot,
};

const basicAxes: Axis[] = [
  {
    key: "x",
    label: "X",
    location: "bottom",
  },
  {
    key: "y",
    label: "Y",
    location: "left",
  },
  {
    key: "y2",
    label: "Y2",
    location: "right",
  },
];

const basicData = {
  a: Array.from({ length: 1000 }, (_, i) => i),
  b: Array.from({ length: 1000 }, (_, i) => Math.sin(i / 20)),
  c: Array.from({ length: 1000 }, (_, i) => Math.sin(i / 21)),
  d: Array.from({ length: 1000 }, (_, i) => Math.sin(i / 20) * 2),
};

const series: Series[] = [
  {
    x: "a",
    y: "b",
    label: "Series 1",
    axis: "y",
  },
  {
    x: "a",
    y: "c",
    label: "Series 2",
    axis: "y2",
  },
  {
    x: "a",
    y: "d",
    label: "Series 3",
    axis: "y",
  },
];

export const Basic: ComponentStory<typeof LinePlot> = () => (
  <Autosize style={{ width: "100%", height: "100%" }} debounce={1}>
    {({ width, height }) => (
      <LinePlot
        width={width}
        height={height}
        axes={basicAxes}
        data={basicData}
        series={series}
      />
    )}
  </Autosize>
);

// eslint-disable-next-line import/no-default-export
export default story;
