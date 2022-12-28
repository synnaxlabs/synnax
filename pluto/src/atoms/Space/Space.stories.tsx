import type { ComponentMeta, ComponentStory } from "@storybook/react";

import { Space, SpaceProps } from ".";

import { Button } from "@/atoms/Button";
import { ComponentSizes } from "@/util/component";

const story: ComponentMeta<typeof Space> = {
  title: "Atoms/Space",
  component: Space,
  argTypes: {
    direction: {
      control: { type: "select" },
      options: ["horizontal", "vertical"],
    },
    size: {
      control: { type: "select" },
      options: ComponentSizes,
    },
    align: {
      control: { type: "select" },
      options: Space.Alignments,
    },
    justify: {
      control: { type: "select" },
      options: Space.Justifications,
    },
  },
};

const Template: ComponentStory<typeof Space> = (args: SpaceProps) => (
  <Space {...args}>
    <Button>Button 1</Button>
    <Button>Button 1</Button>
    <Button>Button 1</Button>
  </Space>
);

export const Basic = Template.bind({});
Basic.args = {
  direction: "horizontal",
  size: "medium",
  align: "center",
  justify: "start",
};

export default story;
