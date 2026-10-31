// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/opcua/device/Browser.css";

import { type rack, status, type Synnax as Client } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Haul } from "@synnaxlabs/lyra/haul";
import { Header } from "@synnaxlabs/lyra/header";
import { useCombinedStateAndRef, useSyncedRef } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Tree } from "@synnaxlabs/lyra/tree";
import { Synnax, TimeSpan } from "@synnaxlabs/pluto";
import { array, type optional } from "@synnaxlabs/x";
import { type ReactElement, useEffect, useState } from "react";

import { retrieveScanTask } from "@/feature/opcua/device/retrieveScanTask";
import { type ConnectionConfig, type Device } from "@/feature/opcua/device/types";
import {
  BROWSE_COMMAND_TYPE,
  nodeIdentity,
  type ScannedNode,
} from "@/feature/opcua/task/types";
import { CSS } from "@/platform/css";

const ICONS: Record<string, ReactElement> = {
  VariableType: <Icon.Type />,
  Variable: <Icon.Variable />,
  ObjectType: <Icon.Type />,
  Object: <Icon.Group />,
};

// A node can sit under more than one parent, so its key names the parent too.
const nodeKey = (node: ScannedNode, parent?: ScannedNode): string =>
  `${nodeIdentity(node)}---${parent == null ? "" : nodeIdentity(parent)}`;

export const HAUL_TYPE = "opc";

export type HaulItem = Haul.Item<typeof HAUL_TYPE, string, ScannedNode>;

const createHaulItem = (node: ScannedNode): HaulItem => ({
  type: HAUL_TYPE,
  key: node.key,
  data: node,
});

export const isHaulItem = (item: Haul.Item): item is HaulItem =>
  item.type === HAUL_TYPE;

export const canDropHaulItem = Haul.canDropOfType<HaulItem>(HAUL_TYPE);

export interface BrowserProps {
  device: Device;
}

const ArrayVariableIcon = Icon.createComposite(Icon.Variable, {
  bottomRight: Icon.Array,
});

const NodeItem = (props: Tree.ItemRenderProps<string>) => {
  const node = List.useItem<string, ScannedNode>(props.itemKey);
  const { getState } = Select.useContext<string>();
  const { getItem } = List.useUtilContext<string, ScannedNode>();
  const { startDrag } = Haul.useDrag({
    type: HAUL_TYPE,
    key: node?.key,
  });
  const handleDragStart = () => {
    if (node == null) return;
    const selected = array.toArray(getState().value);
    if (getItem != null && selected.includes(props.itemKey)) {
      const nodes = getItem(selected);
      startDrag(nodes.map(createHaulItem));
    } else startDrag([createHaulItem(node)]);
  };
  if (node == null) return null;
  const icon = node.isArray ? <ArrayVariableIcon /> : ICONS[node.nodeClass];
  return (
    <Tree.Item {...props} draggable onDragStart={handleDragStart}>
      <Text.Text color={10} gap="small">
        {icon}
        {node.name}
      </Text.Text>
    </Tree.Item>
  );
};

const itemRenderProp = Component.renderProp(NodeItem);

const BROWSE_TIMEOUT = TimeSpan.seconds(10);

const browseNodes = async (
  client: Client,
  rack: rack.Key,
  connection: ConnectionConfig,
  parent?: ScannedNode,
): Promise<ScannedNode[]> => {
  const scanTask = await retrieveScanTask(client, rack);
  const browseStatus = await scanTask.executeCommandSync({
    type: BROWSE_COMMAND_TYPE,
    timeout: BROWSE_TIMEOUT,
    args: { connection, node_id: parent?.nodeId ?? "", field: parent?.field ?? "" },
  });
  if (browseStatus.variant === "error") throw status.toError(browseStatus);
  const { details } = browseStatus;
  if (details?.data == null || !("channels" in details.data)) return [];
  return details.data.channels;
};

export const Browser = ({ device }: BrowserProps) => {
  const [treeNodes, setTreeNodes, treeNodesRef] = useCombinedStateAndRef<Tree.Node[]>(
    [],
  );
  const opcNodesStore = List.useMapData<string, ScannedNode>();
  const [stat, setStat] = useState<status.Status | null>(null);
  const [initialLoading, setInitialLoading] = useState(false);
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const retrieveNodes = (parent?: ScannedNode) => {
    if (client == null) return;
    setStat(status.create({ variant: "loading", message: "Browsing OPC UA nodes" }));
    handleError(async () => {
      let channels: ScannedNode[];
      try {
        channels = await browseNodes(
          client,
          device.rack,
          device.properties.connection,
          parent,
        );
      } catch (e) {
        setStat(status.fromException(e, "Failed to browse OPC UA nodes"));
        return;
      }
      setStat(status.create({ variant: "success", message: "Browsed OPC UA nodes" }));
      // A member's name carries its structure's name, so channels made from members of
      // different structures do not collide.
      const nodes = channels.map((node) => ({
        ...node,
        name:
          parent == null || node.field === ""
            ? node.name
            : `${parent.name}.${node.name}`,
        key: nodeKey(node, parent),
      }));
      opcNodesStore.setItem(nodes);
      const newNodes = nodes.map(({ key }): Tree.Node => ({ key, children: [] }));
      setInitialLoading(false);
      if (parent == null) setTreeNodes(newNodes);
      else
        setTreeNodes([
          ...Tree.setNode({
            tree: treeNodesRef.current,
            destination: parent.key,
            additions: newNodes,
          }),
        ]);
    }, "Failed to browse OPC UA nodes");
  };

  const expand = ({
    clicked,
    action,
  }: optional.Optional<Tree.HandleExpandProps, "clicked">) => {
    if (action === "contract") return;
    retrieveNodes(clicked == null ? undefined : opcNodesStore.getItem(clicked));
  };

  const treeProps = Tree.use({ nodes: treeNodes, onExpand: expand });
  const { shape, clearExpanded } = treeProps;
  const refresh = () => {
    setInitialLoading(true);
    expand({ action: "expand", current: [] });
    clearExpanded();
  };
  const refreshRef = useSyncedRef(refresh);
  useEffect(() => refreshRef.current(), [refreshRef]);
  let content: ReactElement;
  if (stat?.variant === "error") content = <Status.Summary center status={stat} />;
  else if (initialLoading)
    content = (
      <Flex.Box center>
        <Icon.Loading className={CSS.BE("opc-browser", "loading-icon")} color={9} />
      </Flex.Box>
    );
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
            disabled={initialLoading && stat?.variant !== "error"}
            size="medium"
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
