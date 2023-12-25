import { useState, type ReactElement } from "react";

import type { Meta, StoryFn } from "@storybook/react";

import { Align } from "@/align";
import { Pluto } from "@/pluto";

import { SelectOrientation, type OrientationValue } from "./SelectOrientation";
import { FourWayValve, ReliefValve, ThreeWayValve } from "./Symbols";

const story: Meta<typeof SelectOrientation> = {
  title: "OrientationControl",
  component: SelectOrientation,
};

const Example = (): ReactElement => {
  const [value, setValue] = useState<OrientationValue>({
    inner: "top",
    outer: "left",
  });

  return (
    <Align.Space align="start">
      <SelectOrientation value={value} onChange={setValue} />;
      <ReliefValve
        label={{ label: "Label", orientation: value.outer }}
        orientation={value.inner}
        color="#000000"
      />
    </Align.Space>
  );
};

export const Primary: StoryFn<typeof SelectOrientation> = () => <Example />;

// eslint-disable-next-line import/no-default-export
export default story;
