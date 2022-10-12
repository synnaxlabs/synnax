import { ComponentMeta, ComponentStory } from "@storybook/react";
import { ComponentSizes } from "../../util/types";
import Button from "../Button/Button";
import { Space, SpaceAlignments, SpaceJustifications, SpaceProps } from ".";

export default {
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
      options: SpaceAlignments,
    },
    justify: {
      control: { type: "select" },
      options: SpaceJustifications,
    },
  },
} as ComponentMeta<typeof Space>;

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
