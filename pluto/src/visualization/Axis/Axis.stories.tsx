import { ComponentMeta } from "@storybook/react";

import { Axis } from "./Axis";

import { TimeStamp } from "@synnaxlabs/x";

const story: ComponentMeta<typeof Axis> = {
  title: "Visualization/Axis",
  component: Axis,
};

const start = new TimeStamp([2020, 2, 22]);
const end = start.add(TimeStamp.seconds(120));

export const Basic = (): JSX.Element => (
  <svg width="100%" height="100%" style={{ width: "100%", height: "100%" }}>
    <Axis
      range={[0, 1000]}
      type="linear"
      size={800}
      position={{ x: 50, y: 30 }}
      location="left"
    />
    <Axis
      range={[0, 200]}
      type="linear"
      size={800}
      position={{ x: 20, y: 30 }}
      location="left"
    />
    <Axis
      range={[start.valueOf(), end.valueOf()]}
      type="time"
      size={800}
      position={{ x: 60, y: 850 }}
      location="bottom"
    />
  </svg>
);

export default story;
