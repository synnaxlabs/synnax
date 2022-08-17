import { HexagonBar } from "./Hexagon";
import { ComponentMeta, ComponentStory } from "@storybook/react";

export default {
  title: "Metrics/Hexagon",
  component: HexagonBar,
} as ComponentMeta<typeof HexagonBar>;

const Template: ComponentStory<typeof HexagonBar> = (args) => (
  <HexagonBar {...args} />
);

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
