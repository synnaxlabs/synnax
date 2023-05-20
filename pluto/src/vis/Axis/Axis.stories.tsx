// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Meta } from "@storybook/react";
import { TimeStamp, Scale } from "@synnaxlabs/x";

import { Axis } from "./Axis";

const story: Meta<typeof Axis> = {
  title: "Visualization/Axis",
  component: Axis,
};

const start = new TimeStamp([2020, 2, 22]);
const end = start.add(TimeStamp.seconds(120));
const startTwo = new TimeStamp([2020, 2, 22]);
const endTwo = start.add(TimeStamp.milliseconds(120));

export const Basic = (): JSX.Element => (
  <svg width="100%" height="100%" style={{ width: "100%", height: "100%" }}>
    <Axis
      scale={Scale.scale(0, 100)}
      type="linear"
      size={800}
      position={{ x: 50, y: 30 }}
      location="left"
      pixelsPerTick={30}
      showGrid
      height={800}
    />
    <Axis
      scale={Scale.scale(5000000, 10000000)}
      type="linear"
      size={800}
      position={{ x: 20, y: 30 }}
      location="left"
    />
    <Axis
      scale={Scale.scale(0, 0.5)}
      type="linear"
      size={800}
      position={{ x: 850, y: 30 }}
      location="right"
    />
    <Axis
      scale={Scale.scale(start.valueOf(), end.valueOf())}
      type="time"
      size={800}
      position={{ x: 50, y: 830 }}
      location="bottom"
      showGrid
      pixelsPerTick={40}
      height={800}
    />
    <Axis
      scale={Scale.scale(startTwo.valueOf(), endTwo.valueOf())}
      type="time"
      size={800}
      position={{ x: 50, y: 855 }}
      location="bottom"
      pixelsPerTick={40}
      height={800}
    />
  </svg>
);

// eslint-disable-next-line import/no-default-export
export default story;
