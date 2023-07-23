// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Meta, StoryFn } from "@storybook/react";

import { useStaticTabs } from "./Tabs";

import { Tabs } from ".";

const story: Meta<typeof Tabs> = {
  title: "Core/Standard/Tabs",
  component: Tabs,
};

const exampleTabs = [
  {
    tabKey: "tab1",
    name: "Tab 1",
    content: <h1>Tab 1 contents</h1>,
  },
  {
    tabKey: "tab2",
    name: "Tab 2",
    content: <h1>Tab 2 contents</h1>,
  },
];

export const Primary: StoryFn<typeof Tabs> = () => {
  const props = useStaticTabs({ tabs: exampleTabs });
  return (
    <Tabs {...props} size="small" closable>
      {(tab) => <h2>{tab.tabKey}</h2>}
    </Tabs>
  );
};

// eslint-disable-next-line import/no-default-export
export default story;
