import type { ComponentMeta, ComponentStory } from "@storybook/react";
import { AiOutlineDelete } from "react-icons/ai";

import { Button, ButtonProps } from ".";

const story: ComponentMeta<typeof Button> = {
  title: "Atoms/Button",
  component: Button,
  argTypes: {
    variant: {
      options: ["filled", "outlined"],
      control: { type: "select" },
    },
  },
};

const Template = (args: ButtonProps): JSX.Element => <Button {...args} />;

export const Primary: ComponentStory<typeof Button> = Template.bind({});
Primary.args = {
  size: "medium",
  startIcon: <AiOutlineDelete />,
  children: "Button",
};

export const Outlined = (): JSX.Element => (
  <Button variant="outlined" endIcon={<AiOutlineDelete />}>
    Button
  </Button>
);

export default story;
