// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Button,
  Header,
  Icon as PIcon,
  Status,
  Synnax,
  Text,
  TimeSpan,
  Tree,
} from "@synnaxlabs/pluto";
import { type Optional } from "@synnaxlabs/x";
import { useMutation, useQuery } from "@tanstack/react-query";
import { type ReactElement, useCallback, useEffect, useState } from "react";

import { CSS } from "@/css";
import {
  type Device as OPCDevice,
  type ScannerScanCommandResult,
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

  const [loading, setLoading] = useState<string | undefined>(undefined);

  const expand = useMutation({
    mutationFn: async (
      props: Optional<Tree.HandleExpandProps, "clicked"> & { delay?: number },
    ) => {
      if (scanTask?.key == null || props.action === "contract" || device == null)
        return;
      if (props.delay != null)
        await new Promise((resolve) => setTimeout(resolve, props.delay));
      const isRoot = props.clicked == null;
      const nodeID = isRoot ? "" : parseNodeID(props.clicked as string);
      const { connection } = device.properties;
      setLoading(props.clicked as string);
      const res = await scanTask.executeCommandSync<ScannerScanCommandResult>(
        "scan",
        { connection, node_id: nodeID },
        TimeSpan.seconds(10),
      );
      if (res.details == null) return;
      const { channels } = res.details;
      const newNodes = channels.map((node) => ({
        key: nodeKey(node.nodeId, nodeID),
        name: node.name,
        icon: node.isArray ? (
          <PIcon.Icon bottomRight={<Icon.Array />}>
            <Icon.Variable />
          </PIcon.Icon>
        ) : (
          ICONS[node.nodeClass]
        ),
        hasChildren: true,
        haulItems: [{ key: node.nodeId, type: "opc", data: node }],
      }));
      setLoading(undefined);
      setInitialLoading(false);
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

  const treeProps = Tree.use({ nodes, onExpand: expand.mutate });

  const [initialLoading, setInitialLoading] = useState(false);

  const refresh = useCallback(() => {
    if (device == null || scanTask == null) return;
    setInitialLoading(true);
    expand.mutate({ action: "expand", current: [], delay: 200 });
    treeProps.clearExpanded();
  }, [device, scanTask?.key, treeProps.clearExpanded]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  let content: ReactElement;
  if (initialLoading)
    content = (
      <Align.Center style={{ width: "100%", height: "100%" }}>
        <Icon.Loading style={{ fontSize: "5rem" }} color="var(--pluto-gray-l5)" />
      </Align.Center>
    );
  else if (expand.isError)
    content = (
      <Align.Center style={{ width: "100%", height: "100%" }}>
        <Status.Text variant="error" shade={6} level="p">
          Error loading nodes. {expand.error?.message}
        </Status.Text>
      </Align.Center>
    );
  else
    content = (
      <Tree.Tree
        loading={loading}
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
    );

  return (
    <Align.Space
      className={CSS.B("browser")}
      direction="y"
      grow
      bordered
      rounded
      style={{
        overflow: "hidden",
        height: "100%",
      }}
      empty
      background={1}
    >
      <Header.Header level="h4">
        <Header.Title weight={500}>Browser</Header.Title>
        <Header.Actions>
          <Button.Icon
            onClick={refresh}
            disabled={device == null || scanTask == null || initialLoading}
          >
            <Icon.Refresh style={{ color: "var(--pluto-gray-l9)" }} />
          </Button.Icon>
        </Header.Actions>
      </Header.Header>
      <Align.Space direction="y" grow style={{ height: "100%", overflow: "hidden" }}>
        {content}
      </Align.Space>
    </Align.Space>
  );
};
