import { ComponentMeta, ComponentStory } from "@storybook/react";
import { AiOutlineDelete } from "react-icons/ai";
import Button, { ButtonProps } from "./Button";

export default {
  title: "Atoms/Button",
  component: Button,
  argTypes: {
    variant: {
      options: ["filled", "outlined"],
      control: { type: "select" },
    },
  },
} as ComponentMeta<typeof Button>;

const Template = (args: ButtonProps) => <Button {...args} />;

export const Primary: ComponentStory<typeof Button> = Template.bind({});
Primary.args = {
  size: "medium",
  startIcon: <AiOutlineDelete />,
  children: "Button",
};

export const Outlined = () => (
  <Button variant="outlined" endIcon={<AiOutlineDelete />}>
    Button
  </Button>
);
