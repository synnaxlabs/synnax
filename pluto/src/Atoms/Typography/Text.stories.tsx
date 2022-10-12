import { ComponentMeta, ComponentStory } from "@storybook/react";
import { AiFillDatabase, AiOutlineDelete } from "react-icons/ai";
import { Text } from ".";

export default {
  title: "Atoms/Text",
  component: Text,
} as ComponentMeta<typeof Text>;

export const Basic = () => <Text level="h2">Hello</Text>;

export const WithIcon = () => (
  <Text.WithIcon
    startIcon={<AiOutlineDelete />}
    endIcon={<AiFillDatabase />}
    level="h2"
  />
);
