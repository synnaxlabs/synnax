import { ComponentStory } from "@storybook/react";
import { Tabs } from ".";
import { useStaticTabs } from "./Tabs";

export default {
  title: "Atoms/Tabs",
  component: Tabs,
};

const exampleTabs = [
  {
    tabKey: "tab1",
    title: "Tab 1",
    content: <h1>Tab 1 contents</h1>,
  },
  {
    tabKey: "tab2",
    title: "Tab 2",
    content: <h1>Tab 2 contents</h1>,
  },
];

export const Primary: ComponentStory<typeof Tabs> = (args) => {
  const props = useStaticTabs({ tabs: exampleTabs });
  return <Tabs {...props}>{({ tab }) => <h2>{tab.tabKey}</h2>}</Tabs>;
};
