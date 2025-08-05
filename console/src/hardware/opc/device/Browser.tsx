// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/opc/device/Browser.css";

import { UnexpectedError } from "@synnaxlabs/client";
import {
  Button,
  Component,
  Flex,
  Haul,
  Header,
  Icon,
  List,
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
import { type Device } from "@/hardware/opc/device/types";
import {
  SCAN_COMMAND_TYPE,
  SCAN_SCHEMAS,
  SCAN_TYPE,
  type scanConfigZ,
  type ScannedNode,
  type scanStatusDataZ,
  type scanTypeZ,
} from "@/hardware/opc/task/types";

const ICONS: Record<string, ReactElement> = {
  VariableType: <Icon.Type />,
  Variable: <Icon.Variable />,
  ObjectType: <Icon.Type />,
  Object: <Icon.Group />,
};

const nodeKey = (nodeId: string, parentId: string): string => `${nodeId}---${parentId}`;
const parseNodeID = (key: string): string => key.split("---")[0];

export const HAUL_TYPE = "opc";

export interface BrowserProps {
  device: Device;
}

const ArrayVariableIcon = Icon.createComposite(Icon.Variable, {
  bottomRight: Icon.Array,
});

const itemRenderProp = Component.renderProp((props: Tree.ItemRenderProps<string>) => {
  const node = List.useItem<string, ScannedNode>(props.itemKey);
  const { startDrag } = Haul.useDrag({
    type: HAUL_TYPE,
    key: node?.nodeId,
    data: node,
  });
  const handleDragStart = useCallback(() => {
    if (node == null) return;
    startDrag([{ key: node.nodeId, type: HAUL_TYPE, data: node }]);
  }, [startDrag, node]);
  if (node == null) return null;
  const icon = node.isArray ? <ArrayVariableIcon /> : ICONS[node.nodeClass];
  return (
    <Tree.Item {...props} hasChildren draggable onDragStart={handleDragStart}>
      <Text.Text color={10} gap="small">
        {icon}
        {node.name}
      </Text.Text>
    </Tree.Item>
  );
});

export const Browser = ({ device }: BrowserProps) => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const [treeNodes, setTreeNodes] = useState<Tree.Node[]>([]);
  const opcNodesStore = List.useMapData<string, ScannedNode>();
  const { data: scanTask } = useQuery({
    queryKey: [client?.key],
    queryFn: async () => {
      if (client == null) return null;
      const scanTasks = await client.hardware.tasks.retrieve<
        typeof scanTypeZ,
        typeof scanConfigZ,
        typeof scanStatusDataZ
      >({
        types: [SCAN_TYPE],
        rack: device.rack,
        schemas: SCAN_SCHEMAS,
      });
      if (scanTasks.length === 0)
        throw new UnexpectedError(`No scan task found for device ${device.name}`);
      return scanTasks[0];
    },
  });
  const [, setLoading] = useState<string>();
  const expand = useMutation({
    mutationFn: async ({
      action,
      delay,
      clicked,
    }: Optional<Tree.HandleExpandProps, "clicked"> & { delay?: number }) => {
      if (scanTask?.key == null || action === "contract") return;
      if (delay != null) await new Promise((resolve) => setTimeout(resolve, delay));
      const isRoot = clicked == null;
      const nodeID = isRoot ? "" : parseNodeID(clicked);
      const { connection } = device.properties;
      setLoading(clicked);
      const { details } = await scanTask.executeCommandSync(
        SCAN_COMMAND_TYPE,
        TimeSpan.seconds(10),
        { connection, node_id: nodeID },
      );
      if (details?.data == null) return;
      if (!("channels" in details.data)) return;
      const channels = details.data.channels;
      const newNodes = channels.map(
        (node): Tree.Node => ({
          key: nodeKey(node.nodeId, nodeID),
          children: [],
        }),
      );
      opcNodesStore.setItem(
        channels.map((node) => ({ ...node, key: nodeKey(node.nodeId, nodeID) })),
      );
      setLoading(undefined);
      setInitialLoading(false);
      if (isRoot) setTreeNodes(newNodes);
      else
        setTreeNodes([
          ...Tree.setNode({
            tree: [...treeNodes],
            destination: clicked,
            additions: newNodes,
          }),
        ]);
    },
    onError: (error) => handleError(error, "Error loading nodes"),
  });
  const treeProps = Tree.use({
    nodes: treeNodes,
    onExpand: expand.mutate,
  });
  const { clearExpanded } = treeProps;
  const [initialLoading, setInitialLoading] = useState(false);
  const refresh = useCallback(() => {
    if (scanTask == null) return;
    setInitialLoading(true);
    expand.mutate({ action: "expand", current: [], delay: 200 });
    clearExpanded();
  }, [scanTask, clearExpanded]);
  useEffect(refresh, [refresh]);
  const content = initialLoading ? (
    <Flex.Box center>
      <Icon.Loading style={{ fontSize: "5rem" }} color="var(--pluto-gray-l7)" />
    </Flex.Box>
  ) : expand.isError ? (
    <Status.Text center color={10} variant="error">
      Error loading nodes. {expand.error.message}
    </Status.Text>
  ) : (
    <Tree.Tree<string, ScannedNode>
      {...treeProps}
      getItem={opcNodesStore.getItem}
      subscribe={opcNodesStore.subscribe}
    >
      {itemRenderProp}
    </Tree.Tree>
  );
  return (
    <Flex.Box empty className={CSS.B("opc-browser")}>
      <Header.Header>
        <Header.Title weight={500} color={10}>
          Browser
        </Header.Title>
        <Header.Actions>
          <Button.Button
            onClick={refresh}
            disabled={scanTask == null || initialLoading}
            sharp
            contrast={2}
            variant="text"
          >
            <Icon.Refresh />
          </Button.Button>
        </Header.Actions>
      </Header.Header>
      {content}
    </Flex.Box>
  );
};
