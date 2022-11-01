import { ComponentMeta, ComponentStory } from "@storybook/react";
import { AiFillDatabase, AiFillPhone } from "react-icons/ai";
import { Tree, TreeLeaf } from ".";

export default {
  title: "Atoms/Tree",
  component: Tree,
} as ComponentMeta<typeof Tree>;

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

export const Primary: ComponentStory<typeof Tree> = () => <Tree data={nodes} />;
