import type { ComponentMeta, ComponentStory } from "@storybook/react";

import { HexagonBar } from "./Hexagon";

const story: ComponentMeta<typeof HexagonBar> = {
  title: "Metrics/Hexagon",
  component: HexagonBar,
};

const Template: ComponentStory<typeof HexagonBar> = (args) => <HexagonBar {...args} />;

export const Primary = Template.bind({});
Primary.args = {
  strokeWidth: 5,
  width: "50%",
  metrics: [
    {
      name: "Memory",
      value: 10,
      max: 100,
      units: "GB",
    },
    {
      name: "CPU",
      value: 30,
      max: 100,
      units: "%",
    },
    {
      name: "CPU",
      value: 65,
      max: 100,
      units: "%",
    },
  ],
};

export default story;
