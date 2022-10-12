import { ComponentMeta, ComponentStory } from "@storybook/react";
import Resize from "./Resize";

export default {
  title: "Atoms/Resize",
  component: Resize,
} as ComponentMeta<typeof Resize>;

const Template = (args: any) => <Resize {...args}>Resize</Resize>;

export const Primary: ComponentStory<typeof Resize> = Template.bind({});
Primary.args = {
  style: {
    height: "100%",
  },
};
