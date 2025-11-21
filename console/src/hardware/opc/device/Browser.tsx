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
  type Device as PDevice,
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
  useCombinedStateAndRef,
} from "@synnaxlabs/pluto";
import { type optional, type status } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useEffect, useState } from "react";

import { CSS } from "@/css";
import { retrieveScanTask } from "@/hardware/opc/device/retrieveScanTask";
import { type Device } from "@/hardware/opc/device/types";
import { SCAN_COMMAND_TYPE, type ScannedNode } from "@/hardware/opc/task/types";

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
    <Tree.Item {...props} draggable onDragStart={handleDragStart}>
      <Text.Text color={10} gap="small">
        {icon}
        {node.name}
      </Text.Text>
    </Tree.Item>
  );
});

interface RetrieveNodesQuery {
  clicked: { id: string; key: string | undefined };
  device: Device;
}

const { useRetrieveObservable: useRetrieveNodes } = Flux.createRetrieve<
  RetrieveNodesQuery,
  ScannedNode[],
  PDevice.FluxSubStore
>({
  name: "OPC UA Node",
  retrieve: async ({
    client,
    store,
    query: {
      device: {
        rack,
        properties: { connection },
      },
      clicked: { id },
    },
  }) => {
    const scanTask = await retrieveScanTask(client, store, rack);
    const { details, variant, message } = await scanTask.executeCommandSync({
      type: SCAN_COMMAND_TYPE,
      timeout: TimeSpan.seconds(10),
      args: { connection, node_id: id },
    });
    if (variant !== "success") throw new Error(message);
    if (details?.data == null || !("channels" in details.data)) return [];
    return details.data.channels;
  },
});

export const Browser = ({ device }: BrowserProps) => {
  const [treeNodes, setTreeNodes, treeNodesRef] = useCombinedStateAndRef<Tree.Node[]>(
    [],
  );
  const opcNodesStore = List.useMapData<string, ScannedNode>();
  const [status, setStatus] = useState<status.Status | null>(null);
  const { retrieve: retrieveNodes } = useRetrieveNodes({
    onChange: useCallback((result, { clicked: { id, key } }) => {
      setStatus(result.status);
      if (result.variant !== "success") return;
      const isRoot = id === "";
      const { data: channels } = result;
      const newNodes = channels.map(
        (node): Tree.Node => ({ key: nodeKey(node.nodeId, id), children: [] }),
      );
      opcNodesStore.setItem(
        channels.map((node) => ({ ...node, key: nodeKey(node.nodeId, id) })),
      );
      setInitialLoading(false);
      if (isRoot) setTreeNodes(newNodes);
      else
        setTreeNodes([
          ...Tree.setNode({
            tree: treeNodesRef.current,
            destination: key ?? null,
            additions: newNodes,
          }),
        ]);
    }, []),
  });

  const expand = useCallback(
    ({ clicked, action }: optional.Optional<Tree.HandleExpandProps, "clicked">) => {
      if (action === "contract") return;
      retrieveNodes({
        clicked: { key: clicked, id: clicked == null ? "" : parseNodeID(clicked) },
        device,
      });
    },
    [retrieveNodes, device],
  );

  const treeProps = Tree.use({ nodes: treeNodes, onExpand: expand });
  const { shape, clearExpanded } = treeProps;
  const [initialLoading, setInitialLoading] = useState(false);
  const refresh = useCallback(() => {
    setInitialLoading(true);
    expand({ action: "expand", current: [] });
    clearExpanded();
  }, [clearExpanded]);
  useEffect(refresh, [refresh]);
  let content: ReactElement;
  if (status?.variant === "error") content = <Status.Summary center status={status} />;
  else if (initialLoading)
    content = (
      <Flex.Box center>
        <Icon.Loading style={{ fontSize: "5rem" }} color={7} />
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
            disabled={initialLoading && status?.variant !== "error"}
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
