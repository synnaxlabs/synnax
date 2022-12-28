import type { ComponentMeta, ComponentStory } from "@storybook/react";

import { useStaticTabs } from "./Tabs";

import { Tabs } from ".";

const story: ComponentMeta<typeof Tabs> = {
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

export const Primary: ComponentStory<typeof Tabs> = () => {
  const props = useStaticTabs({ tabs: exampleTabs });
  return (
    <Tabs {...props} closable>
      {({ tab }) => <h2>{tab.tabKey}</h2>}
    </Tabs>
  );
};

export default story;
