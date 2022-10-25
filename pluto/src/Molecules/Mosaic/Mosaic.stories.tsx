import { ComponentStory, ComponentMeta } from "@storybook/react";
import { Mosaic } from ".";
import { MosaicLeaf } from "./mosaicTree";
import { useMosaic } from "./useMosaic";

export default {
  title: "Molecules/Mosaic",
  component: Mosaic,
};

const initialTree: MosaicLeaf = {
  key: 0,
  level: 0,
  direction: "horizontal",
  first: {
    level: 1,
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
    level: 1,
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

export const Primary: ComponentStory<typeof Mosaic> = (args) => {
  const { insertTab, ...props } = useMosaic({ initialTree });
  return <Mosaic {...props} />;
};
