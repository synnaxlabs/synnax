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
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Tooltip } from "@synnaxlabs/lyra/tooltip";
import { Tree } from "@synnaxlabs/lyra/tree";
import { Synnax } from "@synnaxlabs/pluto";
import { array, type optional } from "@synnaxlabs/x";
import { type KeyboardEvent, type ReactElement, useEffect, useState } from "react";

import { Details } from "@/feature/opcua/device/Details";
import { createHaulItem, HAUL_TYPE } from "@/feature/opcua/device/haul";
import { executeScanCommand } from "@/feature/opcua/device/retrieveScanTask";
import { type ConnectionConfig, type Device } from "@/feature/opcua/device/types";
import {
  BROWSE_COMMAND_TYPE,
  browseResponseZ,
  nodeIdentity,
  type ScannedNode,
} from "@/feature/opcua/task/types";
import { CSS } from "@/platform/css";

/** A node in the browser, with the error of its last failed browse. */
interface BrowserNode extends ScannedNode {
  error?: string;
}

const ICONS: Record<string, ReactElement> = {
  VariableType: <Icon.Type />,
  Variable: <Icon.Variable />,
  ObjectType: <Icon.Type />,
  Object: <Icon.Group />,
};

// A node can sit under more than one parent, so its key names the parent too.
const nodeKey = (node: ScannedNode, parent?: ScannedNode): string =>
  `${nodeIdentity(node)}---${parent == null ? "" : nodeIdentity(parent)}`;

export interface BrowserProps {
  device: Device;
}

const ArrayVariableIcon = Icon.createComposite(Icon.Variable, {
  bottomRight: Icon.Array,
});

const NodeItem = (props: Tree.ItemRenderProps<string>) => {
  const node = List.useItem<string, BrowserNode>(props.itemKey);
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
      {node.error != null && (
        <Tooltip.Dialog>
          <Text.Text level="small">{node.error}</Text.Text>
          <Text.Text level="small" status="error" aria-label={node.error}>
            <Icon.Warning />
          </Text.Text>
        </Tooltip.Dialog>
      )}
    </Tree.Item>
  );
};

const itemRenderProp = Component.renderProp(NodeItem);

const SEARCH_ICON = <Icon.Search color={9} />;

const browseNodes = async (
  client: Client,
  rack: rack.Key,
  connection: ConnectionConfig,
  parent?: ScannedNode,
): Promise<ScannedNode[]> => {
  const { channels } = await executeScanCommand(client, rack, {
    type: BROWSE_COMMAND_TYPE,
    args: { connection, node_id: parent?.nodeId ?? "", field: parent?.field ?? "" },
    schema: browseResponseZ,
  });
  return channels;
};

/** @returns the nodes that match, or that hold a node that matches. */
const filterNodes = (
  nodes: Tree.Node[],
  matches: (key: string) => boolean,
): Tree.Node[] =>
  nodes.flatMap((node) => {
    const children = filterNodes(node.children ?? [], matches);
    if (children.length === 0 && !matches(node.key)) return [];
    return [{ ...node, children }];
  });

// A typed value with '=' names a node ID, such as ns=2;s=Tank1 or i=2258.
const isNodeID = (value: string): boolean => value.includes("=");

interface Selection {
  nodeId: string;
  field: string;
}

/** A node ID that the user entered, and the tree selection when they entered it. */
interface Lookup {
  nodeId: string;
  selected?: string;
}

export const Browser = ({ device }: BrowserProps) => {
  const [treeNodes, setTreeNodes, treeNodesRef] = useCombinedStateAndRef<Tree.Node[]>(
    [],
  );
  const opcNodesStore = List.useMapData<string, BrowserNode>();
  const [stat, setStat] = useState<status.Status | null>(null);
  const [initialLoading, setInitialLoading] = useState(false);
  const [query, setQuery] = useState("");
  const [lookup, setLookup] = useState<Lookup | null>(null);
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const retrieveNodes = (parent?: BrowserNode) => {
    if (client == null) return;
    if (parent == null)
      setStat(status.create({ variant: "loading", message: "Browsing OPC UA nodes" }));
    handleError(async () => {
      let channels: ScannedNode[] = [];
      let failure: status.Status | null = null;
      try {
        channels = await browseNodes(
          client,
          device.rack,
          device.properties.connection,
          parent,
        );
      } catch (e) {
        failure = status.fromException(e, "Failed to browse OPC UA nodes");
      }
      // A failed child marks its node and leaves the rest of the tree alone.
      if (parent != null)
        opcNodesStore.setItem({ ...parent, error: failure?.description });
      if (failure != null) {
        if (parent == null) setStat(failure);
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
  const { shape, clearExpanded, selected } = treeProps;
  const selectedKey = selected.at(-1);
  const selectedNode =
    selectedKey == null ? undefined : opcNodesStore.getItem(selectedKey);
  // An entered node ID shows until the user selects another node in the tree.
  let selection: Selection | null = null;
  if (lookup != null && lookup.selected === selectedKey)
    selection = { nodeId: lookup.nodeId, field: "" };
  else if (selectedNode != null)
    selection = { nodeId: selectedNode.nodeId, field: selectedNode.field };
  const refresh = () => {
    setInitialLoading(true);
    expand({ action: "expand", current: [] });
    clearExpanded();
  };
  const search = query.trim().toLowerCase();
  const matches = (key: string): boolean => {
    const node = opcNodesStore.getItem(key);
    return (
      node != null &&
      (node.name.toLowerCase().includes(search) ||
        node.nodeId.toLowerCase().includes(search))
    );
  };
  let visibleShape = shape;
  if (search !== "") {
    const filtered = filterNodes(treeNodes, matches);
    visibleShape = Tree.flatten({
      nodes: filtered,
      expanded: Tree.getDescendants(...filtered).map(({ key }) => key),
    });
  }
  const handleSearchKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key !== "Enter" || !isNodeID(query.trim())) return;
    setLookup({ nodeId: query.trim(), selected: selectedKey });
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
        shape={visibleShape}
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
            tooltip="Browse again"
          >
            <Icon.Refresh />
          </Button.Button>
        </Header.Actions>
      </Header.Header>
      <Input.Text
        size="small"
        variant="text"
        value={query}
        onChange={setQuery}
        onKeyDown={handleSearchKeyDown}
        startContent={SEARCH_ICON}
        placeholder="Search, or enter a node ID"
        className={CSS.BE("opc-browser", "search")}
      />
      {content}
      {selection != null && (
        <Details device={device} nodeId={selection.nodeId} field={selection.field} />
      )}
    </Flex.Box>
  );
};
