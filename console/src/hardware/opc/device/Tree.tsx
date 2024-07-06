import { Align, Device, Synnax, TimeSpan, Tree } from "@synnaxlabs/pluto";
import { useMutation, useQuery } from "@tanstack/react-query";
import { nanoid } from "nanoid";
import { useState } from "react";

import {
  type Device as OPCDevice,
  Properties,
  ScannerScanCommandResult,
} from "@/hardware/opc/device/types";

export const Base = () => {
  const client = Synnax.use();
  const [dev, setDev] = useState<null | OPCDevice>(null);
  const [nodes, setNodes] = useState<Tree.Node[]>([]);

  const { data: scanTask } = useQuery({
    queryKey: [client?.key],
    queryFn: async () => {
      if (client == null) return null;
      const rack = await client.hardware.racks.retrieve("sy_node_1_rack");
      return await rack.retrieveTaskByName("opc Scanner");
    },
  });

  const expand = useMutation({
    mutationKey: [client?.key, scanTask?.key],
    mutationFn: async (props: Tree.HandleExpandProps) => {
      if (scanTask?.key == null || props.action === "contract") return;
      const res = await scanTask.executeCommandSync<ScannerScanCommandResult>(
        "scan",
        {
          connection: dev?.properties.connection,
          node_id: props.clicked.split("---")[0],
        },
        TimeSpan.seconds(10),
      );
      console.log(res);
      setNodes([
        ...Tree.setNode({
          tree: [...nodes],
          destination: props.clicked,
          additions: res.details?.channels.map((channel) => {
            return {
              key: `${channel.nodeId}---${props.clicked.split("---")[0]}`, // `---${nanoid()}
              name: channel.name,
              hasChildren: true,
            };
          }),
        }),
      ]);
    },
  });

  const treeProps = Tree.use({
    nodes,
    onExpand: expand.mutate,
  });

  const deviceChange = useMutation({
    mutationKey: [client?.key, scanTask?.key],
    mutationFn: async (device: string) => {
      if (client == null || scanTask == null) return;
      if (device == null) {
        setDev(null);
        setNodes([]);
        return;
      }
      const dev = await client.hardware.devices.retrieve<Properties>(device);
      const res = await scanTask.executeCommandSync<ScannerScanCommandResult>(
        "scan",
        { connection: dev.properties.connection },
        TimeSpan.seconds(10),
      );
      if (res.details?.channels == null) return;
      setNodes(
        res.details.channels.map((channel) => ({
          key: `${channel.nodeId}`, // `---${nanoid()}
          name: channel.name,
          hasChildren: true,
        })),
      );
      setDev(dev);
    },
  });

  return (
    <Align.Space direction="y" grow>
      <Device.SelectSingle value={dev?.key} onChange={deviceChange.mutate} />
      <Tree.Tree {...treeProps} />
    </Align.Space>
  );
};
