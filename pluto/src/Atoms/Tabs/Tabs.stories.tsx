import { ComponentMeta, ComponentStory } from "@storybook/react";
import { Tabs } from ".";
import { Resize } from "../Resize";

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

export const Primary: ComponentStory<typeof Tabs> = (args) => (
  <Resize.Multiple direction="horizontal">
    <Tabs tabs={exampleTabs} selected="tab2">
      {({ tab }) => <h2>{tab.tabKey}</h2>}
    </Tabs>
    <Tabs tabs={exampleTabs} selected="tab2">
      {({ tab }) => <h2>{tab.tabKey}</h2>}
    </Tabs>
  </Resize.Multiple>
);
