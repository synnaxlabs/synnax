// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Meta, StoryFn } from "@storybook/react";

import { Mosaic } from "@/mosaic";

const story: Meta<typeof Mosaic.Mosaic> = {
  title: "Mosaic",
  component: Mosaic.Mosaic,
};

const initialTree: Mosaic.Node = {
  key: 1,
  direction: "x",
  first: {
    key: 2,
    tabs: [
      {
        tabKey: "1",
        name: "Tab 1",
        content: <h1>Tab One Content</h1>,
      },
    ],
  },
  last: {
    key: 3,
    tabs: [
      {
        tabKey: "2",
        name: "Tab 2",
        content: <h1>Tab Two Content</h1>,
      },
      {
        tabKey: "3",
        name: "Tab 3",
        content: <h1>Tab Three Content</h1>,
      },
    ],
  },
};

export const Primary: StoryFn<typeof Mosaic> = () => {
  const props = Mosaic.use({ initialTree, allowRename: true });
  return <Mosaic.Mosaic {...props}>{() => <h1>Dog</h1>}</Mosaic.Mosaic>;
};

// eslint-disable-next-line import/no-default-export
export default story;
