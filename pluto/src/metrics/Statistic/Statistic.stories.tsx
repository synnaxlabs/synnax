import type { ComponentMeta, ComponentStory } from "@storybook/react";

import Statistic from "./Statistic";

const story: ComponentMeta<typeof Statistic> = {
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
};

const Template: ComponentStory<typeof Statistic> = (props) => <Statistic {...props} />;

export const Primary = Template.bind({});
Primary.args = {
  value: 12,
  level: "h1",
  label: "Events",
  variant: "primary",
};

export default story;
