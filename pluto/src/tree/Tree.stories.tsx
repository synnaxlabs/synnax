// // Copyright 2023 Synnax Labs, Inc.
// //
// // Use of this software is governed by the Business Source License included in the file
// // licenses/BSL.txt.
// //
// // As of the Change Date specified in that file, in accordance with the Business Source
// // License, use of this software will be governed by the Apache License, Version 2.0,
// // included in the file licenses/APL.txt.

// import { useState } from "react";

// import type { Meta, StoryFn } from "@storybook/react";
// import { AiFillDatabase, AiFillPhone } from "react-icons/ai";

// import { Haul } from "@/haul";
// import { Tree } from "@/tree";
// import { Triggers } from "@/triggers";

// const story: Meta<typeof Tree.Tree> = {
//   title: "Tree",
//   component: Tree.Tree,
// };

// const NODES: Tree.Node[] = [
//   {
//     key: "cluster",
//     name: "Cluster",
//     icon: <AiFillDatabase />,
//     children: [
//       {
//         key: "node-1",
//         name: "Node 1",
//         children: [
//           {
//             key: "pod-1",
//             name: "Pod 1",
//             icon: <AiFillDatabase />,
//           },
//         ],
//       },
//     ],
//   },
//   {
//     key: "Devices",
//     name: "Devices",
//     icon: <AiFillPhone />,
//     children: [
//       {
//         key: "device-1",
//         name: "Device 1",
//       },
//     ],
//   },
//   {
//     key: "device-2",
//     name: "Device 2",
//     icon: <AiFillPhone />,
//     children: [
//       {
//         key: "device-3",
//         name: "Device 3",
//       },
//     ],
//   },
// ];

// const Component = () => {
//   const props = Tree.use();
//   const [nodes, setNodes] = useState(NODES);
//   const handleDrop = (key: string, items: Haul.Item[]): void => {
//     setNodes([...Tree.moveNode(nodes, key, ...items.map((item) => item.key))]);
//   };
//   return <Tree.Tree nodes={nodes} {...props} onDrop={handleDrop} />;
// };

// export const Primary: StoryFn<typeof Tree> = () => {
//   return (
//     <Haul.Provider>
//       <Triggers.Provider>
//         <Component />
//       </Triggers.Provider>
//     </Haul.Provider>
//   );
// };

// // eslint-disable-next-line import/no-default-export
// export default story;
