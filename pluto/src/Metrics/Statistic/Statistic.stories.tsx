import Statistic from "./Statistic";
import { ComponentMeta, ComponentStory } from "@storybook/react";

export default {
  title: "Metrics/Statistic",
  component: Statistic,
  argTypes: {
    level: {
      control: {
        type: "select",
        options: ["h1", "h2", "h3", "h4", "h5", "p", "small"],
      },
    },
    variant: {
      control: {
        type: "select",
        options: ["primary", "error"],
      },
    },
  },
} as ComponentMeta<typeof Statistic>;

const Template: ComponentStory<typeof Statistic> = (props) => {
  return <Statistic {...props} />;
};

export const Primary = Template.bind({});
Primary.args = {
  value: 12,
  level: "h1",
  label: "Events",
  variant: "primary",
};
