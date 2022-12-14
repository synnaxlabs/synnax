// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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
  return <Tabs {...props}>{({ tab }) => <h2>{tab.tabKey}</h2>}</Tabs>;
};

export default story;
