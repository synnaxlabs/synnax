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

export const Multiple: ComponentStory<typeof Resize.Multiple> = () => {
  return (
    <Resize.Multiple
      initialSizes={[100, 200]}
      style={{ border: "1px solid red" }}
    >
      <h1>Hello From One</h1>
      <h1>Hello From Two</h1>
      <h1>Hello From Three</h1>
    </Resize.Multiple>
  );
};
