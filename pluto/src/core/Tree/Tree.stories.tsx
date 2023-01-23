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

import { useSelectMultiple } from "@/hooks/useSelectMultiple";

const story: ComponentMeta<typeof Tree> = {
  title: "Core/Tree",
  component: Tree,
};

const nodes: TreeLeaf[] = [
  {
    key: "cluster",
    name: "Cluster",
    icon: <AiFillDatabase />,
    children: [
      {
        key: "node-1",
        name: "Node 1",
        children: [
          {
            key: "pod-1",
            name: "Pod 1",
            icon: <AiFillDatabase />,
          },
        ],
      },
    ],
  },
  {
    key: "Devices",
    name: "Devices",
    icon: <AiFillPhone />,
    children: [
      {
        key: "device-1",
        name: "Device 1",
      },
    ],
  },
];

export const Primary: ComponentStory<typeof Tree> = () => {
  const [value, setValue] = useState<readonly string[]>([]);
  const { onSelect } = useSelectMultiple({
    allowMultiple: false,
    value,
    onChange: setValue,
    data: nodes,
  });
  return <Tree data={nodes} value={value} onChange={onSelect} />;
};

// eslint-disable-next-line import/no-default-export
export default story;
