import { ComponentMeta } from "@storybook/react";
import { TimeSpan, TimeStamp } from "@synnaxlabs/x";

import { Axis } from "./Axis";

import { Scale } from "@/spatial";

const story: ComponentMeta<typeof Axis> = {
  title: "Visualization/Axis",
  component: Axis,
};

const start = new TimeStamp([2020, 2, 22]);
const end = start.add(TimeStamp.seconds(120));

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
      scale={Scale.scale(0, 200)}
      type="linear"
      size={800}
      position={{ x: 20, y: 30 }}
      location="left"
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
  </svg>
);

// eslint-disable-next-line import/no-default-export
export default story;
