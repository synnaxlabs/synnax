import { ComponentMeta, ComponentStory } from "@storybook/react";
import { Nav } from ".";
import { NavBarProps } from "./NavBar";

export default {
  title: "Molecules/Nav",
  component: Nav.Bar,
} as ComponentMeta<typeof Nav.Bar>;

const Template = (args: NavBarProps) => <Nav.Bar {...args} />;

export const Left: ComponentStory<typeof Nav.Bar> = Template.bind({});
