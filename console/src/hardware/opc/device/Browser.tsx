// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/opc/device/Browser.css";

import {
  Button,
  Component,
  Flex,
  Flux,
  Haul,
  Header,
  Icon,
  List,
  Status,
  Text,
  TimeSpan,
  Tree,
} from "@synnaxlabs/pluto";
import { type Optional } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useEffect, useState } from "react";

import { CSS } from "@/css";
import { type Device } from "@/hardware/opc/device/types";
import { SCAN_COMMAND_TYPE, type ScannedNode } from "@/hardware/opc/task/types";

import { useRetrieveScanTask } from "./useRetrieveScanTask";

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
  const [treeNodes, setTreeNodes] = useState<Tree.Node[]>([]);
  const opcNodesStore = List.useMapData<string, ScannedNode>();
  const scanTask = useRetrieveScanTask(device.rack);
  const {
    run: expand,
    variant,
    status: { key, ...status },
  } = Flux.useAsyncOperation(
    "OPC Node",
    "Retrieve",
    async ({
      action,
      delay,
      clicked,
    }: Optional<Tree.HandleExpandProps, "clicked"> & { delay?: number }) => {
      if (scanTask?.key == null || action === "contract") return;
      if (delay != null) await new Promise((resolve) => setTimeout(resolve, delay));
      const isRoot = clicked == null;
      const nodeID = isRoot ? "" : parseNodeID(clicked);
      const { connection } = device.properties;
      const { details } = await scanTask.executeCommandSync(
        SCAN_COMMAND_TYPE,
        TimeSpan.seconds(10),
        { connection, node_id: nodeID },
      );
      if (details?.data == null) return;
      if (!("channels" in details.data)) return;
      const channels = details.data.channels;
      const newNodes = channels.map(
        (node): Tree.Node => ({ key: nodeKey(node.nodeId, nodeID), children: [] }),
      );
      opcNodesStore.setItem(
        channels.map((node) => ({ ...node, key: nodeKey(node.nodeId, nodeID) })),
      );
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
  );
  const treeProps = Tree.use({ nodes: treeNodes, onExpand: expand });
  const { shape, clearExpanded } = treeProps;
  const [initialLoading, setInitialLoading] = useState(false);
  const refresh = useCallback(() => {
    if (scanTask == null) return;
    setInitialLoading(true);
    expand({ action: "expand", current: [], delay: 200 });
    clearExpanded();
  }, [scanTask, clearExpanded]);
  useEffect(refresh, [refresh]);
  let content: ReactElement;
  if (initialLoading)
    content = (
      <Flex.Box center>
        <Icon.Loading style={{ fontSize: "5rem" }} color="var(--pluto-gray-l7)" />
      </Flex.Box>
    );
  else if (variant === "error") content = <Status.Summary center {...status} />;
  else
    content = (
      <Tree.Tree
        {...treeProps}
        shape={shape}
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
