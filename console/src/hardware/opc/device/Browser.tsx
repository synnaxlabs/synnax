import { Icon } from "@synnaxlabs/media";
import { Align, Synnax, Text, TimeSpan, Tree } from "@synnaxlabs/pluto";
import { Optional } from "@synnaxlabs/x";
import { useMutation, useQuery } from "@tanstack/react-query";
import { ReactElement, useEffect, useState } from "react";

import { CSS } from "@/css";
import {
  Device as OPCDevice,
  ScannerScanCommandResult,
} from "@/hardware/opc/device/types";

const ICONS: Record<string, ReactElement> = {
  VariableType: <Icon.Type />,
  Variable: <Icon.Variable />,
  ObjectType: <Icon.Type />,
  Object: <Icon.Group />,
};

export interface BrowserProps {
  device?: OPCDevice;
}

const nodeKey = (nodeId: string, parentId: string): string => `${nodeId}---${parentId}`;
const parseNodeID = (key: string): string => key.split("---")[0];

export const Browser = ({ device }: BrowserProps): ReactElement => {
  const client = Synnax.use();
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
    mutationKey: [client?.key, scanTask?.key, device?.key],
    mutationFn: async (props: Optional<Tree.HandleExpandProps, "clicked">) => {
      if (scanTask?.key == null || props.action === "contract" || device == null)
        return;
      const isRoot = props.clicked == null;
      const nodeID = isRoot ? "" : parseNodeID(props.clicked as string);
      const { connection } = device.properties;
      const res = await scanTask.executeCommandSync<ScannerScanCommandResult>(
        "scan",
        { connection, node_id: nodeID },
        TimeSpan.seconds(10),
      );
      console.log(res);
      if (res.details == null) return;
      const { channels } = res.details;
      const newNodes = channels.map((channel) => ({
        key: nodeKey(channel.nodeId, nodeID),
        name: channel.name,
        icon: ICONS[channel.nodeClass],
        hasChildren: true,
        haulItems: [
          {
            key: channel.nodeId,
            type: "opc",
            data: channel,
          },
        ],
      }));
      if (isRoot) setNodes(newNodes);
      else
        setNodes([
          ...Tree.setNode({
            tree: [...nodes],
            destination: props.clicked as string,
            additions: newNodes,
          }),
        ]);
    },
  });

  useEffect(() => {
    if (device == null || scanTask == null) return;
    expand.mutate({ action: "expand", current: [] });
  }, [device, scanTask?.key]);

  const treeProps = Tree.use({ nodes, onExpand: expand.mutate });

  return (
    <Align.Space direction="y" grow style={{ height: "100%", overflow: "hidden" }}>
      <Tree.Tree
        emptyContent={
          <Align.Center>
            <Text.Text shade={6} level="p" style={{ maxWidth: 215 }}>
              {device == null
                ? "No OPC UA Server Selected. Select a server to browse nodes."
                : "No nodes found in OPC UA server."}
            </Text.Text>
          </Align.Center>
        }
        {...treeProps}
      />
    </Align.Space>
  );
};
