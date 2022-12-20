import type { ComponentMeta, ComponentStory } from "@storybook/react";

import { MosaicLeaf } from "./types";
import { useMosaic } from "./useMosaic";

import { Mosaic } from ".";

const story: ComponentMeta<typeof Mosaic> = {
  title: "Molecules/Mosaic",
  component: Mosaic,
};

const initialTree: MosaicLeaf = {
  key: 0,
  direction: "horizontal",
  first: {
    key: 1,
    tabs: [
      {
        tabKey: "1",
        title: "Tab 1",
        content: <h1>Tab One Content</h1>,
      },
    ],
  },
  last: {
    key: 2,
    tabs: [
      {
        tabKey: "2",
        title: "Tab 2",
        content: <h1>Tab Two Content</h1>,
      },
      {
        tabKey: "3",
        title: "Tab 3",
        content: <h1>Tab Three Content</h1>,
      },
    ],
  },
};

export const Primary: ComponentStory<typeof Mosaic> = () => {
  const props = useMosaic({ initialTree, editableTitle: true });
  return <Mosaic {...props} />;
};

export default story;
