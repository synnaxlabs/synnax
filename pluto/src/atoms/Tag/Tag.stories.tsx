import type { ComponentMeta, ComponentStory } from "@storybook/react";

import { Tag } from ".";

const story: ComponentMeta<typeof Tag> = {
  title: "Atoms/Tag",
  component: Tag,
};

const Template: ComponentStory<typeof Tag> = (props) => <Tag {...props} />;

export const Primary: ComponentStory<typeof Tag> = Template.bind({});
Primary.args = {
  children: "Tag",
  onClose: () => undefined,
  variant: "filled",
  size: "medium",
};

export default story;
