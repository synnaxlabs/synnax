// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useState } from "react";

import type { ComponentMeta, ComponentStory } from "@storybook/react";
import { AiFillDatabase, AiFillPhone } from "react-icons/ai";

import { Tree, TreeLeaf } from ".";

const story: ComponentMeta<typeof Tree> = {
  title: "Atoms/Tree",
  component: Tree,
};

const nodes: TreeLeaf[] = [
  {
    key: "cluster",
    title: "Cluster",
    icon: <AiFillDatabase />,
    children: [
      {
        key: "node-1",
        title: "Node 1",
        children: [
          {
            key: "pod-1",
            title: "Pod 1",
            icon: <AiFillDatabase />,
          },
        ],
      },
    ],
  },
  {
    key: "Devices",
    title: "Devices",
    icon: <AiFillPhone />,
    children: [
      {
        key: "device-1",
        title: "Device 1",
      },
    ],
  },
];

export const Primary: ComponentStory<typeof Tree> = () => {
  const [value, setValue] = useState<readonly string[]>([]);
  return <Tree data={nodes} value={value} onChange={setValue} />;
};

// eslint-disable-next-line import/no-default-export
export default story;
