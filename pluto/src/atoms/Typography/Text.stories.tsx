import type { ComponentMeta, ComponentStory } from "@storybook/react";
import { AiFillDatabase, AiOutlineDelete } from "react-icons/ai";

import { Text } from ".";

const story: ComponentMeta<typeof Text> = {
  title: "Atoms/Text",
  component: Text,
};

export const Basic: ComponentStory<typeof Text> = () => <Text level="h2">Hello</Text>;

export const WithIcon: ComponentStory<typeof Text> = () => (
  <Text.WithIcon
    startIcon={<AiOutlineDelete />}
    endIcon={<AiFillDatabase />}
    level="h2"
  />
);

export const Editable: ComponentStory<typeof Text> = () => (
  <Text.Editable level="h1" text="hello"></Text.Editable>
);

export default story;
